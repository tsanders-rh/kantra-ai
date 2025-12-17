package gitutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS with .git",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "HTTPS without .git",
			url:       "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH with .git",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH without .git",
			url:       "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:    "Invalid URL",
			url:     "not-a-github-url",
			wantErr: true,
		},
		{
			name:    "GitLab URL",
			url:     "https://gitlab.com/owner/repo.git",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseGitHubURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOwner, owner)
				assert.Equal(t, tt.wantRepo, repo)
			}
		})
	}
}

func TestGitHubClient_CreatePullRequest(t *testing.T) {
	t.Run("successful PR creation", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "/repos/test-owner/test-repo/pulls", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

			// Send mock response
			w.WriteHeader(http.StatusCreated)
			resp := PullRequestResponse{
				Number:  123,
				HTMLURL: "https://github.com/test-owner/test-repo/pull/123",
				State:   "open",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create PR
		req := PullRequestRequest{
			Title: "Test PR",
			Body:  "Test description",
			Head:  "feature-branch",
			Base:  "main",
		}
		resp, err := client.CreatePullRequest(req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 123, resp.Number)
		assert.Equal(t, "https://github.com/test-owner/test-repo/pull/123", resp.HTMLURL)
		assert.Equal(t, "open", resp.State)
	})

	t.Run("API error response", func(t *testing.T) {
		// Create mock server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			errResp := map[string]interface{}{
				"message": "Validation Failed",
				"errors": []map[string]string{
					{"message": "A pull request already exists"},
				},
			}
			_ = json.NewEncoder(w).Encode(errResp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create PR
		req := PullRequestRequest{
			Title: "Test PR",
			Body:  "Test description",
			Head:  "feature-branch",
			Base:  "main",
		}
		_, err := client.CreatePullRequest(req)

		// Assert error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "422")
		assert.Contains(t, err.Error(), "Validation Failed")
	})

	t.Run("unauthorized error", func(t *testing.T) {
		// Create mock server that returns 401
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			errResp := map[string]string{
				"message": "Bad credentials",
			}
			_ = json.NewEncoder(w).Encode(errResp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "invalid-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create PR
		req := PullRequestRequest{
			Title: "Test PR",
			Body:  "Test description",
			Head:  "feature-branch",
			Base:  "main",
		}
		_, err := client.CreatePullRequest(req)

		// Assert error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "Bad credentials")
	})
}

func TestGitHubClient_GetDefaultBranch(t *testing.T) {
	t.Run("successful default branch retrieval", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "/repos/test-owner/test-repo", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			// Send mock response
			w.WriteHeader(http.StatusOK)
			resp := map[string]string{
				"default_branch": "main",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Get default branch
		branch, err := client.GetDefaultBranch()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("master as default branch", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			resp := map[string]string{
				"default_branch": "master",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Get default branch
		branch, err := client.GetDefaultBranch()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})
}

func TestGitHubClient_CreateCommitStatus(t *testing.T) {
	t.Run("successful status creation - success state", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "/repos/test-owner/test-repo/statuses/abc123", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

			// Verify request body
			var reqBody CommitStatusRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, StatusStateSuccess, reqBody.State)
			assert.Equal(t, "kantra-ai/verify-build", reqBody.Context)
			assert.Equal(t, "Build passed", reqBody.Description)

			// Send mock response
			w.WriteHeader(http.StatusCreated)
			resp := CommitStatusResponse{
				State:       "success",
				Description: "Build passed",
				Context:     "kantra-ai/verify-build",
				CreatedAt:   "2025-01-15T10:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create status
		req := CommitStatusRequest{
			State:       StatusStateSuccess,
			Description: "Build passed",
			Context:     "kantra-ai/verify-build",
		}
		resp, err := client.CreateCommitStatus("abc123", req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "success", resp.State)
		assert.Equal(t, "Build passed", resp.Description)
		assert.Equal(t, "kantra-ai/verify-build", resp.Context)
	})

	t.Run("successful status creation - failure state", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request body
			var reqBody CommitStatusRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, StatusStateFailure, reqBody.State)

			// Send mock response
			w.WriteHeader(http.StatusCreated)
			resp := CommitStatusResponse{
				State:       "failure",
				Description: "Build failed",
				Context:     "kantra-ai/verify-test",
				CreatedAt:   "2025-01-15T10:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create status
		req := CommitStatusRequest{
			State:       StatusStateFailure,
			Description: "Build failed",
			Context:     "kantra-ai/verify-test",
		}
		resp, err := client.CreateCommitStatus("def456", req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "failure", resp.State)
	})

	t.Run("successful status creation - pending state", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request body
			var reqBody CommitStatusRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, StatusStatePending, reqBody.State)

			// Send mock response
			w.WriteHeader(http.StatusCreated)
			resp := CommitStatusResponse{
				State:       "pending",
				Description: "Verification running",
				Context:     "kantra-ai/verify-build",
				CreatedAt:   "2025-01-15T10:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create status
		req := CommitStatusRequest{
			State:       StatusStatePending,
			Description: "Verification running",
			Context:     "kantra-ai/verify-build",
		}
		resp, err := client.CreateCommitStatus("ghi789", req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "pending", resp.State)
	})

	t.Run("unauthorized error", func(t *testing.T) {
		// Create mock server that returns 401
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			errResp := map[string]string{
				"message": "Bad credentials",
			}
			_ = json.NewEncoder(w).Encode(errResp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "invalid-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create status
		req := CommitStatusRequest{
			State:       StatusStateSuccess,
			Description: "Build passed",
			Context:     "kantra-ai/verify-build",
		}
		_, err := client.CreateCommitStatus("abc123", req)

		// Assert error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "Bad credentials")
	})

	t.Run("with target URL", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request body includes target_url
			var reqBody CommitStatusRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, "https://example.com/build/123", reqBody.TargetURL)

			// Send mock response
			w.WriteHeader(http.StatusCreated)
			resp := CommitStatusResponse{
				State:       "success",
				Description: "Build passed",
				Context:     "kantra-ai/verify-build",
				CreatedAt:   "2025-01-15T10:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create status with target URL
		req := CommitStatusRequest{
			State:       StatusStateSuccess,
			Description: "Build passed",
			Context:     "kantra-ai/verify-build",
			TargetURL:   "https://example.com/build/123",
		}
		resp, err := client.CreateCommitStatus("abc123", req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "success", resp.State)
	})
}

