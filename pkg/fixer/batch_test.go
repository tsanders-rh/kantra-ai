package fixer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

	// Create a single test file with multiple incidents
	testFile := filepath.Join(tmpDir, "test.java")
	err := os.WriteFile(testFile, []byte("class Test {}"), 0644)
	require.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file://" + testFile + ":10",
					Success:      true,
					FixedContent: "class TestFixed {}",
					Explanation:  "Fixed line 10",
					Confidence:   0.9,
				},
				{
					IncidentURI:  "file://" + testFile + ":20",
					Success:      true,
					FixedContent: "class TestFixed2 {}",
					Explanation:  "Fixed line 20",
					Confidence:   0.9,
				},
			},
			Success:    true,
			TokensUsed: 200,
			Cost:       0.10,
		},
		nil,
	).Once()

	config := DefaultBatchConfig()
	// GroupByFile is enabled by default - both incidents are in the same file
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID:          "test-violation",
		Description: "Test violation",
		Incidents: []violation.Incident{
			{URI: "file://" + testFile, LineNumber: 10},
			{URI: "file://" + testFile, LineNumber: 20},
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
		testFile := filepath.Join(tmpDir, fmt.Sprintf("test_%d.java", i))
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
	config.GroupByFile = false // Disable file grouping for simpler test expectations
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

	testFile := filepath.Join(tmpDir, "test.java")
	err := os.WriteFile(testFile, []byte("class Test {}"), 0644)
	require.NoError(t, err)

	mockProvider := new(MockProvider)
	mockProvider.On("FixBatch", mock.Anything, mock.Anything).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{
					IncidentURI:  "file://" + testFile + ":10",
					Success:      true,
					FixedContent: "class TestFixed {}",
					Confidence:   0.9,
				},
				{
					IncidentURI:  "file://" + testFile + ":20",
					Success:      false,
					Error:        assert.AnError,
				},
			},
			Success:    false, // Overall failure
			TokensUsed: 150,
			Cost:       0.08,
		},
		nil,
	).Once()

	config := DefaultBatchConfig()
	// GroupByFile is enabled by default - both incidents are in the same file
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID: "test-violation",
		Incidents: []violation.Incident{
			{URI: "file://" + testFile, LineNumber: 10},
			{URI: "file://" + testFile, LineNumber: 20},
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
		name          string
		inputDir      string
		filePath      string
		expected      string
		skipOnWindows bool
	}{
		{
			name:          "absolute path matching inputDir",
			inputDir:      "/workspace/project",
			filePath:      "/workspace/project/src/Main.java",
			expected:      "src/Main.java",
			skipOnWindows: true, // Unix-style absolute paths don't work the same on Windows
		},
		{
			name:          "absolute path not matching inputDir",
			inputDir:      "/workspace/project",
			filePath:      "/other/path/Main.java",
			expected:      "other/path/Main.java",
			skipOnWindows: true, // Unix-style absolute paths don't work the same on Windows
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
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping on Windows: Unix-style absolute paths behave differently")
			}
			result, err := resolveAndValidateFilePath(tt.filePath, tt.inputDir)
			require.NoError(t, err)
			// Normalize to forward slashes for cross-platform comparison
			assert.Equal(t, tt.expected, filepath.ToSlash(result))
		})
	}
}

func TestBatchFixer_Parallelism(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 20 test files to trigger parallel processing
	var incidents []violation.Incident
	for i := 0; i < 20; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("test_%d.java", i))
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
	config.GroupByFile = false // Disable file grouping for simpler test expectations
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

