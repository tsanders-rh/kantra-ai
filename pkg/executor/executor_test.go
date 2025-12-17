package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/ux"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// MockProvider is a mock implementation of provider.Provider
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProvider) FixViolation(ctx context.Context, req provider.FixRequest) (*provider.FixResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.FixResponse), args.Error(1)
}

func (m *MockProvider) EstimateCost(req provider.FixRequest) (float64, error) {
	args := m.Called(req)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockProvider) GeneratePlan(ctx context.Context, req provider.PlanRequest) (*provider.PlanResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.PlanResponse), args.Error(1)
}

func TestNew(t *testing.T) {
	mockProvider := new(MockProvider)

	config := Config{
		Provider:  mockProvider,
		InputPath: "/tmp/test",
	}

	exec, err := New(config)

	assert.NoError(t, err)
	assert.NotNil(t, exec)
	assert.Equal(t, ".kantra-ai-plan.yaml", exec.config.PlanPath)
	assert.Equal(t, ".kantra-ai-state.yaml", exec.config.StatePath)
	assert.NotNil(t, exec.config.Progress)
}

func TestNew_WithCustomPaths(t *testing.T) {
	mockProvider := new(MockProvider)

	config := Config{
		Provider:  mockProvider,
		InputPath: "/tmp/test",
		PlanPath:  "custom-plan.yaml",
		StatePath: "custom-state.yaml",
	}

	exec, err := New(config)

	assert.NoError(t, err)
	assert.Equal(t, "custom-plan.yaml", exec.config.PlanPath)
	assert.Equal(t, "custom-state.yaml", exec.config.StatePath)
}

func TestExecute_BasicFlow(t *testing.T) {
	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test source file
	testFile := filepath.Join(tmpDir, "test.java")
	err = os.WriteFile(testFile, []byte("public class Test {}"), 0644)
	assert.NoError(t, err)

	planPath := filepath.Join(tmpDir, "plan.yaml")
	statePath := filepath.Join(tmpDir, "state.yaml")

	// Create test plan
	plan := createTestPlan()
	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	// Create mock provider
	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()

	// Mock successful batch fix for both incidents
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file:///test.java:10",
					Success:      true,
					FixedContent: "public class TestFixed {}",
					Explanation:  "Fixed incident 1",
					Confidence:   0.9,
				},
				{
					IncidentURI:  "file:///test.java:20",
					Success:      true,
					FixedContent: "public class TestFixed {}",
					Explanation:  "Fixed incident 2",
					Confidence:   0.9,
				},
			},
			Success:    true,
			TokensUsed: 200,
			Cost:       0.10,
		},
		nil,
	)

	// Create executor
	config := Config{
		PlanPath:  planPath,
		StatePath: statePath,
		InputPath: tmpDir,
		Provider:  mockProvider,
		Progress:  &ux.NoOpProgressWriter{},
		DryRun:    true, // Dry run to avoid actual file writes
	}

	exec, err := New(config)
	assert.NoError(t, err)

	// Execute
	ctx := context.Background()
	result, err := exec.Execute(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalPhases)
	assert.Equal(t, 1, result.ExecutedPhases)
	assert.Equal(t, 1, result.CompletedPhases)
	assert.Equal(t, 2, result.SuccessfulFixes)
	assert.Equal(t, 0, result.FailedFixes)
	assert.Equal(t, 0.10, result.TotalCost)
	assert.Equal(t, 200, result.TotalTokens)

	// Verify state file was created
	state, err := planfile.LoadState(statePath)
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, 1, state.ExecutionSummary.CompletedPhases)

	mockProvider.AssertExpectations(t)
}

func TestExecute_WithDeferredPhase(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	planPath := filepath.Join(tmpDir, "plan.yaml")
	statePath := filepath.Join(tmpDir, "state.yaml")

	// Create plan with deferred phase
	plan := createTestPlan()
	plan.Phases[0].Deferred = true
	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()

	config := Config{
		PlanPath:  planPath,
		StatePath: statePath,
		InputPath: tmpDir,
		Provider:  mockProvider,
		Progress:  &ux.NoOpProgressWriter{},
	}

	exec, err := New(config)
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = exec.Execute(ctx)

	// Should fail because no phases to execute
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no phases to execute")

	mockProvider.AssertNotCalled(t, "FixBatch")
}

