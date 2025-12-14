package planner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tsanders/kantra-ai/pkg/planfile"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
	"gopkg.in/yaml.v3"
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
		AnalysisPath: "/tmp/analysis.yaml",
		InputPath:    "/tmp/input",
		Provider:     mockProvider,
	}

	p := New(config)

	assert.NotNil(t, p)
	assert.Equal(t, ".kantra-ai-plan.yaml", p.config.OutputPath)
	assert.Equal(t, "balanced", p.config.RiskTolerance)
}

func TestNew_WithCustomDefaults(t *testing.T) {
	mockProvider := new(MockProvider)

	config := Config{
		AnalysisPath:  "/tmp/analysis.yaml",
		InputPath:     "/tmp/input",
		Provider:      mockProvider,
		OutputPath:    "custom-plan.yaml",
		RiskTolerance: "conservative",
	}

	p := New(config)

	assert.Equal(t, "custom-plan.yaml", p.config.OutputPath)
	assert.Equal(t, "conservative", p.config.RiskTolerance)
}

func TestGenerate_BasicFlow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test analysis file
	analysisPath := filepath.Join(tmpDir, "analysis.yaml")
	analysis := createTestAnalysis()
	err = saveAnalysis(analysis, analysisPath)
	assert.NoError(t, err)

	outputPath := filepath.Join(tmpDir, "plan.yaml")

	// Create mock provider
	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()
	mockProvider.On("GeneratePlan", mock.Anything, mock.Anything).Return(
		&provider.PlanResponse{
			Phases: []provider.PlannedPhase{
				{
					ID:          "phase-1",
					Name:        "Critical Mandatory Fixes",
					Order:       1,
					Risk:        "high",
					Category:    "mandatory",
					EffortRange: [2]int{5, 7},
					Explanation: "High effort mandatory fixes",
					ViolationIDs: []string{"javax-to-jakarta"},
					EstimatedCost:            0.50,
					EstimatedDurationMinutes: 15,
				},
			},
			TokensUsed: 500,
			Cost:       0.10,
		},
		nil,
	).Once()

	config := Config{
		AnalysisPath: analysisPath,
		InputPath:    tmpDir,
		Provider:     mockProvider,
		OutputPath:   outputPath,
	}

	p := New(config)

	ctx := context.Background()
	result, err := p.Generate(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, outputPath, result.PlanPath)
	assert.Equal(t, 1, result.TotalPhases)
	assert.Equal(t, 0.50, result.TotalCost)
	assert.Equal(t, 0.10, result.GenerateCost)
	assert.Equal(t, 500, result.TokensUsed)

	// Verify plan file was created
	plan, err := planfile.LoadPlan(outputPath)
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Len(t, plan.Phases, 1)
	assert.Equal(t, "phase-1", plan.Phases[0].ID)

	mockProvider.AssertExpectations(t)
}

func TestGenerate_WithFilters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	analysisPath := filepath.Join(tmpDir, "analysis.yaml")
	analysis := createTestAnalysisMultipleViolations()
	err = saveAnalysis(analysis, analysisPath)
	assert.NoError(t, err)

	outputPath := filepath.Join(tmpDir, "plan.yaml")

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()
	mockProvider.On("GeneratePlan", mock.Anything, mock.MatchedBy(func(req provider.PlanRequest) bool {
		// Verify only mandatory violations are included
		return len(req.Violations) == 1 && req.Violations[0].ID == "javax-to-jakarta"
	})).Return(
		&provider.PlanResponse{
			Phases: []provider.PlannedPhase{
				{
					ID:            "phase-1",
					Name:          "Mandatory Only",
					Order:         1,
					Risk:          "medium",
					Category:      "mandatory",
					ViolationIDs:  []string{"javax-to-jakarta"},
					EstimatedCost: 0.30,
				},
			},
			TokensUsed: 300,
			Cost:       0.05,
		},
		nil,
	).Once()

	config := Config{
		AnalysisPath:  analysisPath,
		InputPath:     tmpDir,
		Provider:      mockProvider,
		OutputPath:    outputPath,
		Categories:    []string{"mandatory"},
		RiskTolerance: "balanced",
	}

	p := New(config)

	ctx := context.Background()
	result, err := p.Generate(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	mockProvider.AssertExpectations(t)
}

