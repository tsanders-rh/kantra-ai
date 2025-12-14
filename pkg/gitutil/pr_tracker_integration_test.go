package gitutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// setupMockGitHubServer creates a mock GitHub API server
func setupMockGitHubServer(t *testing.T) *httptest.Server {
	prCounter := 1

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle default branch request
		if r.URL.Path == "/repos/test-owner/test-repo" && r.Method == "GET" {
			response := map[string]interface{}{
				"default_branch": "main",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Handle PR creation
		if r.URL.Path == "/repos/test-owner/test-repo/pulls" && r.Method == "POST" {
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			// Return successful PR creation
			response := map[string]interface{}{
				"number":   prCounter,
				"html_url": fmt.Sprintf("https://github.com/test-owner/test-repo/pull/%d", prCounter),
			}
			prCounter++

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(response)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

// setupTestRepoWithRemote creates a git repo with a mock remote
func setupTestRepoWithRemote(t *testing.T, remoteURL string) string {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git user
	configGitUser(t, tmpDir)

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("initial"), 0644))

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	return tmpDir
}

func TestNewGitHubClient_Success(t *testing.T) {
	tmpDir := setupTestRepoWithRemote(t, "https://github.com/test-owner/test-repo.git")

	client, err := NewGitHubClient(tmpDir, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "test-owner", client.owner)
	assert.Equal(t, "test-repo", client.repo)
	assert.Equal(t, "test-token", client.token)
}

func TestNewGitHubClient_Errors(t *testing.T) {
	t.Run("not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := NewGitHubClient(tmpDir, "test-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get remote URL")
	})

	t.Run("not a GitHub URL", func(t *testing.T) {
		tmpDir := setupTestRepoWithRemote(t, "https://gitlab.com/owner/repo.git")
		_, err := NewGitHubClient(tmpDir, "test-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a valid GitHub URL")
	})
}

func TestNewPRTracker_Success(t *testing.T) {
	tmpDir := setupTestRepoWithRemote(t, "https://github.com/test-owner/test-repo.git")

	config := PRConfig{
		Strategy:     PRStrategyAtEnd,
		BranchPrefix: "test-branch",
		GitHubToken:  "test-token",
	}

	tracker, err := NewPRTracker(config, tmpDir, "claude", nil)
	require.NoError(t, err)
	assert.NotNil(t, tracker)
	assert.Equal(t, "claude", tracker.providerName)
}

func TestNewPRTracker_Errors(t *testing.T) {
	t.Run("missing GitHub token", func(t *testing.T) {
		tmpDir := setupTestRepoWithRemote(t, "https://github.com/test-owner/test-repo.git")

		config := PRConfig{
			Strategy:    PRStrategyAtEnd,
			GitHubToken: "", // Empty token
		}

		_, err := NewPRTracker(config, tmpDir, "claude", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub token is required")
	})

	t.Run("invalid GitHub client", func(t *testing.T) {
		tmpDir := t.TempDir() // Not a git repo

		config := PRConfig{
			Strategy:    PRStrategyAtEnd,
			GitHubToken: "test-token",
		}

		_, err := NewPRTracker(config, tmpDir, "claude", nil)
		assert.Error(t, err)
	})
}

func TestPRTracker_Finalize_AtEnd(t *testing.T) {
	// Skip if we can't push (no git configured)
	if os.Getenv("CI") == "" {
		t.Skip("Skipping git push test in local environment")
	}

	server := setupMockGitHubServer(t)
	defer server.Close()

	tmpDir := setupTestRepoWithRemote(t, "https://github.com/test-owner/test-repo.git")

	// Create tracker with mock server
	tracker := &PRTracker{
		config: PRConfig{
			Strategy:     PRStrategyAtEnd,
			BranchPrefix: "test-branch",
			BaseBranch:   "main",
			GitHubToken:  "test-token",
		},
		workingDir:       tmpDir,
		providerName:     "claude",
		fixesByViolation: make(map[string][]FixRecord),
		allFixes:         make([]FixRecord, 0),
		createdPRs:       make([]CreatedPR, 0),
	}

	// Initialize GitHub client with mock server
	tracker.githubClient = &GitHubClient{
		owner:   "test-owner",
		repo:    "test-repo",
		token:   "test-token",
		baseURL: server.URL,
		client:  http.DefaultClient,
	}

	// Track some fixes
	v1 := violation.Violation{ID: "v1", Description: "Test violation"}
	incident := violation.Incident{LineNumber: 10}
	result := &fixer.FixResult{FilePath: "test.java", Cost: 0.01, TokensUsed: 10}

	err := tracker.TrackForPR(v1, incident, result)
	require.NoError(t, err)

	// Create a commit first (PR creation expects commits)
	testFile := filepath.Join(tmpDir, "test.java")
	require.NoError(t, os.WriteFile(testFile, []byte("fixed"), 0644))

	cmd := exec.Command("git", "add", "test.java")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "fix: test violation")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Note: We can't actually test Finalize() without being able to push
	// This test verifies the setup works, but full integration needs a real remote
	assert.Len(t, tracker.fixesByViolation, 1)
}

func TestPRTracker_CreatePRAtEnd_DryRun(t *testing.T) {
	server := setupMockGitHubServer(t)
	defer server.Close()

	tmpDir := setupTestRepoWithRemote(t, "https://github.com/test-owner/test-repo.git")

	tracker := &PRTracker{
		config: PRConfig{
			Strategy:     PRStrategyAtEnd,
			BranchPrefix: "test-branch",
			BaseBranch:   "main",
			GitHubToken:  "test-token",
		},
		workingDir:   tmpDir,
		providerName: "claude",
		fixesByViolation: map[string][]FixRecord{
			"v1": {
				{
					Violation: violation.Violation{ID: "v1", Description: "Test"},
					Incident:  violation.Incident{LineNumber: 10},
					Result:    &fixer.FixResult{FilePath: "test.java", Cost: 0.01, TokensUsed: 10},
					Timestamp: time.Now(),
				},
			},
		},
		allFixes:   make([]FixRecord, 0),
		createdPRs: make([]CreatedPR, 0),
	}

	tracker.githubClient = &GitHubClient{
		owner:   "test-owner",
		repo:    "test-repo",
		token:   "test-token",
		baseURL: server.URL,
		client:  http.DefaultClient,
	}

	// We can test the PR message formatting without actually creating the PR
	title := FormatPRTitleAtEnd(len(tracker.fixesByViolation))
	body := FormatPRBodyAtEnd(tracker.fixesByViolation, tracker.providerName)

	assert.Contains(t, title, "Konveyor")
	assert.Contains(t, body, "v1")
	assert.Contains(t, body, "claude")
}

func TestPRTracker_GetCreatedPRs_AfterTracking(t *testing.T) {
	tracker := &PRTracker{
		fixesByViolation: make(map[string][]FixRecord),
		allFixes:         make([]FixRecord, 0),
		createdPRs: []CreatedPR{
			{Number: 1, URL: "https://github.com/owner/repo/pull/1", BranchName: "branch1"},
		},
	}

	prs := tracker.GetCreatedPRs()
	assert.Len(t, prs, 1)
	assert.Equal(t, 1, prs[0].Number)
}

func TestCreateAndPushBranch_ValidBranchName(t *testing.T) {
	tmpDir := setupTestRepoWithRemote(t, "https://github.com/test-owner/test-repo.git")

	// Test that we can create a branch name
	branchName := fmt.Sprintf("test-branch-%d", time.Now().Unix())

	// Create the branch (but don't push since we can't push to fake remote)
	err := CreateBranch(tmpDir, branchName)
	require.NoError(t, err)

	// Verify branch was created
	cmd := exec.Command("git", "branch", "--list", branchName)
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(output), branchName)
}

