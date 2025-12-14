package planfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewState(t *testing.T) {
	state := NewState(".kantra-ai-plan.yaml", 3)

	assert.Equal(t, StateVersion, state.Version)
	assert.Equal(t, ".kantra-ai-plan.yaml", state.PlanFile)
	assert.Equal(t, 3, state.ExecutionSummary.TotalPhases)
	assert.Equal(t, 0, state.ExecutionSummary.CompletedPhases)
	assert.Equal(t, 3, state.ExecutionSummary.PendingPhases)
	assert.Equal(t, 0.0, state.ExecutionSummary.TotalCost)
	assert.NotNil(t, state.Phases)
	assert.NotNil(t, state.Violations)
}

func TestSaveAndLoadState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test-state.yaml")

	now := time.Now().UTC().Truncate(time.Second)
	originalState := &ExecutionState{
		Version:   StateVersion,
		PlanFile:  ".kantra-ai-plan.yaml",
		StartedAt: now,
		UpdatedAt: now,
		ExecutionSummary: ExecutionSummary{
			TotalPhases:     2,
			CompletedPhases: 1,
			PendingPhases:   1,
			TotalCost:       1.5,
		},
		Phases: []PhaseStatus{
			{
				PhaseID:      "phase-1",
				Status:       StatusCompleted,
				StartedAt:    &now,
				CompletedAt:  &now,
				FixesApplied: 5,
				Cost:         1.5,
			},
		},
		Violations: map[string]ViolationStatus{
			"v1": {
				Status: StatusCompleted,
				Incidents: map[string]IncidentStatus{
					"file:///test.java:10": {
						Status:    StatusCompleted,
						Cost:      0.5,
						Timestamp: now,
					},
				},
			},
		},
	}

	err := SaveState(originalState, statePath)
	require.NoError(t, err)

	loadedState, err := LoadState(statePath)
	require.NoError(t, err)

	assert.Equal(t, originalState.Version, loadedState.Version)
	assert.Equal(t, originalState.PlanFile, loadedState.PlanFile)
	assert.Equal(t, originalState.ExecutionSummary.TotalPhases, loadedState.ExecutionSummary.TotalPhases)
	assert.Len(t, loadedState.Phases, 1)
	assert.Equal(t, "phase-1", loadedState.Phases[0].PhaseID)
	assert.Equal(t, StatusCompleted, loadedState.Phases[0].Status)
	assert.Contains(t, loadedState.Violations, "v1")
}

func TestLoadStateNonexistent(t *testing.T) {
	state, err := LoadState("/nonexistent/state.yaml")
	assert.NoError(t, err)
	assert.Nil(t, state)
}

func TestLoadStateInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(statePath, []byte("invalid: yaml: [[["), 0644)
	require.NoError(t, err)

	_, err = LoadState(statePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse state file")
}

func TestGetPhaseStatus(t *testing.T) {
	state := &ExecutionState{
		Version: StateVersion,
		Phases: []PhaseStatus{
			{PhaseID: "phase-1", Status: StatusCompleted},
			{PhaseID: "phase-2", Status: StatusPending},
		},
	}

	t.Run("existing phase", func(t *testing.T) {
		status := state.GetPhaseStatus("phase-1")
		require.NotNil(t, status)
		assert.Equal(t, "phase-1", status.PhaseID)
		assert.Equal(t, StatusCompleted, status.Status)
	})

	t.Run("nonexistent phase", func(t *testing.T) {
		status := state.GetPhaseStatus("phase-99")
		assert.Nil(t, status)
	})
}

