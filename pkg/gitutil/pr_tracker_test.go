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

		_, err := NewPRTracker(config, "/tmp", "claude", nil)
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
		DryRun:       false,
	}

	assert.Equal(t, PRStrategyPerViolation, config.Strategy)
	assert.Equal(t, "kantra-ai/remediation", config.BranchPrefix)
	assert.Equal(t, "main", config.BaseBranch)
	assert.Equal(t, "ghp_test123", config.GitHubToken)
	assert.False(t, config.DryRun)
}

func TestNewPRTracker_DryRunMode(t *testing.T) {
	t.Run("dry-run mode does not require GitHub token", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		config := PRConfig{
			Strategy:    PRStrategyAtEnd,
			GitHubToken: "", // Empty token is OK in dry-run
			DryRun:      true,
		}

		tracker, err := NewPRTracker(config, tmpDir, "claude", nil)
		require.NoError(t, err)
		assert.NotNil(t, tracker)
		assert.Nil(t, tracker.githubClient) // No GitHub client in dry-run
		assert.True(t, tracker.config.DryRun)
	})

	t.Run("dry-run mode skips GitHub client creation", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		config := PRConfig{
			Strategy:    PRStrategyPerViolation,
			GitHubToken: "test-token",
			DryRun:      true,
		}

		tracker, err := NewPRTracker(config, tmpDir, "openai", nil)
		require.NoError(t, err)
		assert.Nil(t, tracker.githubClient)
		assert.Equal(t, "openai", tracker.providerName)
	})
}

func TestPRTracker_DryRunFinalize(t *testing.T) {
	t.Run("dry-run at-end shows preview without creating branches", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		tracker := &PRTracker{
			config: PRConfig{
				Strategy:     PRStrategyAtEnd,
				BranchPrefix: "test-branch",
				DryRun:       true,
			},
			workingDir:       tmpDir,
			providerName:     "claude",
			fixesByViolation: make(map[string][]FixRecord),
			allFixes:         make([]FixRecord, 0),
			createdPRs:       make([]CreatedPR, 0),
			progress:         &NoOpProgressWriter{},
		}

		// Track a fix
		v := violation.Violation{ID: "v1", Description: "Test"}
		incident := violation.Incident{LineNumber: 10}
		result := &fixer.FixResult{FilePath: "test.java", Cost: 0.01, TokensUsed: 10}
		err := tracker.TrackForPR(v, incident, result)
		require.NoError(t, err)

		// Finalize should succeed without creating branches
		err = tracker.Finalize()
		require.NoError(t, err)

		// Verify a "mock" PR was tracked
		assert.Len(t, tracker.createdPRs, 1)
		assert.Equal(t, 0, tracker.createdPRs[0].Number)
		assert.Contains(t, tracker.createdPRs[0].URL, "DRY RUN")
	})

	t.Run("dry-run per-violation shows multiple PR previews", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		tracker := &PRTracker{
			config: PRConfig{
				Strategy:     PRStrategyPerViolation,
				BranchPrefix: "test-branch",
				DryRun:       true,
			},
			workingDir:       tmpDir,
			providerName:     "claude",
			fixesByViolation: make(map[string][]FixRecord),
			allFixes:         make([]FixRecord, 0),
			createdPRs:       make([]CreatedPR, 0),
			progress:         &NoOpProgressWriter{},
		}

		// Track fixes for two violations
		v1 := violation.Violation{ID: "v1", Description: "Test 1"}
		v2 := violation.Violation{ID: "v2", Description: "Test 2"}

		err := tracker.TrackForPR(v1, violation.Incident{LineNumber: 10}, &fixer.FixResult{FilePath: "test1.java"})
		require.NoError(t, err)
		err = tracker.TrackForPR(v2, violation.Incident{LineNumber: 20}, &fixer.FixResult{FilePath: "test2.java"})
		require.NoError(t, err)

		// Finalize should show previews for both PRs
		err = tracker.Finalize()
		require.NoError(t, err)

		// Verify both "mock" PRs were tracked
		assert.Len(t, tracker.createdPRs, 2)

		// Check that both violation IDs are present (order is non-deterministic due to map iteration)
		violationIDs := []string{tracker.createdPRs[0].ViolationID, tracker.createdPRs[1].ViolationID}
		assert.Contains(t, violationIDs, "v1")
		assert.Contains(t, violationIDs, "v2")
	})

	t.Run("dry-run per-incident shows PR preview for each fix", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		tracker := &PRTracker{
			config: PRConfig{
				Strategy:     PRStrategyPerIncident,
				BranchPrefix: "test-branch",
				DryRun:       true,
			},
			workingDir:       tmpDir,
			providerName:     "claude",
			fixesByViolation: make(map[string][]FixRecord),
			allFixes:         make([]FixRecord, 0),
			createdPRs:       make([]CreatedPR, 0),
			progress:         &NoOpProgressWriter{},
		}

		// Track three fixes
		v := violation.Violation{ID: "v1", Description: "Test"}
		for i := 0; i < 3; i++ {
			err := tracker.TrackForPR(v, violation.Incident{LineNumber: i * 10}, &fixer.FixResult{FilePath: "test.java"})
			require.NoError(t, err)
		}

		// Finalize should show previews for all three PRs
		err := tracker.Finalize()
		require.NoError(t, err)

		// Verify all three "mock" PRs were tracked
		assert.Len(t, tracker.createdPRs, 3)
	})
}

