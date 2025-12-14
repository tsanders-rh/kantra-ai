package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// FixBatch processes multiple incidents of the same violation in one API call.
// This reduces costs and execution time by batching similar fixes together.
func (p *Provider) FixBatch(ctx context.Context, req provider.BatchRequest) (*provider.BatchResponse, error) {
	if len(req.Incidents) == 0 {
		return nil, fmt.Errorf("batch request must contain at least one incident")
	}

	// Build the batch prompt
	prompt := p.buildBatchPrompt(req)

	// Call OpenAI API
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Temperature: p.temperature,
		MaxTokens:   8192,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})

	if err != nil {
		return nil, enhanceAPIError(fmt.Errorf("OpenAI API error: %w", err))
	}

	// Extract response text
	responseText := resp.Choices[0].Message.Content

	// Parse the batch response
	fixes, err := p.parseBatchResponse(responseText, req.Incidents)
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	// Calculate costs
	// Note: For custom providers (Groq, Together, etc.), pricing may vary
	// Currently using GPT-4 pricing as baseline: $30/1M input, $60/1M output
	// Future enhancement: Make pricing configurable per provider preset
	inputTokens := resp.Usage.PromptTokens
	outputTokens := resp.Usage.CompletionTokens
	inputCost := float64(inputTokens) * 30.0 / 1000000.0
	outputCost := float64(outputTokens) * 60.0 / 1000000.0
	cost := inputCost + outputCost

	// Check if all fixes succeeded
	allSuccess := true
	for _, fix := range fixes {
		if !fix.Success {
			allSuccess = false
			break
		}
	}

	return &provider.BatchResponse{
		Fixes:      fixes,
		Success:    allSuccess,
		TokensUsed: resp.Usage.TotalTokens,
		Cost:       cost,
	}, nil
}

// buildBatchPrompt constructs a prompt for fixing multiple incidents together
func (p *Provider) buildBatchPrompt(req provider.BatchRequest) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert code modernization assistant. Fix multiple occurrences of the same violation in a codebase.\n\n")

	// Violation context
	prompt.WriteString(fmt.Sprintf("VIOLATION: %s\n", req.Violation.ID))
	prompt.WriteString(fmt.Sprintf("DESCRIPTION: %s\n\n", req.Violation.Description))

	// Add each incident
	prompt.WriteString(fmt.Sprintf("Fix the following %d incident(s):\n\n", len(req.Incidents)))

	for i, incident := range req.Incidents {
		filePath := incident.GetFilePath()

		prompt.WriteString(fmt.Sprintf("INCIDENT %d:\n", i+1))
		prompt.WriteString(fmt.Sprintf("File: %s\n", filePath))
		prompt.WriteString(fmt.Sprintf("Line: %d\n", incident.LineNumber))
		prompt.WriteString(fmt.Sprintf("Issue: %s\n", incident.Message))

		// Add file content if available
		if content, ok := req.FileContents[filePath]; ok {
			// Show context around the line
			lines := strings.Split(content, "\n")
			start := max(0, incident.LineNumber-5)
			end := min(len(lines), incident.LineNumber+5)

			prompt.WriteString("Code context:\n```")
			prompt.WriteString(req.Language)
			prompt.WriteString("\n")
			for j := start; j < end; j++ {
				prefix := "  "
				if j == incident.LineNumber-1 {
					prefix = "→ " // Mark the problematic line
				}
				prompt.WriteString(fmt.Sprintf("%s%s\n", prefix, lines[j]))
			}
			prompt.WriteString("```\n\n")
		}
	}

	// Output format instructions
	prompt.WriteString("\nFor each incident, provide the complete fixed file content and a brief explanation.\n\n")
	prompt.WriteString("OUTPUT FORMAT (JSON):\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("[\n")
	prompt.WriteString("  {\n")
	prompt.WriteString("    \"incident_uri\": \"file:///path/to/file.java:line\",\n")
	prompt.WriteString("    \"success\": true,\n")
	prompt.WriteString("    \"fixed_content\": \"<complete fixed file content>\",\n")
	prompt.WriteString("    \"explanation\": \"<what was changed>\",\n")
	prompt.WriteString("    \"confidence\": 0.95\n")
	prompt.WriteString("  }\n")
	prompt.WriteString("]\n")
	prompt.WriteString("```\n\n")
	prompt.WriteString("IMPORTANT:\n")
	prompt.WriteString("- Return the COMPLETE fixed file content for each file, not just the changed lines\n")
	prompt.WriteString("- Maintain all existing code that doesn't need changes\n")
	prompt.WriteString("- Preserve formatting and indentation\n")
	prompt.WriteString("- If you cannot fix an incident, set success=false and explain why in explanation\n")

	return prompt.String()
}

// parseBatchResponse parses the JSON response into IncidentFix structs
func (p *Provider) parseBatchResponse(responseText string, incidents []violation.Incident) ([]provider.IncidentFix, error) {
	// Extract JSON from markdown code blocks
	jsonData := extractJSONFromMarkdown(responseText)

	// Parse JSON array
	var rawFixes []struct {
		IncidentURI  string  `json:"incident_uri"`
		Success      bool    `json:"success"`
		FixedContent string  `json:"fixed_content"`
		Explanation  string  `json:"explanation"`
		Confidence   float64 `json:"confidence"`
	}

	if err := json.Unmarshal(jsonData, &rawFixes); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Verify we got a fix for each incident
	if len(rawFixes) != len(incidents) {
		return nil, fmt.Errorf("expected %d fixes but got %d", len(incidents), len(rawFixes))
	}

	// Convert to IncidentFix structs
	fixes := make([]provider.IncidentFix, len(rawFixes))
	for i, raw := range rawFixes {
		fixes[i] = provider.IncidentFix{
			IncidentURI:  raw.IncidentURI,
			Success:      raw.Success,
			FixedContent: raw.FixedContent,
			Explanation:  raw.Explanation,
			Confidence:   raw.Confidence,
		}

		if !raw.Success {
			fixes[i].Error = fmt.Errorf("%s", raw.Explanation)
		}
	}

	return fixes, nil
}

// extractJSONFromMarkdown extracts JSON content from markdown code blocks
func extractJSONFromMarkdown(text string) []byte {
	// Try to find JSON in code blocks first
	re := regexp.MustCompile("```(?:json)?\\s*\\n([\\s\\S]*?)\\n```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return []byte(strings.TrimSpace(matches[1]))
	}

	// Try to find raw JSON array
	re = regexp.MustCompile(`(?s)(\[.*\])`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return []byte(strings.TrimSpace(matches[1]))
	}

	// Return the whole text as last resort
	return []byte(text)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
