package gitutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// validBranchNameRegex matches valid git branch names
	// Allows alphanumeric, dashes, underscores, slashes, and dots
	// but prevents starting with dot, dash, or having consecutive dots
	validBranchNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)
)

// validateBranchName checks if a branch name is safe to use in git commands
func validateBranchName(branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if len(branchName) > 255 {
		return fmt.Errorf("branch name too long (max 255 characters)")
	}
	if strings.Contains(branchName, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}
	if strings.HasPrefix(branchName, ".") {
		return fmt.Errorf("branch name cannot start with '.'")
	}
	if strings.HasPrefix(branchName, "-") {
		return fmt.Errorf("branch name cannot start with '-'")
	}
	if !validBranchNameRegex.MatchString(branchName) {
		return fmt.Errorf("branch name contains invalid characters")
	}
	return nil
}

// validateFilePath checks if a file path is safe to use in git commands
// It prevents path traversal and ensures the path is within the working directory
func validateFilePath(workingDir, filePath string) (string, error) {
	// Clean the path to normalize it
	cleanPath := filepath.Clean(filePath)

	// Resolve to absolute path
	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(workingDir, cleanPath)
	}

	// Get absolute working directory
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve working directory: %w", err)
	}

	// Ensure the file is within the working directory
	if !strings.HasPrefix(absPath, absWorkingDir+string(filepath.Separator)) &&
		absPath != absWorkingDir {
		return "", fmt.Errorf("file path '%s' is outside working directory", filePath)
	}

	// Return the clean relative path
	relPath, err := filepath.Rel(absWorkingDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to make path relative: %w", err)
	}

	return relPath, nil
}

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
	// Validate and sanitize the file path to prevent command injection
	cleanPath, err := validateFilePath(workingDir, filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	cmd := exec.Command("git", "add", cleanPath)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage file %s: %w\nOutput: %s", cleanPath, err, string(output))
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

// ResetChanges resets all uncommitted changes in the working directory
func ResetChanges(workingDir string) error {
	cmd := exec.Command("git", "reset", "--hard", "HEAD")
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reset changes: %w\nOutput: %s", err, string(output))
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
	// Validate branch name to prevent command injection
	if err := validateBranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w\nOutput: %s", branchName, err, string(output))
	}
	return nil
}

// CheckoutBranch checks out an existing branch
func CheckoutBranch(workingDir string, branchName string) error {
	// Validate branch name to prevent command injection
	if err := validateBranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = workingDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w\nOutput: %s", branchName, err, string(output))
	}
	return nil
}

// PushBranch pushes a branch to remote origin
func PushBranch(workingDir string, branchName string) error {
	// Validate branch name to prevent command injection
	if err := validateBranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

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
