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
	Name        string // Provider name: claude, openai
	APIKey      string // API key
	Model       string // Model to use
	Temperature float64 // Temperature (0.0-1.0)
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
