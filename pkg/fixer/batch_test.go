package fixer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestNewBatchFixer(t *testing.T) {
	mockProvider := new(MockProvider)
	config := DefaultBatchConfig()

	bf := NewBatchFixer(mockProvider, "/tmp/test", false, config)

	assert.NotNil(t, bf)
	assert.Equal(t, mockProvider, bf.provider)
	assert.Equal(t, "/tmp/test", bf.inputDir)
	assert.False(t, bf.dryRun)
	assert.Equal(t, config, bf.config)
}

func TestBatchFixer_FixViolationBatch_SingleBatch(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.java")
	testFile2 := filepath.Join(tmpDir, "test2.java")
	err := os.WriteFile(testFile1, []byte("class Test1 {}"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("class Test2 {}"), 0644)
	require.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file://" + testFile1 + ":10",
					Success:      true,
					FixedContent: "class Test1Fixed {}",
					Explanation:  "Fixed test1",
					Confidence:   0.9,
				},
				{
					IncidentURI:  "file://" + testFile2 + ":20",
					Success:      true,
					FixedContent: "class Test2Fixed {}",
					Explanation:  "Fixed test2",
					Confidence:   0.9,
				},
			},
			Success:    true,
			TokensUsed: 200,
			Cost:       0.10,
		},
		nil,
	)

	config := DefaultBatchConfig()
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID:          "test-violation",
		Description: "Test violation",
		Incidents: []violation.Incident{
			{URI: "file://" + testFile1, LineNumber: 10},
			{URI: "file://" + testFile2, LineNumber: 20},
		},
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.True(t, results[1].Success)
	assert.Equal(t, 100, results[0].TokensUsed) // 200/2
	assert.Equal(t, 100, results[1].TokensUsed)
	assert.Equal(t, 0.05, results[0].Cost) // 0.10/2
	assert.Equal(t, 0.05, results[1].Cost)

	mockProvider.AssertExpectations(t)
}

func TestBatchFixer_FixViolationBatch_MultipleBatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 15 test files (will be split into 2 batches: 10 + 5)
	var incidents []violation.Incident
	for i := 0; i < 15; i++ {
		testFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".java")
		err := os.WriteFile(testFile, []byte("class Test {}"), 0644)
		require.NoError(t, err)
		incidents = append(incidents, violation.Incident{
			URI:        "file://" + testFile,
			LineNumber: 10,
		})
	}

	mockProvider := new(MockProvider)

	// First batch (10 incidents)
	firstBatchFixes := make([]provider.IncidentFix, 10)
	for i := 0; i < 10; i++ {
		firstBatchFixes[i] = provider.IncidentFix{
			IncidentURI:  incidents[i].URI + ":10",
			Success:      true,
			FixedContent: "class TestFixed {}",
			Confidence:   0.9,
		}
	}
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 10
	})).Return(
		&provider.BatchResponse{
			Fixes:      firstBatchFixes,
			Success:    true,
			TokensUsed: 1000,
			Cost:       0.50,
		},
		nil,
	).Once()

	// Second batch (5 incidents)
	secondBatchFixes := make([]provider.IncidentFix, 5)
	for i := 0; i < 5; i++ {
		secondBatchFixes[i] = provider.IncidentFix{
			IncidentURI:  incidents[10+i].URI + ":10",
			Success:      true,
			FixedContent: "class TestFixed {}",
			Confidence:   0.9,
		}
	}
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 5
	})).Return(
		&provider.BatchResponse{
			Fixes:      secondBatchFixes,
			Success:    true,
			TokensUsed: 500,
			Cost:       0.25,
		},
		nil,
	).Once()

	config := DefaultBatchConfig()
	config.MaxBatchSize = 10
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID:        "test-violation",
		Incidents: incidents,
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	require.NoError(t, err)
	assert.Len(t, results, 15)

	// Verify all succeeded
	for _, result := range results {
		assert.True(t, result.Success)
	}

	mockProvider.AssertExpectations(t)
}

func TestBatchFixer_FixViolationBatch_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()

	testFile1 := filepath.Join(tmpDir, "test1.java")
	testFile2 := filepath.Join(tmpDir, "test2.java")
	err := os.WriteFile(testFile1, []byte("class Test1 {}"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("class Test2 {}"), 0644)
	require.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file://" + testFile1 + ":10",
					Success:      true,
					FixedContent: "class Test1Fixed {}",
					Confidence:   0.9,
				},
				{
					IncidentURI:  "file://" + testFile2 + ":20",
					Success:      false,
					Error:        assert.AnError,
				},
			},
			Success:    false, // Overall failure
			TokensUsed: 150,
			Cost:       0.08,
		},
		nil,
	)

	config := DefaultBatchConfig()
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID: "test-violation",
		Incidents: []violation.Incident{
			{URI: "file://" + testFile1, LineNumber: 10},
			{URI: "file://" + testFile2, LineNumber: 20},
		},
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.False(t, results[1].Success)
	assert.NotNil(t, results[1].Error)

	mockProvider.AssertExpectations(t)
}

func TestBatchFixer_FixViolationBatch_BatchingDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.java")
	err := os.WriteFile(testFile, []byte("class Test {}"), 0644)
	require.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("FixViolation", mock.Anything, mock.Anything).Return(
		&provider.FixResponse{
			Success:      true,
			FixedContent: "class TestFixed {}",
			TokensUsed:   100,
			Cost:         0.05,
		},
		nil,
	)

	config := DefaultBatchConfig()
	config.Enabled = false // Disable batching
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID: "test-violation",
		Incidents: []violation.Incident{
			{URI: "file://" + testFile, LineNumber: 10},
		},
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)

	// Should NOT call FixBatch, should call FixViolation instead
	mockProvider.AssertNotCalled(t, "FixBatch")
	mockProvider.AssertExpectations(t)
}