func TestGenerate_ProviderError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	analysisPath := filepath.Join(tmpDir, "analysis.yaml")
	analysis := createTestAnalysis()
	err = saveAnalysis(analysis, analysisPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider").Maybe()
	mockProvider.On("GeneratePlan", mock.Anything, mock.Anything).Return(
		nil,
		assert.AnError,
	).Once()

	config := Config{
		AnalysisPath: analysisPath,
		InputPath:    tmpDir,
		Provider:     mockProvider,
		OutputPath:   filepath.Join(tmpDir, "plan.yaml"),
	}

	p := New(config)

	ctx := context.Background()
	result, err := p.Generate(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to generate plan")

	mockProvider.AssertExpectations(t)
}

func TestGenerate_NoViolations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-test-*")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	analysisPath := filepath.Join(tmpDir, "analysis.yaml")
	analysis := &violation.Analysis{
		Violations: []violation.Violation{},
	}
	err = saveAnalysis(analysis, analysisPath)
	assert.NoError(t, err)

	mockProvider := new(MockProvider)

	config := Config{
		AnalysisPath: analysisPath,
		InputPath:    tmpDir,
		Provider:     mockProvider,
		OutputPath:   filepath.Join(tmpDir, "plan.yaml"),
	}

	p := New(config)

	ctx := context.Background()
	result, err := p.Generate(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no violations match")

	mockProvider.AssertNotCalled(t, "GeneratePlan")
}

func TestBuildPlan(t *testing.T) {
	violations := []violation.Violation{
		{
			ID:          "test-violation-1",
			Description: "Test violation 1",
			Category:    "mandatory",
			Effort:      5,
			Incidents: []violation.Incident{
				{URI: "file:///test.java", LineNumber: 10},
			},
		},
	}

	providerResp := &provider.PlanResponse{
		Phases: []provider.PlannedPhase{
			{
				ID:                       "phase-1",
				Name:                     "Test Phase",
				Order:                    1,
				Risk:                     "medium",
				Category:                 "mandatory",
				EffortRange:              [2]int{3, 7},
				Explanation:              "Test explanation",
				ViolationIDs:             []string{"test-violation-1"},
				EstimatedCost:            0.25,
				EstimatedDurationMinutes: 10,
			},
		},
	}

	p := &Planner{
		config: Config{
			Provider: &MockProvider{},
		},
	}

	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("test-provider")
	p.config.Provider = mockProvider

	plan := p.buildPlan(providerResp, violations)

	assert.NotNil(t, plan)
	assert.Len(t, plan.Phases, 1)
	assert.Equal(t, "phase-1", plan.Phases[0].ID)
	assert.Equal(t, "Test Phase", plan.Phases[0].Name)
	assert.Equal(t, planfile.RiskMedium, plan.Phases[0].Risk)
	assert.Len(t, plan.Phases[0].Violations, 1)
	assert.Equal(t, "test-violation-1", plan.Phases[0].Violations[0].ViolationID)
	assert.Equal(t, 1, plan.Phases[0].Violations[0].IncidentCount)
}

func TestMapRiskLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected planfile.RiskLevel
	}{
		{"low", planfile.RiskLow},
		{"medium", planfile.RiskMedium},
		{"high", planfile.RiskHigh},
		{"unknown", planfile.RiskMedium}, // default
		{"", planfile.RiskMedium},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapRiskLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func saveAnalysis(analysis *violation.Analysis, path string) error {
	data, err := yaml.Marshal(analysis)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func createTestAnalysis() *violation.Analysis {
	return &violation.Analysis{
		Violations: []violation.Violation{
			{
				ID:          "javax-to-jakarta",
				Description: "Replace javax.* with jakarta.*",
				Category:    "mandatory",
				Effort:      7,
				Incidents: []violation.Incident{
					{
						URI:        "file:///src/Servlet.java",
						LineNumber: 10,
						Message:    "Replace javax.servlet",
					},
					{
						URI:        "file:///src/Controller.java",
						LineNumber: 20,
						Message:    "Replace javax.servlet.http",
					},
				},
			},
		},
	}
}

func createTestAnalysisMultipleViolations() *violation.Analysis {
	return &violation.Analysis{
		Violations: []violation.Violation{
			{
				ID:          "javax-to-jakarta",
				Description: "Replace javax.* with jakarta.*",
				Category:    "mandatory",
				Effort:      7,
				Incidents: []violation.Incident{
					{URI: "file:///src/Servlet.java", LineNumber: 10},
				},
			},
			{
				ID:          "logger-update",
				Description: "Update logger usage",
				Category:    "optional",
				Effort:      3,
				Incidents: []violation.Incident{
					{URI: "file:///src/Logger.java", LineNumber: 5},
				},
			},
		},
	}
}
