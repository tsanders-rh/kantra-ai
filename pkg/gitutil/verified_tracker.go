package gitutil

import (
	"fmt"
	"time"

	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/verifier"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// VerifiedCommitTracker wraps CommitTracker with verification support
type VerifiedCommitTracker struct {
	commitTracker *CommitTracker
	verifier      *verifier.Verifier
	verifyConfig  verifier.Config
	stats         VerificationStats
	githubClient  *GitHubClient // Optional: for reporting status checks
	workingDir    string
}

// VerificationStats tracks verification outcomes
type VerificationStats struct {
	TotalVerifications int
	PassedVerifications int
	FailedVerifications int
	SkippedFixes       int // Fixes skipped due to verification failure
}

// NewVerifiedCommitTracker creates a commit tracker with verification
func NewVerifiedCommitTracker(
	commitStrategy CommitStrategy,
	workingDir string,
	providerName string,
	verifyConfig verifier.Config,
) (*VerifiedCommitTracker, error) {
	return NewVerifiedCommitTrackerWithGitHub(commitStrategy, workingDir, providerName, verifyConfig, nil)
}

// NewVerifiedCommitTrackerWithGitHub creates a commit tracker with verification and optional GitHub status checks
func NewVerifiedCommitTrackerWithGitHub(
	commitStrategy CommitStrategy,
	workingDir string,
	providerName string,
	verifyConfig verifier.Config,
	githubClient *GitHubClient,
) (*VerifiedCommitTracker, error) {
	// Create verifier if verification is enabled
	var v *verifier.Verifier
	var err error
	if verifyConfig.Type != verifier.VerificationNone {
		v, err = verifier.NewVerifier(verifyConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create verifier: %w", err)
		}
	}

	return &VerifiedCommitTracker{
		commitTracker: NewCommitTracker(commitStrategy, workingDir, providerName),
		verifier:      v,
		verifyConfig:  verifyConfig,
		stats:         VerificationStats{},
		githubClient:  githubClient,
		workingDir:    workingDir,
	}, nil
}

// TrackFix records a fix and verifies it based on the strategy
func (vct *VerifiedCommitTracker) TrackFix(v violation.Violation, incident violation.Incident, result *fixer.FixResult) error {
	// If no verification, just track the fix
	if vct.verifier == nil {
		return vct.commitTracker.TrackFix(v, incident, result)
	}

	// Determine if we should verify now based on strategy
	shouldVerify := vct.shouldVerifyNow(v, incident)

	// Track the fix first
	if err := vct.commitTracker.TrackFix(v, incident, result); err != nil {
		return err
	}

	// Run verification if needed
	if shouldVerify {
		return vct.runVerification()
	}

	return nil
}

// Finalize commits any pending fixes and runs final verification if needed
func (vct *VerifiedCommitTracker) Finalize() error {
	// For at-end strategy, verify before final commit
	if vct.verifier != nil && vct.verifyConfig.Strategy == verifier.StrategyAtEnd {
		// Don't commit yet - we need to verify first
		if err := vct.runVerification(); err != nil {
			return err
		}
	}

	// Now finalize commits
	if err := vct.commitTracker.Finalize(); err != nil {
		return err
	}

	return nil
}

// shouldVerifyNow determines if verification should run now
func (vct *VerifiedCommitTracker) shouldVerifyNow(v violation.Violation, incident violation.Incident) bool {
	if vct.verifier == nil {
		return false
	}

	switch vct.verifyConfig.Strategy {
	case verifier.StrategyPerFix:
		// Verify after every fix
		return true
	case verifier.StrategyPerViolation:
		// Verify when we're done with a violation
		// This is tricky - we need to know if this is the last fix for the violation
		// For now, we'll verify at Finalize() for per-violation
		return false
	case verifier.StrategyAtEnd:
		// Only verify at Finalize()
		return false
	default:
		return false
	}
}

// runVerification runs the verification and handles the result
func (vct *VerifiedCommitTracker) runVerification() error {
	vct.stats.TotalVerifications++

	// Report pending status to GitHub if enabled
	if vct.githubClient != nil {
		vct.reportPendingStatus()
	}

	result, err := vct.verifier.Verify()
	if err != nil {
		// Report error status to GitHub if enabled
		if vct.githubClient != nil {
			vct.reportErrorStatus(err)
		}
		return fmt.Errorf("verification error: %w", err)
	}

	if result.Success {
		vct.stats.PassedVerifications++
		// Report success status to GitHub if enabled
		if vct.githubClient != nil {
			vct.reportSuccessStatus(result)
		}
		return nil
	}

	// Verification failed
	vct.stats.FailedVerifications++

	// Report failure status to GitHub if enabled
	if vct.githubClient != nil {
		vct.reportFailureStatus(result)
	}

	// Handle failure based on configuration
	if vct.verifyConfig.FailFast {
		return fmt.Errorf("verification failed (fail-fast enabled):\n%s\n\nCommand: %s\nError: %v",
			result.Output, result.Command, result.Error)
	}

	// Log failure but continue
	fmt.Printf("\n⚠ Verification failed (continuing):\n")
	fmt.Printf("  Command: %s\n", result.Command)
	fmt.Printf("  Duration: %s\n", result.Duration)
	if result.Error != nil {
		fmt.Printf("  Error: %v\n", result.Error)
	}
	fmt.Printf("  Output (last 500 chars):\n")
	output := result.Output
	if len(output) > 500 {
		output = "..." + output[len(output)-500:]
	}
	fmt.Printf("  %s\n\n", output)

	// For now, we'll revert the last commit if verification fails
	// In the future, we might want more sophisticated rollback
	if err := vct.revertLastChange(); err != nil {
		return fmt.Errorf("failed to revert changes after verification failure: %w", err)
	}

	vct.stats.SkippedFixes++
	return nil
}

// revertLastChange reverts the most recent uncommitted changes
func (vct *VerifiedCommitTracker) revertLastChange() error {
	// For per-fix strategy, we need to revert uncommitted changes
	// This is a simplified implementation
	if err := ResetChanges(vct.commitTracker.workingDir); err != nil {
		return fmt.Errorf("failed to reset changes: %w", err)
	}
	return nil
}

// GetStats returns the verification statistics
func (vct *VerifiedCommitTracker) GetStats() VerificationStats {
	return vct.stats
}

// GetCommitTracker returns the underlying commit tracker
func (vct *VerifiedCommitTracker) GetCommitTracker() *CommitTracker {
	return vct.commitTracker
}

// reportPendingStatus reports a pending verification status to GitHub
func (vct *VerifiedCommitTracker) reportPendingStatus() {
	sha, err := GetCurrentCommitSHA(vct.workingDir)
	if err != nil {
		fmt.Printf("Warning: failed to get commit SHA for status check: %v\n", err)
		return
	}

	context := vct.getStatusContext()
	description := vct.getVerificationDescription() + " - running..."

	req := CommitStatusRequest{
		State:       StatusStatePending,
		Description: description,
		Context:     context,
	}

	if _, err := vct.githubClient.CreateCommitStatus(sha, req); err != nil {
		fmt.Printf("Warning: failed to report pending status to GitHub: %v\n", err)
	}
}

// reportSuccessStatus reports a successful verification status to GitHub
func (vct *VerifiedCommitTracker) reportSuccessStatus(result *verifier.Result) {
	sha, err := GetCurrentCommitSHA(vct.workingDir)
	if err != nil {
		fmt.Printf("Warning: failed to get commit SHA for status check: %v\n", err)
		return
	}

	context := vct.getStatusContext()
	description := fmt.Sprintf("%s passed (%s)", vct.getVerificationDescription(), result.Duration.Round(100*time.Millisecond))

	req := CommitStatusRequest{
		State:       StatusStateSuccess,
		Description: description,
		Context:     context,
	}

	if _, err := vct.githubClient.CreateCommitStatus(sha, req); err != nil {
		fmt.Printf("Warning: failed to report success status to GitHub: %v\n", err)
	} else {
		fmt.Printf("✓ Reported verification success to GitHub\n")
	}
}

// reportFailureStatus reports a failed verification status to GitHub
func (vct *VerifiedCommitTracker) reportFailureStatus(result *verifier.Result) {
	sha, err := GetCurrentCommitSHA(vct.workingDir)
	if err != nil {
		fmt.Printf("Warning: failed to get commit SHA for status check: %v\n", err)
		return
	}

	context := vct.getStatusContext()
	description := fmt.Sprintf("%s failed", vct.getVerificationDescription())

	req := CommitStatusRequest{
		State:       StatusStateFailure,
		Description: description,
		Context:     context,
	}

	if _, err := vct.githubClient.CreateCommitStatus(sha, req); err != nil {
		fmt.Printf("Warning: failed to report failure status to GitHub: %v\n", err)
	} else {
		fmt.Printf("✗ Reported verification failure to GitHub\n")
	}
}

// reportErrorStatus reports an error during verification to GitHub
func (vct *VerifiedCommitTracker) reportErrorStatus(verifyErr error) {
	sha, err := GetCurrentCommitSHA(vct.workingDir)
	if err != nil {
		fmt.Printf("Warning: failed to get commit SHA for status check: %v\n", err)
		return
	}

	context := vct.getStatusContext()
	description := fmt.Sprintf("%s encountered an error", vct.getVerificationDescription())

	req := CommitStatusRequest{
		State:       StatusStateError,
		Description: description,
		Context:     context,
	}

	if _, err := vct.githubClient.CreateCommitStatus(sha, req); err != nil {
		fmt.Printf("Warning: failed to report error status to GitHub: %v\n", err)
	}
}

// getStatusContext returns the context string for GitHub status checks
// Format: "kantra-ai/verify-{type}"
func (vct *VerifiedCommitTracker) getStatusContext() string {
	verifyType := "verification"
	switch vct.verifyConfig.Type {
	case verifier.VerificationBuild:
		verifyType = "build"
	case verifier.VerificationTest:
		verifyType = "test"
	}
	return fmt.Sprintf("kantra-ai/verify-%s", verifyType)
}

// getVerificationDescription returns a human-readable description of the verification type
func (vct *VerifiedCommitTracker) getVerificationDescription() string {
	switch vct.verifyConfig.Type {
	case verifier.VerificationBuild:
		return "Build verification"
	case verifier.VerificationTest:
		return "Test verification"
	default:
		return "Verification"
	}
}
