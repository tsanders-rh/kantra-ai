package gitutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	// GitHubAPITimeout is the timeout for GitHub API requests
	GitHubAPITimeout = 30 * time.Second

	// maxResponseSize is the maximum size of GitHub API responses to prevent memory exhaustion
	// GitHub API responses are typically small, 10MB is generous
	maxResponseSize = 10 * 1024 * 1024 // 10MB
)

// GitHubClient handles GitHub API interactions
type GitHubClient struct {
	token   string
	owner   string
	repo    string
	baseURL string
	client  *http.Client
}

// PullRequestRequest represents a GitHub PR creation request
type PullRequestRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"` // branch name
	Base  string `json:"base"` // target branch
}

// PullRequestResponse represents a GitHub PR creation response
type PullRequestResponse struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
}

// GitHubError represents an error from the GitHub API
type GitHubError struct {
	Message string `json:"message"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
	StatusCode int
}

func (e *GitHubError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("GitHub API error (HTTP %d): %s - %s", e.StatusCode, e.Message, e.Errors[0].Message)
	}
	return fmt.Sprintf("GitHub API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(workingDir string, token string) (*GitHubClient, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// Get remote URL
	remoteURL, err := GetRemoteURL(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote URL: %w", err)
	}

	// Parse owner and repo from URL
	owner, repo, err := ParseGitHubURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	return &GitHubClient{
		token:   token,
		owner:   owner,
		repo:    repo,
		baseURL: "https://api.github.com",
		client: &http.Client{
			Timeout: GitHubAPITimeout,
		},
	}, nil
}

// ParseGitHubURL extracts owner and repo from a GitHub remote URL
// Supports: https://github.com/owner/repo.git, git@github.com:owner/repo.git
func ParseGitHubURL(remoteURL string) (owner, repo string, err error) {
	// Remove trailing whitespace
	remoteURL = strings.TrimSpace(remoteURL)

	// HTTPS format: https://github.com/owner/repo.git
	httpsRegex := regexp.MustCompile(`https?://github\.com/([^/]+)/([^/]+?)(\.git)?$`)
	if matches := httpsRegex.FindStringSubmatch(remoteURL); matches != nil {
		return matches[1], matches[2], nil
	}

	// SSH format: git@github.com:owner/repo.git
	sshRegex := regexp.MustCompile(`git@github\.com:([^/]+)/([^/]+?)(\.git)?$`)
	if matches := sshRegex.FindStringSubmatch(remoteURL); matches != nil {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("not a valid GitHub URL: %s", remoteURL)
}

// CreatePullRequest creates a new pull request on GitHub
func (c *GitHubClient) CreatePullRequest(req PullRequestRequest) (*PullRequestResponse, error) {
	// Build API URL
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, c.owner, c.repo)

	// Marshal request body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request with retry logic
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		resp, err = c.client.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}

		// Success or non-retriable error
		if resp.StatusCode != http.StatusServiceUnavailable &&
			resp.StatusCode != http.StatusBadGateway &&
			resp.StatusCode != http.StatusGatewayTimeout {
			break
		}

		// Close response body before retrying
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d (attempt %d)", resp.StatusCode, attempt+1)
	}

	if resp == nil {
		return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
	}
	// Ensure response body is always closed (whether we broke or completed loop)
	defer resp.Body.Close()

	// Read response body with size limit to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusCreated {
		var ghErr GitHubError
		if err := json.Unmarshal(respBody, &ghErr); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		ghErr.StatusCode = resp.StatusCode
		return nil, &ghErr
	}

	// Parse successful response
	var prResp PullRequestResponse
	if err := json.Unmarshal(respBody, &prResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &prResp, nil
}

// GetDefaultBranch gets the default branch (main/master) from GitHub
func (c *GitHubClient) GetDefaultBranch() (string, error) {
	// Build API URL
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, c.owner, c.repo)

	// Create HTTP request
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to get repository info: %w", err)
	}
	defer resp.Body.Close()

	// Read response with size limit to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var ghErr GitHubError
		if err := json.Unmarshal(respBody, &ghErr); err != nil {
			return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		ghErr.StatusCode = resp.StatusCode
		return "", &ghErr
	}

	// Parse response to get default branch
	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(respBody, &repoInfo); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return repoInfo.DefaultBranch, nil
}
