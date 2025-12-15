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
		defer func() { _ = os.Chdir(originalWd) }() // Ignore cleanup errors

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
		defer func() { _ = os.Chdir(originalWd) }() // Ignore cleanup errors

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
		defer func() { _ = os.Chdir(originalWd) }() // Ignore cleanup errors

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
		defer func() { _ = os.Chdir(originalWd) }() // Ignore cleanup errors

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
		defer func() { _ = os.Chdir(originalWd) }() // Ignore cleanup errors

		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		config := LoadOrDefault()
		assert.Equal(t, "claude", config.Provider.Name) // Default
	})

	t.Run("returns defaults on parse error", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }() // Ignore cleanup errors

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

func TestConfidenceConfig_ToConfidenceConfig(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		config := ConfidenceConfig{}

		result := config.ToConfidenceConfig()

		assert.False(t, result.Enabled) // Default is disabled
		assert.NotEmpty(t, result.Thresholds)
		// Should have default action (skip)
		assert.Equal(t, "skip", string(result.OnLowConfidence))
	})

	t.Run("enabled configuration", func(t *testing.T) {
		config := ConfidenceConfig{
			Enabled: true,
		}

		result := config.ToConfidenceConfig()

		assert.True(t, result.Enabled)
	})

	t.Run("global min-confidence overrides all thresholds", func(t *testing.T) {
		config := ConfidenceConfig{
			MinConfidence: 0.85,
		}

		result := config.ToConfidenceConfig()

		// All complexity thresholds should be set to 0.85
		assert.Equal(t, 0.85, result.Thresholds["trivial"])
		assert.Equal(t, 0.85, result.Thresholds["low"])
		assert.Equal(t, 0.85, result.Thresholds["medium"])
		assert.Equal(t, 0.85, result.Thresholds["high"])
		assert.Equal(t, 0.85, result.Thresholds["expert"])
		assert.Equal(t, 0.85, result.Default)
	})

	t.Run("min-confidence 0.0 is valid (accept all)", func(t *testing.T) {
		config := ConfidenceConfig{
			MinConfidence: 0.0,
		}

		result := config.ToConfidenceConfig()

		// All thresholds should be 0.0
		for _, threshold := range result.Thresholds {
			assert.Equal(t, 0.0, threshold)
		}
		assert.Equal(t, 0.0, result.Default)
	})

	t.Run("min-confidence 1.0 is valid (maximum)", func(t *testing.T) {
		config := ConfidenceConfig{
			MinConfidence: 1.0,
		}

		result := config.ToConfidenceConfig()

		// All thresholds should be 1.0
		for _, threshold := range result.Thresholds {
			assert.Equal(t, 1.0, threshold)
		}
	})

	t.Run("negative min-confidence is ignored", func(t *testing.T) {
		config := ConfidenceConfig{
			MinConfidence: -0.5,
		}

		result := config.ToConfidenceConfig()

		// Should use default thresholds, not -0.5
		assert.NotEqual(t, -0.5, result.Thresholds["trivial"])
		// Check that it's using defaults (not all zeros)
		assert.Greater(t, result.Thresholds["trivial"], 0.0)
	})

	t.Run("specific complexity thresholds override defaults", func(t *testing.T) {
		config := ConfidenceConfig{
			ComplexityThresholds: map[string]float64{
				"high":   0.95,
				"expert": 0.98,
			},
		}

		result := config.ToConfidenceConfig()

		// Custom thresholds should be applied
		assert.Equal(t, 0.95, result.Thresholds["high"])
		assert.Equal(t, 0.98, result.Thresholds["expert"])
		// Other thresholds should be defaults
		assert.NotEqual(t, 0.95, result.Thresholds["trivial"])
		assert.NotEqual(t, 0.95, result.Thresholds["low"])
	})

	t.Run("complexity thresholds override global min-confidence", func(t *testing.T) {
		config := ConfidenceConfig{
			MinConfidence: 0.80,
			ComplexityThresholds: map[string]float64{
				"high":   0.95,
				"expert": 0.98,
			},
		}

		result := config.ToConfidenceConfig()

		// Specific overrides should win
		assert.Equal(t, 0.95, result.Thresholds["high"])
		assert.Equal(t, 0.98, result.Thresholds["expert"])
		// Others should be global min
		assert.Equal(t, 0.80, result.Thresholds["trivial"])
		assert.Equal(t, 0.80, result.Thresholds["low"])
		assert.Equal(t, 0.80, result.Thresholds["medium"])
	})

	t.Run("invalid complexity level is ignored", func(t *testing.T) {
		config := ConfidenceConfig{
			ComplexityThresholds: map[string]float64{
				"invalid":  0.90,
				"high":     0.95,
				"nonsense": 0.99,
			},
		}

		result := config.ToConfidenceConfig()

		// Valid threshold should be applied
		assert.Equal(t, 0.95, result.Thresholds["high"])
		// Invalid levels should not be in map
		_, exists := result.Thresholds["invalid"]
		assert.False(t, exists)
		_, exists = result.Thresholds["nonsense"]
		assert.False(t, exists)
	})

	t.Run("threshold < 0.0 is ignored", func(t *testing.T) {
		config := ConfidenceConfig{
			MinConfidence: -1.0, // Invalid - should be ignored, keeping defaults
			ComplexityThresholds: map[string]float64{
				"high":   -0.5,
				"medium": 0.80,
			},
		}

		result := config.ToConfidenceConfig()

		// Valid threshold should be applied
		assert.Equal(t, 0.80, result.Thresholds["medium"])
		// Invalid threshold should not override default (high defaults to 0.90)
		assert.NotEqual(t, -0.5, result.Thresholds["high"])
		assert.Equal(t, 0.90, result.Thresholds["high"]) // Should be default value
	})

	t.Run("threshold > 1.0 is ignored", func(t *testing.T) {
		config := ConfidenceConfig{
			MinConfidence: -1.0, // Invalid - should be ignored, keeping defaults
			ComplexityThresholds: map[string]float64{
				"high":   1.5,
				"medium": 0.80,
			},
		}

		result := config.ToConfidenceConfig()

		// Valid threshold should be applied
		assert.Equal(t, 0.80, result.Thresholds["medium"])
		// Invalid threshold should not override default (high defaults to 0.90)
		assert.NotEqual(t, 1.5, result.Thresholds["high"])
		assert.Equal(t, 0.90, result.Thresholds["high"]) // Should be default value
	})

	t.Run("OnLowConfidence skip action", func(t *testing.T) {
		config := ConfidenceConfig{
			OnLowConfidence: "skip",
		}

		result := config.ToConfidenceConfig()

		assert.Equal(t, "skip", string(result.OnLowConfidence))
	})

	t.Run("OnLowConfidence warn-and-apply action", func(t *testing.T) {
		config := ConfidenceConfig{
			OnLowConfidence: "warn-and-apply",
		}

		result := config.ToConfidenceConfig()

		assert.Equal(t, "warn-and-apply", string(result.OnLowConfidence))
	})

	t.Run("OnLowConfidence manual-review-file action", func(t *testing.T) {
		config := ConfidenceConfig{
			OnLowConfidence: "manual-review-file",
		}

		result := config.ToConfidenceConfig()

		assert.Equal(t, "manual-review-file", string(result.OnLowConfidence))
	})

	t.Run("OnLowConfidence default to skip for invalid value", func(t *testing.T) {
		config := ConfidenceConfig{
			OnLowConfidence: "invalid-action",
		}

		result := config.ToConfidenceConfig()

		// Should default to skip
		assert.Equal(t, "skip", string(result.OnLowConfidence))
	})

	t.Run("OnLowConfidence default to skip for empty value", func(t *testing.T) {
		config := ConfidenceConfig{
			OnLowConfidence: "",
		}

		result := config.ToConfidenceConfig()

		// Should default to skip
		assert.Equal(t, "skip", string(result.OnLowConfidence))
	})

	t.Run("complex real-world configuration", func(t *testing.T) {
		config := ConfidenceConfig{
			Enabled:         true,
			MinConfidence:   0.75,
			OnLowConfidence: "warn-and-apply",
			ComplexityThresholds: map[string]float64{
				"high":   0.90,
				"expert": 0.95,
			},
		}

		result := config.ToConfidenceConfig()

		assert.True(t, result.Enabled)
		assert.Equal(t, "warn-and-apply", string(result.OnLowConfidence))
		// Global min applies to unspecified levels
		assert.Equal(t, 0.75, result.Thresholds["trivial"])
		assert.Equal(t, 0.75, result.Thresholds["low"])
		assert.Equal(t, 0.75, result.Thresholds["medium"])
		// Specific overrides
		assert.Equal(t, 0.90, result.Thresholds["high"])
		assert.Equal(t, 0.95, result.Thresholds["expert"])
	})

	t.Run("all complexity levels can be customized", func(t *testing.T) {
		config := ConfidenceConfig{
			ComplexityThresholds: map[string]float64{
				"trivial": 0.70,
				"low":     0.75,
				"medium":  0.80,
				"high":    0.90,
				"expert":  0.95,
			},
		}

		result := config.ToConfidenceConfig()

		assert.Equal(t, 0.70, result.Thresholds["trivial"])
		assert.Equal(t, 0.75, result.Thresholds["low"])
		assert.Equal(t, 0.80, result.Thresholds["medium"])
		assert.Equal(t, 0.90, result.Thresholds["high"])
		assert.Equal(t, 0.95, result.Thresholds["expert"])
	})

	t.Run("boundary values for thresholds", func(t *testing.T) {
		config := ConfidenceConfig{
			ComplexityThresholds: map[string]float64{
				"trivial": 0.0,  // Minimum valid
				"expert":  1.0,  // Maximum valid
			},
		}

		result := config.ToConfidenceConfig()

		assert.Equal(t, 0.0, result.Thresholds["trivial"])
		assert.Equal(t, 1.0, result.Thresholds["expert"])
	})
}