func TestPRTracker_addLowConfidenceComments(t *testing.T) {
	t.Run("disabled when threshold is 0", func(t *testing.T) {
		tracker := &PRTracker{
			config: PRConfig{
				CommentThreshold: 0, // Disabled
			},
			progress: &NoOpProgressWriter{},
		}

		// Create fixes with low confidence
		fixes := []FixRecord{
			{
				Violation: violation.Violation{ID: "v1", Description: "Test violation"},
				Incident:  violation.Incident{LineNumber: 10, URI: "file://test.java:10"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.5},
			},
		}

		// Should return nil without creating any comments
		err := tracker.addLowConfidenceComments(123, fixes)
		assert.NoError(t, err)
	})

	t.Run("disabled in dry-run mode", func(t *testing.T) {
		tracker := &PRTracker{
			config: PRConfig{
				CommentThreshold: 0.8,
				DryRun:           true, // Dry-run
			},
			progress: &NoOpProgressWriter{},
		}

		// Create fixes with low confidence
		fixes := []FixRecord{
			{
				Violation: violation.Violation{ID: "v1", Description: "Test violation"},
				Incident:  violation.Incident{LineNumber: 10, URI: "file://test.java:10"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.5},
			},
		}

		// Should return nil without creating any comments
		err := tracker.addLowConfidenceComments(123, fixes)
		assert.NoError(t, err)
	})

	t.Run("no comments when all fixes above threshold", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		// Create test file and commit it
		testFile := tmpDir + "/test.java"
		err := createAndCommitFile(t, tmpDir, testFile, "public class Test {}")
		require.NoError(t, err)

		tracker := &PRTracker{
			config: PRConfig{
				CommentThreshold: 0.8,
				DryRun:           false,
			},
			workingDir: tmpDir,
			progress:   &NoOpProgressWriter{},
		}

		// All fixes have high confidence (above threshold)
		fixes := []FixRecord{
			{
				Violation: violation.Violation{ID: "v1", Description: "Test violation"},
				Incident:  violation.Incident{LineNumber: 1, URI: "file://test.java:1"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.9},
			},
			{
				Violation: violation.Violation{ID: "v2", Description: "Test violation 2"},
				Incident:  violation.Incident{LineNumber: 2, URI: "file://test.java:2"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.85},
			},
		}

		// Should return nil - no low-confidence fixes
		err = tracker.addLowConfidenceComments(123, fixes)
		assert.NoError(t, err)
	})

	t.Run("filters fixes by confidence threshold", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		// Create test file and commit it
		testFile := tmpDir + "/test.java"
		err := createAndCommitFile(t, tmpDir, testFile, "public class Test {}")
		require.NoError(t, err)

		// Track the number of comment creation attempts
		commentCount := 0
		mockClient := &mockGitHubClientForComments{
			createReviewCommentFunc: func(prNumber int, req ReviewCommentRequest) (*ReviewCommentResponse, error) {
				commentCount++
				return &ReviewCommentResponse{
					ID:      commentCount,
					Body:    req.Body,
					Path:    req.Path,
					Line:    req.Line,
					HTMLURL: "https://github.com/test/test/pull/123#discussion_r" + string(rune(commentCount)),
				}, nil
			},
		}

		tracker := &PRTracker{
			config: PRConfig{
				CommentThreshold: 0.8, // Only comment on fixes below 0.8
				DryRun:           false,
			},
			workingDir:   tmpDir,
			githubClient: mockClient,
			progress:     &NoOpProgressWriter{},
		}

		// Mix of high and low confidence fixes
		fixes := []FixRecord{
			{
				Violation: violation.Violation{ID: "v1", Description: "Test violation"},
				Incident:  violation.Incident{LineNumber: 1, URI: "file://test.java:1"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.9}, // Above threshold - no comment
			},
			{
				Violation: violation.Violation{ID: "v2", Description: "Test violation 2"},
				Incident:  violation.Incident{LineNumber: 2, URI: "file://test.java:2"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.7}, // Below threshold - comment
			},
			{
				Violation: violation.Violation{ID: "v3", Description: "Test violation 3"},
				Incident:  violation.Incident{LineNumber: 3, URI: "file://test.java:3"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.5}, // Below threshold - comment
			},
			{
				Violation: violation.Violation{ID: "v4", Description: "Test violation 4"},
				Incident:  violation.Incident{LineNumber: 4, URI: "file://test.java:4"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0}, // No confidence - no comment
			},
		}

		err = tracker.addLowConfidenceComments(123, fixes)
		assert.NoError(t, err)

		// Should have created comments for only the 2 low-confidence fixes (0.7 and 0.5)
		assert.Equal(t, 2, commentCount, "should create comments for fixes below threshold with confidence > 0")
	})

	t.Run("gracefully handles comment creation errors", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		// Create test file and commit it
		testFile := tmpDir + "/test.java"
		err := createAndCommitFile(t, tmpDir, testFile, "public class Test {}")
		require.NoError(t, err)

		// Mock client that fails on second comment
		commentCount := 0
		mockClient := &mockGitHubClientForComments{
			createReviewCommentFunc: func(prNumber int, req ReviewCommentRequest) (*ReviewCommentResponse, error) {
				commentCount++
				if commentCount == 2 {
					return nil, &GitHubError{
						Message:    "Validation Failed",
						StatusCode: 422,
					}
				}
				return &ReviewCommentResponse{
					ID:      commentCount,
					Body:    req.Body,
					Path:    req.Path,
					Line:    req.Line,
					HTMLURL: "https://github.com/test/test/pull/123#discussion_r" + string(rune(commentCount)),
				}, nil
			},
		}

		tracker := &PRTracker{
			config: PRConfig{
				CommentThreshold: 0.8,
				DryRun:           false,
			},
			workingDir:   tmpDir,
			githubClient: mockClient,
			progress:     &NoOpProgressWriter{},
		}

		fixes := []FixRecord{
			{
				Violation: violation.Violation{ID: "v1", Description: "Test violation"},
				Incident:  violation.Incident{LineNumber: 1, URI: "file://test.java:1"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.7},
			},
			{
				Violation: violation.Violation{ID: "v2", Description: "Test violation 2"},
				Incident:  violation.Incident{LineNumber: 2, URI: "file://test.java:2"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.6},
			},
			{
				Violation: violation.Violation{ID: "v3", Description: "Test violation 3"},
				Incident:  violation.Incident{LineNumber: 3, URI: "file://test.java:3"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.5},
			},
		}

		// Should not return error even though second comment failed
		err = tracker.addLowConfidenceComments(123, fixes)
		assert.NoError(t, err, "should gracefully handle comment creation errors")

		// All three comments should have been attempted
		assert.Equal(t, 3, commentCount)
	})

	t.Run("comment body contains expected information", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		configGitUser(t, tmpDir)

		// Create test file and commit it
		testFile := tmpDir + "/test.java"
		err := createAndCommitFile(t, tmpDir, testFile, "public class Test {}")
		require.NoError(t, err)

		var capturedComment ReviewCommentRequest
		mockClient := &mockGitHubClientForComments{
			createReviewCommentFunc: func(prNumber int, req ReviewCommentRequest) (*ReviewCommentResponse, error) {
				capturedComment = req
				return &ReviewCommentResponse{
					ID:      1,
					Body:    req.Body,
					Path:    req.Path,
					Line:    req.Line,
					HTMLURL: "https://github.com/test/test/pull/123#discussion_r1",
				}, nil
			},
		}

		tracker := &PRTracker{
			config: PRConfig{
				CommentThreshold: 0.8,
				DryRun:           false,
			},
			workingDir:   tmpDir,
			githubClient: mockClient,
			progress:     &NoOpProgressWriter{},
		}

		fixes := []FixRecord{
			{
				Violation: violation.Violation{ID: "deprecated-api", Description: "Use of deprecated API"},
				Incident:  violation.Incident{LineNumber: 10, URI: "file://test.java:10"},
				Result:    &fixer.FixResult{FilePath: "test.java", Confidence: 0.65},
			},
		}

		err = tracker.addLowConfidenceComments(123, fixes)
		require.NoError(t, err)

		// Verify comment body contains expected information
		assert.Contains(t, capturedComment.Body, "⚠️", "should have warning emoji")
		assert.Contains(t, capturedComment.Body, "65%", "should show confidence percentage")
		assert.Contains(t, capturedComment.Body, "deprecated-api", "should include violation ID")
		assert.Contains(t, capturedComment.Body, "Use of deprecated API", "should include violation description")
		assert.Contains(t, capturedComment.Body, "review carefully", "should include review guidance")

		// Verify comment metadata
		assert.Equal(t, "test.java", capturedComment.Path)
		assert.Equal(t, 10, capturedComment.Line)
		assert.Equal(t, "RIGHT", capturedComment.Side)
	})
}

// mockGitHubClientForComments is a mock implementation of GitHubClient for testing comment creation
type mockGitHubClientForComments struct {
	createReviewCommentFunc func(prNumber int, req ReviewCommentRequest) (*ReviewCommentResponse, error)
}

func (m *mockGitHubClientForComments) CreatePullRequest(req PullRequestRequest) (*PullRequestResponse, error) {
	return nil, nil
}

func (m *mockGitHubClientForComments) GetDefaultBranch() (string, error) {
	return "main", nil
}

func (m *mockGitHubClientForComments) CreateCommitStatus(sha string, req CommitStatusRequest) (*CommitStatusResponse, error) {
	return nil, nil
}

func (m *mockGitHubClientForComments) CreateReviewComment(prNumber int, req ReviewCommentRequest) (*ReviewCommentResponse, error) {
	if m.createReviewCommentFunc != nil {
		return m.createReviewCommentFunc(prNumber, req)
	}
	return nil, nil
}
