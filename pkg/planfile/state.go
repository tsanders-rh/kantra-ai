package planfile

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const StateVersion = "1.0"

// LoadState reads execution state from a YAML file
func LoadState(path string) (*ExecutionState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ExecutionState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if err := ValidateState(&state); err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	return &state, nil
}

// SaveState writes execution state to a YAML file
func SaveState(state *ExecutionState, path string) error {
	if err := ValidateState(state); err != nil {
		return fmt.Errorf("invalid state: %w", err)
	}

	state.UpdatedAt = time.Now()

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// NewState creates a new execution state
func NewState(planFile string, totalPhases int) *ExecutionState {
	now := time.Now()
	return &ExecutionState{
		Version:   StateVersion,
		PlanFile:  planFile,
		StartedAt: now,
		UpdatedAt: now,
		ExecutionSummary: ExecutionSummary{
			TotalPhases:     totalPhases,
			CompletedPhases: 0,
			PendingPhases:   totalPhases,
			TotalCost:       0.0,
		},
		Phases:     make([]PhaseStatus, 0),
		Violations: make(map[string]ViolationStatus),
	}
}

// GetPhaseStatus returns the status for a phase, or nil if not found
func (s *ExecutionState) GetPhaseStatus(phaseID string) *PhaseStatus {
	for i := range s.Phases {
		if s.Phases[i].PhaseID == phaseID {
			return &s.Phases[i]
		}
	}
	return nil
}

// UpdatePhaseStatus updates or adds a phase status
func (s *ExecutionState) UpdatePhaseStatus(status PhaseStatus) {
	for i := range s.Phases {
		if s.Phases[i].PhaseID == status.PhaseID {
			s.Phases[i] = status
			s.updateSummary()
			return
		}
	}
	s.Phases = append(s.Phases, status)
	s.updateSummary()
}

// MarkPhaseStarted marks a phase as in progress
func (s *ExecutionState) MarkPhaseStarted(phaseID string) {
	now := time.Now()
	status := s.GetPhaseStatus(phaseID)
	if status == nil {
		s.UpdatePhaseStatus(PhaseStatus{
			PhaseID:   phaseID,
			Status:    StatusInProgress,
			StartedAt: &now,
		})
	} else {
		status.Status = StatusInProgress
		if status.StartedAt == nil {
			status.StartedAt = &now
		}
		s.UpdatePhaseStatus(*status)
	}
}

// MarkPhaseCompleted marks a phase as completed
func (s *ExecutionState) MarkPhaseCompleted(phaseID string) {
	now := time.Now()
	status := s.GetPhaseStatus(phaseID)
	if status != nil {
		status.Status = StatusCompleted
		status.CompletedAt = &now
		s.UpdatePhaseStatus(*status)
	}
}

// MarkPhaseFailed marks a phase as failed
func (s *ExecutionState) MarkPhaseFailed(phaseID string) {
	status := s.GetPhaseStatus(phaseID)
	if status != nil {
		status.Status = StatusFailed
		s.UpdatePhaseStatus(*status)
	}
}

// RecordIncidentFix records a successful fix for an incident
func (s *ExecutionState) RecordIncidentFix(violationID, incidentURI string, cost float64) {
	if s.Violations == nil {
		s.Violations = make(map[string]ViolationStatus)
	}

	violationStatus, exists := s.Violations[violationID]
	if !exists {
		violationStatus = ViolationStatus{
			Status:    StatusInProgress,
			Incidents: make(map[string]IncidentStatus),
		}
	}

	violationStatus.Incidents[incidentURI] = IncidentStatus{
		Status:    StatusCompleted,
		Cost:      cost,
		Timestamp: time.Now(),
	}

	allCompleted := true
	for _, incident := range violationStatus.Incidents {
		if incident.Status != StatusCompleted {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		violationStatus.Status = StatusCompleted
	}

	s.Violations[violationID] = violationStatus
	s.ExecutionSummary.TotalCost += cost
}

// RecordIncidentFailure records a failed fix attempt
func (s *ExecutionState) RecordIncidentFailure(phaseID, violationID, incidentURI, errorMsg string) {
	if s.Violations == nil {
		s.Violations = make(map[string]ViolationStatus)
	}

	violationStatus, exists := s.Violations[violationID]
	if !exists {
		violationStatus = ViolationStatus{
			Status:    StatusFailed,
			Incidents: make(map[string]IncidentStatus),
		}
	}

	violationStatus.Incidents[incidentURI] = IncidentStatus{
		Status:    StatusFailed,
		Timestamp: time.Now(),
	}
	violationStatus.Status = StatusFailed

	s.Violations[violationID] = violationStatus

	s.LastFailure = &FailureInfo{
		PhaseID:     phaseID,
		ViolationID: violationID,
		IncidentURI: incidentURI,
		Error:       errorMsg,
	}
}

// HasFailures returns true if there are any failed incidents
func (s *ExecutionState) HasFailures() bool {
	return s.LastFailure != nil
}

// updateSummary recalculates the execution summary
func (s *ExecutionState) updateSummary() {
	completed := 0
	pending := 0

	for _, phase := range s.Phases {
		switch phase.Status {
		case StatusCompleted:
			completed++
		case StatusPending:
			pending++
		}
	}

	s.ExecutionSummary.CompletedPhases = completed
	s.ExecutionSummary.PendingPhases = pending
}