func TestGitHubClient_CreateReviewComment(t *testing.T) {
	t.Run("successful review comment creation", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "/repos/test-owner/test-repo/pulls/123/comments", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

			// Verify request body
			var reqBody ReviewCommentRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, "⚠️ Low confidence fix", reqBody.Body)
			assert.Equal(t, "abc123", reqBody.CommitID)
			assert.Equal(t, "src/Main.java", reqBody.Path)
			assert.Equal(t, 42, reqBody.Line)
			assert.Equal(t, "RIGHT", reqBody.Side)

			// Send mock response
			w.WriteHeader(http.StatusCreated)
			resp := ReviewCommentResponse{
				ID:        456,
				Body:      "⚠️ Low confidence fix",
				Path:      "src/Main.java",
				Line:      42,
				CommitID:  "abc123",
				HTMLURL:   "https://github.com/test-owner/test-repo/pull/123#discussion_r456",
				CreatedAt: "2025-01-15T10:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create review comment
		req := ReviewCommentRequest{
			Body:     "⚠️ Low confidence fix",
			CommitID: "abc123",
			Path:     "src/Main.java",
			Line:     42,
			Side:     "RIGHT",
		}
		resp, err := client.CreateReviewComment(123, req)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 456, resp.ID)
		assert.Equal(t, "⚠️ Low confidence fix", resp.Body)
		assert.Equal(t, "src/Main.java", resp.Path)
		assert.Equal(t, 42, resp.Line)
		assert.Equal(t, "abc123", resp.CommitID)
		assert.Contains(t, resp.HTMLURL, "discussion_r456")
	})

	t.Run("unauthorized error", func(t *testing.T) {
		// Create mock server that returns 401
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			errResp := map[string]string{
				"message": "Bad credentials",
			}
			_ = json.NewEncoder(w).Encode(errResp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "invalid-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create review comment
		req := ReviewCommentRequest{
			Body:     "Test comment",
			CommitID: "abc123",
			Path:     "file.go",
			Line:     10,
			Side:     "RIGHT",
		}
		_, err := client.CreateReviewComment(123, req)

		// Assert error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "Bad credentials")
	})

	t.Run("validation error - invalid path", func(t *testing.T) {
		// Create mock server that returns 422
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			errResp := map[string]interface{}{
				"message": "Validation Failed",
				"errors": []map[string]string{
					{"message": "path does not exist in diff"},
				},
			}
			_ = json.NewEncoder(w).Encode(errResp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create review comment with invalid path
		req := ReviewCommentRequest{
			Body:     "Test comment",
			CommitID: "abc123",
			Path:     "nonexistent.go",
			Line:     10,
			Side:     "RIGHT",
		}
		_, err := client.CreateReviewComment(123, req)

		// Assert error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "422")
		assert.Contains(t, err.Error(), "Validation Failed")
	})

	t.Run("retry on 503", func(t *testing.T) {
		attempts := 0
		// Create mock server that fails twice then succeeds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			// Third attempt succeeds
			w.WriteHeader(http.StatusCreated)
			resp := ReviewCommentResponse{
				ID:       789,
				Body:     "Test comment",
				Path:     "file.go",
				Line:     10,
				CommitID: "abc123",
				HTMLURL:  "https://github.com/test-owner/test-repo/pull/123#discussion_r789",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := &GitHubClient{
			token:   "test-token",
			owner:   "test-owner",
			repo:    "test-repo",
			baseURL: server.URL,
			client:  server.Client(),
		}

		// Create review comment
		req := ReviewCommentRequest{
			Body:     "Test comment",
			CommitID: "abc123",
			Path:     "file.go",
			Line:     10,
			Side:     "RIGHT",
		}
		resp, err := client.CreateReviewComment(123, req)

		// Assert success after retries
		require.NoError(t, err)
		assert.Equal(t, 789, resp.ID)
		assert.Equal(t, 3, attempts, "should have retried twice before succeeding")
	})
}
