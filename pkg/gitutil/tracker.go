// Package gitutil provides Git integration for kantra-ai, including commit tracking,
// pull request creation, and verification workflows. It supports multiple commit strategies
// (per-incident, per-violation, at-end) and integrates with GitHub's API for automated PR creation.
package gitutil

import (
	"fmt"
	"time"

	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// CommitStrategy defines when to create git commits
type CommitStrategy int

const (
	// StrategyNone means no commits (disabled)
	StrategyNone CommitStrategy = iota
	// StrategyPerViolation creates one commit per violation type
	StrategyPerViolation
	// StrategyPerIncident creates one commit per incident
	StrategyPerIncident
	// StrategyAtEnd creates one commit at the end with all fixes
	StrategyAtEnd
)

// ParseStrategy parses a strategy string into a CommitStrategy
func ParseStrategy(s string) (CommitStrategy, error) {
	switch s {
	case "per-violation":
		return StrategyPerViolation, nil
	case "per-incident":
		return StrategyPerIncident, nil
	case "at-end":
		return StrategyAtEnd, nil
	default:
		return StrategyNone, fmt.Errorf("invalid commit strategy: %s (must be one of: per-violation, per-incident, at-end)", s)
	}
}

// FixRecord represents a single successful fix
type FixRecord struct {
	Violation violation.Violation
	Incident  violation.Incident
	Result    fixer.FixResult // Store value, not pointer, to avoid aliasing bugs
	Timestamp time.Time
	PhaseID   string // Optional: Phase ID for per-phase PR strategy
}

// CommitInfo represents information about a created commit
type CommitInfo struct {
	SHA         string    // Commit SHA
	Message     string    // Commit message
	ViolationID string    // Violation ID (for per-violation commits)
	PhaseID     string    // Phase ID (for per-phase commits)
	FileCount   int       // Number of files changed in this commit
	Timestamp   time.Time // When the commit was created
}

// PRInfo represents information about a created pull request
type PRInfo struct {
	Number      int       // PR number
	URL         string    // PR URL
	Title       string    // PR title
	BranchName  string    // Branch name
	ViolationID string    // Violation ID (for per-violation PRs)
	PhaseID     string    // Phase ID (for per-phase PRs)
	CommitSHAs  []string  // List of commit SHAs included in this PR
	Timestamp   time.Time // When the PR was created
}

// CommitTracker tracks successful fixes and creates git commits based on strategy
type CommitTracker struct {
	strategy         CommitStrategy
	workingDir       string
	providerName     string
	fixesByViolation map[string][]FixRecord
	allFixes         []FixRecord
	lastViolationID  string
	commits          []CommitInfo // Track all created commits
}

// NewCommitTracker creates a new CommitTracker
func NewCommitTracker(strategy CommitStrategy, workingDir string, providerName string) *CommitTracker {
	return &CommitTracker{
		strategy:         strategy,
		workingDir:       workingDir,
		providerName:     providerName,
		fixesByViolation: make(map[string][]FixRecord),
		allFixes:         make([]FixRecord, 0),
		lastViolationID:  "",
		commits:          make([]CommitInfo, 0),
	}
}

// TrackFix records a successful fix and potentially creates a commit
func (ct *CommitTracker) TrackFix(v violation.Violation, incident violation.Incident, result *fixer.FixResult) error {
	record := FixRecord{
		Violation: v,
		Incident:  incident,
		Result:    *result, // Dereference pointer to store value copy
		Timestamp: time.Now(),
	}

	switch ct.strategy {
	case StrategyPerViolation:
		return ct.trackForPerViolation(record)
	case StrategyPerIncident:
		return ct.commitPerIncident(record)
	case StrategyAtEnd:
		return ct.trackForAtEnd(record)
	default:
		return nil
	}
}

// trackForPerViolation accumulates fixes and commits when violation changes
func (ct *CommitTracker) trackForPerViolation(record FixRecord) error {
	violationID := record.Violation.ID

	// If this is a new violation and we have pending fixes for the previous one, commit them
	if ct.lastViolationID != "" && ct.lastViolationID != violationID {
		if err := ct.commitViolation(ct.lastViolationID); err != nil {
			return err
		}
	}

	// Add this fix to the violation's list
	ct.fixesByViolation[violationID] = append(ct.fixesByViolation[violationID], record)
	ct.lastViolationID = violationID

	return nil
}

// trackForAtEnd accumulates all fixes for final commit
func (ct *CommitTracker) trackForAtEnd(record FixRecord) error {
	violationID := record.Violation.ID
	ct.fixesByViolation[violationID] = append(ct.fixesByViolation[violationID], record)
	ct.allFixes = append(ct.allFixes, record)
	return nil
}

// commitPerIncident immediately commits a single fix
func (ct *CommitTracker) commitPerIncident(record FixRecord) error {
	// Stage the file
	if err := StageFile(ct.workingDir, record.Result.FilePath); err != nil {
		return fmt.Errorf("failed to stage file for per-incident commit: %w", err)
	}

	// Check if there are actually any staged changes
	hasChanges, err := HasStagedChanges(ct.workingDir)
	if err != nil {
		return fmt.Errorf("failed to check for staged changes: %w", err)
	}

	if !hasChanges {
		// No changes to commit (file was already committed)
		fmt.Printf("  ⏭️  Skipping commit for %s (no changes)\n", record.Result.FilePath)
		return nil
	}

	// Create commit message
	message := FormatPerIncidentMessage(
		record.Violation.ID,
		record.Violation.Description,
		record.Result.FilePath,
		record.Incident.LineNumber,
		record.Result.Cost,
		record.Result.TokensUsed,
		ct.providerName,
	)

	// Create commit
	sha, err := CreateCommit(ct.workingDir, message)
	if err != nil {
		return fmt.Errorf("failed to create per-incident commit: %w", err)
	}

	// Track commit info
	ct.commits = append(ct.commits, CommitInfo{
		SHA:         sha,
		Message:     message,
		ViolationID: record.Violation.ID,
		PhaseID:     record.PhaseID,
		FileCount:   1,
		Timestamp:   time.Now(),
	})

	fmt.Printf("  📝 Created commit for %s\n", record.Result.FilePath)
	return nil
}

// commitViolation commits all fixes for a specific violation
func (ct *CommitTracker) commitViolation(violationID string) error {
	fixes, exists := ct.fixesByViolation[violationID]
	if !exists || len(fixes) == 0 {
		return nil
	}

	// Stage all files for this violation
	for _, fix := range fixes {
		if err := StageFile(ct.workingDir, fix.Result.FilePath); err != nil {
			return fmt.Errorf("failed to stage file for violation commit: %w", err)
		}
	}

	// Check if there are actually any staged changes
	hasChanges, err := HasStagedChanges(ct.workingDir)
	if err != nil {
		return fmt.Errorf("failed to check for staged changes: %w", err)
	}

	if !hasChanges {
		// No changes to commit (file was already committed by previous violation)
		fmt.Printf("⏭️  Skipping commit for violation %s (no changes to commit)\n", violationID)
		// Clear the fixes for this violation
		delete(ct.fixesByViolation, violationID)
		return nil
	}

	// Create commit message
	message := FormatPerViolationMessage(
		fixes[0].Violation.ID,
		fixes[0].Violation.Description,
		fixes[0].Violation.Category,
		fixes[0].Violation.Effort,
		fixes,
		ct.providerName,
	)

	// Create commit
	sha, err := CreateCommit(ct.workingDir, message)
	if err != nil {
		return fmt.Errorf("failed to create violation commit: %w", err)
	}

	// Track commit info
	ct.commits = append(ct.commits, CommitInfo{
		SHA:         sha,
		Message:     message,
		ViolationID: violationID,
		PhaseID:     fixes[0].PhaseID, // Use phase ID from first fix (all should be same)
		FileCount:   len(fixes),
		Timestamp:   time.Now(),
	})

	fmt.Printf("📝 Created commit for violation %s (%d files)\n", violationID, len(fixes))

	// Clear the fixes for this violation
	delete(ct.fixesByViolation, violationID)

	return nil
}

// Finalize commits any remaining fixes based on strategy
func (ct *CommitTracker) Finalize() error {
	switch ct.strategy {
	case StrategyPerViolation:
		// Commit the last violation if there are pending fixes
		if ct.lastViolationID != "" {
			if err := ct.commitViolation(ct.lastViolationID); err != nil {
				return err
			}
		}
	case StrategyAtEnd:
		return ct.commitAtEnd()
	case StrategyPerIncident:
		// Nothing to do - commits were created incrementally
		return nil
	default:
		return nil
	}
	return nil
}

// commitAtEnd commits all accumulated fixes in one commit
func (ct *CommitTracker) commitAtEnd() error {
	if len(ct.allFixes) == 0 {
		return nil
	}

	// Stage all files
	stagedFiles := make(map[string]bool)
	for _, fix := range ct.allFixes {
		if !stagedFiles[fix.Result.FilePath] {
			if err := StageFile(ct.workingDir, fix.Result.FilePath); err != nil {
				return fmt.Errorf("failed to stage file for at-end commit: %w", err)
			}
			stagedFiles[fix.Result.FilePath] = true
		}
	}

	// Create commit message
	message := FormatAtEndMessage(ct.fixesByViolation, ct.providerName)

	// Create commit
	sha, err := CreateCommit(ct.workingDir, message)
	if err != nil {
		return fmt.Errorf("failed to create at-end commit: %w", err)
	}

	// Track commit info
	ct.commits = append(ct.commits, CommitInfo{
		SHA:         sha,
		Message:     message,
		ViolationID: "", // At-end commits don't have a single violation ID
		PhaseID:     "", // At-end commits don't have a single phase ID
		FileCount:   len(stagedFiles),
		Timestamp:   time.Now(),
	})

	fmt.Printf("📝 Created batch commit (%d violations, %d files)\n",
		len(ct.fixesByViolation), len(stagedFiles))

	return nil
}

// GetCommits returns the list of created commits
func (ct *CommitTracker) GetCommits() []CommitInfo {
	return ct.commits
}
