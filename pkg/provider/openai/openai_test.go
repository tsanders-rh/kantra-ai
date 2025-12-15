package openai

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestNew(t *testing.T) {
	t.Run("with API key in config", func(t *testing.T) {
		config := provider.Config{
			APIKey:      "test-api-key",
			Model:       "gpt-4-turbo",
			Temperature: 0.3,
		}

		p, err := New(config)
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "gpt-4-turbo", p.model)
		assert.Equal(t, float32(0.3), p.temperature)
	})

	t.Run("with default model", func(t *testing.T) {
		config := provider.Config{
			APIKey: "test-api-key",
		}

		p, err := New(config)
		require.NoError(t, err)
		assert.Equal(t, "gpt-4", p.model)
		assert.Equal(t, float32(0.2), p.temperature) // Default temperature
	})

	t.Run("with environment variable", func(t *testing.T) {
		// Set env var
		os.Setenv("OPENAI_API_KEY", "env-api-key")
		defer os.Unsetenv("OPENAI_API_KEY")

		config := provider.Config{}
		p, err := New(config)
		require.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("missing API key", func(t *testing.T) {
		// Ensure env var is not set
		os.Unsetenv("OPENAI_API_KEY")

		config := provider.Config{}
		_, err := New(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OPENAI_API_KEY environment variable is not set")
		assert.Contains(t, err.Error(), "https://platform.openai.com/api-keys")
	})
}

func TestProvider_Name(t *testing.T) {
	config := provider.Config{APIKey: "test"}
	p, err := New(config)
	require.NoError(t, err)

	assert.Equal(t, "openai", p.Name())
}

func TestProvider_EstimateCost(t *testing.T) {
	config := provider.Config{APIKey: "test"}
	p, err := New(config)
	require.NoError(t, err)

	req := provider.FixRequest{
		Violation: violation.Violation{
			ID: "test",
		},
	}

	cost, err := p.EstimateCost(req)
	require.NoError(t, err)

	// Estimate: 2000 input tokens * $30/1M + 1000 output tokens * $60/1M
	expectedCost := (2000.0 * 30.0 / 1000000.0) + (1000.0 * 60.0 / 1000000.0)
	assert.InDelta(t, expectedCost, cost, 0.0001) // Use InDelta for float comparison
	assert.Greater(t, cost, 0.0)
}

// NOTE: buildPrompt tests removed - prompts now generated via configurable templates
// Prompt generation is tested indirectly through FixViolation integration tests
