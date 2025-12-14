package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderPresets(t *testing.T) {
	tests := []struct {
		name         string
		presetName   string
		expectedURL  string
		expectExists bool
	}{
		{
			name:         "groq preset exists",
			presetName:   "groq",
			expectedURL:  "https://api.groq.com/openai/v1",
			expectExists: true,
		},
		{
			name:         "together preset exists",
			presetName:   "together",
			expectedURL:  "https://api.together.xyz/v1",
			expectExists: true,
		},
		{
			name:         "anyscale preset exists",
			presetName:   "anyscale",
			expectedURL:  "https://api.endpoints.anyscale.com/v1",
			expectExists: true,
		},
		{
			name:         "perplexity preset exists",
			presetName:   "perplexity",
			expectedURL:  "https://api.perplexity.ai",
			expectExists: true,
		},
		{
			name:         "ollama preset exists",
			presetName:   "ollama",
			expectedURL:  "http://localhost:11434/v1",
			expectExists: true,
		},
		{
			name:         "lmstudio preset exists",
			presetName:   "lmstudio",
			expectedURL:  "http://localhost:1234/v1",
			expectExists: true,
		},
		{
			name:         "openrouter preset exists",
			presetName:   "openrouter",
			expectedURL:  "https://openrouter.ai/api/v1",
			expectExists: true,
		},
		{
			name:         "unknown preset",
			presetName:   "unknown",
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset, exists := ProviderPresets[tt.presetName]

			assert.Equal(t, tt.expectExists, exists)

			if exists {
				assert.Equal(t, tt.expectedURL, preset.BaseURL)
				assert.NotEmpty(t, preset.Description)
				assert.NotEmpty(t, preset.DefaultModel)
			}
		})
	}
}

func TestProviderPreset_Structure(t *testing.T) {
	// Verify all presets have required fields
	require.NotEmpty(t, ProviderPresets, "ProviderPresets should not be empty")

	for name, preset := range ProviderPresets {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, preset.BaseURL, "BaseURL should not be empty")
			assert.NotEmpty(t, preset.Description, "Description should not be empty")
			assert.NotEmpty(t, preset.DefaultModel, "DefaultModel should not be empty")

			// Verify BaseURL format
			assert.Contains(t, preset.BaseURL, "://", "BaseURL should be a valid URL")
		})
	}
}

func TestConfig_BaseURL(t *testing.T) {
	tests := []struct {
		name            string
		config          Config
		expectedBaseURL string
	}{
		{
			name: "config with base URL",
			config: Config{
				Name:        "custom",
				BaseURL:     "https://custom.api.com/v1",
				Model:       "custom-model",
				Temperature: 0.5,
			},
			expectedBaseURL: "https://custom.api.com/v1",
		},
		{
			name: "config without base URL",
			config: Config{
				Name:        "openai",
				Model:       "gpt-4",
				Temperature: 0.2,
			},
			expectedBaseURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedBaseURL, tt.config.BaseURL)
		})
	}
}
