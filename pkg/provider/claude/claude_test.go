package claude

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
		assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY not set")
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

func TestBuildPrompt(t *testing.T) {
	req := provider.FixRequest{
		Violation: violation.Violation{
			ID:          "violation-001",
			Description: "Replace javax with jakarta",
			Category:    "mandatory",
			Rule: violation.Rule{
				ID:      "javax-to-jakarta",
				Message: "Migrate to jakarta namespace",
			},
		},
		Incident: violation.Incident{
			URI:        "file:///src/Test.java",
			LineNumber: 10,
			CodeSnip:   "import javax.servlet.*;",
		},
		FileContent: "package test;\nimport javax.servlet.*;\npublic class Test {}",
		Language:    "java",
	}

	prompt := buildPrompt(req)

	// Verify prompt contains all key information
	assert.Contains(t, prompt, "VIOLATION DETAILS")
	assert.Contains(t, prompt, "Category: mandatory")
	assert.Contains(t, prompt, "Description: Replace javax with jakarta")
	assert.Contains(t, prompt, "Rule: javax-to-jakarta")
	assert.Contains(t, prompt, "Rule Message: Migrate to jakarta namespace")

	assert.Contains(t, prompt, "FILE LOCATION")
	assert.Contains(t, prompt, "File: /src/Test.java")
	assert.Contains(t, prompt, "Line: 10")

	assert.Contains(t, prompt, "CURRENT CODE SNIPPET")
	assert.Contains(t, prompt, "import javax.servlet.*;")

	assert.Contains(t, prompt, "FULL FILE CONTENT")
	assert.Contains(t, prompt, "package test;")

	assert.Contains(t, prompt, "TASK")
	assert.Contains(t, prompt, "valid java code")

	// Verify formatting instructions
	assert.Contains(t, prompt, "Return the ENTIRE file content")
	assert.Contains(t, prompt, "Do not include markdown formatting")
	assert.Contains(t, prompt, "Do not include explanations")
}

func TestBuildPrompt_DifferentLanguages(t *testing.T) {
	languages := []string{"java", "python", "go", "javascript"}

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
			URI:        "file:///test.java",
			LineNumber: 5,
			CodeSnip:   "code snippet",
		},
		FileContent: "full file",
		Language:    "java",
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
