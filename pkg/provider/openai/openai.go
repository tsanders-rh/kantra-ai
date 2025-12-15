package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/tsanders/kantra-ai/pkg/prompt"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/provider/common"
)

const (
	// DefaultMaxTokens is the default maximum tokens for fix generation
	DefaultMaxTokens = 4096
	// PlanningMaxTokens is the maximum tokens for plan generation (requires more output)
	PlanningMaxTokens = 8192
)

// Provider implements the OpenAI provider
type Provider struct {
	client      *openai.Client
	model       string
	temperature float32
	templates   *prompt.Templates
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

	// Load templates (use defaults if not provided)
	templates := config.Templates
	if templates == nil {
		var err error
		templates, err = prompt.Load(prompt.Config{
			Provider: "openai",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to load default templates: %w", err)
		}
	}

	return &Provider{
		client:      client,
		model:       model,
		temperature: temperature,
		templates:   templates,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "openai"
}

// FixViolation sends the violation to OpenAI and gets a fix
func (p *Provider) FixViolation(ctx context.Context, req provider.FixRequest) (*provider.FixResponse, error) {
	// Build prompt from template
	data := prompt.BuildSingleFixData(req)
	promptText, err := p.templates.SingleFix.RenderSingleFix(data)
	if err != nil {
		return &provider.FixResponse{
			Success: false,
			Error:   fmt.Errorf("failed to render prompt template: %w", err),
		}, nil
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Temperature: p.temperature,
		MaxTokens:   DefaultMaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: promptText,
			},
		},
	})

	if err != nil {
		return &provider.FixResponse{
			Success: false,
			Error:   enhanceAPIError(err),
		}, nil
	}

	responseText := resp.Choices[0].Message.Content

	// Parse JSON response
	type Response struct {
		FixedContent string  `json:"fixed_content"`
		Confidence   float64 `json:"confidence"`
		Explanation  string  `json:"explanation"`
	}

	// Try to extract JSON from response (may be wrapped in markdown)
	jsonData := extractJSONFromMarkdown(responseText)

	var parsedResp Response
	if err := json.Unmarshal(jsonData, &parsedResp); err != nil {
		// If JSON parsing fails, fall back to treating entire response as code with default confidence
		inputCost := float64(resp.Usage.PromptTokens) * 30.0 / 1000000.0
		outputCost := float64(resp.Usage.CompletionTokens) * 60.0 / 1000000.0
		return &provider.FixResponse{
			Success:      true,
			FixedContent: responseText,
			Explanation:  "Fixed by GPT-4 (JSON parse failed, using raw response)",
			Confidence:   0.85, // Default when JSON parsing fails
			TokensUsed:   resp.Usage.TotalTokens,
			Cost:         inputCost + outputCost,
		}, nil
	}

	// Validate confidence range
	if parsedResp.Confidence < 0.0 || parsedResp.Confidence > 1.0 {
		parsedResp.Confidence = 0.85 // Clamp to reasonable default
	}

	// Calculate cost (GPT-4 pricing: $30/$60 per 1M tokens)
	inputCost := float64(resp.Usage.PromptTokens) * 30.0 / 1000000.0
	outputCost := float64(resp.Usage.CompletionTokens) * 60.0 / 1000000.0
	totalCost := inputCost + outputCost

	return &provider.FixResponse{
		Success:      true,
		FixedContent: parsedResp.FixedContent,
		Explanation:  parsedResp.Explanation,
		Confidence:   parsedResp.Confidence,
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

// enhanceAPIError adds helpful context to OpenAI API errors using the common error handler.
func enhanceAPIError(err error) error {
	return common.EnhanceAPIError(err, common.ProviderErrorContext{
		ProviderName:      "OpenAI",
		APIKeysURL:        "https://platform.openai.com/api-keys",
		StatusPageURL:     "https://status.openai.com",
		BillingURL:        "https://platform.openai.com/account/billing",
		AlternateProvider: "claude",
	})
}

// GeneratePlan generates a phased migration plan using OpenAI
func (p *Provider) GeneratePlan(ctx context.Context, req provider.PlanRequest) (*provider.PlanResponse, error) {
	return &provider.PlanResponse{
		Error: fmt.Errorf("plan generation not yet implemented for OpenAI provider - use --provider=claude"),
	}, nil
}


