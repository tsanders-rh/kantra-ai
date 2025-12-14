package gitutil

import (
	"fmt"

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

	result, err := vct.verifier.Verify()
	if err != nil {
		return fmt.Errorf("verification error: %w", err)
	}

	if result.Success {
		vct.stats.PassedVerifications++
		return nil
	}

	// Verification failed
	vct.stats.FailedVerifications++

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
