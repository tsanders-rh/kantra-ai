package claude

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tsanders/kantra-ai/pkg/provider"
)

// Provider implements the Claude AI provider
type Provider struct {
	client      *anthropic.Client
	model       string
	temperature float64
}

// New creates a new Claude provider
func New(config provider.Config) (*Provider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	model := config.Model
	if model == "" {
		model = "claude-sonnet-4-20250514" // Default to latest Sonnet
	}

	temperature := config.Temperature
	if temperature == 0 {
		temperature = 0.2 // Low temperature for code fixes
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Provider{
		client:      client,
		model:       model,
		temperature: temperature,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "claude"
}

// FixViolation sends the violation to Claude and gets a fix
func (p *Provider) FixViolation(ctx context.Context, req provider.FixRequest) (*provider.FixResponse, error) {
	prompt := buildPrompt(req)

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.F(p.model),
		MaxTokens:   anthropic.F(int64(4096)),
		Temperature: anthropic.F(p.temperature),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return &provider.FixResponse{
			Success: false,
			Error:   fmt.Errorf("claude API error: %w", err),
		}, nil
	}

	// Extract the fixed content from Claude's response
	var fixedContent string
	for _, block := range message.Content {
		if block.Type == "text" {
			fixedContent = block.Text
		}
	}

	// Calculate cost (rough estimate based on Sonnet 4 pricing)
	// $3 per 1M input tokens, $15 per 1M output tokens
	inputCost := float64(message.Usage.InputTokens) * 3.0 / 1000000.0
	outputCost := float64(message.Usage.OutputTokens) * 15.0 / 1000000.0
	totalCost := inputCost + outputCost

	return &provider.FixResponse{
		Success:      true,
		FixedContent: fixedContent,
		Explanation:  "Fixed by Claude",
		Confidence:   0.85, // Claude doesn't provide confidence scores
		TokensUsed:   int(message.Usage.InputTokens + message.Usage.OutputTokens),
		Cost:         totalCost,
	}, nil
}

// EstimateCost estimates the cost for fixing a violation
func (p *Provider) EstimateCost(req provider.FixRequest) (float64, error) {
	// Rough estimate: ~2000 tokens input + ~1000 tokens output
	// Using Sonnet 4 pricing: $3/$15 per 1M tokens
	estimatedInputTokens := 2000.0
	estimatedOutputTokens := 1000.0

	inputCost := estimatedInputTokens * 3.0 / 1000000.0
	outputCost := estimatedOutputTokens * 15.0 / 1000000.0

	return inputCost + outputCost, nil
}

// buildPrompt constructs the prompt for Claude
func buildPrompt(req provider.FixRequest) string {
	return fmt.Sprintf(`You are a code migration assistant helping fix violations found by Konveyor static analysis.

VIOLATION DETAILS:
Category: %s
Description: %s
Rule: %s
Rule Message: %s

FILE LOCATION:
File: %s
Line: %d

CURRENT CODE SNIPPET:
%s

FULL FILE CONTENT:
%s

TASK:
Fix this violation by modifying the code. Return ONLY the complete fixed file content, with no explanation or markdown code blocks. The output must be valid %s code that can be written directly to the file.

IMPORTANT:
- Return the ENTIRE file content, not just the changed lines
- Do not include markdown formatting or code blocks
- Do not include explanations before or after the code
- Ensure the fix is syntactically correct
- Preserve all other code unchanged`,
		req.Violation.Category,
		req.Violation.Description,
		req.Violation.Rule.ID,
		req.Violation.Rule.Message,
		req.Incident.GetFilePath(),
		req.Incident.LineNumber,
		req.Incident.CodeSnip,
		req.FileContent,
		req.Language,
	)
}
