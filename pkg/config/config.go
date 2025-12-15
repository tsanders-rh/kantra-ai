package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsanders/kantra-ai/pkg/confidence"
	"gopkg.in/yaml.v3"
)

// Config represents the kantra-ai configuration
type Config struct {
	// Provider settings
	Provider ProviderConfig `yaml:"provider"`

	// Input/Output paths
	Paths PathsConfig `yaml:"paths"`

	// Cost and effort limits
	Limits LimitsConfig `yaml:"limits"`

	// Filtering options
	Filters FiltersConfig `yaml:"filters"`

	// Git integration
	Git GitConfig `yaml:"git"`

	// Verification settings
	Verification VerificationConfig `yaml:"verification"`

	// Confidence threshold settings
	Confidence ConfidenceConfig `yaml:"confidence"`

	// Prompt template settings
	Prompts PromptsConfig `yaml:"prompts"`

	// General settings
	DryRun bool `yaml:"dry-run"`
}

// ProviderConfig holds AI provider settings
type ProviderConfig struct {
	Name  string `yaml:"name"`  // claude, openai
	Model string `yaml:"model"` // optional, provider-specific model
}

// PathsConfig holds input/output path settings
type PathsConfig struct {
	Analysis string `yaml:"analysis"` // Path to Konveyor output.yaml
	Input    string `yaml:"input"`    // Path to source code directory
}

// LimitsConfig holds cost and effort limits
type LimitsConfig struct {
	MaxCost   float64 `yaml:"max-cost"`   // Maximum cost in USD
	MaxEffort int     `yaml:"max-effort"` // Maximum effort level (0 = no limit)
}

// FiltersConfig holds violation filtering options
type FiltersConfig struct {
	Categories   []string `yaml:"categories"`    // Filter by category (mandatory, optional, potential)
	ViolationIDs []string `yaml:"violation-ids"` // Filter by specific violation IDs
}

// GitConfig holds git integration settings
type GitConfig struct {
	CommitStrategy string `yaml:"commit-strategy"` // per-violation, per-incident, at-end
	CreatePR       bool   `yaml:"create-pr"`       // Automatically create pull requests
	BranchPrefix   string `yaml:"branch-prefix"`   // Custom branch name prefix
}

// VerificationConfig holds build/test verification settings
type VerificationConfig struct {
	Enabled  bool   `yaml:"enabled"`   // Enable verification
	Type     string `yaml:"type"`      // build, test
	Strategy string `yaml:"strategy"`  // per-fix, per-violation, at-end
	Command  string `yaml:"command"`   // Custom verification command
	FailFast bool   `yaml:"fail-fast"` // Stop on first failure
}

// ConfidenceConfig holds confidence threshold settings
type ConfidenceConfig struct {
	Enabled           bool               `yaml:"enabled"`             // Enable confidence filtering
	MinConfidence     float64            `yaml:"min-confidence"`      // Global minimum confidence (overrides complexity thresholds)
	OnLowConfidence   string             `yaml:"on-low-confidence"`   // skip, warn-and-apply, manual-review-file
	ComplexityThresholds map[string]float64 `yaml:"complexity-thresholds,omitempty"` // Override specific complexity thresholds
}

// PromptsConfig holds custom prompt template paths
type PromptsConfig struct {
	SingleFixTemplate string `yaml:"single-fix-template"` // Path to custom single-fix prompt template (base/fallback)
	BatchFixTemplate  string `yaml:"batch-fix-template"`  // Path to custom batch-fix prompt template (base/fallback)
	LanguageTemplates map[string]LanguageTemplateConfig `yaml:"language-templates,omitempty"` // Language-specific template overrides
}