func TestBatchFixer_FixViolationBatch_FileGrouping(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 3 test files with multiple incidents each
	testFile1 := filepath.Join(tmpDir, "file1.java")
	testFile2 := filepath.Join(tmpDir, "file2.java")
	testFile3 := filepath.Join(tmpDir, "file3.java")

	err := os.WriteFile(testFile1, []byte("class File1 {}"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("class File2 {}"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile3, []byte("class File3 {}"), 0644)
	require.NoError(t, err)

	mockProvider := new(MockProvider)

	// Expect 3 separate batches (one per file) due to file grouping
	// Batch 1: file1.java (2 incidents)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 2 && req.Incidents[0].GetFilePath() == testFile1
	})).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{IncidentURI: "file://" + testFile1 + ":10", Success: true, FixedContent: "fixed", Confidence: 0.9},
				{IncidentURI: "file://" + testFile1 + ":20", Success: true, FixedContent: "fixed", Confidence: 0.9},
			},
			Success:    true,
			TokensUsed: 100,
			Cost:       0.05,
		},
		nil,
	).Once()

	// Batch 2: file2.java (3 incidents)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 3 && req.Incidents[0].GetFilePath() == testFile2
	})).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{IncidentURI: "file://" + testFile2 + ":10", Success: true, FixedContent: "fixed", Confidence: 0.9},
				{IncidentURI: "file://" + testFile2 + ":20", Success: true, FixedContent: "fixed", Confidence: 0.9},
				{IncidentURI: "file://" + testFile2 + ":30", Success: true, FixedContent: "fixed", Confidence: 0.9},
			},
			Success:    true,
			TokensUsed: 150,
			Cost:       0.075,
		},
		nil,
	).Once()

	// Batch 3: file3.java (1 incident)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 1 && req.Incidents[0].GetFilePath() == testFile3
	})).Return(
		&provider.BatchResponse{
			Fixes: []provider.IncidentFix{
				{IncidentURI: "file://" + testFile3 + ":10", Success: true, FixedContent: "fixed", Confidence: 0.9},
			},
			Success:    true,
			TokensUsed: 50,
			Cost:       0.025,
		},
		nil,
	).Once()

	config := DefaultBatchConfig()
	config.MaxBatchSize = 10 // Large enough to fit all incidents from each file
	// GroupByFile is enabled by default
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID: "test-violation",
		Incidents: []violation.Incident{
			{URI: "file://" + testFile1, LineNumber: 10},
			{URI: "file://" + testFile1, LineNumber: 20},
			{URI: "file://" + testFile2, LineNumber: 10},
			{URI: "file://" + testFile2, LineNumber: 20},
			{URI: "file://" + testFile2, LineNumber: 30},
			{URI: "file://" + testFile3, LineNumber: 10},
		},
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	require.NoError(t, err)
	assert.Len(t, results, 6)

	// Verify all succeeded
	for _, result := range results {
		assert.True(t, result.Success)
	}

	mockProvider.AssertExpectations(t)
}

func TestBatchFixer_FixViolationBatch_FileGroupingWithSizeBatching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that will require multiple batches due to MaxBatchSize
	testFile := filepath.Join(tmpDir, "large.java")
	err := os.WriteFile(testFile, []byte("class Large {}"), 0644)
	require.NoError(t, err)

	mockProvider := new(MockProvider)

	// Create 25 incidents for the same file with MaxBatchSize=10
	// Should create 3 batches: 10 + 10 + 5
	var incidents []violation.Incident
	for i := 0; i < 25; i++ {
		incidents = append(incidents, violation.Incident{
			URI:        "file://" + testFile,
			LineNumber: i + 1,
		})
	}

	// First batch (10 incidents)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 10
	})).Return(
		&provider.BatchResponse{
			Fixes: func() []provider.IncidentFix {
				fixes := make([]provider.IncidentFix, 10)
				for i := 0; i < 10; i++ {
					fixes[i] = provider.IncidentFix{
						IncidentURI:  "file://" + testFile + ":" + string(rune('0'+i+1)),
						Success:      true,
						FixedContent: "fixed",
						Confidence:   0.9,
					}
				}
				return fixes
			}(),
			Success:    true,
			TokensUsed: 500,
			Cost:       0.25,
		},
		nil,
	).Once()

	// Second batch (10 incidents)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 10
	})).Return(
		&provider.BatchResponse{
			Fixes: func() []provider.IncidentFix {
				fixes := make([]provider.IncidentFix, 10)
				for i := 0; i < 10; i++ {
					fixes[i] = provider.IncidentFix{
						IncidentURI:  "file://" + testFile + ":" + string(rune('0'+i+11)),
						Success:      true,
						FixedContent: "fixed",
						Confidence:   0.9,
					}
				}
				return fixes
			}(),
			Success:    true,
			TokensUsed: 500,
			Cost:       0.25,
		},
		nil,
	).Once()

	// Third batch (5 incidents)
	mockProvider.On("FixBatch", mock.Anything, mock.MatchedBy(func(req provider.BatchRequest) bool {
		return len(req.Incidents) == 5
	})).Return(
		&provider.BatchResponse{
			Fixes: func() []provider.IncidentFix {
				fixes := make([]provider.IncidentFix, 5)
				for i := 0; i < 5; i++ {
					fixes[i] = provider.IncidentFix{
						IncidentURI:  "file://" + testFile + ":" + string(rune('0'+i+21)),
						Success:      true,
						FixedContent: "fixed",
						Confidence:   0.9,
					}
				}
				return fixes
			}(),
			Success:    true,
			TokensUsed: 250,
			Cost:       0.125,
		},
		nil,
	).Once()

	config := DefaultBatchConfig()
	config.MaxBatchSize = 10
	// GroupByFile is enabled by default
	bf := NewBatchFixer(mockProvider, tmpDir, true, config)

	v := violation.Violation{
		ID:        "test-violation",
		Incidents: incidents,
	}

	results, err := bf.FixViolationBatch(context.Background(), v)

	require.NoError(t, err)
	assert.Len(t, results, 25)

	// Verify all succeeded
	for _, result := range results {
		assert.True(t, result.Success)
	}

	mockProvider.AssertExpectations(t)
}

