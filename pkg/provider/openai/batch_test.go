package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestBuildBatchPrompt(t *testing.T) {
	p := &Provider{
		model:       "gpt-4",
		temperature: 0.2,
	}

	req := provider.BatchRequest{
		Violation: violation.Violation{
			ID:          "test-violation",
			Description: "Test violation description",
		},
		Incidents: []violation.Incident{
			{
				URI:        "file:///test1.java",
				LineNumber: 10,
				Message:    "Test message 1",
			},
			{
				URI:        "file:///test2.java",
				LineNumber: 20,
				Message:    "Test message 2",
			},
		},
		FileContents: map[string]string{
			"/test1.java": "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11",
			"/test2.java": "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\nline13\nline14\nline15\nline16\nline17\nline18\nline19\nline20\nline21",
		},
		Language: "java",
	}

	prompt := p.buildBatchPrompt(req)

	// Verify prompt structure
	assert.Contains(t, prompt, "code modernization assistant")
	assert.Contains(t, prompt, "VIOLATION: test-violation")
	assert.Contains(t, prompt, "DESCRIPTION: Test violation description")
	assert.Contains(t, prompt, "Fix the following 2 incident(s)")
	assert.Contains(t, prompt, "INCIDENT 1:")
	assert.Contains(t, prompt, "File: /test1.java")
	assert.Contains(t, prompt, "Line: 10")
	assert.Contains(t, prompt, "INCIDENT 2:")
	assert.Contains(t, prompt, "File: /test2.java")
	assert.Contains(t, prompt, "Line: 20")
	assert.Contains(t, prompt, "OUTPUT FORMAT (JSON)")
	assert.Contains(t, prompt, "incident_uri")
	assert.Contains(t, prompt, "fixed_content")
	assert.Contains(t, prompt, "COMPLETE fixed file content")
}

func TestParseBatchResponse(t *testing.T) {
	p := &Provider{}

	incidents := []violation.Incident{
		{URI: "file:///test1.java", LineNumber: 10},
		{URI: "file:///test2.java", LineNumber: 20},
	}

	t.Run("valid JSON response", func(t *testing.T) {
		responseText := "```json\n" +
			"[\n" +
			"  {\n" +
			"    \"incident_uri\": \"file:///test1.java:10\",\n" +
			"    \"success\": true,\n" +
			"    \"fixed_content\": \"fixed content 1\",\n" +
			"    \"explanation\": \"Fixed issue 1\",\n" +
			"    \"confidence\": 0.95\n" +
			"  },\n" +
			"  {\n" +
			"    \"incident_uri\": \"file:///test2.java:20\",\n" +
			"    \"success\": true,\n" +
			"    \"fixed_content\": \"fixed content 2\",\n" +
			"    \"explanation\": \"Fixed issue 2\",\n" +
			"    \"confidence\": 0.90\n" +
			"  }\n" +
			"]\n" +
			"```\n"
		fixes, err := p.parseBatchResponse(responseText, incidents)

		require.NoError(t, err)
		require.Len(t, fixes, 2)

		assert.Equal(t, "file:///test1.java:10", fixes[0].IncidentURI)
		assert.True(t, fixes[0].Success)
		assert.Equal(t, "fixed content 1", fixes[0].FixedContent)
		assert.Equal(t, "Fixed issue 1", fixes[0].Explanation)
		assert.Equal(t, 0.95, fixes[0].Confidence)
		assert.Nil(t, fixes[0].Error)

		assert.Equal(t, "file:///test2.java:20", fixes[1].IncidentURI)
		assert.True(t, fixes[1].Success)
	})

	t.Run("JSON without code blocks", func(t *testing.T) {
		responseText := `[
  {
    "incident_uri": "file:///test1.java:10",
    "success": true,
    "fixed_content": "fixed content 1",
    "explanation": "Fixed issue 1",
    "confidence": 0.95
  },
  {
    "incident_uri": "file:///test2.java:20",
    "success": true,
    "fixed_content": "fixed content 2",
    "explanation": "Fixed issue 2",
    "confidence": 0.90
  }
]`
		fixes, err := p.parseBatchResponse(responseText, incidents)

		require.NoError(t, err)
		require.Len(t, fixes, 2)
		assert.True(t, fixes[0].Success)
		assert.True(t, fixes[1].Success)
	})

	t.Run("partial failure", func(t *testing.T) {
		responseText := "```json\n" +
			"[\n" +
			"  {\n" +
			"    \"incident_uri\": \"file:///test1.java:10\",\n" +
			"    \"success\": true,\n" +
			"    \"fixed_content\": \"fixed content 1\",\n" +
			"    \"explanation\": \"Fixed issue 1\",\n" +
			"    \"confidence\": 0.95\n" +
			"  },\n" +
			"  {\n" +
			"    \"incident_uri\": \"file:///test2.java:20\",\n" +
			"    \"success\": false,\n" +
			"    \"fixed_content\": \"\",\n" +
			"    \"explanation\": \"Could not parse code\",\n" +
			"    \"confidence\": 0.0\n" +
			"  }\n" +
			"]\n" +
			"```\n"
		fixes, err := p.parseBatchResponse(responseText, incidents)

		require.NoError(t, err)
		require.Len(t, fixes, 2)
		assert.True(t, fixes[0].Success)
		assert.Nil(t, fixes[0].Error)
		assert.False(t, fixes[1].Success)
		assert.NotNil(t, fixes[1].Error)
		assert.Contains(t, fixes[1].Error.Error(), "Could not parse code")
	})

	t.Run("wrong number of fixes", func(t *testing.T) {
		responseText := "```json\n" +
			"[\n" +
			"  {\n" +
			"    \"incident_uri\": \"file:///test1.java:10\",\n" +
			"    \"success\": true,\n" +
			"    \"fixed_content\": \"fixed content 1\",\n" +
			"    \"explanation\": \"Fixed issue 1\",\n" +
			"    \"confidence\": 0.95\n" +
			"  }\n" +
			"]\n" +
			"```\n"
		_, err := p.parseBatchResponse(responseText, incidents)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected 2 fixes but got 1")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		responseText := "not valid json"

		_, err := p.parseBatchResponse(responseText, incidents)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON in code block",
			input:    "Here is the JSON:\n" + "```json\n[{\"key\": \"value\"}]\n```",
			expected: "[{\"key\": \"value\"}]",
		},
		{
			name:     "JSON in code block without json marker",
			input:    "Here is the JSON:\n" + "```\n[{\"key\": \"value\"}]\n```",
			expected: "[{\"key\": \"value\"}]",
		},
		{
			name:     "raw JSON array",
			input:    "Some text [{\"key\": \"value\"}] more text",
			expected: "[{\"key\": \"value\"}]",
		},
		{
			name:     "complex JSON",
			input:    "```json\n[\n  {\n    \"nested\": {\n      \"key\": \"value\"\n    }\n  }\n]\n```",
			expected: "[\n  {\n    \"nested\": {\n      \"key\": \"value\"\n    }\n  }\n]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.input)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestMinMax(t *testing.T) {
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 5, min(5, 5))

	assert.Equal(t, 10, max(5, 10))
	assert.Equal(t, 10, max(10, 5))
	assert.Equal(t, 5, max(5, 5))
}
