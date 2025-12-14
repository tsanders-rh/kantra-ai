package config

import (
	"fmt"
	"os"
	"path/filepath"

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
