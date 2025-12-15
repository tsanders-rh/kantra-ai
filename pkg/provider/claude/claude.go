package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tsanders/kantra-ai/pkg/prompt"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/provider/common"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

const (
	// DefaultMaxTokens is the default maximum tokens for fix generation
	DefaultMaxTokens = 4096
	// PlanningMaxTokens is the maximum tokens for plan generation (requires more output)
	PlanningMaxTokens = 8192
)

// Provider implements the Claude AI provider
type Provider struct {
	client      *anthropic.Client
	model       string
	temperature float64
	templates   *prompt.Templates
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

	// Load templates (use defaults if not provided)
	templates := config.Templates
	if templates == nil {
		var err error
		templates, err = prompt.Load(prompt.Config{
			Provider: "claude",
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
	return "claude"
}

// FixViolation sends the violation to Claude and gets a fix
func (p *Provider) FixViolation(ctx context.Context, req provider.FixRequest) (*provider.FixResponse, error) {
	// Build prompt from template
	data := provider.BuildSingleFixData(req)
	// Select language-specific template or fall back to base template
	tmpl := p.templates.GetSingleFixTemplate(data.Language)
	promptText, err := tmpl.RenderSingleFix(data)
	if err != nil {
		return &provider.FixResponse{
			Success: false,
			Error:   fmt.Errorf("failed to render prompt template: %w", err),
		}, nil
	}

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.F(p.model),
		MaxTokens:   anthropic.F(int64(DefaultMaxTokens)),
		Temperature: anthropic.F(p.temperature),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptText)),
		}),
	})

	if err != nil {
		return &provider.FixResponse{
			Success: false,
			Error:   enhanceAPIError(err),
		}, nil
	}

	// Extract the response text from Claude's response
	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
		}
	}

	// Parse JSON response
	type Response struct {
		FixedContent string  `json:"fixed_content"`
		Confidence   float64 `json:"confidence"`
		Explanation  string  `json:"explanation"`
	}

	// Try to extract JSON from response (may be wrapped in markdown)
	jsonData := extractJSONFromMarkdown(responseText)

	var resp Response
	if err := json.Unmarshal(jsonData, &resp); err != nil {
		// If JSON parsing fails, fall back to treating entire response as code with default confidence
		inputCost := float64(message.Usage.InputTokens) * 3.0 / 1000000.0
		outputCost := float64(message.Usage.OutputTokens) * 15.0 / 1000000.0
		return &provider.FixResponse{
			Success:      true,
			FixedContent: responseText,
			Explanation:  "Fixed by Claude (JSON parse failed, using raw response)",
			Confidence:   0.85, // Default when JSON parsing fails
			TokensUsed:   int(message.Usage.InputTokens + message.Usage.OutputTokens),
			Cost:         inputCost + outputCost,
		}, nil
	}

	// Validate confidence range
	if resp.Confidence < 0.0 || resp.Confidence > 1.0 {
		resp.Confidence = 0.85 // Clamp to reasonable default
	}

	// Calculate cost (Sonnet 4 pricing: $3/1M input, $15/1M output)
	inputCost := float64(message.Usage.InputTokens) * 3.0 / 1000000.0
	outputCost := float64(message.Usage.OutputTokens) * 15.0 / 1000000.0
	totalCost := inputCost + outputCost

	return &provider.FixResponse{
		Success:      true,
		FixedContent: resp.FixedContent,
		Explanation:  resp.Explanation,
		Confidence:   resp.Confidence,
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

// enhanceAPIError adds helpful context to Claude API errors using the common error handler.
func enhanceAPIError(err error) error {
	return common.EnhanceAPIError(err, common.ProviderErrorContext{
		ProviderName:      "Claude",
		APIKeysURL:        "https://console.anthropic.com/settings/keys",
		StatusPageURL:     "https://status.anthropic.com",
		AlternateProvider: "openai",
	})
}

// GeneratePlan generates a phased migration plan using Claude
func (p *Provider) GeneratePlan(ctx context.Context, req provider.PlanRequest) (*provider.PlanResponse, error) {
	prompt := buildPlanPrompt(req)

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.F(p.model),
		MaxTokens:   anthropic.F(int64(PlanningMaxTokens)), // Higher limit for planning
		Temperature: anthropic.F(0.3),                      // Slightly higher for creativity in planning
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return &provider.PlanResponse{
			Error: enhanceAPIError(err),
		}, nil
	}

	// Extract the JSON response from Claude
	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
		}
	}

	// Parse the response into phases
	phases, err := parsePlanResponse(responseText, req.Violations)
	if err != nil {
		return &provider.PlanResponse{
			Error: fmt.Errorf("failed to parse plan response: %w", err),
		}, nil
	}

	// Calculate cost
	inputCost := float64(message.Usage.InputTokens) * 3.0 / 1000000.0
	outputCost := float64(message.Usage.OutputTokens) * 15.0 / 1000000.0
	totalCost := inputCost + outputCost

	return &provider.PlanResponse{
		Phases:     phases,
		TokensUsed: int(message.Usage.InputTokens + message.Usage.OutputTokens),
		Cost:       totalCost,
	}, nil
}

