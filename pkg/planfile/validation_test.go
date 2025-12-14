package planfile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestValidatePlan(t *testing.T) {
	t.Run("valid plan", func(t *testing.T) {
		plan := &Plan{
			Version: PlanVersion,
			Metadata: PlanMetadata{
				Provider:        "claude",
				TotalViolations: 1,
			},
			Phases: []Phase{
				{
					ID:          "phase-1",
					Name:        "Test Phase",
					Order:       1,
					Risk:        RiskMedium,
					Category:    "mandatory",
					EffortRange: [2]int{1, 3},
					Violations: []PlannedViolation{
						{
							ViolationID:   "v1",
							Description:   "Test",
							IncidentCount: 1,
							Incidents: []violation.Incident{
								{URI: "file:///test.java", LineNumber: 10},
							},
						},
					},
					EstimatedCost: 1.0,
				},
			},
		}

		err := ValidatePlan(plan)
		assert.NoError(t, err)
	})

	t.Run("nil plan", func(t *testing.T) {
		err := ValidatePlan(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plan is nil")
	})

	t.Run("missing version", func(t *testing.T) {
		plan := &Plan{
			Metadata: PlanMetadata{Provider: "claude"},
			Phases:   []Phase{{ID: "p1", Name: "Phase 1"}},
		}
		err := ValidatePlan(plan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("unsupported version", func(t *testing.T) {
		plan := &Plan{
			Version:  "2.0",
			Metadata: PlanMetadata{Provider: "claude"},
			Phases:   []Phase{{ID: "p1", Name: "Phase 1"}},
		}
		err := ValidatePlan(plan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported plan version")
	})

	t.Run("missing provider", func(t *testing.T) {
		plan := &Plan{
			Version: PlanVersion,
			Phases:  []Phase{{ID: "p1", Name: "Phase 1"}},
		}
		err := ValidatePlan(plan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider is required")
	})

	t.Run("no phases", func(t *testing.T) {
		plan := &Plan{
			Version:  PlanVersion,
			Metadata: PlanMetadata{Provider: "claude"},
			Phases:   []Phase{},
		}
		err := ValidatePlan(plan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one phase")
	})

	t.Run("duplicate phase IDs", func(t *testing.T) {
		plan := &Plan{
			Version:  PlanVersion,
			Metadata: PlanMetadata{Provider: "claude"},
			Phases: []Phase{
				{
					ID:       "phase-1",
					Name:     "Phase 1",
					Risk:     RiskLow,
					Category: "mandatory",
					Violations: []PlannedViolation{
						{ViolationID: "v1", Description: "Test", IncidentCount: 0},
					},
				},
				{
					ID:       "phase-1",
					Name:     "Phase 2",
					Risk:     RiskLow,
					Category: "mandatory",
					Violations: []PlannedViolation{
						{ViolationID: "v2", Description: "Test", IncidentCount: 0},
					},
				},
			},
		}
		err := ValidatePlan(plan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate phase ID")
	})
}

func TestValidatePhase(t *testing.T) {
	t.Run("missing phase ID", func(t *testing.T) {
		phase := &Phase{
			Name:     "Test",
			Risk:     RiskLow,
			Category: "mandatory",
		}
		err := validatePhase(phase, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "phase ID is required")
	})

	t.Run("missing phase name", func(t *testing.T) {
		phase := &Phase{
			ID:       "p1",
			Risk:     RiskLow,
			Category: "mandatory",
		}
		err := validatePhase(phase, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "phase name is required")
	})

	t.Run("negative order", func(t *testing.T) {
		phase := &Phase{
			ID:       "p1",
			Name:     "Test",
			Order:    -1,
			Risk:     RiskLow,
			Category: "mandatory",
		}
		err := validatePhase(phase, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "order must be non-negative")
	})

	t.Run("invalid risk level", func(t *testing.T) {
		phase := &Phase{
			ID:       "p1",
			Name:     "Test",
			Risk:     RiskLevel("invalid"),
			Category: "mandatory",
		}
		err := validatePhase(phase, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid risk level")
	})

	t.Run("missing category", func(t *testing.T) {
		phase := &Phase{
			ID:   "p1",
			Name: "Test",
			Risk: RiskLow,
		}
		err := validatePhase(phase, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "category is required")
	})

	t.Run("no violations", func(t *testing.T) {
		phase := &Phase{
			ID:         "p1",
			Name:       "Test",
			Risk:       RiskLow,
			Category:   "mandatory",
			Violations: []PlannedViolation{},
		}
		err := validatePhase(phase, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one violation")
	})

	t.Run("negative cost", func(t *testing.T) {
		phase := &Phase{
			ID:       "p1",
			Name:     "Test",
			Risk:     RiskLow,
			Category: "mandatory",
			Violations: []PlannedViolation{
				{ViolationID: "v1", Description: "Test", IncidentCount: 0},
			},
			EstimatedCost: -1.0,
		}
		err := validatePhase(phase, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cost must be non-negative")
	})
}

func TestValidatePlannedViolation(t *testing.T) {
	t.Run("missing violation ID", func(t *testing.T) {
		v := &PlannedViolation{
			Description:   "Test",
			IncidentCount: 0,
		}
		err := validatePlannedViolation(v, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "violation ID is required")
	})

	t.Run("missing description", func(t *testing.T) {
		v := &PlannedViolation{
			ViolationID:   "v1",
			IncidentCount: 0,
		}
		err := validatePlannedViolation(v, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "description is required")
	})

	t.Run("negative incident count", func(t *testing.T) {
		v := &PlannedViolation{
			ViolationID:   "v1",
			Description:   "Test",
			IncidentCount: -1,
		}
		err := validatePlannedViolation(v, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incident count must be non-negative")
	})

	t.Run("incident count mismatch", func(t *testing.T) {
		v := &PlannedViolation{
			ViolationID:   "v1",
			Description:   "Test",
			IncidentCount: 2,
			Incidents: []violation.Incident{
				{URI: "file:///test.java"},
			},
		}
		err := validatePlannedViolation(v, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incident count mismatch")
	})
}

func TestValidateState(t *testing.T) {
	now := time.Now()

	t.Run("valid state", func(t *testing.T) {
		state := &ExecutionState{
			Version:   StateVersion,
			PlanFile:  "plan.yaml",
			StartedAt: now,
			UpdatedAt: now,
			ExecutionSummary: ExecutionSummary{
				TotalPhases: 1,
			},
			Phases:     []PhaseStatus{},
			Violations: map[string]ViolationStatus{},
		}

		err := ValidateState(state)
		assert.NoError(t, err)
	})

	t.Run("nil state", func(t *testing.T) {
		err := ValidateState(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state is nil")
	})

	t.Run("missing version", func(t *testing.T) {
		state := &ExecutionState{
			PlanFile:  "plan.yaml",
			StartedAt: now,
			UpdatedAt: now,
		}
		err := ValidateState(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("unsupported version", func(t *testing.T) {
		state := &ExecutionState{
			Version:   "2.0",
			PlanFile:  "plan.yaml",
			StartedAt: now,
			UpdatedAt: now,
		}
		err := ValidateState(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported state version")
	})

	t.Run("missing plan file", func(t *testing.T) {
		state := &ExecutionState{
			Version:   StateVersion,
			StartedAt: now,
			UpdatedAt: now,
		}
		err := ValidateState(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plan file reference is required")
	})

	t.Run("zero started_at", func(t *testing.T) {
		state := &ExecutionState{
			Version:   StateVersion,
			PlanFile:  "plan.yaml",
			UpdatedAt: now,
		}
		err := ValidateState(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "started_at timestamp is required")
	})

	t.Run("zero updated_at", func(t *testing.T) {
		state := &ExecutionState{
			Version:   StateVersion,
			PlanFile:  "plan.yaml",
			StartedAt: now,
		}
		err := ValidateState(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "updated_at timestamp is required")
	})

	t.Run("updated_at before started_at", func(t *testing.T) {
		state := &ExecutionState{
			Version:   StateVersion,
			PlanFile:  "plan.yaml",
			StartedAt: now,
			UpdatedAt: now.Add(-1 * time.Hour),
		}
		err := ValidateState(state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "updated_at cannot be before started_at")
	})
}

func TestValidatePhaseStatus(t *testing.T) {
	now := time.Now()

	t.Run("valid phase status", func(t *testing.T) {
		status := &PhaseStatus{
			PhaseID: "p1",
			Status:  StatusPending,
		}
		err := validatePhaseStatus(status)
		assert.NoError(t, err)
	})

	t.Run("missing phase ID", func(t *testing.T) {
		status := &PhaseStatus{
			Status: StatusPending,
		}
		err := validatePhaseStatus(status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "phase ID is required")
	})

	t.Run("invalid status", func(t *testing.T) {
		status := &PhaseStatus{
			PhaseID: "p1",
			Status:  StatusType("invalid"),
		}
		err := validatePhaseStatus(status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("negative fixes applied", func(t *testing.T) {
		status := &PhaseStatus{
			PhaseID:      "p1",
			Status:       StatusCompleted,
			FixesApplied: -1,
		}
		err := validatePhaseStatus(status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fixes applied must be non-negative")
	})

	t.Run("negative cost", func(t *testing.T) {
		status := &PhaseStatus{
			PhaseID: "p1",
			Status:  StatusCompleted,
			Cost:    -1.0,
		}
		err := validatePhaseStatus(status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cost must be non-negative")
	})

	t.Run("completed without completed_at", func(t *testing.T) {
		status := &PhaseStatus{
			PhaseID: "p1",
			Status:  StatusCompleted,
		}
		err := validatePhaseStatus(status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "completed_at is required")
	})

	t.Run("completed_at before started_at", func(t *testing.T) {
		startedAt := now
		completedAt := now.Add(-1 * time.Hour)
		status := &PhaseStatus{
			PhaseID:     "p1",
			Status:      StatusCompleted,
			StartedAt:   &startedAt,
			CompletedAt: &completedAt,
		}
		err := validatePhaseStatus(status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "completed_at cannot be before started_at")
	})
}

func TestValidateViolationStatus(t *testing.T) {
	now := time.Now()

	t.Run("valid violation status", func(t *testing.T) {
		status := &ViolationStatus{
			Status: StatusCompleted,
			Incidents: map[string]IncidentStatus{
				"file:///test.java:10": {
					Status:    StatusCompleted,
					Cost:      0.5,
					Timestamp: now,
				},
			},
		}
		err := validateViolationStatus(status, "v1")
		assert.NoError(t, err)
	})

	t.Run("invalid violation status", func(t *testing.T) {
		status := &ViolationStatus{
			Status:    StatusType("invalid"),
			Incidents: map[string]IncidentStatus{},
		}
		err := validateViolationStatus(status, "v1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("invalid incident status", func(t *testing.T) {
		status := &ViolationStatus{
			Status: StatusInProgress,
			Incidents: map[string]IncidentStatus{
				"file:///test.java:10": {
					Status:    StatusType("invalid"),
					Timestamp: now,
				},
			},
		}
		err := validateViolationStatus(status, "v1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("negative incident cost", func(t *testing.T) {
		status := &ViolationStatus{
			Status: StatusCompleted,
			Incidents: map[string]IncidentStatus{
				"file:///test.java:10": {
					Status:    StatusCompleted,
					Cost:      -1.0,
					Timestamp: now,
				},
			},
		}
		err := validateViolationStatus(status, "v1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cost must be non-negative")
	})

	t.Run("zero timestamp", func(t *testing.T) {
		status := &ViolationStatus{
			Status: StatusCompleted,
			Incidents: map[string]IncidentStatus{
				"file:///test.java:10": {
					Status: StatusCompleted,
					Cost:   0.5,
				},
			},
		}
		err := validateViolationStatus(status, "v1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp is required")
	})
}