func TestGitHubClient_WithMockServer(t *testing.T) {
	server := setupMockGitHubServer(t)
	defer server.Close()

	client := &GitHubClient{
		owner:   "test-owner",
		repo:    "test-repo",
		token:   "test-token",
		baseURL: server.URL,
		client:  http.DefaultClient,
	}

	t.Run("GetDefaultBranch", func(t *testing.T) {
		branch, err := client.GetDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("CreatePullRequest", func(t *testing.T) {
		req := PullRequestRequest{
			Title: "Test PR",
			Body:  "Test body",
			Head:  "test-branch",
			Base:  "main",
		}

		pr, err := client.CreatePullRequest(req)
		require.NoError(t, err)
		assert.Equal(t, 1, pr.Number)
		assert.Contains(t, pr.HTMLURL, "/pull/1")
	})

	t.Run("CreatePullRequest multiple times", func(t *testing.T) {
		req := PullRequestRequest{
			Title: "Test PR 2",
			Body:  "Test body 2",
			Head:  "test-branch-2",
			Base:  "main",
		}

		pr, err := client.CreatePullRequest(req)
		require.NoError(t, err)
		assert.Equal(t, 2, pr.Number) // Counter should increment
	})
}

func TestGitHubClient_ErrorHandling(t *testing.T) {
	t.Run("401 Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Bad credentials",
			})
		}))
		defer server.Close()

		client := &GitHubClient{
			owner:   "test-owner",
			repo:    "test-repo",
			token:   "bad-token",
			baseURL: server.URL,
			client:  http.DefaultClient,
		}

		req := PullRequestRequest{
			Title: "Test",
			Body:  "Test",
			Head:  "test",
			Base:  "main",
		}

		_, err := client.CreatePullRequest(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "Bad credentials")
	})

	t.Run("422 Validation Failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Validation Failed",
				"errors": []map[string]string{
					{"message": "A pull request already exists"},
				},
			})
		}))
		defer server.Close()

		client := &GitHubClient{
			owner:   "test-owner",
			repo:    "test-repo",
			token:   "test-token",
			baseURL: server.URL,
			client:  http.DefaultClient,
		}

		req := PullRequestRequest{
			Title: "Test",
			Body:  "Test",
			Head:  "test",
			Base:  "main",
		}

		_, err := client.CreatePullRequest(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "422")
		assert.Contains(t, err.Error(), "Validation Failed")
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GetDefaultBranch with error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := &GitHubClient{
			owner:   "test-owner",
			repo:    "nonexistent",
			token:   "test-token",
			baseURL: server.URL,
			client:  http.DefaultClient,
		}

		_, err := client.GetDefaultBranch()
		require.Error(t, err)
	})
}

func TestGitHubClient_RetryLogic(t *testing.T) {
	t.Run("Retry on 503 then succeed", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts == 1 {
				// First attempt: service unavailable
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			// Second attempt: success
			if r.URL.Path == "/repos/test-owner/test-repo/pulls" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"number":   1,
					"html_url": "https://github.com/test-owner/test-repo/pull/1",
				})
			}
		}))
		defer server.Close()

		client := &GitHubClient{
			owner:   "test-owner",
			repo:    "test-repo",
			token:   "test-token",
			baseURL: server.URL,
			client: &http.Client{
				Timeout: 5 * time.Second,
			},
		}

		req := PullRequestRequest{
			Title: "Test PR",
			Body:  "Test",
			Head:  "test",
			Base:  "main",
		}

		pr, err := client.CreatePullRequest(req)
		require.NoError(t, err)
		assert.Equal(t, 1, pr.Number)
		assert.Equal(t, 2, attempts) // Should have retried once
	})
}
