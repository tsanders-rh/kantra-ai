package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

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

var (
	// Compiled regexes for JSON extraction (compiled once at package init time)
	jsonCodeBlockRegex = regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*([\[{].*?[\]}])\s*` + "```")
	jsonArrayRegex     = regexp.MustCompile(`(?s)(\[.*\])`)
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

// isRateLimitError checks if an error is a rate limit error (429)
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return regexp.MustCompile(`(?i)rate.limit|429|too many requests`).MatchString(errStr)
}

// GeneratePlan generates a phased migration plan using Claude
// If there are too many violations, it batches them to avoid rate limits
func (p *Provider) GeneratePlan(ctx context.Context, req provider.PlanRequest) (*provider.PlanResponse, error) {
	const maxViolationsPerBatch = 15 // Keep batches small to avoid token limits

	// If violations fit in one batch, use direct approach
	if len(req.Violations) <= maxViolationsPerBatch {
		return p.generatePlanDirect(ctx, req)
	}

	// Otherwise, batch the violations
	fmt.Printf("📦 Batching %d violations into smaller groups to avoid rate limits...\n", len(req.Violations))
	return p.generatePlanBatched(ctx, req, maxViolationsPerBatch)
}

// generatePlanDirect generates a plan directly without batching
func (p *Provider) generatePlanDirect(ctx context.Context, req provider.PlanRequest) (*provider.PlanResponse, error) {
	prompt := buildPlanPrompt(req)

	// Retry logic for rate limits
	var message *anthropic.Message
	var err error
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		message, err = p.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       anthropic.F(p.model),
			MaxTokens:   anthropic.F(int64(PlanningMaxTokens)), // Higher limit for planning
			Temperature: anthropic.F(0.3),                      // Slightly higher for creativity in planning
			Messages: anthropic.F([]anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			}),
		})

		// Success - break out of retry loop
		if err == nil {
			break
		}

		// Check if it's a rate limit error
		if !isRateLimitError(err) {
			// Not a rate limit error, return immediately
			return &provider.PlanResponse{
				Error: enhanceAPIError(err),
			}, nil
		}

		// Rate limit error - check if we should retry
		if attempt == maxRetries {
			// Max retries reached
			return &provider.PlanResponse{
				Error: enhanceAPIError(err),
			}, nil
		}

		// Calculate backoff delay (exponential: 30s, 60s, 90s)
		backoff := time.Duration(30*(attempt+1)) * time.Second
		fmt.Printf("   ⏳ Rate limit hit, waiting %ds before retry %d/%d...\n",
			int(backoff.Seconds()), attempt+1, maxRetries)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return &provider.PlanResponse{
				Error: ctx.Err(),
			}, nil
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

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

// generatePlanBatched splits violations into batches, generates mini-plans, and merges them
func (p *Provider) generatePlanBatched(ctx context.Context, req provider.PlanRequest, batchSize int) (*provider.PlanResponse, error) {
	// Split violations into batches
	batches := batchViolations(req.Violations, batchSize)
	fmt.Printf("   Split into %d batches\n\n", len(batches))

	var allPhases []provider.PlannedPhase
	var totalTokens int
	var totalCost float64

	// Generate a mini-plan for each batch
	for i, batch := range batches {
		// Update progress bar
		updateBatchProgress(i, len(batches), "Processing")

		// Add delay between batches to respect rate limits
		if i > 0 {
			delay := calculateBatchDelay(totalTokens, i)

			// Show waiting status in progress bar
			updateBatchProgress(i, len(batches), fmt.Sprintf("Waiting %ds", int(delay.Seconds())))

			select {
			case <-ctx.Done():
				fmt.Println()
				return &provider.PlanResponse{
					Error: ctx.Err(),
				}, nil
			case <-time.After(delay):
				// Continue
			}
		}

		batchReq := provider.PlanRequest{
			Violations:    batch,
			MaxPhases:     req.MaxPhases,
			RiskTolerance: req.RiskTolerance,
		}

		resp, err := p.generatePlanDirect(ctx, batchReq)
		if err != nil {
			fmt.Println()
			return &provider.PlanResponse{
				Error: fmt.Errorf("failed to generate plan for batch %d: %w", i+1, err),
			}, nil
		}
		if resp.Error != nil {
			fmt.Println()
			return resp, nil
		}

		allPhases = append(allPhases, resp.Phases...)
		totalTokens += resp.TokensUsed
		totalCost += resp.Cost
	}

	// Complete the progress bar
	updateBatchProgress(len(batches), len(batches), "Complete")
	fmt.Println()

	// Merge and reorganize phases
	fmt.Printf("\n   Merging %d phases from all batches...\n", len(allPhases))
	mergedPhases := mergePhases(allPhases, req.MaxPhases)
	fmt.Printf("✓ Generated %d final phases\n", len(mergedPhases))

	return &provider.PlanResponse{
		Phases:     mergedPhases,
		TokensUsed: totalTokens,
		Cost:       totalCost,
	}, nil
}

// calculateBatchDelay calculates how long to wait between batches to respect rate limits
// Claude has a 30,000 tokens/minute limit, so we need to space requests accordingly
func calculateBatchDelay(tokensSoFar int, batchIndex int) time.Duration {
	// Conservative delay: wait 20 seconds between batches
	// This ensures we stay well under 30k tokens/minute
	// (each batch is ~8-12k tokens, so 20s gives us plenty of headroom)
	baseDelay := 20 * time.Second

	// If we've already used a lot of tokens, wait a bit longer
	if tokensSoFar > 20000 {
		return 30 * time.Second
	}

	return baseDelay
}

