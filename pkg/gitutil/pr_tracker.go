package gitutil

import (
	"fmt"
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
	case "at-end":
		return PRStrategyAtEnd, nil
	default:
		return PRStrategyNone, fmt.Errorf("invalid PR strategy: %s", s)
	}
}

// PRConfig holds PR creation configuration
type PRConfig struct {
	Strategy     PRStrategy
	BranchPrefix string // Base name for branches
	BaseBranch   string // Target branch (empty = auto-detect)
	GitHubToken  string
}

// PendingPR represents a PR that needs to be created
type PendingPR struct {
	ViolationID string
	BranchName  string
	Fixes       []FixRecord
}

// CreatedPR represents a successfully created PR
type CreatedPR struct {
	Number      int
	URL         string
	BranchName  string
	ViolationID string
}

// PRTracker manages PR creation aligned with commit strategy
type PRTracker struct {
	config        PRConfig
	workingDir    string
	providerName  string
	githubClient  *GitHubClient
	originalBranch string

	// Track fixes for PR creation
	fixesByViolation map[string][]FixRecord
	allFixes         []FixRecord

	// Track created PRs
	createdPRs []CreatedPR
}

// NewPRTracker creates a new PR tracker
func NewPRTracker(config PRConfig, workingDir string, providerName string) (*PRTracker, error) {
	// Validate config
	if config.GitHubToken == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// Create GitHub client
	githubClient, err := NewGitHubClient(workingDir, config.GitHubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Get current branch to restore later
	currentBranch, err := GetCurrentBranch(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	return &PRTracker{
		config:           config,
		workingDir:       workingDir,
		providerName:     providerName,
		githubClient:     githubClient,
		originalBranch:   currentBranch,
		fixesByViolation: make(map[string][]FixRecord),
		allFixes:         make([]FixRecord, 0),
		createdPRs:       make([]CreatedPR, 0),
	}, nil
}

// TrackForPR records that a fix should be included in a PR
func (pt *PRTracker) TrackForPR(v violation.Violation, incident violation.Incident, result *fixer.FixResult) error {
	record := FixRecord{
		Violation: v,
		Incident:  incident,
		Result:    result,
		Timestamp: time.Now(),
	}

	// Track by violation
	violationID := v.ID
	pt.fixesByViolation[violationID] = append(pt.fixesByViolation[violationID], record)

	// Track all fixes
	pt.allFixes = append(pt.allFixes, record)

	return nil
}

// Finalize creates all pending PRs based on strategy
func (pt *PRTracker) Finalize() error {
	// Determine base branch (target for PR)
	baseBranch := pt.config.BaseBranch
	if baseBranch == "" {
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
	}

	// Create PRs based on strategy
	switch pt.config.Strategy {
	case PRStrategyPerViolation:
		return pt.createPRsPerViolation(baseBranch)
	case PRStrategyPerIncident:
		return pt.createPRsPerIncident(baseBranch)
	case PRStrategyAtEnd:
		return pt.createPRAtEnd(baseBranch)
	default:
		return fmt.Errorf("unsupported PR strategy: %d", pt.config.Strategy)
	}
}

// createPRsPerViolation creates one PR for each violation
func (pt *PRTracker) createPRsPerViolation(baseBranch string) error {
	timestamp := time.Now().Unix()

	for violationID, fixes := range pt.fixesByViolation {
		if len(fixes) == 0 {
			continue
		}

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

		// Track created PR
		pt.createdPRs = append(pt.createdPRs, CreatedPR{
			Number:      pr.Number,
			URL:         pr.HTMLURL,
			BranchName:  branchName,
			ViolationID: violationID,
		})

		// Return to original branch for next PR
		if err := CheckoutBranch(pt.workingDir, pt.originalBranch); err != nil {
			return fmt.Errorf("failed to return to original branch: %w", err)
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

		// Track created PR
		pt.createdPRs = append(pt.createdPRs, CreatedPR{
			Number:      pr.Number,
			URL:         pr.HTMLURL,
			BranchName:  branchName,
			ViolationID: fix.Violation.ID,
		})

		// Return to original branch for next PR
		if err := CheckoutBranch(pt.workingDir, pt.originalBranch); err != nil {
			return fmt.Errorf("failed to return to original branch: %w", err)
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

	// Track created PR
	pt.createdPRs = append(pt.createdPRs, CreatedPR{
		Number:     pr.Number,
		URL:        pr.HTMLURL,
		BranchName: branchName,
	})

	// Return to original branch
	if err := CheckoutBranch(pt.workingDir, pt.originalBranch); err != nil {
		return fmt.Errorf("failed to return to original branch: %w", err)
	}

	return nil
}

// createAndPushBranch creates a new branch from current HEAD and pushes it
func (pt *PRTracker) createAndPushBranch(branchName string) error {
	// Create branch
	if err := CreateBranch(pt.workingDir, branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Push branch
	if err := PushBranch(pt.workingDir, branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// createPR creates a pull request on GitHub
func (pt *PRTracker) createPR(title, body, head, base string) (*PullRequestResponse, error) {
	req := PullRequestRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}

	return pt.githubClient.CreatePullRequest(req)
}

// GetCreatedPRs returns the list of created PRs
func (pt *PRTracker) GetCreatedPRs() []CreatedPR {
	return pt.createdPRs
}