func TestExecute_SpecificPhase(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test source files
	err = os.WriteFile(filepath.Join(tmpDir, "test1.java"), []byte("class Test1 {}"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "test2.java"), []byte("class Test2 {}"), 0644)
	assert.NoError(t, err)

	planPath := filepath.Join(tmpDir, "plan.yaml")
	statePath := filepath.Join(tmpDir, "state.yaml")

	// Create plan with 2 phases
	plan := createTestPlanMultiPhase()
	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file:///test2.java:20",
					Success:      true,
					FixedContent: "class Test2Fixed {}",
					Explanation:  "Fixed",
					Confidence:   0.9,
				},
			},
			Success:    true,
			TokensUsed: 50,
			Cost:       0.03,
		},
		nil,
	).Once()

	config := Config{
		PlanPath:  planPath,
		StatePath: statePath,
		InputPath: tmpDir,
		Provider:  mockProvider,
		PhaseID:   "phase-2",
		Progress:  &ux.NoOpProgressWriter{},
		DryRun:    true,
	}

	exec, err := New(config)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := exec.Execute(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 2, result.TotalPhases)
	assert.Equal(t, 1, result.ExecutedPhases) // Only phase-2
	assert.Equal(t, 1, result.SuccessfulFixes)

	mockProvider.AssertExpectations(t)
}

func TestExecute_WithFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test source file
	err = os.WriteFile(filepath.Join(tmpDir, "test.java"), []byte("class Test {}"), 0644)
	assert.NoError(t, err)

	planPath := filepath.Join(tmpDir, "plan.yaml")
	statePath := filepath.Join(tmpDir, "state.yaml")

	plan := createTestPlan()
	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()

	// Batch with one successful fix and one failed fix
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file:///test.java:10",
					Success:      true,
					FixedContent: "class TestFixed {}",
					Explanation:  "Fixed",
					Confidence:   0.9,
				},
				{
					IncidentURI:  "file:///test.java:20",
					Success:      false,
					Error:        assert.AnError,
				},
			},
			Success:    false, // Overall success is false if any fix failed
			TokensUsed: 100,
			Cost:       0.05,
		},
		nil,
	).Once()

	config := Config{
		PlanPath:  planPath,
		StatePath: statePath,
		InputPath: tmpDir,
		Provider:  mockProvider,
		Progress:  &ux.NoOpProgressWriter{},
		DryRun:    true,
	}

	exec, err := New(config)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := exec.Execute(ctx)

	assert.NoError(t, err) // Execution continues despite failures
	assert.Equal(t, 1, result.SuccessfulFixes)
	assert.Equal(t, 1, result.FailedFixes)

	// Check state tracks the failure
	state, err := planfile.LoadState(statePath)
	assert.NoError(t, err)
	assert.NotNil(t, state.LastFailure)

	mockProvider.AssertExpectations(t)
}

func TestExecute_Resume(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test source file
	err = os.WriteFile(filepath.Join(tmpDir, "test.java"), []byte("class Test {}"), 0644)
	assert.NoError(t, err)

	planPath := filepath.Join(tmpDir, "plan.yaml")
	statePath := filepath.Join(tmpDir, "state.yaml")

	plan := createTestPlan()
	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	// Create state with partial completion
	state := planfile.NewState(planPath, 1)
	now := time.Now()
	state.StartedAt = now
	state.UpdatedAt = now.Add(time.Second) // UpdatedAt must be after StartedAt

	// Mark first incident as completed
	state.RecordIncidentFix("test-violation-1", "file:///test.java:10", 0.05)

	// Mark second incident as failed
	state.RecordIncidentFailure("phase-1", "test-violation-1", "file:///test.java:20", "AI timeout")

	err = planfile.SaveState(state, statePath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()

	// Should only call FixBatch once with just the failed incident
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file:///test.java:20",
					Success:      true,
					FixedContent: "class TestFixed {}",
					Explanation:  "Fixed on retry",
					Confidence:   0.9,
				},
			},
			Success:    true,
			TokensUsed: 100,
			Cost:       0.05,
		},
		nil,
	).Once()

	config := Config{
		PlanPath:  planPath,
		StatePath: statePath,
		InputPath: tmpDir,
		Provider:  mockProvider,
		Progress:  &ux.NoOpProgressWriter{},
		Resume:    true,
		DryRun:    true,
	}

	exec, err := New(config)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := exec.Execute(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.SuccessfulFixes) // Only retried the failed one

	mockProvider.AssertExpectations(t)
}

