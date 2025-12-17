package gitutil

import (
	"fmt"
	"strings"
	"time"

	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// PRStrategy defines when to create pull requests
type PRStrategy int

const (
	// PRStrategyNone means no PRs (disabled)
	PRStrategyNone PRStrategy = iota
	// PRStrategyPerViolation creates one PR per violation
	PRStrategyPerViolation
	// PRStrategyPerIncident creates one PR per incident
	PRStrategyPerIncident
	// PRStrategyPerPhase creates one PR per phase
	PRStrategyPerPhase
	// PRStrategyAtEnd creates one PR with all fixes
	PRStrategyAtEnd
)

// ParsePRStrategy parses a commit strategy string into a PRStrategy
func ParsePRStrategy(s string) (PRStrategy, error) {
	switch s {
	case "per-violation":
		return PRStrategyPerViolation, nil
	case "per-incident":
		return PRStrategyPerIncident, nil
	case "per-phase":
		return PRStrategyPerPhase, nil
	case "at-end":
		return PRStrategyAtEnd, nil
	default:
		return PRStrategyNone, fmt.Errorf("invalid PR strategy: %s", s)
	}
}

// PRConfig holds PR creation configuration
type PRConfig struct {
	Strategy         PRStrategy
	BranchPrefix     string  // Base name for branches
	BaseBranch       string  // Target branch (empty = auto-detect)
	GitHubToken      string
	DryRun           bool    // If true, show what would be done without actually doing it
	CommentThreshold float64 // Add inline comments for fixes with confidence below this (0.0-1.0, 0 = disabled)
}

// PendingPR represents a PR that needs to be created
type PendingPR struct {
	ViolationID string
	PhaseID     string // Phase ID for per-phase strategy
	BranchName  string
	Fixes       []FixRecord
}

// CreatedPR represents a successfully created PR
type CreatedPR struct {
	Number      int
	URL         string
	BranchName  string
	ViolationID string
	PhaseID     string // Phase ID for per-phase strategy
}

// GitHubClientInterface defines the methods needed from GitHubClient for PR operations
type GitHubClientInterface interface {
	CreatePullRequest(req PullRequestRequest) (*PullRequestResponse, error)
	GetDefaultBranch() (string, error)
	CreateCommitStatus(sha string, req CommitStatusRequest) (*CommitStatusResponse, error)
	CreateReviewComment(prNumber int, req ReviewCommentRequest) (*ReviewCommentResponse, error)
}

// PRTracker manages PR creation aligned with commit strategy
type PRTracker struct {
	config         PRConfig
	workingDir     string
	providerName   string
	githubClient   GitHubClientInterface
	originalBranch string
	progress       ProgressWriter

	// Track fixes for PR creation
	fixesByViolation map[string][]FixRecord
	fixesByPhase     map[string][]FixRecord // For per-phase strategy
	allFixes         []FixRecord

	// Track created PRs
	createdPRs []CreatedPR
}

// NewPRTracker creates a new PR tracker for managing GitHub pull request creation.
//
// The PR tracker coordinates with git commits to create pull requests on GitHub
// based on the configured strategy (per-violation, per-incident, or at-end).
//
// Parameters:
//   - config: PR configuration including strategy, branch naming, and GitHub token
//   - workingDir: Path to the git repository working directory
//   - providerName: Name of the AI provider used for fixes (for PR body metadata)
//   - progress: Optional progress reporter (use nil for no progress output)
//
// Returns:
//   - A configured PRTracker ready to track fixes and create PRs
//   - An error if the GitHub token is missing or GitHub client creation fails
//
// Example:
//
//	config := gitutil.PRConfig{
//	    Strategy:     gitutil.PRStrategyPerViolation,
//	    BranchPrefix: "kantra-ai/remediation",
//	    GitHubToken:  os.Getenv("GITHUB_TOKEN"),
//	}
//	progress := &gitutil.StdoutProgressWriter{}
//	tracker, err := gitutil.NewPRTracker(config, "/path/to/repo", "claude", progress)
func NewPRTracker(config PRConfig, workingDir string, providerName string, progress ProgressWriter) (*PRTracker, error) {
	var githubClient *GitHubClient
	var currentBranch string
	var err error

	// Skip GitHub client creation in dry-run mode
	if !config.DryRun {
		// Validate config
		if config.GitHubToken == "" {
			return nil, fmt.Errorf("GitHub token is required")
		}

		// Create GitHub client
		githubClient, err = NewGitHubClient(workingDir, config.GitHubToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client: %w", err)
		}

		// Get current branch to restore later
		currentBranch, err = GetCurrentBranch(workingDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Use NoOp progress writer if none provided
	if progress == nil {
		progress = &NoOpProgressWriter{}
	}

	return &PRTracker{
		config:           config,
		workingDir:       workingDir,
		providerName:     providerName,
		githubClient:     githubClient,
		originalBranch:   currentBranch,
		progress:         progress,
		fixesByViolation: make(map[string][]FixRecord),
		fixesByPhase:     make(map[string][]FixRecord),
		allFixes:         make([]FixRecord, 0),
		createdPRs:       make([]CreatedPR, 0),
	}, nil
}

// TrackForPR records that a fix should be included in a pull request.
//
// Call this method after each successful fix to track it for PR creation.
// Fixes are organized by violation ID for per-violation PRs and also
// stored in order for per-incident and at-end strategies.
//
// Parameters:
//   - v: The violation that was fixed
//   - incident: The specific incident that was fixed
//   - result: The fix result containing file path, cost, and tokens used
//
// Returns nil (currently never returns an error, but signature allows for future validation)
func (pt *PRTracker) TrackForPR(v violation.Violation, incident violation.Incident, result *fixer.FixResult) error {
	return pt.TrackForPRWithPhase(v, incident, result, "")
}

// TrackForPRWithPhase records that a fix should be included in a pull request with phase information.
//
// This method extends TrackForPR to support per-phase PR strategies by tracking which phase
// the fix belongs to. Use this when you need per-phase PRs.
//
// Parameters:
//   - v: The violation that was fixed
//   - incident: The specific incident that was fixed
//   - result: The fix result containing file path, cost, and tokens used
//   - phaseID: The ID of the phase this fix belongs to (for per-phase strategy)
//
// Returns nil (currently never returns an error, but signature allows for future validation)
func (pt *PRTracker) TrackForPRWithPhase(v violation.Violation, incident violation.Incident, result *fixer.FixResult, phaseID string) error {
	record := FixRecord{
		Violation: v,
		Incident:  incident,
		Result:    result,
		Timestamp: time.Now(),
		PhaseID:   phaseID,
	}

	// Track by violation
	violationID := v.ID
	pt.fixesByViolation[violationID] = append(pt.fixesByViolation[violationID], record)

	// Track by phase (if phase ID provided)
	if phaseID != "" {
		pt.fixesByPhase[phaseID] = append(pt.fixesByPhase[phaseID], record)
	}

	// Track all fixes
	pt.allFixes = append(pt.allFixes, record)

	return nil
}

// Finalize creates pull requests on GitHub based on the configured strategy.
//
// This method should be called after all fixes have been tracked via TrackForPR.
// It will create branches, push them to GitHub, and create pull requests according
// to the strategy:
//   - PRStrategyPerViolation: One PR per violation type (groups all incidents)
//   - PRStrategyPerIncident: One PR per individual incident/file
//   - PRStrategyAtEnd: Single PR with all fixes combined
//
// Progress is reported via the ProgressWriter, and created PRs can be retrieved
// afterwards using GetCreatedPRs().
//
// In dry-run mode, this method will print what would be done without actually
// creating branches, pushing to GitHub, or creating pull requests.
//
// Returns an error if branch creation, pushing, or PR creation fails. The error
// will include helpful messages for common failure scenarios.
func (pt *PRTracker) Finalize() error {
	// Determine base branch (target for PR)
	baseBranch := pt.config.BaseBranch
	if baseBranch == "" {
		if !pt.config.DryRun && pt.githubClient != nil {
			// Try to get from GitHub API
			branch, err := pt.githubClient.GetDefaultBranch()
			if err != nil {
				// Fallback to local detection
				branch, err = GetDefaultBranch(pt.workingDir)
				if err != nil {
					// Final fallback
					baseBranch = "main"
				} else {
					baseBranch = branch
				}
			} else {
				baseBranch = branch
			}
		} else {
			// In dry-run mode or if GitHub client is nil, use local detection
			branch, err := GetDefaultBranch(pt.workingDir)
			if err != nil {
				baseBranch = "main" // Final fallback
			} else {
				baseBranch = branch
			}
		}
	}

	if pt.config.DryRun {
		pt.progress.Printf("\n=== DRY RUN MODE: Pull Request Preview ===\n")
		pt.progress.Printf("Base branch: %s\n", baseBranch)
	}

	// Create PRs based on strategy
	switch pt.config.Strategy {
	case PRStrategyPerViolation:
		return pt.createPRsPerViolation(baseBranch)
	case PRStrategyPerIncident:
		return pt.createPRsPerIncident(baseBranch)
	case PRStrategyPerPhase:
		return pt.createPRsPerPhase(baseBranch)
	case PRStrategyAtEnd:
		return pt.createPRAtEnd(baseBranch)
	default:
		return fmt.Errorf("unsupported PR strategy: %d", pt.config.Strategy)
	}
}

// createPRsPerViolation creates one PR for each violation
func (pt *PRTracker) createPRsPerViolation(baseBranch string) error {
	timestamp := time.Now().Unix()

	prCount := len(pt.fixesByViolation)
	currentPR := 0

	for violationID, fixes := range pt.fixesByViolation {
		if len(fixes) == 0 {
			continue
		}

		currentPR++
		pt.progress.Printf("\n[%d/%d] Creating PR for violation: %s\n", currentPR, prCount, violationID)

		// Generate branch name
		branchName := fmt.Sprintf("%s-%s-%d", pt.config.BranchPrefix, violationID, timestamp)

		// Create and push branch
		if err := pt.createAndPushBranch(branchName); err != nil {
			return fmt.Errorf("failed to create branch for violation %s: %w", violationID, err)
		}

		// Create PR
		violation := fixes[0].Violation
		title := FormatPRTitleForViolation(violationID, violation.Description)
		body := FormatPRBodyForViolation(
			violationID,
			violation.Description,
			violation.Category,
			violation.Effort,
			fixes,
			pt.providerName,
		)

		pr, err := pt.createPR(title, body, branchName, baseBranch)
		if err != nil {
			return fmt.Errorf("failed to create PR for violation %s: %w", violationID, err)
		}

		// Add inline comments for low-confidence fixes
		if err := pt.addLowConfidenceComments(pr.Number, fixes); err != nil {
			pt.progress.Printf("  Warning: failed to add low-confidence comments: %v\n", err)
		}

		// Track created PR
		pt.createdPRs = append(pt.createdPRs, CreatedPR{
			Number:      pr.Number,
			URL:         pr.HTMLURL,
			BranchName:  branchName,
			ViolationID: violationID,
		})

		// Return to original branch for next PR (skip in dry-run)
		if !pt.config.DryRun {
			if err := CheckoutBranch(pt.workingDir, pt.originalBranch); err != nil {
				return fmt.Errorf("failed to return to original branch: %w", err)
			}
		}
	}

	return nil
}

// createPRsPerIncident creates one PR for each incident
func (pt *PRTracker) createPRsPerIncident(baseBranch string) error {
	timestamp := time.Now().Unix()

	for i, fix := range pt.allFixes {
		// Generate branch name
		branchName := fmt.Sprintf("%s-%s-%d-%d",
			pt.config.BranchPrefix,
			fix.Violation.ID,
			timestamp,
			i)

		// Create and push branch
		if err := pt.createAndPushBranch(branchName); err != nil {
			return fmt.Errorf("failed to create branch for incident %d: %w", i, err)
		}

		// Create PR
		title := FormatPRTitleForIncident(
			fix.Violation.ID,
			fix.Violation.Description,
			fix.Result.FilePath,
		)
		body := FormatPRBodyForIncident(
			fix.Violation.ID,
			fix.Violation.Description,
			fix.Result.FilePath,
			fix.Incident.LineNumber,
			fix.Result.Cost,
			fix.Result.TokensUsed,
			pt.providerName,
		)

		pr, err := pt.createPR(title, body, branchName, baseBranch)
		if err != nil {
			return fmt.Errorf("failed to create PR for incident %d: %w", i, err)
		}

		// Add inline comments for low-confidence fixes
		if err := pt.addLowConfidenceComments(pr.Number, []FixRecord{fix}); err != nil {
			pt.progress.Printf("  Warning: failed to add low-confidence comments: %v\n", err)
		}

		// Track created PR
		pt.createdPRs = append(pt.createdPRs, CreatedPR{
			Number:      pr.Number,
			URL:         pr.HTMLURL,
			BranchName:  branchName,
			ViolationID: fix.Violation.ID,
		})

		// Return to original branch for next PR (skip in dry-run)
		if !pt.config.DryRun {
			if err := CheckoutBranch(pt.workingDir, pt.originalBranch); err != nil {
				return fmt.Errorf("failed to return to original branch: %w", err)
			}
		}
	}

	return nil
}

// createPRsPerPhase creates one PR for each phase
func (pt *PRTracker) createPRsPerPhase(baseBranch string) error {
	timestamp := time.Now().Unix()

	prCount := len(pt.fixesByPhase)
	currentPR := 0

	for phaseID, fixes := range pt.fixesByPhase {
		if len(fixes) == 0 {
			continue
		}

		currentPR++
		pt.progress.Printf("\n[%d/%d] Creating PR for phase: %s\n", currentPR, prCount, phaseID)

		// Generate branch name
		branchName := fmt.Sprintf("%s-%s-%d", pt.config.BranchPrefix, phaseID, timestamp)

		// Create and push branch
		if err := pt.createAndPushBranch(branchName); err != nil {
			return fmt.Errorf("failed to create branch for phase %s: %w", phaseID, err)
		}

		// Group fixes by violation for the PR body
		fixesByViolation := make(map[string][]FixRecord)
		for _, fix := range fixes {
			violationID := fix.Violation.ID
			fixesByViolation[violationID] = append(fixesByViolation[violationID], fix)
		}

		// Create PR
		title := FormatPRTitleForPhase(phaseID, len(fixesByViolation))
		body := FormatPRBodyForPhase(phaseID, fixesByViolation, pt.providerName)

		pr, err := pt.createPR(title, body, branchName, baseBranch)
		if err != nil {
			return fmt.Errorf("failed to create PR for phase %s: %w", phaseID, err)
		}

		// Add inline comments for low-confidence fixes
		if err := pt.addLowConfidenceComments(pr.Number, fixes); err != nil {
			pt.progress.Printf("  Warning: failed to add low-confidence comments: %v\n", err)
		}

		// Track created PR
		pt.createdPRs = append(pt.createdPRs, CreatedPR{
			Number:     pr.Number,
			URL:        pr.HTMLURL,
			BranchName: branchName,
			PhaseID:    phaseID,
		})

		// Return to original branch for next PR (skip in dry-run)
		if !pt.config.DryRun {
			if err := CheckoutBranch(pt.workingDir, pt.originalBranch); err != nil {
				return fmt.Errorf("failed to return to original branch: %w", err)
			}
		}
	}

	return nil
}

// createPRAtEnd creates a single PR with all fixes
func (pt *PRTracker) createPRAtEnd(baseBranch string) error {
	if len(pt.allFixes) == 0 {
		return nil // No fixes to create PR for
	}

	timestamp := time.Now().Unix()
	branchName := fmt.Sprintf("%s-%d", pt.config.BranchPrefix, timestamp)

	// Create and push branch
	if err := pt.createAndPushBranch(branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Create PR
	title := FormatPRTitleAtEnd(len(pt.fixesByViolation))
	body := FormatPRBodyAtEnd(pt.fixesByViolation, pt.providerName)

	pr, err := pt.createPR(title, body, branchName, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// Add inline comments for low-confidence fixes
	if err := pt.addLowConfidenceComments(pr.Number, pt.allFixes); err != nil {
		pt.progress.Printf("  Warning: failed to add low-confidence comments: %v\n", err)
	}

	// Track created PR
	pt.createdPRs = append(pt.createdPRs, CreatedPR{
		Number:     pr.Number,
		URL:        pr.HTMLURL,
		BranchName: branchName,
	})

	// Return to original branch (skip in dry-run)
	if !pt.config.DryRun {
		if err := CheckoutBranch(pt.workingDir, pt.originalBranch); err != nil {
			return fmt.Errorf("failed to return to original branch: %w", err)
		}
	}

	return nil
}

// createAndPushBranch creates a new branch from current HEAD and pushes it to the remote.
// Reports progress and provides helpful error messages for common failure scenarios.
//
// In dry-run mode, this method prints what would be done without actually creating
// or pushing the branch.
//
// Common errors and their causes:
//   - Branch already exists: Suggests deletion command
//   - SSH key not configured: Suggests HTTPS remote or SSH setup
//   - No write access (403): Suggests checking token scope
//   - Network errors: Suggests checking internet connection
func (pt *PRTracker) createAndPushBranch(branchName string) error {
	if pt.config.DryRun {
		pt.progress.Printf("  [DRY RUN] Would create branch: %s\n", branchName)
		pt.progress.Printf("  [DRY RUN] Would push to remote\n")
		return nil
	}

	// Create branch
	pt.progress.Printf("  Creating branch: %s\n", branchName)
	if err := CreateBranch(pt.workingDir, branchName); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("branch '%s' already exists - delete it first with: git branch -D %s", branchName, branchName)
		}
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Push branch
	pt.progress.Printf("  Pushing to remote...\n")
	if err := PushBranch(pt.workingDir, branchName); err != nil {
		// Provide helpful error messages for common push failures
		errStr := err.Error()
		if strings.Contains(errStr, "Permission denied") || strings.Contains(errStr, "publickey") {
			return fmt.Errorf("push failed: SSH key not configured\n"+
				"  Either:\n"+
				"  1. Use HTTPS remote: git remote set-url origin https://github.com/OWNER/REPO.git\n"+
				"  2. Or setup SSH key: https://docs.github.com/en/authentication/connecting-to-github-with-ssh")
		}
		if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
			return fmt.Errorf("push failed: No write access to repository\n"+
				"  Check that your GITHUB_TOKEN has 'repo' scope")
		}
		if strings.Contains(errStr, "Could not resolve host") || strings.Contains(errStr, "network") {
			return fmt.Errorf("push failed: Network error\n"+
				"  Check your internet connection")
		}
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// createPR creates a pull request on GitHub via the GitHub API.
// Reports progress and provides helpful error messages for common API errors.
//
// In dry-run mode, this method prints the PR details without actually creating it.
//
// Parameters:
//   - title: PR title
//   - body: PR body (markdown formatted)
//   - head: Source branch name
//   - base: Target branch name (usually "main" or "master")
//
// Common errors and their causes:
//   - No commits (422): Branch has no commits vs base branch
//   - PR already exists (422): A PR already exists for this head branch
//   - Other GitHub API errors: Authentication, permissions, validation failures
func (pt *PRTracker) createPR(title, body, head, base string) (*PullRequestResponse, error) {
	if pt.config.DryRun {
		pt.progress.Printf("  [DRY RUN] Would create pull request:\n")
		pt.progress.Printf("    Title: %s\n", title)
		pt.progress.Printf("    Base: %s <- Head: %s\n", base, head)
		pt.progress.Printf("    Body preview (first 200 chars):\n")
		bodyPreview := body
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		pt.progress.Printf("    %s\n", strings.ReplaceAll(bodyPreview, "\n", "\n    "))

		// Return mock response for dry-run
		return &PullRequestResponse{
			Number:  0,
			HTMLURL: "[DRY RUN - PR would be created here]",
		}, nil
	}

	pt.progress.Printf("  Creating pull request...\n")

	req := PullRequestRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}

	pr, err := pt.githubClient.CreatePullRequest(req)
	if err != nil {
		// Provide better error messages for common GitHub API errors
		if ghErr, ok := err.(*GitHubError); ok {
			switch ghErr.StatusCode {
			case 422:
				if strings.Contains(ghErr.Message, "No commits") {
					return nil, fmt.Errorf("no commits to create PR from\n"+
						"  This usually means:\n"+
						"  1. The fixes were already committed to the base branch, or\n"+
						"  2. Git commits failed earlier in the process")
				}
				if strings.Contains(ghErr.Message, "already exists") {
					return nil, fmt.Errorf("a pull request already exists for branch '%s'\n"+
						"  Either close the existing PR or use a different branch name with --branch", head)
				}
			}
		}
		return nil, err
	}

	return pr, nil
}

// addLowConfidenceComments adds inline comments to PR for fixes with low confidence
func (pt *PRTracker) addLowConfidenceComments(prNumber int, fixes []FixRecord) error {
	// Skip if commenting is disabled or in dry-run mode
	if pt.config.CommentThreshold == 0 || pt.config.DryRun {
		return nil
	}

	// Get current commit SHA from the branch HEAD
	commitSHA, err := GetCurrentCommitSHA(pt.workingDir)
	if err != nil {
		return fmt.Errorf("failed to get commit SHA: %w", err)
	}

	// Filter for low-confidence fixes
	lowConfidenceFixes := []FixRecord{}
	for _, fix := range fixes {
		if fix.Result != nil && fix.Result.Confidence > 0 && fix.Result.Confidence < pt.config.CommentThreshold {
			lowConfidenceFixes = append(lowConfidenceFixes, fix)
		}
	}

	if len(lowConfidenceFixes) == 0 {
		return nil
	}

	pt.progress.Printf("  Adding %d inline comment(s) for low-confidence fixes...\n", len(lowConfidenceFixes))

	// Create comments for each low-confidence fix
	successCount := 0
	for _, fix := range lowConfidenceFixes {
		// Format confidence percentage
		confidencePct := int(fix.Result.Confidence * 100)

		// Create comment body
		commentBody := fmt.Sprintf("⚠️ **Low Confidence Fix (%d%%)**\n\n"+
			"This fix was generated with lower confidence than usual. Please review carefully to ensure:\n"+
			"- The fix correctly addresses the violation\n"+
			"- No unintended side effects are introduced\n"+
			"- The logic remains semantically correct\n\n"+
			"**Violation:** %s\n"+
			"**Description:** %s",
			confidencePct,
			fix.Violation.ID,
			fix.Violation.Description,
		)

		// Create review comment
		req := ReviewCommentRequest{
			Body:     commentBody,
			CommitID: commitSHA,
			Path:     fix.Result.FilePath,
			Line:     fix.Incident.LineNumber,
			Side:     "RIGHT", // Comment on the new version (after changes)
		}

		_, err := pt.githubClient.CreateReviewComment(prNumber, req)
		if err != nil {
			// Log warning but don't fail the PR creation
			pt.progress.Printf("    Warning: failed to add comment for %s:%d: %v\n",
				fix.Result.FilePath, fix.Incident.LineNumber, err)
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		pt.progress.Printf("  Added %d/%d inline comments successfully\n", successCount, len(lowConfidenceFixes))
	}

	return nil
}

// GetCreatedPRs returns the list of created PRs
func (pt *PRTracker) GetCreatedPRs() []CreatedPR {
	return pt.createdPRs
}
