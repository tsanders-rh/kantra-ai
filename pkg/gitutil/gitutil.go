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