func TestGetPhasesToExecute(t *testing.T) {
	plan := createTestPlanMultiPhase()

	tests := []struct {
		name           string
		phaseID        string
		deferred       bool
		expectedCount  int
		expectedPhases []string
	}{
		{
			name:           "all phases",
			phaseID:        "",
			deferred:       false,
			expectedCount:  2,
			expectedPhases: []string{"phase-1", "phase-2"},
		},
		{
			name:           "specific phase",
			phaseID:        "phase-1",
			deferred:       false,
			expectedCount:  1,
			expectedPhases: []string{"phase-1"},
		},
		{
			name:           "skip deferred",
			phaseID:        "",
			deferred:       true,
			expectedCount:  1,
			expectedPhases: []string{"phase-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.deferred {
				plan.Phases[0].Deferred = true
			} else {
				plan.Phases[0].Deferred = false
			}

			exec := &Executor{
				plan: plan,
				state: planfile.NewState("test.yaml", len(plan.Phases)),
				config: Config{
					PhaseID: tt.phaseID,
				},
			}

			phases := exec.getPhasesToExecute()

			assert.Len(t, phases, tt.expectedCount)
			for i, expectedID := range tt.expectedPhases {
				assert.Equal(t, expectedID, phases[i].ID)
			}
		})
	}
}

func TestBuildViolation(t *testing.T) {
	exec := &Executor{}

	plannedViolation := planfile.PlannedViolation{
		ViolationID:   "test-id",
		Description:   "test description",
		Category:      "mandatory",
		Effort:        5,
		IncidentCount: 2,
		Incidents: []violation.Incident{
			{
				URI:        "file:///test.java",
				LineNumber: 10,
				Message:    "test message",
			},
		},
	}

	result := exec.buildViolation(plannedViolation)

	assert.Equal(t, "test-id", result.ID)
	assert.Equal(t, "test description", result.Description)
	assert.Equal(t, "mandatory", result.Category)
	assert.Equal(t, 5, result.Effort)
	assert.Len(t, result.Incidents, 1)
	assert.Equal(t, "test-id", result.Rule.ID)
	assert.Equal(t, "test description", result.Rule.Message)
}

// Helper functions

func createTestPlan() *planfile.Plan {
	plan := planfile.NewPlan("test-provider", 1)
	plan.Metadata.CreatedAt = time.Now()

	plan.Phases = []planfile.Phase{
		{
			ID:           "phase-1",
			Name:         "Test Phase",
			Order:        1,
			Risk:         planfile.RiskLow,
			Category:     "mandatory",
			EffortRange:  [2]int{1, 3},
			Explanation:  "Test explanation",
			EstimatedCost: 0.10,
			Deferred:     false,
			Violations: []planfile.PlannedViolation{
				{
					ViolationID:   "test-violation-1",
					Description:   "Test violation",
					Category:      "mandatory",
					Effort:        3,
					IncidentCount: 2,
					Incidents: []violation.Incident{
						{
							URI:        "file:///test.java",
							LineNumber: 10,
							Message:    "Test incident 1",
						},
						{
							URI:        "file:///test.java",
							LineNumber: 20,
							Message:    "Test incident 2",
						},
					},
				},
			},
		},
	}

	return plan
}

