package openai

import (
	"context"
	"fmt"
	"os"
	"strings"

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
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set\n\n" +
			"To use OpenAI:\n" +
			"  1. Get an API key from: https://platform.openai.com/api-keys\n" +
			"  2. Export it as an environment variable:\n" +
			"     export OPENAI_API_KEY=sk-...\n" +
			"  3. Or set it in your shell profile (~/.bashrc, ~/.zshrc)\n\n" +
			"Alternatively, use Claude instead:\n" +
			"  --provider=claude")
	}

	model := config.Model
	if model == "" {
		model = openai.GPT4 // Default to GPT-4
	}

	temperature := float32(config.Temperature)
	if temperature == 0 {
		temperature = 0.2 // Low temperature for code fixes
	}

	// Create client configuration
	clientConfig := openai.DefaultConfig(apiKey)

	// Support custom base URLs for OpenAI-compatible APIs (Groq, Ollama, etc.)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

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
			Error:   enhanceAPIError(err),
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

// enhanceAPIError adds helpful context to OpenAI API errors
func enhanceAPIError(err error) error {
	errMsg := err.Error()

	// Check for common error patterns
	if contains(errMsg, "401") || contains(errMsg, "unauthorized") || contains(errMsg, "invalid api key") {
		return fmt.Errorf("OpenAI API authentication failed: %w\n\n"+
			"Possible causes:\n"+
			"  - Invalid or expired API key\n"+
			"  - API key revoked or deleted\n\n"+
			"To fix:\n"+
			"  1. Verify your API key at: https://platform.openai.com/api-keys\n"+
			"  2. Ensure OPENAI_API_KEY is set correctly\n"+
			"  3. Try generating a new API key", err)
	}

	if contains(errMsg, "429") || contains(errMsg, "rate limit") {
		return fmt.Errorf("OpenAI API rate limit exceeded: %w\n\n"+
			"You've made too many requests or exceeded your quota.\n\n"+
			"To fix:\n"+
			"  1. Wait a few minutes and try again\n"+
			"  2. Check your usage and billing: https://platform.openai.com/usage\n"+
			"  3. Upgrade your OpenAI plan or add credits", err)
	}

	if contains(errMsg, "insufficient_quota") || contains(errMsg, "quota") {
		return fmt.Errorf("OpenAI API quota exceeded: %w\n\n"+
			"You've reached your account spending limit.\n\n"+
			"To fix:\n"+
			"  1. Add credits to your account: https://platform.openai.com/account/billing\n"+
			"  2. Check your usage limits and upgrade if needed\n"+
			"  3. Or use --provider=claude instead", err)
	}

	if contains(errMsg, "timeout") || contains(errMsg, "deadline exceeded") {
		return fmt.Errorf("OpenAI API request timed out: %w\n\n"+
			"The request took too long to complete.\n\n"+
			"To fix:\n"+
			"  1. Check your internet connection\n"+
			"  2. Try again - this is often a temporary issue\n"+
			"  3. If persistent, reduce file size or complexity", err)
	}

	if contains(errMsg, "connection") || contains(errMsg, "network") || contains(errMsg, "dial") {
		return fmt.Errorf("network error connecting to OpenAI API: %w\n\n"+
			"Unable to reach OpenAI's servers.\n\n"+
			"To fix:\n"+
			"  1. Check your internet connection\n"+
			"  2. Verify you can reach: https://api.openai.com\n"+
			"  3. Check if your firewall/proxy is blocking the connection\n"+
			"  4. Try again in a few moments", err)
	}

	if contains(errMsg, "500") || contains(errMsg, "502") || contains(errMsg, "503") {
		return fmt.Errorf("OpenAI API server error: %w\n\n"+
			"OpenAI's API is experiencing issues.\n\n"+
			"To fix:\n"+
			"  1. Wait a few minutes and try again\n"+
			"  2. Check OpenAI's status page: https://status.openai.com\n"+
			"  3. If urgent, try --provider=claude instead", err)
	}

	// Generic API error
	return fmt.Errorf("OpenAI API error: %w\n\n"+
		"Check the error message above for details.\n"+
		"Visit https://platform.openai.com/docs for API documentation.", err)
}

// GeneratePlan generates a phased migration plan using OpenAI
func (p *Provider) GeneratePlan(ctx context.Context, req provider.PlanRequest) (*provider.PlanResponse, error) {
	return &provider.PlanResponse{
		Error: fmt.Errorf("plan generation not yet implemented for OpenAI provider - use --provider=claude"),
	}, nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

