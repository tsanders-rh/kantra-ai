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