func createTestPlanMultiPhase() *planfile.Plan {
	plan := planfile.NewPlan("test-provider", 2)
	plan.Metadata.CreatedAt = time.Now()

	plan.Phases = []planfile.Phase{
		{
			ID:           "phase-1",
			Name:         "Phase 1",
			Order:        1,
			Risk:         planfile.RiskLow,
			Category:     "mandatory",
			EffortRange:  [2]int{1, 3},
			EstimatedCost: 0.05,
			Violations: []planfile.PlannedViolation{
				{
					ViolationID:   "violation-1",
					Description:   "Violation 1",
					Category:      "mandatory",
					Effort:        2,
					IncidentCount: 1,
					Incidents: []violation.Incident{
						{URI: "file:///test1.java", LineNumber: 10},
					},
				},
			},
		},
		{
			ID:           "phase-2",
			Name:         "Phase 2",
			Order:        2,
			Risk:         planfile.RiskMedium,
			Category:     "optional",
			EffortRange:  [2]int{4, 7},
			EstimatedCost: 0.08,
			Violations: []planfile.PlannedViolation{
				{
					ViolationID:   "violation-2",
					Description:   "Violation 2",
					Category:      "optional",
					Effort:        5,
					IncidentCount: 1,
					Incidents: []violation.Incident{
						{URI: "file:///test2.java", LineNumber: 20},
					},
				},
			},
		},
	}

	return plan
}

func (m *MockProvider) FixBatch(ctx context.Context, req provider.BatchRequest) (*provider.BatchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.BatchResponse), args.Error(1)
}

func TestExecute_DeduplicationDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test source file
	err = os.WriteFile(filepath.Join(tmpDir, "test.java"), []byte("class Test {}"), 0644)
	assert.NoError(t, err)

	planPath := filepath.Join(tmpDir, "plan.yaml")
	statePath := filepath.Join(tmpDir, "state.yaml")

	// Create plan with duplicate incidents (same file, line, and violation)
	plan := planfile.NewPlan("test-provider", 1)
	plan.Metadata.CreatedAt = time.Now()
	plan.Phases = []planfile.Phase{
		{
			ID:            "phase-1",
			Name:          "Test Phase",
			Order:         1,
			Risk:          planfile.RiskLow,
			Category:      "mandatory",
			EffortRange:   [2]int{1, 3},
			Explanation:   "Test explanation",
			EstimatedCost: 0.10,
			Violations: []planfile.PlannedViolation{
				{
					ViolationID:   "test-violation-1",
					Description:   "Test violation",
					Category:      "mandatory",
					Effort:        3,
					IncidentCount: 4,
					Incidents: []violation.Incident{
						// First unique incident
						{URI: "file:///test.java", LineNumber: 10, Message: "Test 1"},
						// Duplicate of first (same file, line, violation)
						{URI: "file:///test.java", LineNumber: 10, Message: "Test 1 duplicate"},
						// Second unique incident
						{URI: "file:///test.java", LineNumber: 20, Message: "Test 2"},
						// Another duplicate of first
						{URI: "file:///test.java", LineNumber: 10, Message: "Test 1 duplicate again"},
					},
				},
			},
		},
	}

	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()

	// Mock should only be called with 2 unique incidents (not 4)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		// Verify only 2 incidents sent (duplicates filtered out)
		return len(req.Incidents) == 2
	})).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file:///test.java:10",
					Success:      true,
					FixedContent: "class TestFixed {}",
					Explanation:  "Fixed incident 1",
					Confidence:   0.9,
				},
				{
					IncidentURI:  "file:///test.java:20",
					Success:      true,
					FixedContent: "class TestFixed {}",
					Explanation:  "Fixed incident 2",
					Confidence:   0.9,
				},
			},
			Success:    true,
			TokensUsed: 100,
			Cost:       0.05,
		},
		nil,
	).Once()

	config := Config{
		PlanPath:  planPath,
		StatePath: statePath,
		InputPath: tmpDir,
		Provider:  mockProvider,
		Progress:  &ux.NoOpProgressWriter{},
		DryRun:    true,
	}

	exec, err := New(config)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := exec.Execute(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalPhases)
	assert.Equal(t, 1, result.ExecutedPhases)
	assert.Equal(t, 1, result.CompletedPhases)
	assert.Equal(t, 2, result.SuccessfulFixes)  // Only 2 unique incidents fixed
	assert.Equal(t, 0, result.FailedFixes)
	assert.Equal(t, 0, result.SkippedFixes)
	assert.Equal(t, 2, result.DuplicateFixes)   // 2 duplicates skipped
	assert.Equal(t, 0.05, result.TotalCost)
	assert.Equal(t, 100, result.TotalTokens)

	mockProvider.AssertExpectations(t)
}

