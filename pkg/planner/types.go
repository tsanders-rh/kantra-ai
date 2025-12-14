// Package planner generates AI-powered migration plans from Konveyor violations.
//
// The planner analyzes violations and uses AI to group them into logical phases
// based on risk, category, and effort. Each phase includes AI-generated explanations
// of why violations are grouped together and estimated costs/durations.
package planner

import (
	"github.com/tsanders/kantra-ai/pkg/provider"
)

// Config holds configuration for plan generation.
type Config struct {
	AnalysisPath  string   // Path to Konveyor output.yaml
	InputPath     string   // Path to source code directory
	Provider      provider.Provider
	OutputPath    string   // Where to save the plan (default: .kantra-ai-plan.yaml)
	MaxPhases     int      // Maximum number of phases (0 = auto)
	RiskTolerance string   // conservative | balanced | aggressive
	Categories    []string // Filter by categories
	ViolationIDs  []string // Filter by violation IDs
	MaxEffort     int      // Only include violations with effort <= this value
	Interactive   bool     // Enable interactive approval mode
}

// Result contains the result of plan generation with cost and phase metrics.
type Result struct {
	PlanPath     string  // Path where plan was saved
	TotalPhases  int     // Number of phases generated
	TotalCost    float64 // Estimated total cost
	TokensUsed   int     // Tokens consumed for plan generation
	GenerateCost float64 // Cost to generate the plan
}
