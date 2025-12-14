package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "claude", config.Provider.Name)
	assert.Equal(t, "", config.Provider.Model)
	assert.Equal(t, float64(0), config.Limits.MaxCost)
	assert.Equal(t, 0, config.Limits.MaxEffort)
	assert.False(t, config.Verification.Enabled)
	assert.Equal(t, "test", config.Verification.Type)
	assert.Equal(t, "at-end", config.Verification.Strategy)
	assert.True(t, config.Verification.FailFast)
	assert.False(t, config.DryRun)
}

func TestLoad(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".kantra-ai.yaml")

		configContent := `
provider:
  name: openai
  model: gpt-4

paths:
  analysis: ./analysis/output.yaml
  input: ./src

limits:
  max-cost: 5.00
  max-effort: 3

filters:
  categories:
    - mandatory
    - optional
  violation-ids:
    - test-001
    - test-002

git:
  commit-strategy: per-violation
  create-pr: true
  branch-prefix: feature/fixes

verification:
  enabled: true
  type: build
  strategy: per-fix
  command: make test
  fail-fast: false

dry-run: true
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "openai", config.Provider.Name)
		assert.Equal(t, "gpt-4", config.Provider.Model)
		assert.Equal(t, "./analysis/output.yaml", config.Paths.Analysis)
		assert.Equal(t, "./src", config.Paths.Input)
		assert.Equal(t, 5.00, config.Limits.MaxCost)
		assert.Equal(t, 3, config.Limits.MaxEffort)
		assert.Equal(t, []string{"mandatory", "optional"}, config.Filters.Categories)
		assert.Equal(t, []string{"test-001", "test-002"}, config.Filters.ViolationIDs)
		assert.Equal(t, "per-violation", config.Git.CommitStrategy)
		assert.True(t, config.Git.CreatePR)
		assert.Equal(t, "feature/fixes", config.Git.BranchPrefix)
		assert.True(t, config.Verification.Enabled)
		assert.Equal(t, "build", config.Verification.Type)
		assert.Equal(t, "per-fix", config.Verification.Strategy)
		assert.Equal(t, "make test", config.Verification.Command)
		assert.False(t, config.Verification.FailFast)
		assert.True(t, config.DryRun)
	})

	t.Run("partial config file with defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".kantra-ai.yaml")

		configContent := `
provider:
  name: claude

limits:
  max-cost: 10.00
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := Load(configPath)
		require.NoError(t, err)

		// Specified values
		assert.Equal(t, "claude", config.Provider.Name)
		assert.Equal(t, 10.00, config.Limits.MaxCost)

		// Default values
		assert.Equal(t, "", config.Provider.Model)
		assert.Equal(t, 0, config.Limits.MaxEffort)
		assert.False(t, config.DryRun)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := Load("/nonexistent/config.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".kantra-ai.yaml")

		invalidYAML := `
provider:
  name: claude
  invalid yaml here [[[
`
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		_, err = Load(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
}

func TestFindConfigFile(t *testing.T) {
	t.Run("finds config in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		configPath := filepath.Join(tmpDir, ".kantra-ai.yaml")
		err = os.WriteFile(configPath, []byte("provider:\n  name: claude\n"), 0644)
		require.NoError(t, err)

		found := FindConfigFile()
		assert.Equal(t, ".kantra-ai.yaml", found)
	})

	t.Run("prefers .yaml over .yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create both files
		yamlPath := filepath.Join(tmpDir, ".kantra-ai.yaml")
		ymlPath := filepath.Join(tmpDir, ".kantra-ai.yml")
		err = os.WriteFile(yamlPath, []byte("provider:\n  name: claude\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(ymlPath, []byte("provider:\n  name: openai\n"), 0644)
		require.NoError(t, err)

		found := FindConfigFile()
		assert.Equal(t, ".kantra-ai.yaml", found) // Should prefer .yaml
	})

	t.Run("returns empty string when no config found", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		found := FindConfigFile()
		assert.Equal(t, "", found)
	})
}

func TestLoadOrDefault(t *testing.T) {
	t.Run("loads config when found", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		configContent := `
provider:
  name: openai
  model: gpt-4
`
		configPath := filepath.Join(tmpDir, ".kantra-ai.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config := LoadOrDefault()
		assert.Equal(t, "openai", config.Provider.Name)
		assert.Equal(t, "gpt-4", config.Provider.Model)
	})

	t.Run("returns defaults when no config found", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		config := LoadOrDefault()
		assert.Equal(t, "claude", config.Provider.Name) // Default
	})

	t.Run("returns defaults on parse error", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create invalid config
		configPath := filepath.Join(tmpDir, ".kantra-ai.yaml")
		err = os.WriteFile(configPath, []byte("invalid yaml [[["), 0644)
		require.NoError(t, err)

		config := LoadOrDefault()
		assert.Equal(t, "claude", config.Provider.Name) // Should fall back to defaults
	})
}

func TestFileExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)

		assert.True(t, fileExists(filePath))
	})

	t.Run("returns false for nonexistent file", func(t *testing.T) {
		assert.False(t, fileExists("/nonexistent/file.txt"))
	})

	t.Run("returns false for directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		// fileExists returns true for directories too (os.Stat succeeds)
		assert.True(t, fileExists(tmpDir))
	})
}