func TestExecute_DeduplicationAcrossViolations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test source files
	err = os.WriteFile(filepath.Join(tmpDir, "test1.java"), []byte("class Test1 {}"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "test2.java"), []byte("class Test2 {}"), 0644)
	assert.NoError(t, err)

	planPath := filepath.Join(tmpDir, "plan.yaml")
	statePath := filepath.Join(tmpDir, "state.yaml")

	// Create plan with 2 violations that share incidents
	plan := planfile.NewPlan("test-provider", 1)
	plan.Metadata.CreatedAt = time.Now()
	plan.Phases = []planfile.Phase{
		{
			ID:            "phase-1",
			Name:          "Test Phase",
			Order:         1,
			Risk:          planfile.RiskLow,
			Category:      "mandatory",
			EffortRange:   [2]int{1, 3},
			EstimatedCost: 0.15,
			Violations: []planfile.PlannedViolation{
				{
					ViolationID:   "violation-1",
					Description:   "Violation 1",
					Category:      "mandatory",
					Effort:        2,
					IncidentCount: 2,
					Incidents: []violation.Incident{
						{URI: "file:///test1.java", LineNumber: 10, Message: "V1 incident 1"},
						{URI: "file:///test2.java", LineNumber: 20, Message: "V1 incident 2"},
					},
				},
				{
					ViolationID:   "violation-2",
					Description:   "Violation 2",
					Category:      "mandatory",
					Effort:        2,
					IncidentCount: 2,
					Incidents: []violation.Incident{
						// Same location as violation-1 but different violation ID - NOT a duplicate
						{URI: "file:///test1.java", LineNumber: 10, Message: "V2 incident 1"},
						{URI: "file:///test2.java", LineNumber: 30, Message: "V2 incident 2"},
					},
				},
			},
		},
	}

	err = planfile.SavePlan(plan, planPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()

	// First batch - violation-1 (2 incidents)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return req.Violation.ID == "violation-1" && len(req.Incidents) == 2
	})).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{IncidentURI: "file:///test1.java:10", Success: true, FixedContent: "fixed", Confidence: 0.9},
				{IncidentURI: "file:///test2.java:20", Success: true, FixedContent: "fixed", Confidence: 0.9},
			},
			Success:    true,
			TokensUsed: 100,
			Cost:       0.05,
		},
		nil,
	).Once()

	// Second batch - violation-2 (2 incidents, no duplicates since violation ID differs)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return req.Violation.ID == "violation-2" && len(req.Incidents) == 2
	})).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{IncidentURI: "file:///test1.java:10", Success: true, FixedContent: "fixed", Confidence: 0.9},
				{IncidentURI: "file:///test2.java:30", Success: true, FixedContent: "fixed", Confidence: 0.9},
			},
			Success:    true,
			TokensUsed: 100,
			Cost:       0.05,
		},
		nil,
	).Once()

	// Disable file grouping for this test to test deduplication across violations
	batchConfig := fixer.DefaultBatchConfig()
	batchConfig.GroupByFile = false

	config := Config{
		PlanPath:    planPath,
		StatePath:   statePath,
		InputPath:   tmpDir,
		Provider:    mockProvider,
		Progress:    &ux.NoOpProgressWriter{},
		DryRun:      true,
		BatchConfig: batchConfig,
	}

	exec, err := New(config)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := exec.Execute(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 4, result.SuccessfulFixes) // All 4 incidents fixed (no duplicates across violations)
	assert.Equal(t, 0, result.DuplicateFixes)  // No duplicates since violation IDs differ

	mockProvider.AssertExpectations(t)
}
