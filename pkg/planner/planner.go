package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// Planner generates migration plans from violations
type Planner struct {
	config Config
}

// New creates a new Planner
func New(config Config) *Planner {
	// Set defaults
	if config.OutputPath == "" {
		config.OutputPath = ".kantra-ai-plan.yaml"
	}
	if config.RiskTolerance == "" {
		config.RiskTolerance = "balanced"
	}

	return &Planner{
		config: config,
	}
}

// Generate creates a migration plan
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

	// Save plan to file
	if err := planfile.SavePlan(plan, p.config.OutputPath); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	return &Result{
		PlanPath:     p.config.OutputPath,
		TotalPhases:  len(plan.Phases),
		TotalCost:    plan.GetTotalCost(),
		TokensUsed:   planResp.TokensUsed,
		GenerateCost: planResp.Cost,
	}, nil
}

// buildPlan converts provider response to planfile.Plan
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
					ViolationID:   v.ID,
					Description:   v.Description,
					Category:      v.Category,
					Effort:        v.Effort,
					IncidentCount: len(v.Incidents),
					Incidents:     v.Incidents,
				}
				phase.Violations = append(phase.Violations, plannedViolation)
			}
		}

		plan.Phases = append(plan.Phases, phase)
	}

	return plan
}

// mapRiskLevel converts string risk level to planfile.RiskLevel
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
