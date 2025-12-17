package planner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tsanders/kantra-ai/pkg/confidence"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// Planner generates AI-powered migration plans from violations.
type Planner struct {
	config Config
}

// New creates a new Planner with the given configuration.
// It sets default values for OutputPath and RiskTolerance if not provided.
func New(config Config) *Planner {
	// Set defaults
	if config.OutputPath == "" {
		config.OutputPath = ".kantra-ai-plan"
	}
	if config.RiskTolerance == "" {
		config.RiskTolerance = "balanced"
	}

	return &Planner{
		config: config,
	}
}

// Generate creates an AI-powered migration plan by analyzing violations,
// filtering based on configuration, and using the AI provider to group
// violations into phases with risk assessment and explanations.
// If Interactive mode is enabled, prompts the user to approve/defer each phase.
func (p *Planner) Generate(ctx context.Context) (*Result, error) {
	// Load violations from analysis file
	analysis, err := violation.LoadAnalysis(p.config.AnalysisPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load violations: %w", err)
	}

	// Apply filters using the Analysis method
	filtered := analysis.FilterViolations(p.config.ViolationIDs, p.config.Categories, p.config.MaxEffort)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no violations match the specified filters")
	}

	// Call AI provider to generate plan
	planReq := provider.PlanRequest{
		Violations:    filtered,
		MaxPhases:     p.config.MaxPhases,
		RiskTolerance: p.config.RiskTolerance,
	}

	planResp, err := p.config.Provider.GeneratePlan(ctx, planReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}
	if planResp.Error != nil {
		return nil, planResp.Error
	}

	// Convert provider response to planfile.Plan
	plan := p.buildPlan(planResp, filtered)

	// Run interactive approval if enabled
	if p.config.Interactive {
		approval := NewInteractiveApproval(plan)
		if err := approval.Run(); err != nil {
			return nil, fmt.Errorf("interactive approval failed: %w", err)
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(p.config.OutputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// OutputPath is now a directory, save plan.yaml inside it
	planPath := filepath.Join(p.config.OutputPath, "plan.yaml")
	if err := planfile.SavePlan(plan, planPath); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	return &Result{
		Plan:         plan,
		PlanPath:     planPath,
		TotalPhases:  len(plan.Phases),
		TotalCost:    plan.GetTotalCost(),
		TokensUsed:   planResp.TokensUsed,
		GenerateCost: planResp.Cost,
	}, nil
}

// buildPlan converts the AI provider's response into a planfile.Plan structure.
// It maps violations from the provider response to the plan format and sets metadata.
func (p *Planner) buildPlan(resp *provider.PlanResponse, violations []violation.Violation) *planfile.Plan {
	plan := planfile.NewPlan(p.config.Provider.Name(), len(violations))
	plan.Metadata.CreatedAt = time.Now()

	// Create a map for quick violation lookup
	violationMap := make(map[string]violation.Violation)
	for _, v := range violations {
		violationMap[v.ID] = v
	}

	// Convert provider phases to planfile phases
	for _, providerPhase := range resp.Phases {
		phase := planfile.Phase{
			ID:                       providerPhase.ID,
			Name:                     providerPhase.Name,
			Order:                    providerPhase.Order,
			Risk:                     mapRiskLevel(providerPhase.Risk),
			Category:                 providerPhase.Category,
			EffortRange:              providerPhase.EffortRange,
			Explanation:              providerPhase.Explanation,
			Violations:               make([]planfile.PlannedViolation, 0),
			EstimatedCost:            providerPhase.EstimatedCost,
			EstimatedDurationMinutes: providerPhase.EstimatedDurationMinutes,
			Deferred:                 false,
		}

		// Add violations to phase
		for _, violationID := range providerPhase.ViolationIDs {
			if v, ok := violationMap[violationID]; ok {
				plannedViolation := planfile.PlannedViolation{
					ViolationID:         v.ID,
					Description:         v.Description,
					Category:            v.Category,
					Effort:              v.Effort,
					MigrationComplexity: v.MigrationComplexity,
					ManualReviewRequired: isHighComplexity(v.MigrationComplexity, v.Effort),
					IncidentCount:       len(v.Incidents),
					Incidents:           v.Incidents,
				}
				phase.Violations = append(phase.Violations, plannedViolation)
			}
		}

		plan.Phases = append(plan.Phases, phase)
	}

	return plan
}

// mapRiskLevel converts a string risk level ("low", "medium", "high")
// to a planfile.RiskLevel constant. Returns RiskMedium as default for unknown values.
func mapRiskLevel(risk string) planfile.RiskLevel {
	switch risk {
	case "low":
		return planfile.RiskLow
	case "medium":
		return planfile.RiskMedium
	case "high":
		return planfile.RiskHigh
	default:
		return planfile.RiskMedium
	}
}

// isHighComplexity determines if a violation has high or expert complexity
// and requires manual review. Uses effort level as fallback if no complexity metadata.
func isHighComplexity(migrationComplexity string, effort int) bool {
	return confidence.IsHighComplexity(migrationComplexity, effort, true)
}
