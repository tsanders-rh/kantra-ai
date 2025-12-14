package planfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestNewPlan(t *testing.T) {
	plan := NewPlan("claude", 42)

	assert.Equal(t, PlanVersion, plan.Version)
	assert.Equal(t, "claude", plan.Metadata.Provider)
	assert.Equal(t, 42, plan.Metadata.TotalViolations)
	assert.NotNil(t, plan.Phases)
	assert.Len(t, plan.Phases, 0)
}

func TestSaveAndLoadPlan(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "test-plan.yaml")

	originalPlan := &Plan{
		Version: PlanVersion,
		Metadata: PlanMetadata{
			CreatedAt:       time.Now().UTC().Truncate(time.Second),
			Provider:        "claude",
			TotalViolations: 5,
		},
		Phases: []Phase{
			{
				ID:          "phase-1",
				Name:        "Critical Fixes",
				Order:       1,
				Risk:        RiskHigh,
				Category:    "mandatory",
				EffortRange: [2]int{5, 7},
				Explanation: "High effort mandatory fixes",
				Violations: []PlannedViolation{
					{
						ViolationID:   "v1",
						Description:   "Test violation",
						Category:      "mandatory",
						Effort:        6,
						IncidentCount: 2,
						Incidents: []violation.Incident{
							{URI: "file:///test.java", LineNumber: 10, Message: "Fix this"},
							{URI: "file:///test.java", LineNumber: 20, Message: "Fix that"},
						},
					},
				},
				EstimatedCost:            1.5,
				EstimatedDurationMinutes: 10,
				Deferred:                 false,
			},
		},
	}

	err := SavePlan(originalPlan, planPath)
	require.NoError(t, err)

	loadedPlan, err := LoadPlan(planPath)
	require.NoError(t, err)

	assert.Equal(t, originalPlan.Version, loadedPlan.Version)
	assert.Equal(t, originalPlan.Metadata.Provider, loadedPlan.Metadata.Provider)
	assert.Equal(t, originalPlan.Metadata.TotalViolations, loadedPlan.Metadata.TotalViolations)
	assert.Len(t, loadedPlan.Phases, 1)
	assert.Equal(t, "phase-1", loadedPlan.Phases[0].ID)
	assert.Equal(t, "Critical Fixes", loadedPlan.Phases[0].Name)
	assert.Equal(t, RiskHigh, loadedPlan.Phases[0].Risk)
	assert.Len(t, loadedPlan.Phases[0].Violations, 1)
	assert.Equal(t, "v1", loadedPlan.Phases[0].Violations[0].ViolationID)
	assert.Len(t, loadedPlan.Phases[0].Violations[0].Incidents, 2)
}

func TestLoadPlanNonexistent(t *testing.T) {
	_, err := LoadPlan("/nonexistent/plan.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read plan file")
}

func TestLoadPlanInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(planPath, []byte("invalid: yaml: [[["), 0644)
	require.NoError(t, err)

	_, err = LoadPlan(planPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse plan file")
}

func TestGetPhaseByID(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion,
		Metadata: PlanMetadata{
			Provider: "claude",
		},
		Phases: []Phase{
			{ID: "phase-1", Name: "Phase 1"},
			{ID: "phase-2", Name: "Phase 2"},
		},
	}

	t.Run("existing phase", func(t *testing.T) {
		phase, err := plan.GetPhaseByID("phase-1")
		require.NoError(t, err)
		assert.Equal(t, "phase-1", phase.ID)
		assert.Equal(t, "Phase 1", phase.Name)
	})

	t.Run("nonexistent phase", func(t *testing.T) {
		_, err := plan.GetPhaseByID("phase-99")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "phase not found")
	})
}

func TestGetActivePhases(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion,
		Metadata: PlanMetadata{
			Provider: "claude",
		},
		Phases: []Phase{
			{ID: "phase-1", Name: "Phase 1", Deferred: false},
			{ID: "phase-2", Name: "Phase 2", Deferred: true},
			{ID: "phase-3", Name: "Phase 3", Deferred: false},
		},
	}

	active := plan.GetActivePhases()
	assert.Len(t, active, 2)
	assert.Equal(t, "phase-1", active[0].ID)
	assert.Equal(t, "phase-3", active[1].ID)
}

func TestGetTotalIncidents(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion,
		Metadata: PlanMetadata{
			Provider: "claude",
		},
		Phases: []Phase{
			{
				ID:       "phase-1",
				Deferred: false,
				Violations: []PlannedViolation{
					{IncidentCount: 5},
					{IncidentCount: 3},
				},
			},
			{
				ID:       "phase-2",
				Deferred: true,
				Violations: []PlannedViolation{
					{IncidentCount: 10},
				},
			},
			{
				ID:       "phase-3",
				Deferred: false,
				Violations: []PlannedViolation{
					{IncidentCount: 2},
				},
			},
		},
	}

	total := plan.GetTotalIncidents()
	assert.Equal(t, 10, total)
}

func TestGetTotalCost(t *testing.T) {
	plan := &Plan{
		Version: PlanVersion,
		Metadata: PlanMetadata{
			Provider: "claude",
		},
		Phases: []Phase{
			{ID: "phase-1", EstimatedCost: 1.5, Deferred: false},
			{ID: "phase-2", EstimatedCost: 2.0, Deferred: true},
			{ID: "phase-3", EstimatedCost: 0.5, Deferred: false},
		},
	}

	total := plan.GetTotalCost()
	assert.Equal(t, 2.0, total)
}
