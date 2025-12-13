package gitutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CommitStrategy
		wantErr bool
	}{
		{
			name:    "per-violation",
			input:   "per-violation",
			want:    StrategyPerViolation,
			wantErr: false,
		},
		{
			name:    "per-incident",
			input:   "per-incident",
			want:    StrategyPerIncident,
			wantErr: false,
		},
		{
			name:    "at-end",
			input:   "at-end",
			want:    StrategyAtEnd,
			wantErr: false,
		},
		{
			name:    "invalid strategy",
			input:   "invalid",
			want:    StrategyNone,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    StrategyNone,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStrategy(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewCommitTracker(t *testing.T) {
	tracker := NewCommitTracker(StrategyPerViolation, "/test/dir", "claude")

	assert.NotNil(t, tracker)
	assert.Equal(t, StrategyPerViolation, tracker.strategy)
	assert.Equal(t, "/test/dir", tracker.workingDir)
	assert.Equal(t, "claude", tracker.providerName)
	assert.NotNil(t, tracker.fixesByViolation)
	assert.NotNil(t, tracker.allFixes)
	assert.Equal(t, "", tracker.lastViolationID)
}

func TestCommitTracker_TrackFix_AtEnd(t *testing.T) {
	tracker := NewCommitTracker(StrategyAtEnd, "/test/dir", "claude")

	v := violation.Violation{
		ID:          "test-001",
		Description: "Test violation",
		Category:    "mandatory",
		Effort:      1,
	}

	incident := violation.Incident{
		URI:        "file:///test.java",
		LineNumber: 10,
	}

	result := &fixer.FixResult{
		ViolationID: v.ID,
		FilePath:    "test.java",
		Cost:        0.01,
		TokensUsed:  100,
		Success:     true,
	}

	// Track a fix
	err := tracker.TrackFix(v, incident, result)
	assert.NoError(t, err)

	// Verify fix was added to allFixes
	assert.Len(t, tracker.allFixes, 1)
	assert.Equal(t, v.ID, tracker.allFixes[0].Violation.ID)

	// Verify fix was added to fixesByViolation
	assert.Len(t, tracker.fixesByViolation["test-001"], 1)
}

func TestCommitTracker_TrackFix_PerViolation(t *testing.T) {
	t.Run("track multiple incidents same violation", func(t *testing.T) {
		tracker := NewCommitTracker(StrategyPerViolation, "/test/dir", "claude")

		v1 := violation.Violation{
			ID:       "violation-001",
			Category: "mandatory",
		}

		// Track first violation
		err := tracker.trackForPerViolation(FixRecord{
			Violation: v1,
			Result:    &fixer.FixResult{FilePath: "file1.java"},
		})
		assert.NoError(t, err)
		assert.Equal(t, "violation-001", tracker.lastViolationID)
		assert.Len(t, tracker.fixesByViolation["violation-001"], 1)

		// Track another incident of same violation
		err = tracker.trackForPerViolation(FixRecord{
			Violation: v1,
			Result:    &fixer.FixResult{FilePath: "file2.java"},
		})
		assert.NoError(t, err)
		assert.Len(t, tracker.fixesByViolation["violation-001"], 2)
		assert.Equal(t, "violation-001", tracker.lastViolationID)
	})

	t.Run("track accumulates fixes for same violation", func(t *testing.T) {
		tracker := NewCommitTracker(StrategyPerViolation, "/test/dir", "claude")

		v1 := violation.Violation{ID: "v1"}

		// Errors expected since /test/dir doesn't exist, but we're testing tracking logic
		_ = tracker.trackForPerViolation(FixRecord{Violation: v1, Result: &fixer.FixResult{}})
		_ = tracker.trackForPerViolation(FixRecord{Violation: v1, Result: &fixer.FixResult{}})
		_ = tracker.trackForPerViolation(FixRecord{Violation: v1, Result: &fixer.FixResult{}})

		// All three fixes should be accumulated for v1
		assert.Len(t, tracker.fixesByViolation["v1"], 3)
		assert.Equal(t, "v1", tracker.lastViolationID)
	})
}

func TestCommitTracker_Integration(t *testing.T) {
	t.Run("at-end strategy integration", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		tracker := NewCommitTracker(StrategyAtEnd, tmpDir, "claude")

		// Configure git user
		configGitUser(t, tmpDir)

		// Create test files and track fixes
		violations := []violation.Violation{
			{ID: "v1", Description: "Violation 1", Category: "mandatory", Effort: 1},
			{ID: "v2", Description: "Violation 2", Category: "optional", Effort: 2},
		}

		for i, v := range violations {
			filename := fmt.Sprintf("file%d.txt", i+1)
			filepath := filepath.Join(tmpDir, filename)
			err := os.WriteFile(filepath, []byte("fixed content"), 0644)
			require.NoError(t, err)

			incident := violation.Incident{URI: "file://" + filepath}
			result := &fixer.FixResult{
				FilePath:   filename,
				Cost:       0.01,
				TokensUsed: 100,
				Success:    true,
			}

			err = tracker.TrackFix(v, incident, result)
			require.NoError(t, err)
		}

		// Finalize should create one commit
		err := tracker.Finalize()
		assert.NoError(t, err)

		// Verify commit was created
		cmd := exec.Command("git", "log", "--oneline")
		cmd.Dir = tmpDir
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "Batch remediation")
	})
}

func TestFormatPerViolationMessage_EmptyFixes(t *testing.T) {
	// Edge case: empty fixes list
	message := FormatPerViolationMessage(
		"test-id",
		"description",
		"mandatory",
		1,
		[]FixRecord{},
		"claude",
	)

	assert.Contains(t, message, "test-id")
	assert.Contains(t, message, "Incidents Fixed: 0")
	assert.Contains(t, message, "Total Cost: $0.0000")
	assert.Contains(t, message, "Total Tokens: 0")
}

// configGitUser configures git user for a test repository
func configGitUser(t *testing.T, dir string) {
	cmd := exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
}
