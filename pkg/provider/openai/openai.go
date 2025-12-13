package openai

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/tsanders/kantra-ai/pkg/provider"
)

// Provider implements the OpenAI provider
type Provider struct {
	client      *openai.Client
	model       string
	temperature float32
}

// New creates a new OpenAI provider
func New(config provider.Config) (*Provider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	model := config.Model
	if model == "" {
		model = openai.GPT4 // Default to GPT-4
	}

	temperature := float32(config.Temperature)
	if temperature == 0 {
		temperature = 0.2 // Low temperature for code fixes
	}

	client := openai.NewClient(apiKey)

	return &Provider{
		client:      client,
		model:       model,
		temperature: temperature,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "openai"
}

// FixViolation sends the violation to OpenAI and gets a fix
func (p *Provider) FixViolation(ctx context.Context, req provider.FixRequest) (*provider.FixResponse, error) {
	prompt := buildPrompt(req)

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Temperature: p.temperature,
		MaxTokens:   4096,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})

	if err != nil {
		return &provider.FixResponse{
			Success: false,
			Error:   fmt.Errorf("openai API error: %w", err),
		}, nil
	}

	fixedContent := resp.Choices[0].Message.Content

	// Calculate cost (GPT-4 pricing: $30/$60 per 1M tokens)
	inputCost := float64(resp.Usage.PromptTokens) * 30.0 / 1000000.0
	outputCost := float64(resp.Usage.CompletionTokens) * 60.0 / 1000000.0
	totalCost := inputCost + outputCost

	return &provider.FixResponse{
		Success:      true,
		FixedContent: fixedContent,
		Explanation:  "Fixed by GPT-4",
		Confidence:   0.85,
		TokensUsed:   resp.Usage.TotalTokens,
		Cost:         totalCost,
	}, nil
}

// EstimateCost estimates the cost for fixing a violation
func (p *Provider) EstimateCost(req provider.FixRequest) (float64, error) {
	// Rough estimate: ~2000 tokens input + ~1000 tokens output
	// Using GPT-4 pricing: $30/$60 per 1M tokens
	estimatedInputTokens := 2000.0
	estimatedOutputTokens := 1000.0

	inputCost := estimatedInputTokens * 30.0 / 1000000.0
	outputCost := estimatedOutputTokens * 60.0 / 1000000.0

	return inputCost + outputCost, nil
}

// buildPrompt constructs the prompt for OpenAI
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
