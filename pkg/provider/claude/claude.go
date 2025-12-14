package claude

import (
	"context"
	"fmt"
	"os"
	"strings"

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
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set\n\n" +
			"To use Claude (Anthropic):\n" +
			"  1. Get an API key from: https://console.anthropic.com/settings/keys\n" +
			"  2. Export it as an environment variable:\n" +
			"     export ANTHROPIC_API_KEY=sk-ant-...\n" +
			"  3. Or set it in your shell profile (~/.bashrc, ~/.zshrc)\n\n" +
			"Alternatively, use OpenAI instead:\n" +
			"  --provider=openai")
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
			Error:   enhanceAPIError(err),
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

// enhanceAPIError adds helpful context to Claude API errors
func enhanceAPIError(err error) error {
	errMsg := err.Error()

	// Check for common error patterns
	if contains(errMsg, "401") || contains(errMsg, "unauthorized") || contains(errMsg, "invalid api key") {
		return fmt.Errorf("Claude API authentication failed: %w\n\n"+
			"Possible causes:\n"+
			"  - Invalid or expired API key\n"+
			"  - API key revoked or deleted\n\n"+
			"To fix:\n"+
			"  1. Verify your API key at: https://console.anthropic.com/settings/keys\n"+
			"  2. Ensure ANTHROPIC_API_KEY is set correctly\n"+
			"  3. Try generating a new API key", err)
	}

	if contains(errMsg, "429") || contains(errMsg, "rate limit") {
		return fmt.Errorf("Claude API rate limit exceeded: %w\n\n"+
			"You've made too many requests in a short period.\n\n"+
			"To fix:\n"+
			"  1. Wait a few minutes and try again\n"+
			"  2. Reduce the number of violations being fixed\n"+
			"  3. Upgrade your Anthropic API plan for higher limits", err)
	}

	if contains(errMsg, "timeout") || contains(errMsg, "deadline exceeded") {
		return fmt.Errorf("Claude API request timed out: %w\n\n"+
			"The request took too long to complete.\n\n"+
			"To fix:\n"+
			"  1. Check your internet connection\n"+
			"  2. Try again - this is often a temporary issue\n"+
			"  3. If persistent, reduce file size or complexity", err)
	}

	if contains(errMsg, "connection") || contains(errMsg, "network") || contains(errMsg, "dial") {
		return fmt.Errorf("network error connecting to Claude API: %w\n\n"+
			"Unable to reach Anthropic's servers.\n\n"+
			"To fix:\n"+
			"  1. Check your internet connection\n"+
			"  2. Verify you can reach: https://api.anthropic.com\n"+
			"  3. Check if your firewall/proxy is blocking the connection\n"+
			"  4. Try again in a few moments", err)
	}

	if contains(errMsg, "500") || contains(errMsg, "502") || contains(errMsg, "503") {
		return fmt.Errorf("Claude API server error: %w\n\n"+
			"Anthropic's API is experiencing issues.\n\n"+
			"To fix:\n"+
			"  1. Wait a few minutes and try again\n"+
			"  2. Check Anthropic's status page: https://status.anthropic.com\n"+
			"  3. If urgent, try --provider=openai instead", err)
	}

	// Generic API error
	return fmt.Errorf("Claude API error: %w\n\n"+
		"Check the error message above for details.\n"+
		"Visit https://docs.anthropic.com for API documentation.", err)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