// updateBatchProgress displays a progress bar for batch processing
func updateBatchProgress(current, total int, status string) {
	// Calculate progress (0.0 to 1.0)
	progress := float64(current) / float64(total)

	// Build progress bar (40 characters wide)
	barWidth := 40
	filled := int(progress * float64(barWidth))
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	// Calculate percentage
	percentage := int(progress * 100)

	// Print progress bar
	fmt.Printf("\r   Progress [%s] %d%% | Batch %d/%d | %s     ",
		bar, percentage, current, total, status)
}

// batchViolations splits violations into batches of specified size
func batchViolations(violations []violation.Violation, batchSize int) [][]violation.Violation {
	var batches [][]violation.Violation
	for i := 0; i < len(violations); i += batchSize {
		end := i + batchSize
		if end > len(violations) {
			end = len(violations)
		}
		batches = append(batches, violations[i:end])
	}
	return batches
}

// mergePhases merges mini-plans into a cohesive final plan
func mergePhases(phases []provider.PlannedPhase, maxPhases int) []provider.PlannedPhase {
	if len(phases) == 0 {
		return phases
	}

	// Group phases by category and risk level
	type phaseKey struct {
		category string
		risk     string
	}

	groups := make(map[phaseKey][]provider.PlannedPhase)
	for _, phase := range phases {
		key := phaseKey{
			category: phase.Category,
			risk:     phase.Risk,
		}
		groups[key] = append(groups[key], phase)
	}

	// Merge phases within each group
	var merged []provider.PlannedPhase
	for key, groupPhases := range groups {
		if len(groupPhases) == 1 {
			merged = append(merged, groupPhases[0])
			continue
		}

		// Merge multiple phases into one
		mergedPhase := provider.PlannedPhase{
			ID:                       fmt.Sprintf("phase-%s-%s", key.category, key.risk),
			Name:                     fmt.Sprintf("%s - %s Risk", key.category, key.risk),
			Risk:                     key.risk,
			Category:                 key.category,
			ViolationIDs:             []string{},
			EstimatedCost:            0,
			EstimatedDurationMinutes: 0,
		}

		// Aggregate from all phases in this group
		minEffort := 10
		maxEffort := 0
		for _, phase := range groupPhases {
			mergedPhase.ViolationIDs = append(mergedPhase.ViolationIDs, phase.ViolationIDs...)
			mergedPhase.EstimatedCost += phase.EstimatedCost
			mergedPhase.EstimatedDurationMinutes += phase.EstimatedDurationMinutes

			if phase.EffortRange[0] < minEffort {
				minEffort = phase.EffortRange[0]
			}
			if phase.EffortRange[1] > maxEffort {
				maxEffort = phase.EffortRange[1]
			}
		}

		mergedPhase.EffortRange = [2]int{minEffort, maxEffort}
		mergedPhase.Explanation = fmt.Sprintf("Merged %d phases with %s violations at %s risk level.",
			len(groupPhases), key.category, key.risk)

		merged = append(merged, mergedPhase)
	}

	// Sort by priority: mandatory > optional > potential, then high risk > medium > low
	sortPhasesByPriority(merged)

	// Reassign order and IDs
	for i := range merged {
		merged[i].Order = i + 1
		merged[i].ID = fmt.Sprintf("phase-%d", i+1)
	}

	// Limit to maxPhases if specified
	if maxPhases > 0 && len(merged) > maxPhases {
		merged = merged[:maxPhases]
	}

	return merged
}

// sortPhasesByPriority sorts phases by category and risk
func sortPhasesByPriority(phases []provider.PlannedPhase) {
	// Simple bubble sort for small slices
	for i := 0; i < len(phases); i++ {
		for j := i + 1; j < len(phases); j++ {
			if phasePriority(phases[i]) > phasePriority(phases[j]) {
				phases[i], phases[j] = phases[j], phases[i]
			}
		}
	}
}

// phasePriority returns a priority score (lower = higher priority)
func phasePriority(phase provider.PlannedPhase) int {
	priority := 0

	// Category priority (mandatory = 0, optional = 100, potential = 200)
	switch phase.Category {
	case "mandatory":
		priority += 0
	case "optional":
		priority += 100
	case "potential":
		priority += 200
	default:
		priority += 300
	}

	// Risk priority (high = 0, medium = 10, low = 20)
	switch phase.Risk {
	case "high":
		priority += 0
	case "medium":
		priority += 10
	case "low":
		priority += 20
	default:
		priority += 30
	}

	return priority
}

// buildPlanPrompt constructs the prompt for plan generation
func buildPlanPrompt(req provider.PlanRequest) string {
	// Create a lightweight version of violations (without full incident details)
	// to avoid exceeding token limits
	type lightweightViolation struct {
		ID                  string `json:"id"`
		Description         string `json:"description"`
		Category            string `json:"category"`
		Effort              int    `json:"effort"`
		IncidentCount       int    `json:"incident_count"`
		MigrationComplexity string `json:"migration_complexity,omitempty"`
	}

	lightViolations := make([]lightweightViolation, len(req.Violations))
	for i, v := range req.Violations {
		lightViolations[i] = lightweightViolation{
			ID:                  v.ID,
			Description:         v.Description,
			Category:            v.Category,
			Effort:              v.Effort,
			IncidentCount:       len(v.Incidents),
			MigrationComplexity: v.MigrationComplexity,
		}
	}

	violationsJSON, _ := json.MarshalIndent(lightViolations, "", "  ")

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
	// Try to extract JSON from markdown code blocks using pre-compiled regex
	matches := jsonCodeBlockRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	// If no code blocks, try to find JSON array or object directly using pre-compiled regex
	matches = jsonArrayRegex.FindStringSubmatch(text)
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

