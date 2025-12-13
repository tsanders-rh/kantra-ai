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