// LanguageTemplateConfig holds template paths for a specific language
type LanguageTemplateConfig struct {
	SingleFix string `yaml:"single-fix"` // Path to language-specific single-fix template
	BatchFix  string `yaml:"batch-fix"`  // Path to language-specific batch-fix template
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Provider: ProviderConfig{
			Name: "claude",
		},
		Limits: LimitsConfig{
			MaxCost:   0, // No limit
			MaxEffort: 0, // No limit
		},
		Verification: VerificationConfig{
			Enabled:  false,
			Type:     "test",
			Strategy: "at-end",
			FailFast: true,
		},
		Confidence: ConfidenceConfig{
			Enabled:         false, // Disabled by default for backward compatibility
			MinConfidence:   0.0,   // 0.0 means use complexity-based thresholds
			OnLowConfidence: "skip",
		},
		DryRun: false,
	}
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w\n\n"+
			"Please check that the file is valid YAML and follows the expected format.\n"+
			"See README.md for example configuration.", path, err)
	}

	return config, nil
}

// FindConfigFile searches for a config file in common locations
// Returns the path to the first config file found, or empty string if none found
func FindConfigFile() string {
	// Check current directory first
	candidates := []string{
		".kantra-ai.yaml",
		".kantra-ai.yml",
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		for _, candidate := range candidates {
			path := filepath.Join(homeDir, candidate)
			if fileExists(path) {
				return path
			}
		}
	}

	return ""
}

// LoadOrDefault attempts to load a config file, falling back to defaults
func LoadOrDefault() *Config {
	configPath := FindConfigFile()
	if configPath == "" {
		return DefaultConfig()
	}

	config, err := Load(configPath)
	if err != nil {
		// Log the error but return defaults
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config from %s: %v\n", configPath, err)
		fmt.Fprintf(os.Stderr, "Using default configuration.\n\n")
		return DefaultConfig()
	}

	return config
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ToConfidenceConfig converts ConfidenceConfig to confidence.Config
func (c *ConfidenceConfig) ToConfidenceConfig() confidence.Config {
	conf := confidence.DefaultConfig()

	// Apply user configuration
	conf.Enabled = c.Enabled

	// If global min-confidence is set, validate and apply it to all complexity levels
	// Note: We check >= 0.0 to allow explicitly setting 0.0 (accept all fixes)
	// Negative values are invalid and ignored
	if c.MinConfidence >= 0.0 && c.MinConfidence <= 1.0 {
		// Validate upper bound
		if c.MinConfidence > 1.0 {
			fmt.Fprintf(os.Stderr, "Warning: min-confidence %.2f > 1.0, clamping to 1.0\n", c.MinConfidence)
			c.MinConfidence = 1.0
		}

		// Apply to all thresholds (0.0 means accept all)
		for level := range conf.Thresholds {
			conf.Thresholds[level] = c.MinConfidence
		}
		conf.Default = c.MinConfidence
	} else if c.MinConfidence < 0.0 {
		fmt.Fprintf(os.Stderr, "Warning: min-confidence %.2f < 0.0, ignoring (use 0.0-1.0)\n", c.MinConfidence)
	}

	// Override specific complexity thresholds if provided
	if len(c.ComplexityThresholds) > 0 {
		for level, threshold := range c.ComplexityThresholds {
			// Validate complexity level
			if !confidence.IsValidComplexity(level) {
				fmt.Fprintf(os.Stderr, "Warning: invalid complexity level '%s', skipping. Valid levels: %v\n",
					level, confidence.ValidComplexityLevels())
				continue
			}

			// Validate threshold range
			if threshold < 0.0 || threshold > 1.0 {
				fmt.Fprintf(os.Stderr, "Warning: threshold %.2f for %s out of range [0.0, 1.0], skipping\n",
					threshold, level)
				continue
			}

			conf.Thresholds[level] = threshold
		}
	}

	// Set action
	switch c.OnLowConfidence {
	case "skip":
		conf.OnLowConfidence = confidence.ActionSkip
	case "warn-and-apply":
		conf.OnLowConfidence = confidence.ActionWarnAndApply
	case "manual-review-file":
		conf.OnLowConfidence = confidence.ActionManualReviewFile
	default:
		conf.OnLowConfidence = confidence.ActionSkip
	}

	return conf
}
