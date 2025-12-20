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

func TestCommitTracker_GetCommits(t *testing.T) {
	t.Run("returns empty list when no commits", func(t *testing.T) {
		tracker := NewCommitTracker(StrategyAtEnd, "/test/dir", "claude")
		commits := tracker.GetCommits()
		assert.Empty(t, commits)
	})

	t.Run("tracks commits from per-incident strategy", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)
		tracker := NewCommitTracker(StrategyPerIncident, tmpDir, "claude")

		// Create and track a fix
		filename := "test.txt"
		filepath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filepath, []byte("fixed"), 0644)
		require.NoError(t, err)

		v := violation.Violation{ID: "v1", Description: "Test", Category: "mandatory", Effort: 1}
		incident := violation.Incident{URI: "file://" + filepath, LineNumber: 10}
		result := &fixer.FixResult{FilePath: filename, Cost: 0.01, TokensUsed: 100, Success: true}

		err = tracker.TrackFix(v, incident, result)
		require.NoError(t, err)

		// Verify commit was tracked
		commits := tracker.GetCommits()
		require.Len(t, commits, 1)
		assert.NotEmpty(t, commits[0].SHA)
		assert.Equal(t, "v1", commits[0].ViolationID)
		assert.Equal(t, 1, commits[0].FileCount)
		assert.Contains(t, commits[0].Message, "v1")
		assert.NotZero(t, commits[0].Timestamp)
	})

	t.Run("tracks commits from per-violation strategy", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)
		tracker := NewCommitTracker(StrategyPerViolation, tmpDir, "claude")

		// Create and track multiple fixes for same violation
		v1 := violation.Violation{ID: "v1", Description: "Test 1", Category: "mandatory", Effort: 1}
		for i := 1; i <= 2; i++ {
			filename := fmt.Sprintf("test%d.txt", i)
			filepath := filepath.Join(tmpDir, filename)
			err := os.WriteFile(filepath, []byte("fixed"), 0644)
			require.NoError(t, err)

			incident := violation.Incident{URI: "file://" + filepath}
			result := &fixer.FixResult{FilePath: filename, Success: true}
			err = tracker.TrackFix(v1, incident, result)
			require.NoError(t, err)
		}

		// Create and track fix for different violation
		v2 := violation.Violation{ID: "v2", Description: "Test 2", Category: "optional", Effort: 2}
		filename := "test3.txt"
		filepath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filepath, []byte("fixed"), 0644)
		require.NoError(t, err)

		incident := violation.Incident{URI: "file://" + filepath}
		result := &fixer.FixResult{FilePath: filename, Success: true}
		err = tracker.TrackFix(v2, incident, result)
		require.NoError(t, err)

		// Finalize to create commits
		err = tracker.Finalize()
		require.NoError(t, err)

		// Verify commits were tracked - should have 2 commits (one per violation)
		commits := tracker.GetCommits()
		require.Len(t, commits, 2)

		// First commit should be for v1 with 2 files
		assert.Equal(t, "v1", commits[0].ViolationID)
		assert.Equal(t, 2, commits[0].FileCount)
		assert.NotEmpty(t, commits[0].SHA)

		// Second commit should be for v2 with 1 file
		assert.Equal(t, "v2", commits[1].ViolationID)
		assert.Equal(t, 1, commits[1].FileCount)
		assert.NotEmpty(t, commits[1].SHA)

		// SHAs should be different
		assert.NotEqual(t, commits[0].SHA, commits[1].SHA)
	})

	t.Run("tracks commit from at-end strategy", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)
		tracker := NewCommitTracker(StrategyAtEnd, tmpDir, "claude")

		// Create and track fixes for multiple violations
		for i := 1; i <= 3; i++ {
			filename := fmt.Sprintf("test%d.txt", i)
			filepath := filepath.Join(tmpDir, filename)
			err := os.WriteFile(filepath, []byte("fixed"), 0644)
			require.NoError(t, err)

			v := violation.Violation{ID: fmt.Sprintf("v%d", i), Description: fmt.Sprintf("Test %d", i), Category: "mandatory", Effort: 1}
			incident := violation.Incident{URI: "file://" + filepath}
			result := &fixer.FixResult{FilePath: filename, Success: true}
			err = tracker.TrackFix(v, incident, result)
			require.NoError(t, err)
		}

		// Finalize to create single commit
		err := tracker.Finalize()
		require.NoError(t, err)

		// Verify single commit was tracked with all files
		commits := tracker.GetCommits()
		require.Len(t, commits, 1)
		assert.NotEmpty(t, commits[0].SHA)
		assert.Equal(t, "", commits[0].ViolationID) // at-end commits don't have a single violation ID
		assert.Equal(t, 3, commits[0].FileCount)
		assert.Contains(t, commits[0].Message, "Batch remediation")
	})
}

func TestCommitTracker_CommitInfoFields(t *testing.T) {
	tmpDir := createTestGitRepo(t)
	configGitUser(t, tmpDir)
	tracker := NewCommitTracker(StrategyPerIncident, tmpDir, "claude")

	// Create and track a fix
	filename := "test.txt"
	filepath := filepath.Join(tmpDir, filename)
	err := os.WriteFile(filepath, []byte("fixed content"), 0644)
	require.NoError(t, err)

	v := violation.Violation{ID: "test-violation", Description: "Test violation", Category: "mandatory", Effort: 5}
	incident := violation.Incident{URI: "file://" + filepath, LineNumber: 42}
	result := &fixer.FixResult{
		ViolationID: v.ID,
		FilePath:    filename,
		Cost:        0.05,
		TokensUsed:  500,
		Success:     true,
	}

	err = tracker.TrackFix(v, incident, result)
	require.NoError(t, err)

	commits := tracker.GetCommits()
	require.Len(t, commits, 1)

	commit := commits[0]

	// Verify all CommitInfo fields are populated correctly
	assert.Len(t, commit.SHA, 40, "SHA should be 40 characters (full git SHA)")
	assert.NotEmpty(t, commit.Message, "Message should not be empty")
	assert.Contains(t, commit.Message, "test-violation", "Message should contain violation ID")
	assert.Contains(t, commit.Message, "Test violation", "Message should contain violation description")
	assert.Equal(t, "test-violation", commit.ViolationID, "ViolationID should match")
	assert.Equal(t, "", commit.PhaseID, "PhaseID should be empty for this test")
	assert.Equal(t, 1, commit.FileCount, "FileCount should be 1")
	assert.NotZero(t, commit.Timestamp, "Timestamp should be set")
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
