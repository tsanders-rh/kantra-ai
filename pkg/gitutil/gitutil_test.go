package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestGitRepo creates a temporary git repository for testing
func createTestGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	require.NoError(t, err, "failed to init git repo")

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run() // Ignore error, not critical for test setup

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	_ = cmd.Run() // Ignore error, not critical for test setup

	return tmpDir
}

// createAndCommitFile creates a file with content and commits it to the git repo
func createAndCommitFile(t *testing.T, repoDir, filePath, content string) error {
	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}

	// Git add
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Git commit
	cmd = exec.Command("git", "commit", "-m", "test commit")
	cmd.Dir = repoDir
	return cmd.Run()
}

func TestIsGitRepository(t *testing.T) {
	t.Run("directory with .git", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)
		assert.True(t, IsGitRepository(tmpDir))
	})

	t.Run("directory without .git", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.False(t, IsGitRepository(tmpDir))
	})

	t.Run("non-existent directory", func(t *testing.T) {
		assert.False(t, IsGitRepository("/nonexistent/directory"))
	})
}

func TestIsGitInstalled(t *testing.T) {
	// This test will pass on systems with git installed
	// which is required for running the tool anyway
	result := IsGitInstalled()
	assert.True(t, result, "git should be installed for tests to run")
}

func TestHasUncommittedChanges(t *testing.T) {
	t.Run("clean repository", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)

		// Create and commit a file to have a valid repo
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("initial"), 0644)
		require.NoError(t, err)

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "commit", "-m", "initial commit")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		hasChanges, err := HasUncommittedChanges(tmpDir)
		require.NoError(t, err)
		assert.False(t, hasChanges)
	})

	t.Run("repository with modified files", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)

		// Create and commit a file
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("initial"), 0644)
		require.NoError(t, err)

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "commit", "-m", "initial commit")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		// Modify the file
		err = os.WriteFile(testFile, []byte("modified"), 0644)
		require.NoError(t, err)

		hasChanges, err := HasUncommittedChanges(tmpDir)
		require.NoError(t, err)
		assert.True(t, hasChanges)
	})

	t.Run("repository with staged files", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)

		// Create and stage a file
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		hasChanges, err := HasUncommittedChanges(tmpDir)
		require.NoError(t, err)
		assert.True(t, hasChanges)
	})
}

func TestStageFile(t *testing.T) {
	t.Run("stage existing file", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)

		// Create a file
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		// Stage the file
		err = StageFile(tmpDir, "test.txt")
		assert.NoError(t, err)

		// Verify it's staged
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = tmpDir
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "A  test.txt")
	})

	t.Run("stage non-existent file", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)

		err := StageFile(tmpDir, "nonexistent.txt")
		assert.Error(t, err)
	})
}

func TestCreateCommit(t *testing.T) {
	t.Run("create commit with staged changes", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)

		// Configure git user for this repo
		cmd := exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		// Create and stage a file
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		err = StageFile(tmpDir, "test.txt")
		require.NoError(t, err)

		// Create commit
		err = CreateCommit(tmpDir, "Test commit message")
		assert.NoError(t, err)

		// Verify commit was created
		cmd = exec.Command("git", "log", "--oneline")
		cmd.Dir = tmpDir
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "Test commit message")
	})

	t.Run("create commit with no staged changes", func(t *testing.T) {
		tmpDir := createTestGitRepo(t)

		// Configure git user
		cmd := exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		// Try to commit with nothing staged
		err := CreateCommit(tmpDir, "Empty commit")
		assert.Error(t, err)
	})
}
