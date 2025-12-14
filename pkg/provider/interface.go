package provider

import (
	"context"

	"github.com/tsanders/kantra-ai/pkg/violation"
)

// Provider defines the interface for AI-powered code remediation
type Provider interface {
	// Name returns the provider name (e.g., "claude", "openai")
	Name() string

	// FixViolation requests a fix for a specific incident of a violation
	FixViolation(ctx context.Context, req FixRequest) (*FixResponse, error)

	// EstimateCost estimates the cost for fixing a violation (in USD)
	EstimateCost(req FixRequest) (float64, error)

	// GeneratePlan generates a phased migration plan from violations
	GeneratePlan(ctx context.Context, req PlanRequest) (*PlanResponse, error)

	// FixBatch requests fixes for multiple incidents of the same violation in one API call
	FixBatch(ctx context.Context, req BatchRequest) (*BatchResponse, error)
}

// FixRequest contains all the context needed to fix a violation
type FixRequest struct {
	Violation   violation.Violation
	Incident    violation.Incident
	FileContent string // Full file content
	Language    string // Programming language (java, python, go, etc.)
}

// FixResponse contains the AI's fix attempt
type FixResponse struct {
	Success      bool    // Whether the fix was successful
	FixedContent string  // The fixed file content
	Explanation  string  // AI's explanation of what was changed
	Confidence   float64 // Confidence score (0.0-1.0)
	TokensUsed   int     // Number of tokens consumed
	Cost         float64 // Cost in USD
	Error        error   // Error if fix failed
}

// Config holds provider configuration
type Config struct {
	Name        string  // Provider name: claude, openai, or preset (groq, ollama, etc.)
	APIKey      string  // API key
	Model       string  // Model to use
	Temperature float64 // Temperature (0.0-1.0)
	BaseURL     string  // Custom base URL for OpenAI-compatible APIs
}

// PlanRequest contains the context needed to generate a migration plan
type PlanRequest struct {
	Violations      []violation.Violation // All violations to plan for
	MaxPhases       int                   // Maximum number of phases (0 = auto)
	RiskTolerance   string                // conservative | balanced | aggressive
}

// PlanResponse contains the generated migration plan
type PlanResponse struct {
	Phases      []PlannedPhase // Phases in recommended execution order
	TokensUsed  int            // Number of tokens consumed
	Cost        float64        // Cost in USD
	Error       error          // Error if plan generation failed
}

// PlannedPhase represents a phase in the migration plan
type PlannedPhase struct {
	ID                       string                // Unique phase identifier
	Name                     string                // Human-readable phase name
	Order                    int                   // Execution order (1-based)
	Risk                     string                // low | medium | high
	Category                 string                // Violation category
	EffortRange              [2]int                // Min and max effort levels
	Explanation              string                // AI explanation of why these are grouped
	ViolationIDs             []string              // Violation IDs in this phase
	EstimatedCost            float64               // Estimated cost for this phase
	EstimatedDurationMinutes int                   // Estimated time in minutes
}

// BatchRequest contains multiple incidents to fix in one API call
type BatchRequest struct {
	Violation    violation.Violation   // Shared violation context
	Incidents    []violation.Incident  // Multiple incidents to fix together
	FileContents map[string]string     // file path → file content
	Language     string                // Programming language
}

// BatchResponse contains fixes for multiple incidents
type BatchResponse struct {
	Fixes      []IncidentFix // One fix per incident
	Success    bool          // Overall success (true only if all succeeded)
	TokensUsed int           // Total tokens consumed
	Cost       float64       // Total cost in USD
	Error      error         // Error if batch processing failed
}

// IncidentFix represents a single fix within a batch response
type IncidentFix struct {
	IncidentURI  string  // Which incident this fixes (matches incident.URI)
	Success      bool    // Whether this specific fix succeeded
	FixedContent string  // Fixed file content
	Explanation  string  // AI's explanation of the change
	Confidence   float64 // Confidence score (0.0-1.0)
	Error        error   // Error if this fix failed
}

// ProviderPresets maps provider names to their OpenAI-compatible base URLs
// This allows users to use --provider=groq instead of manually setting base URLs
var ProviderPresets = map[string]ProviderPreset{
	"groq": {
		BaseURL:     "https://api.groq.com/openai/v1",
		Description: "Groq - Fast inference with Llama, Mixtral, and Gemma models",
		DefaultModel: "llama-3.1-70b-versatile",
	},
	"together": {
		BaseURL:     "https://api.together.xyz/v1",
		Description: "Together AI - Open source models (Llama, Mixtral, Qwen, etc.)",
		DefaultModel: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
	},
	"anyscale": {
		BaseURL:     "https://api.endpoints.anyscale.com/v1",
		Description: "Anyscale - Llama, Mistral, and Mixtral models",
		DefaultModel: "meta-llama/Meta-Llama-3.1-70B-Instruct",
	},
	"perplexity": {
		BaseURL:     "https://api.perplexity.ai",
		Description: "Perplexity AI - Llama and Mistral models with online context",
		DefaultModel: "llama-3.1-sonar-large-128k-online",
	},
	"ollama": {
		BaseURL:     "http://localhost:11434/v1",
		Description: "Ollama - Local models (requires Ollama running locally)",
		DefaultModel: "codellama",
	},
	"lmstudio": {
		BaseURL:     "http://localhost:1234/v1",
		Description: "LM Studio - Local models (requires LM Studio running locally)",
		DefaultModel: "local-model",
	},
	"openrouter": {
		BaseURL:     "https://openrouter.ai/api/v1",
		Description: "OpenRouter - Access to 100+ models through one API",
		DefaultModel: "meta-llama/llama-3.1-70b-instruct",
	},
}

// ProviderPreset contains configuration for a provider preset
type ProviderPreset struct {
	BaseURL      string // OpenAI-compatible base URL
	Description  string // Human-readable description
	DefaultModel string // Default model for this provider
}