func TestBatchFixer_CreateBatches_FileGrouping(t *testing.T) {
	config := DefaultBatchConfig()
	config.MaxBatchSize = 10
	// GroupByFile is enabled by default
	bf := NewBatchFixer(nil, "/tmp", false, config)

	// Create incidents across 3 files
	incidents := []violation.Incident{
		{URI: "file:///file1.java", LineNumber: 10},
		{URI: "file:///file1.java", LineNumber: 20},
		{URI: "file:///file1.java", LineNumber: 30},
		{URI: "file:///file2.java", LineNumber: 10},
		{URI: "file:///file2.java", LineNumber: 20},
		{URI: "file:///file3.java", LineNumber: 10},
	}

	v := violation.Violation{
		ID:        "test",
		Incidents: incidents,
	}

	batches := bf.createBatches(v)

	// Should create 3 batches (one per file)
	assert.Len(t, batches, 3)

	// Verify each batch contains incidents from only one file
	fileGroups := make(map[string]int)
	for _, batch := range batches {
		filePath := batch.incidents[0].GetFilePath()
		fileGroups[filePath]++

		// All incidents in batch should be from the same file
		for _, incident := range batch.incidents {
			assert.Equal(t, filePath, incident.GetFilePath())
		}
	}

	// Verify we have batches for all 3 files
	assert.Len(t, fileGroups, 3)
}

func TestBatchFixer_CreateBatches_FileGroupingWithSizeSplit(t *testing.T) {
	config := DefaultBatchConfig()
	config.MaxBatchSize = 5
	// GroupByFile is enabled by default
	bf := NewBatchFixer(nil, "/tmp", false, config)

	// Create 12 incidents for the same file (should split into 3 batches: 5+5+2)
	incidents := make([]violation.Incident, 12)
	for i := 0; i < 12; i++ {
		incidents[i] = violation.Incident{
			URI:        "file:///large.java",
			LineNumber: i + 1,
		}
	}

	v := violation.Violation{
		ID:        "test",
		Incidents: incidents,
	}

	batches := bf.createBatches(v)

	// Should create 3 batches for the same file
	assert.Len(t, batches, 3)
	assert.Len(t, batches[0].incidents, 5)
	assert.Len(t, batches[1].incidents, 5)
	assert.Len(t, batches[2].incidents, 2)

	// All batches should be for the same file
	for _, batch := range batches {
		for _, incident := range batch.incidents {
			assert.Equal(t, "/large.java", incident.GetFilePath())
		}
	}
}

func TestDefaultBatchConfig(t *testing.T) {
	config := DefaultBatchConfig()

	assert.Equal(t, 10, config.MaxBatchSize)
	assert.Equal(t, 8, config.Parallelism)
	assert.True(t, config.Enabled)
	assert.True(t, config.GroupByFile)
	assert.Equal(t, 0, config.MaxTokensPerBatch) // Disabled by default
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
