package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

var (
	// Compiled regexes for batch JSON extraction (compiled once at package init time)
	batchJSONCodeBlockRegex = regexp.MustCompile("```(?:json)?\\s*\\n([\\s\\S]*?)\\n```")
	batchJSONArrayRegex     = regexp.MustCompile(`(?s)(\[.*\])`)
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

	// Call Claude API
	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.F(p.model),
		MaxTokens:   anthropic.F(int64(PlanningMaxTokens)), // Higher limit for batch processing
		Temperature: anthropic.F(p.temperature),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptText)),
		}),
	})

	if err != nil {
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	// Extract response text
	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
		}
	}

	// Parse the batch response
	fixes, err := p.parseBatchResponse(responseText, req.Incidents)
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	// Calculate costs (Sonnet 4 pricing: $3/1M input, $15/1M output)
	inputTokens := message.Usage.InputTokens
	outputTokens := message.Usage.OutputTokens
	inputCost := float64(inputTokens) * 3.0 / 1000000.0
	outputCost := float64(outputTokens) * 15.0 / 1000000.0
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
		TokensUsed: int(inputTokens + outputTokens),
		Cost:       cost,
	}, nil
}

// parseBatchResponse parses Claude's JSON response into IncidentFix structs
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
	// Try to find JSON in code blocks first using pre-compiled regex
	matches := batchJSONCodeBlockRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return []byte(strings.TrimSpace(matches[1]))
	}

	// Try to find raw JSON array using pre-compiled regex
	matches = batchJSONArrayRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return []byte(strings.TrimSpace(matches[1]))
	}

	// Return the whole text as last resort
	return []byte(text)
}
