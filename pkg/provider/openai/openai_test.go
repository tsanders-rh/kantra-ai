package openai

import (
	"os"
	"strings"
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

func TestBuildPrompt(t *testing.T) {
	req := provider.FixRequest{
		Violation: violation.Violation{
			ID:          "violation-002",
			Description: "Update deprecated API",
			Category:    "optional",
			Rule: violation.Rule{
				ID:      "api-deprecation",
				Message: "Use new API version",
			},
		},
		Incident: violation.Incident{
			URI:        "file:///src/Service.py",
			LineNumber: 25,
			CodeSnip:   "old_api.call()",
		},
		FileContent: "# Service file\nold_api.call()\nprint('done')",
		Language:    "python",
	}

	prompt := buildPrompt(req)

	// Verify prompt contains all key information
	assert.Contains(t, prompt, "VIOLATION DETAILS")
	assert.Contains(t, prompt, "Category: optional")
	assert.Contains(t, prompt, "Description: Update deprecated API")
	assert.Contains(t, prompt, "Rule: api-deprecation")
	assert.Contains(t, prompt, "Rule Message: Use new API version")

	assert.Contains(t, prompt, "FILE LOCATION")
	assert.Contains(t, prompt, "File: /src/Service.py")
	assert.Contains(t, prompt, "Line: 25")

	assert.Contains(t, prompt, "CURRENT CODE SNIPPET")
	assert.Contains(t, prompt, "old_api.call()")

	assert.Contains(t, prompt, "FULL FILE CONTENT")
	assert.Contains(t, prompt, "# Service file")

	assert.Contains(t, prompt, "TASK")
	assert.Contains(t, prompt, "valid python code")

	// Verify formatting instructions
	assert.Contains(t, prompt, "Return the ENTIRE file content")
	assert.Contains(t, prompt, "Do not include markdown formatting")
}

func TestBuildPrompt_DifferentLanguages(t *testing.T) {
	languages := []string{"java", "python", "go", "javascript", "typescript"}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			req := provider.FixRequest{
				Violation: violation.Violation{
					ID:   "test",
					Rule: violation.Rule{},
				},
				Incident:    violation.Incident{},
				FileContent: "test content",
				Language:    lang,
			}

			prompt := buildPrompt(req)
			assert.Contains(t, prompt, "valid "+lang+" code")
		})
	}
}

func TestBuildPrompt_Structure(t *testing.T) {
	req := provider.FixRequest{
		Violation: violation.Violation{
			ID:          "test",
			Description: "Test description",
			Category:    "mandatory",
			Rule: violation.Rule{
				ID:      "rule-id",
				Message: "Rule message",
			},
		},
		Incident: violation.Incident{
			URI:        "file:///test.go",
			LineNumber: 15,
			CodeSnip:   "code snippet",
		},
		FileContent: "full file content",
		Language:    "go",
	}

	prompt := buildPrompt(req)

	// Verify sections appear in order
	violationIdx := strings.Index(prompt, "VIOLATION DETAILS")
	locationIdx := strings.Index(prompt, "FILE LOCATION")
	snippetIdx := strings.Index(prompt, "CURRENT CODE SNIPPET")
	contentIdx := strings.Index(prompt, "FULL FILE CONTENT")
	taskIdx := strings.Index(prompt, "TASK")
	importantIdx := strings.Index(prompt, "IMPORTANT")

	assert.Less(t, violationIdx, locationIdx)
	assert.Less(t, locationIdx, snippetIdx)
	assert.Less(t, snippetIdx, contentIdx)
	assert.Less(t, contentIdx, taskIdx)
	assert.Less(t, taskIdx, importantIdx)
}

func TestBuildPrompt_EmptyValues(t *testing.T) {
	req := provider.FixRequest{
		Violation: violation.Violation{
			ID:   "",
			Rule: violation.Rule{},
		},
		Incident:    violation.Incident{},
		FileContent: "",
		Language:    "",
	}

	// Should not panic with empty values
	prompt := buildPrompt(req)
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "VIOLATION DETAILS")
}
