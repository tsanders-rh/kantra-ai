package gitutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestParsePRStrategy(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    PRStrategy
		wantErr bool
	}{
		{
			name:    "per-violation",
			input:   "per-violation",
			want:    PRStrategyPerViolation,
			wantErr: false,
		},
		{
			name:    "per-incident",
			input:   "per-incident",
			want:    PRStrategyPerIncident,
			wantErr: false,
		},
		{
			name:    "at-end",
			input:   "at-end",
			want:    PRStrategyAtEnd,
			wantErr: false,
		},
		{
			name:    "invalid strategy",
			input:   "invalid",
			want:    PRStrategyNone,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    PRStrategyNone,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePRStrategy(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPRTracker_TrackForPR(t *testing.T) {
	t.Run("tracks fixes by violation", func(t *testing.T) {
		tracker := &PRTracker{
			fixesByViolation: make(map[string][]FixRecord),
			allFixes:         make([]FixRecord, 0),
		}

		v1 := violation.Violation{ID: "v1", Description: "Test"}
		incident1 := violation.Incident{LineNumber: 10}
		result1 := &fixer.FixResult{FilePath: "file1.java", Cost: 0.05, TokensUsed: 100}

		err := tracker.TrackForPR(v1, incident1, result1)
		require.NoError(t, err)

		// Verify tracking
		assert.Len(t, tracker.fixesByViolation["v1"], 1)
		assert.Len(t, tracker.allFixes, 1)
		assert.Equal(t, "v1", tracker.allFixes[0].Violation.ID)
	})

	t.Run("accumulates multiple fixes for same violation", func(t *testing.T) {
		tracker := &PRTracker{
			fixesByViolation: make(map[string][]FixRecord),
			allFixes:         make([]FixRecord, 0),
		}

		v := violation.Violation{ID: "v1"}

		// Add three fixes for same violation
		for i := 0; i < 3; i++ {
			incident := violation.Incident{LineNumber: i * 10}
			result := &fixer.FixResult{FilePath: "file.java", Cost: 0.01, TokensUsed: 10}
			err := tracker.TrackForPR(v, incident, result)
			require.NoError(t, err)
		}

		assert.Len(t, tracker.fixesByViolation["v1"], 3)
		assert.Len(t, tracker.allFixes, 3)
	})

	t.Run("tracks multiple violations separately", func(t *testing.T) {
		tracker := &PRTracker{
			fixesByViolation: make(map[string][]FixRecord),
			allFixes:         make([]FixRecord, 0),
		}

		v1 := violation.Violation{ID: "v1"}
		v2 := violation.Violation{ID: "v2"}

		err := tracker.TrackForPR(v1, violation.Incident{}, &fixer.FixResult{})
		require.NoError(t, err)

		err = tracker.TrackForPR(v2, violation.Incident{}, &fixer.FixResult{})
		require.NoError(t, err)

		err = tracker.TrackForPR(v1, violation.Incident{}, &fixer.FixResult{})
		require.NoError(t, err)

		assert.Len(t, tracker.fixesByViolation["v1"], 2)
		assert.Len(t, tracker.fixesByViolation["v2"], 1)
		assert.Len(t, tracker.allFixes, 3)
	})
}

func TestPRTracker_GetCreatedPRs(t *testing.T) {
	tracker := &PRTracker{
		createdPRs: []CreatedPR{
			{Number: 1, URL: "https://github.com/owner/repo/pull/1", BranchName: "branch1"},
			{Number: 2, URL: "https://github.com/owner/repo/pull/2", BranchName: "branch2"},
		},
	}

	prs := tracker.GetCreatedPRs()
	assert.Len(t, prs, 2)
	assert.Equal(t, 1, prs[0].Number)
	assert.Equal(t, 2, prs[1].Number)
}

func TestNewPRTracker_Validation(t *testing.T) {
	t.Run("missing GitHub token", func(t *testing.T) {
		config := PRConfig{
			Strategy:    PRStrategyAtEnd,
			GitHubToken: "", // Empty token
		}

		_, err := NewPRTracker(config, "/tmp", "claude")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub token is required")
	})
}

func TestPRTracker_Integration(t *testing.T) {
	t.Run("tracks and retrieves fixes correctly", func(t *testing.T) {
		// Create a real git repo for this test
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		// Note: We can't test the full Finalize flow without a real GitHub repo
		// and token, but we can test the tracking logic

		tracker := &PRTracker{
			config: PRConfig{
				Strategy:     PRStrategyPerViolation,
				BranchPrefix: "test",
				GitHubToken:  "fake-token",
			},
			workingDir:       tmpDir,
			providerName:     "claude",
			fixesByViolation: make(map[string][]FixRecord),
			allFixes:         make([]FixRecord, 0),
			createdPRs:       make([]CreatedPR, 0),
		}

		// Track some fixes
		v1 := violation.Violation{
			ID:          "v1",
			Description: "Test violation 1",
			Category:    "mandatory",
			Effort:      1,
		}
		v2 := violation.Violation{
			ID:          "v2",
			Description: "Test violation 2",
			Category:    "optional",
			Effort:      2,
		}

		// Add fixes
		for i := 0; i < 3; i++ {
			incident := violation.Incident{LineNumber: i * 10}
			result := &fixer.FixResult{
				FilePath:   "file.java",
				Cost:       0.01,
				TokensUsed: 10,
			}

			if i < 2 {
				err := tracker.TrackForPR(v1, incident, result)
				require.NoError(t, err)
			} else {
				err := tracker.TrackForPR(v2, incident, result)
				require.NoError(t, err)
			}
		}

		// Verify state
		assert.Len(t, tracker.fixesByViolation, 2)
		assert.Len(t, tracker.fixesByViolation["v1"], 2)
		assert.Len(t, tracker.fixesByViolation["v2"], 1)
		assert.Len(t, tracker.allFixes, 3)

		// Verify fix details
		assert.Equal(t, "v1", tracker.allFixes[0].Violation.ID)
		assert.Equal(t, "v1", tracker.allFixes[1].Violation.ID)
		assert.Equal(t, "v2", tracker.allFixes[2].Violation.ID)
	})
}

func TestPendingPR_Structure(t *testing.T) {
	// Test that PendingPR can be created and used correctly
	pr := PendingPR{
		ViolationID: "test-001",
		BranchName:  "kantra-ai/fix-test-001",
		Fixes: []FixRecord{
			{
				Violation: violation.Violation{ID: "test-001"},
				Incident:  violation.Incident{LineNumber: 42},
				Result:    &fixer.FixResult{FilePath: "test.java"},
				Timestamp: time.Now(),
			},
		},
	}

	assert.Equal(t, "test-001", pr.ViolationID)
	assert.Equal(t, "kantra-ai/fix-test-001", pr.BranchName)
	assert.Len(t, pr.Fixes, 1)
}

func TestCreatedPR_Structure(t *testing.T) {
	// Test that CreatedPR can be created and used correctly
	pr := CreatedPR{
		Number:      42,
		URL:         "https://github.com/owner/repo/pull/42",
		BranchName:  "feature/test",
		ViolationID: "v1",
	}

	assert.Equal(t, 42, pr.Number)
	assert.Equal(t, "https://github.com/owner/repo/pull/42", pr.URL)
	assert.Equal(t, "feature/test", pr.BranchName)
	assert.Equal(t, "v1", pr.ViolationID)
}

func TestPRConfig_Structure(t *testing.T) {
	// Test that PRConfig can be created correctly
	config := PRConfig{
		Strategy:     PRStrategyPerViolation,
		BranchPrefix: "kantra-ai/remediation",
		BaseBranch:   "main",
		GitHubToken:  "ghp_test123",
	}

	assert.Equal(t, PRStrategyPerViolation, config.Strategy)
	assert.Equal(t, "kantra-ai/remediation", config.BranchPrefix)
	assert.Equal(t, "main", config.BaseBranch)
	assert.Equal(t, "ghp_test123", config.GitHubToken)
}