// buildPlanPrompt constructs the prompt for plan generation
func buildPlanPrompt(req provider.PlanRequest) string {
	// Convert violations to JSON for the prompt
	violationsJSON, _ := json.MarshalIndent(req.Violations, "", "  ")

	maxPhases := req.MaxPhases
	if maxPhases == 0 {
		maxPhases = 5 // Default to 5 phases
	}

	return fmt.Sprintf(`You are a migration planning expert helping create a phased migration plan for code violations found by Konveyor static analysis.

VIOLATIONS TO ANALYZE:
%s

REQUIREMENTS:
1. Group violations into %d logical phases (or fewer if appropriate)
2. Prioritize phases by: category (mandatory > optional > potential) > effort level
3. For each phase provide:
   - A clear, descriptive name
   - Risk level assessment (low/medium/high)
   - Explanation of WHY these violations are grouped together
   - Recommended execution order
   - Violation IDs to include in this phase
   - Estimated cost per phase ($0.05-0.15 per incident typically)
   - Estimated duration in minutes

GROUPING STRATEGY:
- Group by category first (mandatory, optional, potential)
- Within each category, group by effort level (high effort separate from low effort)
- Consider dependencies and risk
- Explain the reasoning for each grouping

RISK TOLERANCE: %s
- conservative: Smaller phases, lower risk, more phases
- balanced: Moderate phase sizes, mixed complexity
- aggressive: Larger phases, higher efficiency, fewer phases

OUTPUT FORMAT: Return a valid JSON array of phases:
[
  {
    "id": "phase-1",
    "name": "Critical Mandatory Fixes - High Effort",
    "order": 1,
    "risk": "high",
    "category": "mandatory",
    "effort_range": [5, 7],
    "explanation": "These violations require significant refactoring of core APIs...",
    "violation_ids": ["javax-to-jakarta-001", "javax-to-jakarta-002"],
    "estimated_cost": 2.45,
    "estimated_duration_minutes": 15
  }
]

Return ONLY the JSON array with no additional text or markdown formatting.`,
		string(violationsJSON),
		maxPhases,
		req.RiskTolerance)
}

// parsePlanResponse parses Claude's JSON response into PlannedPhase structs
func parsePlanResponse(responseText string, violations []violation.Violation) ([]provider.PlannedPhase, error) {
	// Extract JSON from response (handle markdown code blocks if present)
	jsonStr := extractJSON(responseText)

	var rawPhases []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawPhases); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w\nResponse: %s", err, responseText)
	}

	phases := make([]provider.PlannedPhase, 0, len(rawPhases))
	for _, raw := range rawPhases {
		phase := provider.PlannedPhase{
			ID:          getString(raw, "id"),
			Name:        getString(raw, "name"),
			Order:       getInt(raw, "order"),
			Risk:        getString(raw, "risk"),
			Category:    getString(raw, "category"),
			Explanation: getString(raw, "explanation"),
		}

		// Parse effort_range as [min, max]
		if effortRange, ok := raw["effort_range"].([]interface{}); ok && len(effortRange) >= 2 {
			if min, ok := effortRange[0].(float64); ok {
				phase.EffortRange[0] = int(min)
			}
			if max, ok := effortRange[1].(float64); ok {
				phase.EffortRange[1] = int(max)
			}
		}

		// Parse violation_ids
		if ids, ok := raw["violation_ids"].([]interface{}); ok {
			phase.ViolationIDs = make([]string, 0, len(ids))
			for _, id := range ids {
				if str, ok := id.(string); ok {
					phase.ViolationIDs = append(phase.ViolationIDs, str)
				}
			}
		}

		phase.EstimatedCost = getFloat(raw, "estimated_cost")
		phase.EstimatedDurationMinutes = getInt(raw, "estimated_duration_minutes")

		phases = append(phases, phase)
	}

	return phases, nil
}

// extractJSON extracts JSON from a response that might contain markdown code blocks
func extractJSON(text string) string {
	// Try to extract JSON from markdown code blocks
	re := regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*([\[{].*?[\]}])\s*` + "```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	// If no code blocks, try to find JSON array or object directly
	re = regexp.MustCompile(`(?s)(\[.*\])`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	// Return original text if no JSON pattern found
	return text
}

// Helper functions for safe type conversion from map[string]interface{}
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0.0
}