func TestUpdatePhaseStatus(t *testing.T) {
	state := NewState(".kantra-ai-plan.yaml", 2)

	t.Run("add new phase status", func(t *testing.T) {
		now := time.Now()
		state.UpdatePhaseStatus(PhaseStatus{
			PhaseID:      "phase-1",
			Status:       StatusCompleted,
			StartedAt:    &now,
			CompletedAt:  &now,
			FixesApplied: 5,
			Cost:         1.0,
		})

		assert.Len(t, state.Phases, 1)
		status := state.GetPhaseStatus("phase-1")
		require.NotNil(t, status)
		assert.Equal(t, StatusCompleted, status.Status)
		assert.Equal(t, 5, status.FixesApplied)
	})

	t.Run("update existing phase status", func(t *testing.T) {
		state.UpdatePhaseStatus(PhaseStatus{
			PhaseID:      "phase-1",
			Status:       StatusCompleted,
			FixesApplied: 10,
			Cost:         2.0,
		})

		assert.Len(t, state.Phases, 1)
		status := state.GetPhaseStatus("phase-1")
		require.NotNil(t, status)
		assert.Equal(t, 10, status.FixesApplied)
		assert.Equal(t, 2.0, status.Cost)
	})
}

func TestMarkPhaseLifecycle(t *testing.T) {
	state := NewState(".kantra-ai-plan.yaml", 1)

	t.Run("mark phase started", func(t *testing.T) {
		state.MarkPhaseStarted("phase-1")

		status := state.GetPhaseStatus("phase-1")
		require.NotNil(t, status)
		assert.Equal(t, StatusInProgress, status.Status)
		assert.NotNil(t, status.StartedAt)
	})

	t.Run("mark phase completed", func(t *testing.T) {
		state.MarkPhaseCompleted("phase-1")

		status := state.GetPhaseStatus("phase-1")
		require.NotNil(t, status)
		assert.Equal(t, StatusCompleted, status.Status)
		assert.NotNil(t, status.CompletedAt)
	})

	t.Run("mark phase failed", func(t *testing.T) {
		state.MarkPhaseStarted("phase-2")
		state.MarkPhaseFailed("phase-2")

		status := state.GetPhaseStatus("phase-2")
		require.NotNil(t, status)
		assert.Equal(t, StatusFailed, status.Status)
	})
}

func TestRecordIncidentFix(t *testing.T) {
	state := NewState(".kantra-ai-plan.yaml", 1)

	state.RecordIncidentFix("v1", "file:///test.java:10", 0.5)

	assert.Contains(t, state.Violations, "v1")
	violation := state.Violations["v1"]
	assert.Equal(t, StatusCompleted, violation.Status)
	assert.Contains(t, violation.Incidents, "file:///test.java:10")

	incident := violation.Incidents["file:///test.java:10"]
	assert.Equal(t, StatusCompleted, incident.Status)
	assert.Equal(t, 0.5, incident.Cost)
	assert.Equal(t, 0.5, state.ExecutionSummary.TotalCost)
}

func TestRecordIncidentFixAllCompleted(t *testing.T) {
	state := NewState(".kantra-ai-plan.yaml", 1)

	state.RecordIncidentFix("v1", "file:///test.java:10", 0.5)
	state.RecordIncidentFix("v1", "file:///test.java:20", 0.3)

	violation := state.Violations["v1"]
	assert.Equal(t, StatusCompleted, violation.Status)
	assert.Equal(t, 0.8, state.ExecutionSummary.TotalCost)
}

func TestRecordIncidentFailure(t *testing.T) {
	state := NewState(".kantra-ai-plan.yaml", 1)

	state.RecordIncidentFailure("phase-1", "v1", "file:///test.java:10", "AI timeout")

	assert.Contains(t, state.Violations, "v1")
	violation := state.Violations["v1"]
	assert.Equal(t, StatusFailed, violation.Status)

	incident := violation.Incidents["file:///test.java:10"]
	assert.Equal(t, StatusFailed, incident.Status)

	require.NotNil(t, state.LastFailure)
	assert.Equal(t, "phase-1", state.LastFailure.PhaseID)
	assert.Equal(t, "v1", state.LastFailure.ViolationID)
	assert.Equal(t, "file:///test.java:10", state.LastFailure.IncidentURI)
	assert.Equal(t, "AI timeout", state.LastFailure.Error)
}

func TestHasFailures(t *testing.T) {
	state := NewState(".kantra-ai-plan.yaml", 1)

	assert.False(t, state.HasFailures())

	state.RecordIncidentFailure("phase-1", "v1", "file:///test.java:10", "error")

	assert.True(t, state.HasFailures())
}
