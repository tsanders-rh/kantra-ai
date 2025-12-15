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

	// Build batch prompt from template
	data := provider.BuildBatchFixData(req)
	// Select language-specific template or fall back to base template
	tmpl := p.templates.GetBatchFixTemplate(data.Language)
	promptText, err := tmpl.RenderBatchFix(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render batch prompt template: %w", err)
	}

	// Call OpenAI API
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Temperature: p.temperature,
		MaxTokens:   8192,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: promptText,
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
