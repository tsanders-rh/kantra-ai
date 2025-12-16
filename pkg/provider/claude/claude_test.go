package claude

import (
	"errors"
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
			Model:       "claude-3-5-sonnet-20250201",
			Temperature: 0.3,
		}

		p, err := New(config)
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "claude-3-5-sonnet-20250201", p.model)
		assert.Equal(t, 0.3, p.temperature)
	})

	t.Run("with default model", func(t *testing.T) {
		config := provider.Config{
			APIKey: "test-api-key",
		}

		p, err := New(config)
		require.NoError(t, err)
		assert.Equal(t, "claude-sonnet-4-20250514", p.model)
		assert.Equal(t, 0.2, p.temperature) // Default temperature
	})

	t.Run("with environment variable", func(t *testing.T) {
		// Set env var
		os.Setenv("ANTHROPIC_API_KEY", "env-api-key")
		defer os.Unsetenv("ANTHROPIC_API_KEY")

		config := provider.Config{}
		p, err := New(config)
		require.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("missing API key", func(t *testing.T) {
		// Ensure env var is not set
		os.Unsetenv("ANTHROPIC_API_KEY")

		config := provider.Config{}
		_, err := New(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY environment variable is not set")
		assert.Contains(t, err.Error(), "https://console.anthropic.com/settings/keys")
	})
}

func TestProvider_Name(t *testing.T) {
	config := provider.Config{APIKey: "test"}
	p, err := New(config)
	require.NoError(t, err)

	assert.Equal(t, "claude", p.Name())
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

	// Estimate: 2000 input tokens * $3/1M + 1000 output tokens * $15/1M
	expectedCost := (2000.0 * 3.0 / 1000000.0) + (1000.0 * 15.0 / 1000000.0)
	assert.InDelta(t, expectedCost, cost, 0.0001) // Use InDelta for float comparison
	assert.Greater(t, cost, 0.0)
}

// NOTE: buildPrompt tests removed - prompts now generated via configurable templates
// Prompt generation is tested indirectly through FixViolation integration tests

func TestEnhanceAPIError(t *testing.T) {
	t.Run("401 authentication error", func(t *testing.T) {
		err := enhanceAPIError(assert.AnError)
		assert.Error(t, err)
		// Should wrap original error
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("wraps error with Claude context", func(t *testing.T) {
		originalErr := errors.New("API error")
		enhanced := enhanceAPIError(originalErr)

		assert.Error(t, enhanced)
		assert.ErrorIs(t, enhanced, originalErr)
		// The error should contain Claude-specific information
		// (actual enhancement is tested in pkg/provider/common/errors_test.go)
	})
}
