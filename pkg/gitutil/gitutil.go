package gitutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsGitRepository checks if the given directory is a git repository
func IsGitRepository(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// HasUncommittedChanges checks if there are uncommitted changes in the repository
func HasUncommittedChanges(dir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// StageFile stages a specific file for commit
func StageFile(workingDir string, filePath string) error {
	cmd := exec.Command("git", "add", filePath)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage file %s: %w\nOutput: %s", filePath, err, string(output))
	}
	return nil
}

// CreateCommit creates a git commit with the given message
func CreateCommit(workingDir string, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create commit: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// IsGitInstalled checks if git is installed and available in PATH
func IsGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(workingDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// CreateBranch creates and checks out a new branch
func CreateBranch(workingDir string, branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w\nOutput: %s", branchName, err, string(output))
	}
	return nil
}

// CheckoutBranch checks out an existing branch
func CheckoutBranch(workingDir string, branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w\nOutput: %s", branchName, err, string(output))
	}
	return nil
}

// PushBranch pushes a branch to remote origin
func PushBranch(workingDir string, branchName string) error {
	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push branch %s: %w\nOutput: %s", branchName, err, string(output))
	}
	return nil
}

// GetRemoteURL gets the URL for the 'origin' remote
func GetRemoteURL(workingDir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDefaultBranch attempts to determine the default branch (main/master)
// by checking refs/remotes/origin/HEAD
func GetDefaultBranch(workingDir string) (string, error) {
	// Try to get the default branch from remote HEAD
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err == nil {
		// Output is like "origin/main", extract "main"
		branch := strings.TrimSpace(string(output))
		if strings.HasPrefix(branch, "origin/") {
			return strings.TrimPrefix(branch, "origin/"), nil
		}
		return branch, nil
	}

	// Fallback: try common default branches
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "show-ref", "--verify", fmt.Sprintf("refs/remotes/origin/%s", branch))
		cmd.Dir = workingDir
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}
