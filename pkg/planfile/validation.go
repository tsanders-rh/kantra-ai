package planfile

import (
	"fmt"
)

// ValidatePlan validates a migration plan structure for correctness.
// Checks version, provider, phase structure, and ensures no duplicate phase IDs.
func ValidatePlan(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}

	if plan.Version == "" {
		return fmt.Errorf("plan version is required")
	}

	if plan.Version != PlanVersion {
		return fmt.Errorf("unsupported plan version: %s (expected %s)", plan.Version, PlanVersion)
	}

	if plan.Metadata.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	if len(plan.Phases) == 0 {
		return fmt.Errorf("plan must have at least one phase")
	}

	phaseIDs := make(map[string]bool)
	for i, phase := range plan.Phases {
		if err := validatePhase(&phase, i); err != nil {
			return fmt.Errorf("phase %d: %w", i, err)
		}

		if phaseIDs[phase.ID] {
			return fmt.Errorf("duplicate phase ID: %s", phase.ID)
		}
		phaseIDs[phase.ID] = true
	}

	return nil
}

// validatePhase validates a single phase
func validatePhase(phase *Phase, index int) error {
	if phase.ID == "" {
		return fmt.Errorf("phase ID is required")
	}

	if phase.Name == "" {
		return fmt.Errorf("phase name is required")
	}

	if phase.Order < 0 {
		return fmt.Errorf("phase order must be non-negative")
	}

	if !isValidRiskLevel(phase.Risk) {
		return fmt.Errorf("invalid risk level: %s (must be low, medium, or high)", phase.Risk)
	}

	if phase.Category == "" {
		return fmt.Errorf("phase category is required")
	}

	if len(phase.Violations) == 0 {
		return fmt.Errorf("phase must have at least one violation")
	}

	for i, violation := range phase.Violations {
		if err := validatePlannedViolation(&violation, i); err != nil {
			return fmt.Errorf("violation %d: %w", i, err)
		}
	}

	if phase.EstimatedCost < 0 {
		return fmt.Errorf("estimated cost must be non-negative")
	}

	return nil
}

// validatePlannedViolation validates a planned violation
func validatePlannedViolation(violation *PlannedViolation, index int) error {
	if violation.ViolationID == "" {
		return fmt.Errorf("violation ID is required")
	}

	if violation.Description == "" {
		return fmt.Errorf("description is required")
	}

	if violation.IncidentCount < 0 {
		return fmt.Errorf("incident count must be non-negative")
	}

	if len(violation.Incidents) > 0 && violation.IncidentCount != len(violation.Incidents) {
		return fmt.Errorf("incident count mismatch: count=%d, incidents=%d",
			violation.IncidentCount, len(violation.Incidents))
	}

	return nil
}

// isValidRiskLevel checks if a risk level is valid
func isValidRiskLevel(risk RiskLevel) bool {
	switch risk {
	case RiskLow, RiskMedium, RiskHigh:
		return true
	default:
		return false
	}
}

// ValidateState validates an execution state structure for correctness.
// Checks version, timestamps, phase statuses, and violation tracking consistency.
func ValidateState(state *ExecutionState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	if state.Version == "" {
		return fmt.Errorf("state version is required")
	}

	if state.Version != StateVersion {
		return fmt.Errorf("unsupported state version: %s (expected %s)", state.Version, StateVersion)
	}

	if state.PlanFile == "" {
		return fmt.Errorf("plan file reference is required")
	}

	if state.StartedAt.IsZero() {
		return fmt.Errorf("started_at timestamp is required")
	}

	if state.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at timestamp is required")
	}

	if state.UpdatedAt.Before(state.StartedAt) {
		return fmt.Errorf("updated_at cannot be before started_at")
	}

	if err := validateExecutionSummary(&state.ExecutionSummary); err != nil {
		return fmt.Errorf("execution summary: %w", err)
	}

	for i, phase := range state.Phases {
		if err := validatePhaseStatus(&phase); err != nil {
			return fmt.Errorf("phase status %d: %w", i, err)
		}
	}

	for violationID, violation := range state.Violations {
		if err := validateViolationStatus(&violation, violationID); err != nil {
			return fmt.Errorf("violation %s: %w", violationID, err)
		}
	}

	return nil
}

// validateExecutionSummary validates an execution summary
func validateExecutionSummary(summary *ExecutionSummary) error {
	if summary.TotalPhases < 0 {
		return fmt.Errorf("total phases must be non-negative")
	}

	if summary.CompletedPhases < 0 {
		return fmt.Errorf("completed phases must be non-negative")
	}

	if summary.PendingPhases < 0 {
		return fmt.Errorf("pending phases must be non-negative")
	}

	if summary.TotalCost < 0 {
		return fmt.Errorf("total cost must be non-negative")
	}

	return nil
}

// validatePhaseStatus validates a phase status
func validatePhaseStatus(status *PhaseStatus) error {
	if status.PhaseID == "" {
		return fmt.Errorf("phase ID is required")
	}

	if !isValidStatusType(status.Status) {
		return fmt.Errorf("invalid status: %s", status.Status)
	}

	if status.FixesApplied < 0 {
		return fmt.Errorf("fixes applied must be non-negative")
	}

	if status.Cost < 0 {
		return fmt.Errorf("cost must be non-negative")
	}

	if status.Status == StatusCompleted && status.CompletedAt == nil {
		return fmt.Errorf("completed_at is required for completed phase")
	}

	if status.CompletedAt != nil && status.StartedAt != nil {
		if status.CompletedAt.Before(*status.StartedAt) {
			return fmt.Errorf("completed_at cannot be before started_at")
		}
	}

	return nil
}

// validateViolationStatus validates a violation status
func validateViolationStatus(status *ViolationStatus, violationID string) error {
	if !isValidStatusType(status.Status) {
		return fmt.Errorf("invalid status: %s", status.Status)
	}

	for uri, incident := range status.Incidents {
		if !isValidStatusType(incident.Status) {
			return fmt.Errorf("incident %s: invalid status: %s", uri, incident.Status)
		}

		if incident.Cost < 0 {
			return fmt.Errorf("incident %s: cost must be non-negative", uri)
		}

		if incident.Timestamp.IsZero() {
			return fmt.Errorf("incident %s: timestamp is required", uri)
		}
	}

	return nil
}

// isValidStatusType checks if a status type is valid
func isValidStatusType(status StatusType) bool {
	switch status {
	case StatusPending, StatusInProgress, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}
