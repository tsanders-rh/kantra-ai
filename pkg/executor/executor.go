package executor

import (
	"context"
	"fmt"

	"github.com/tsanders/kantra-ai/pkg/confidence"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/gitutil"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/ux"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// Executor executes migration plans with state tracking and resume capability.
type Executor struct {
	config Config
	plan   *planfile.Plan
	state  *planfile.ExecutionState
}

// New creates a new Executor with the given configuration.
// It sets default values for PlanPath, StatePath, Progress, and BatchConfig if not provided.
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
	if config.BatchConfig.MaxBatchSize == 0 && config.BatchConfig.Parallelism == 0 {
		// Use default batch config if not specified
		config.BatchConfig = fixer.DefaultBatchConfig()
	}

	return &Executor{
		config: config,
	}, nil
}

// Execute runs the plan execution, processing violations phase-by-phase.
// It loads the plan and state files, determines which phases to execute,
// and runs fixes for each incident. State is saved after each phase to
// enable resume capability. Returns detailed execution results and metrics.
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

	// Initialize confidence stats if enabled
	if e.config.ConfidenceConfig.Enabled {
		result.ConfidenceStats = confidence.NewStats()
	}

	// Execute phases
	for _, phase := range phasesToExecute {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		phaseResult := e.executePhase(ctx, &phase)

		result.ExecutedPhases++
		result.TotalFixes += phaseResult.SuccessfulFixes + phaseResult.FailedFixes
		result.SuccessfulFixes += phaseResult.SuccessfulFixes
		result.FailedFixes += phaseResult.FailedFixes
		result.SkippedFixes += phaseResult.SkippedFixes
		result.DuplicateFixes += phaseResult.DuplicateFixes
		result.TotalCost += phaseResult.Cost
		result.TotalTokens += phaseResult.Tokens

		// Merge phase confidence stats into overall stats
		if result.ConfidenceStats != nil && phaseResult.ConfidenceStats != nil {
			result.ConfidenceStats.TotalFixes += phaseResult.ConfidenceStats.TotalFixes
			result.ConfidenceStats.AppliedFixes += phaseResult.ConfidenceStats.AppliedFixes
			result.ConfidenceStats.SkippedFixes += phaseResult.ConfidenceStats.SkippedFixes

			// Ensure the ByComplexity map is initialized
			if result.ConfidenceStats.ByComplexity == nil {
				result.ConfidenceStats.ByComplexity = make(map[string]*confidence.ComplexityStats)
			}

			// Merge complexity-level stats with nil checks
			if phaseResult.ConfidenceStats.ByComplexity != nil {
				for complexity, phaseComplexityStats := range phaseResult.ConfidenceStats.ByComplexity {
					// Skip nil entries
					if phaseComplexityStats == nil {
						continue
					}

					if _, ok := result.ConfidenceStats.ByComplexity[complexity]; !ok {
						result.ConfidenceStats.ByComplexity[complexity] = &confidence.ComplexityStats{}
					}
					result.ConfidenceStats.ByComplexity[complexity].Total += phaseComplexityStats.Total
					result.ConfidenceStats.ByComplexity[complexity].Applied += phaseComplexityStats.Applied
					result.ConfidenceStats.ByComplexity[complexity].Skipped += phaseComplexityStats.Skipped
				}
			}
		}

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

	// Finalize git commits if enabled
	if e.config.VerifiedTracker != nil && !e.config.DryRun {
		if err := e.config.VerifiedTracker.Finalize(); err != nil {
			e.config.Progress.Error("Failed to finalize verified commits: %v", err)
		}
	} else if e.config.CommitTracker != nil && !e.config.DryRun {
		if err := e.config.CommitTracker.Finalize(); err != nil {
			e.config.Progress.Error("Failed to finalize commits: %v", err)
		}
	}

	// Finalize PR creation if enabled
	if e.config.PRTracker != nil && !e.config.DryRun {
		if err := e.config.PRTracker.Finalize(); err != nil {
			e.config.Progress.Error("Failed to finalize PR: %v", err)
		}
	}

	// Collect commit information from trackers
	if e.config.VerifiedTracker != nil {
		result.Commits = e.config.VerifiedTracker.GetCommitTracker().GetCommits()
	} else if e.config.CommitTracker != nil {
		result.Commits = e.config.CommitTracker.GetCommits()
	}

	// Collect PR information from tracker
	if e.config.PRTracker != nil {
		createdPRs := e.config.PRTracker.GetCreatedPRs()
		result.PRs = make([]gitutil.PRInfo, len(createdPRs))
		for i, pr := range createdPRs {
			result.PRs[i] = gitutil.PRInfo{
				Number:      pr.Number,
				URL:         pr.URL,
				Title:       pr.Title,
				BranchName:  pr.BranchName,
				ViolationID: pr.ViolationID,
				PhaseID:     pr.PhaseID,
				CommitSHAs:  pr.CommitSHAs,
				Timestamp:   pr.Timestamp,
			}
		}
	}

	return result, nil
}

// getPhasesToExecute determines which phases should be executed based on
// configuration filters (PhaseID, deferred status) and resume state.
// Returns a list of phases to execute in order.
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

