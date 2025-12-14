package planner

import (
	"github.com/tsanders/kantra-ai/pkg/provider"
)

// Config holds configuration for plan generation
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

// Result contains the result of plan generation
type Result struct {
	PlanPath     string  // Path where plan was saved
	TotalPhases  int     // Number of phases generated
	TotalCost    float64 // Estimated total cost
	TokensUsed   int     // Tokens consumed for plan generation
	GenerateCost float64 // Cost to generate the plan
}
