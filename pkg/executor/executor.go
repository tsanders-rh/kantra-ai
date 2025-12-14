package executor

import (
	"context"
	"fmt"

	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/ux"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// Executor executes migration plans
type Executor struct {
	config Config
	plan   *planfile.Plan
	state  *planfile.ExecutionState
}

// New creates a new Executor
func New(config Config) (*Executor, error) {
	// Set defaults
	if config.PlanPath == "" {
		config.PlanPath = ".kantra-ai-plan.yaml"
	}
	if config.StatePath == "" {
		config.StatePath = ".kantra-ai-state.yaml"
	}
	if config.Progress == nil {
		config.Progress = &ux.NoOpProgressWriter{}
	}

	return &Executor{
		config: config,
	}, nil
}

// Execute runs the plan execution
func (e *Executor) Execute(ctx context.Context) (*Result, error) {
	// Load plan
	plan, err := planfile.LoadPlan(e.config.PlanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}
	e.plan = plan

	// Load or create state
	state, err := planfile.LoadState(e.config.StatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	if state == nil {
		// Create new state
		state = planfile.NewState(e.config.PlanPath, len(plan.Phases))
	}
	e.state = state

	// Check for resume
	if e.state.HasFailures() && e.config.Resume {
		e.config.Progress.Info("Resuming from last failure...")
		e.config.Progress.Info("Last failure: %s at %s",
			e.state.LastFailure.ViolationID,
			e.state.LastFailure.IncidentURI)
	}

	// Determine which phases to execute
	phasesToExecute := e.getPhasesToExecute()
	if len(phasesToExecute) == 0 {
		return nil, fmt.Errorf("no phases to execute")
	}

	result := &Result{
		TotalPhases: len(plan.Phases),
		StatePath:   e.config.StatePath,
	}

	// Execute phases
	for _, phase := range phasesToExecute {
		phaseResult := e.executePhase(ctx, &phase)

		result.ExecutedPhases++
		result.TotalFixes += phaseResult.SuccessfulFixes + phaseResult.FailedFixes
		result.SuccessfulFixes += phaseResult.SuccessfulFixes
		result.FailedFixes += phaseResult.FailedFixes
		result.TotalCost += phaseResult.Cost
		result.TotalTokens += phaseResult.Tokens

		if phaseResult.Error != nil {
			result.FailedPhases++
			e.config.Progress.Error("Phase %s failed: %v", phase.ID, phaseResult.Error)

			// Save state and return on failure
			if err := planfile.SaveState(e.state, e.config.StatePath); err != nil {
				return result, fmt.Errorf("phase failed and could not save state: %w", err)
			}
			return result, phaseResult.Error
		}

		result.CompletedPhases++

		// Save state after each phase
		if err := planfile.SaveState(e.state, e.config.StatePath); err != nil {
			return result, fmt.Errorf("failed to save state: %w", err)
		}
	}

	return result, nil
}

// getPhasesToExecute determines which phases should be executed
func (e *Executor) getPhasesToExecute() []planfile.Phase {
	phases := make([]planfile.Phase, 0)

	for _, phase := range e.plan.Phases {
		// Skip deferred phases
		if phase.Deferred {
			continue
		}

		// If specific phase requested, only execute that one
		if e.config.PhaseID != "" && phase.ID != e.config.PhaseID {
			continue
		}

		// Skip already completed phases (unless resuming)
		if !e.config.Resume {
			phaseStatus := e.state.GetPhaseStatus(phase.ID)
			if phaseStatus != nil && phaseStatus.Status == planfile.StatusCompleted {
				continue
			}
		}

		phases = append(phases, phase)
	}

	return phases
}

// executePhase executes a single phase
func (e *Executor) executePhase(ctx context.Context, phase *planfile.Phase) PhaseResult {
	result := PhaseResult{
		PhaseID:   phase.ID,
		PhaseName: phase.Name,
	}

	e.config.Progress.StartPhase(phase.Name)
	e.state.MarkPhaseStarted(phase.ID)

	// Create fixer
	f := fixer.New(e.config.Provider, e.config.InputPath, e.config.DryRun)

	// Execute fixes for each violation in the phase
	for _, plannedViolation := range phase.Violations {
		// Check if we should skip this violation (already completed)
		violationStatus, exists := e.state.Violations[plannedViolation.ViolationID]
		if exists && violationStatus.Status == planfile.StatusCompleted && !e.config.Resume {
			continue
		}

		// Fix each incident
		for _, incident := range plannedViolation.Incidents {
			incidentURI := incident.URI

			// Skip if already completed
			if exists {
				if incidentStatus, ok := violationStatus.Incidents[incidentURI]; ok {
					if incidentStatus.Status == planfile.StatusCompleted {
						continue
					}
				}
			}

			// Build violation object for fixer
			violation := e.buildViolation(plannedViolation)

			// Attempt fix
			fixResult, err := f.FixIncident(ctx, violation, incident)

			if err != nil || !fixResult.Success {
				result.FailedFixes++
				errorMsg := ""
				if err != nil {
					errorMsg = err.Error()
				} else if fixResult.Error != nil {
					errorMsg = fixResult.Error.Error()
				}

				e.state.RecordIncidentFailure(phase.ID, plannedViolation.ViolationID, incidentURI, errorMsg)

				// Continue to next incident
				continue
			}

			// Record successful fix
			result.SuccessfulFixes++
			result.Cost += fixResult.Cost
			result.Tokens += fixResult.TokensUsed

			e.state.RecordIncidentFix(plannedViolation.ViolationID, incidentURI, fixResult.Cost)
		}
	}

	// Mark phase as completed
	e.state.MarkPhaseCompleted(phase.ID)

	// Update phase status with results
	phaseStatus := e.state.GetPhaseStatus(phase.ID)
	if phaseStatus != nil {
		phaseStatus.FixesApplied = result.SuccessfulFixes
		phaseStatus.Cost = result.Cost
		e.state.UpdatePhaseStatus(*phaseStatus)
	}

	e.config.Progress.EndPhase()

	return result
}

// buildViolation constructs a violation.Violation from a PlannedViolation
func (e *Executor) buildViolation(pv planfile.PlannedViolation) violation.Violation {
	return violation.Violation{
		ID:          pv.ViolationID,
		Description: pv.Description,
		Category:    pv.Category,
		Effort:      pv.Effort,
		Incidents:   pv.Incidents,
		Rule: violation.Rule{
			ID:      pv.ViolationID,
			Message: pv.Description,
		},
	}
}