// executePhase executes a single phase by processing violations using batch processing
// when enabled. It tracks successes and failures in the state file and returns detailed
// metrics for the phase.
func (e *Executor) executePhase(ctx context.Context, phase *planfile.Phase) PhaseResult {
	result := PhaseResult{
		PhaseID:   phase.ID,
		PhaseName: phase.Name,
	}

	e.config.Progress.StartPhase(phase.Name)
	e.state.MarkPhaseStarted(phase.ID)

	// Create batch fixer with confidence configuration
	batchFixer := fixer.NewBatchFixerWithConfidence(
		e.config.Provider,
		e.config.InputPath,
		e.config.DryRun,
		e.config.BatchConfig,
		e.config.ConfidenceConfig,
	)

	// Create stats tracker for confidence filtering (if enabled)
	var confidenceStats *confidence.Stats
	if e.config.ConfidenceConfig.Enabled {
		confidenceStats = confidence.NewStats()
	}

	// Track seen incidents to detect duplicates
	// Key format: "violationID:filePath:lineNumber"
	seenIncidents := make(map[string]bool)

	// Execute fixes for each violation in the phase
	for _, plannedViolation := range phase.Violations {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			return result
		default:
		}

		// Check if we should skip this violation (already completed)
		violationStatus, exists := e.state.Violations[plannedViolation.ViolationID]
		if exists && violationStatus.Status == planfile.StatusCompleted && !e.config.Resume {
			continue
		}

		// Filter incidents that need fixing
		incidentsToFix := make([]violation.Incident, 0, len(plannedViolation.Incidents))
		skippedCount := 0
		duplicateCount := 0
		for _, incident := range plannedViolation.Incidents {
			incidentURI := incident.URI

			// Skip if already completed
			if exists {
				if incidentStatus, ok := violationStatus.Incidents[incidentURI]; ok {
					if incidentStatus.Status == planfile.StatusCompleted {
						skippedCount++
						continue
					}
				}
			}

			// Check for duplicate (same file + line + violation)
			incidentKey := fmt.Sprintf("%s:%s:%d", plannedViolation.ViolationID, incident.GetFilePath(), incident.LineNumber)
			if seenIncidents[incidentKey] {
				duplicateCount++
				continue
			}
			seenIncidents[incidentKey] = true

			incidentsToFix = append(incidentsToFix, incident)
		}

		// Track skipped incidents
		result.SkippedFixes += skippedCount
		result.DuplicateFixes += duplicateCount

		if len(incidentsToFix) == 0 {
			// All incidents already fixed or duplicates - skip this violation
			if (skippedCount > 0 || duplicateCount > 0) && e.config.Progress != nil {
				msg := ""
				if skippedCount > 0 && duplicateCount > 0 {
					msg = fmt.Sprintf("   ⏭️  Skipped %d already-fixed and %d duplicate incident(s) for %s",
						skippedCount, duplicateCount, plannedViolation.ViolationID)
				} else if skippedCount > 0 {
					msg = fmt.Sprintf("   ⏭️  Skipped %d already-fixed incident(s) for %s",
						skippedCount, plannedViolation.ViolationID)
				} else {
					msg = fmt.Sprintf("   ⏭️  Skipped %d duplicate incident(s) for %s",
						duplicateCount, plannedViolation.ViolationID)
				}
				e.config.Progress.Info(msg)
			}
			continue
		}

		// Build violation object with incidents to fix
		v := e.buildViolation(plannedViolation)
		v.Incidents = incidentsToFix

		// Process violation using batch fixer
		fixResults, err := batchFixer.FixViolationBatch(ctx, v)

		if err != nil {
			// If entire batch failed, mark all incidents as failed
			for _, incident := range incidentsToFix {
				result.FailedFixes++
				e.state.RecordIncidentFailure(phase.ID, plannedViolation.ViolationID, incident.URI, err.Error())
			}
			continue
		}

		// Process individual fix results
		for i, fixResult := range fixResults {
			incident := incidentsToFix[i]
			incidentURI := incident.URI

			// Track confidence filtering stats
			if confidenceStats != nil {
				applied := fixResult.Success && !fixResult.SkippedLowConfidence
				confidenceStats.RecordFix(v.MigrationComplexity, applied)
			}

			if !fixResult.Success {
				result.FailedFixes++
				errorMsg := ""
				if fixResult.Error != nil {
					errorMsg = fixResult.Error.Error()
				}
				e.state.RecordIncidentFailure(phase.ID, plannedViolation.ViolationID, incidentURI, errorMsg)
				continue
			}

			// Record successful fix
			result.SuccessfulFixes++
			result.Cost += fixResult.Cost
			result.Tokens += fixResult.TokensUsed

			e.state.RecordIncidentFix(plannedViolation.ViolationID, incidentURI, fixResult.Cost)

			// Track for git commit if enabled
			if e.config.VerifiedTracker != nil && !e.config.DryRun {
				if err := e.config.VerifiedTracker.TrackFix(v, incident, &fixResult); err != nil {
					e.config.Progress.Error("Git commit/verification failed: %v", err)
				}
			} else if e.config.CommitTracker != nil && !e.config.DryRun {
				if err := e.config.CommitTracker.TrackFix(v, incident, &fixResult); err != nil {
					e.config.Progress.Error("Git commit failed: %v", err)
				}
			}

			// Track for PR if enabled
			if e.config.PRTracker != nil && !e.config.DryRun {
				if err := e.config.PRTracker.TrackForPR(v, incident, &fixResult); err != nil {
					e.config.Progress.Error("PR tracking failed: %v", err)
				}
			}
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

	// Store confidence stats in result
	result.ConfidenceStats = confidenceStats

	return result
}

// buildViolation constructs a violation.Violation from a planfile.PlannedViolation.
// This converts the plan's violation representation into the format expected by the fixer.
func (e *Executor) buildViolation(pv planfile.PlannedViolation) violation.Violation {
	return violation.Violation{
		ID:                  pv.ViolationID,
		Description:         pv.Description,
		Category:            pv.Category,
		Effort:              pv.Effort,
		MigrationComplexity: pv.MigrationComplexity,
		Incidents:           pv.Incidents,
		Rule: violation.Rule{
			ID:      pv.ViolationID,
			Message: pv.Description,
		},
	}
}