func TestBatchFixer_FixViolationBatch_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	mockProvider := new(MockProvider)

	config := DefaultBatchConfig()
	bf := NewBatchFixer(mockProvider, tmpDir, false, config)

	v := violation.Violation{
		ID: "test-violation",
		Incidents: []violation.Incident{
			{URI: "file:///nonexistent/file.java", LineNumber: 10},
		},
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	// Should not return error from FixViolationBatch, but results should contain failed fix
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.False(t, results[0].Success)
	assert.NotNil(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "failed to read file")

	mockProvider.AssertNotCalled(t, "FixBatch")
}

func TestBatchFixer_CreateBatches(t *testing.T) {
	tests := []struct {
		name             string
		incidentCount    int
		maxBatchSize     int
		expectedBatches  int
		expectedLastSize int
	}{
		{
			name:             "exact fit",
			incidentCount:    10,
			maxBatchSize:     10,
			expectedBatches:  1,
			expectedLastSize: 10,
		},
		{
			name:             "multiple batches",
			incidentCount:    25,
			maxBatchSize:     10,
			expectedBatches:  3,
			expectedLastSize: 5,
		},
		{
			name:             "single incident",
			incidentCount:    1,
			maxBatchSize:     10,
			expectedBatches:  1,
			expectedLastSize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultBatchConfig()
			config.MaxBatchSize = tt.maxBatchSize
			bf := NewBatchFixer(nil, "/tmp", false, config)

			incidents := make([]violation.Incident, tt.incidentCount)
			for i := 0; i < tt.incidentCount; i++ {
				incidents[i] = violation.Incident{
					URI:        "file:///test.java",
					LineNumber: i + 1,
				}
			}

			v := violation.Violation{
				ID:        "test",
				Incidents: incidents,
			}

			batches := bf.createBatches(v)

			assert.Len(t, batches, tt.expectedBatches)
			assert.Len(t, batches[len(batches)-1].incidents, tt.expectedLastSize)
		})
	}
}

func TestBatchFixer_ResolveFilePath(t *testing.T) {
	tests := []struct {
		name     string
		inputDir string
		filePath string
		expected string
	}{
		{
			name:     "absolute path matching inputDir",
			inputDir: "/workspace/project",
			filePath: "/workspace/project/src/Main.java",
			expected: "src/Main.java",
		},
		{
			name:     "absolute path not matching inputDir",
			inputDir: "/workspace/project",
			filePath: "/other/path/Main.java",
			expected: "other/path/Main.java",
		},
		{
			name:     "relative path",
			inputDir: "/workspace/project",
			filePath: "src/Main.java",
			expected: "src/Main.java",
		},
		{
			name:     "path with leading slashes",
			inputDir: "/workspace/project",
			filePath: "///src/Main.java",
			expected: "src/Main.java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveAndValidateFilePath(tt.filePath, tt.inputDir)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBatchFixer_Parallelism(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 20 test files to trigger parallel processing
	var incidents []violation.Incident
	for i := 0; i < 20; i++ {
		testFile := filepath.Join(tmpDir, "test_"+string(rune('a'+i))+".java")
		err := os.WriteFile(testFile, []byte("class Test {}"), 0644)
		require.NoError(t, err)
		incidents = append(incidents, violation.Incident{
			URI:        "file://" + testFile,
			LineNumber: 10,
		})
	}

	mockProvider := new(MockProvider)

	// Should create 2 batches (10 + 10)
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
				{IncidentURI: "file:///test:10", Success: true, FixedContent: "fixed"},
			},
			Success:    true,
			TokensUsed: 1000,
			Cost:       0.50,
		},
		nil,
	).Times(2)

	config := DefaultBatchConfig()
	config.MaxBatchSize = 10
	config.Parallelism = 4
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID:        "test-violation",
		Incidents: incidents,
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	require.NoError(t, err)
	assert.Len(t, results, 20)

	mockProvider.AssertExpectations(t)
}

func TestDefaultBatchConfig(t *testing.T) {
	config := DefaultBatchConfig()

	assert.Equal(t, 10, config.MaxBatchSize)
	assert.Equal(t, 4, config.Parallelism)
	assert.True(t, config.Enabled)
}

func TestGetFilePathFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "file URI with line number",
			uri:      "file:///path/to/file.java:123",
			expected: "/path/to/file.java",
		},
		{
			name:     "file URI without line number",
			uri:      "file:///path/to/file.java",
			expected: "/path/to/file.java",
		},
		{
			name:     "plain path with line number",
			uri:      "/path/to/file.java:456",
			expected: "/path/to/file.java",
		},
		{
			name:     "plain path without line number",
			uri:      "/path/to/file.java",
			expected: "/path/to/file.java",
		},
		{
			name:     "path with colon in name (not line number)",
			uri:      "/path/to/file:name.java",
			expected: "/path/to/file:name.java",
		},
		{
			name:     "empty string",
			uri:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFilePathFromURI(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a smaller", 5, 10, 5},
		{"b smaller", 10, 5, 5},
		{"equal", 5, 5, 5},
		{"negative", -5, 10, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
