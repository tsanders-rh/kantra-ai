package fixer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tsanders/kantra-ai/pkg/confidence"
	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
	"gopkg.in/yaml.v3"
)

// MockProvider is a mock implementation of the Provider interface
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

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{"Java file", "Test.java", "java"},
		{"Python file", "test.py", "python"},
		{"Go file", "main.go", "go"},
		{"JavaScript file", "app.js", "javascript"},
		{"TypeScript file", "app.ts", "typescript"},
		{"Ruby file", "script.rb", "ruby"},
		{"XML file", "config.xml", "xml"},
		{"YAML file", "config.yaml", "yaml"},
		{"YML file", "config.yml", "yaml"},
		{"Unknown extension", "file.xyz", "unknown"},
		{"No extension", "README", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLanguage(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCleanResponse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Java code block",
			input: "```java\npublic class Test {}\n```",
			want:  "public class Test {}",
		},
		{
			name:  "Python code block",
			input: "```python\ndef test():\n    pass\n```",
			want:  "def test():\n    pass",
		},
		{
			name:  "Go code block",
			input: "```go\npackage main\n```",
			want:  "package main",
		},
		{
			name:  "JavaScript code block",
			input: "```javascript\nconsole.log('hello');\n```",
			want:  "console.log('hello');",
		},
		{
			name:  "TypeScript code block",
			input: "```typescript\nconst x: number = 5;\n```",
			want:  "const x: number = 5;",
		},
		{
			name:  "Generic code block",
			input: "```\nsome code\n```",
			want:  "some code",
		},
		{
			name:  "Plain text no markers",
			input: "plain text content",
			want:  "plain text content",
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanResponse(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFixer_New(t *testing.T) {
	mockProvider := new(MockProvider)
	mockProvider.On("Name").Return("mock")

	fixer := New(mockProvider, "/test/input", false)

	assert.NotNil(t, fixer)
	assert.Equal(t, mockProvider, fixer.provider)
	assert.Equal(t, "/test/input", fixer.inputDir)
	assert.False(t, fixer.dryRun)

	// Test with dry-run enabled
	fixerDryRun := New(mockProvider, "/test/input", true)
	assert.True(t, fixerDryRun.dryRun)
}

func TestFixer_FixIncident(t *testing.T) {
	t.Run("successful fix", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.java")
		originalContent := "import javax.servlet.*;"
		err := os.WriteFile(testFile, []byte(originalContent), 0644)
		require.NoError(t, err)

		// Create mock provider
		mockProvider := new(MockProvider)
		fixedContent := "import jakarta.servlet.*;"
		mockProvider.On("FixViolation", mock.Anything, mock.Anything).Return(&provider.FixResponse{
			Success:      true,
			FixedContent: fixedContent,
			Explanation:  "Replaced javax with jakarta",
			Cost:         0.01,
			TokensUsed:   100,
		}, nil)

		// Create fixer
		fixer := New(mockProvider, tmpDir, false)

		// Create test violation and incident
		v := violation.Violation{
			ID:          "test-violation",
			Description: "Test violation",
		}
		incident := violation.Incident{
			URI:        "file://" + testFile,
			LineNumber: 1,
		}

		// Fix the incident
		result, err := fixer.FixIncident(context.Background(), v, incident)

		// Assertions
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "test.java", result.FilePath)
		assert.Equal(t, 0.01, result.Cost)
		assert.Equal(t, 100, result.TokensUsed)

		// Verify file was updated
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, fixedContent, string(content))

		mockProvider.AssertExpectations(t)
	})

	t.Run("dry-run mode doesn't write file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.py")
		originalContent := "print('old')"
		err := os.WriteFile(testFile, []byte(originalContent), 0644)
		require.NoError(t, err)

		mockProvider := new(MockProvider)
		fixedContent := "print('new')"
		mockProvider.On("FixViolation", mock.Anything, mock.Anything).Return(&provider.FixResponse{
			Success:      true,
			FixedContent: fixedContent,
			Explanation:  "Updated print statement",
			Cost:         0.01,
			TokensUsed:   50,
		}, nil)

		// Create fixer with dry-run enabled
		fixer := New(mockProvider, tmpDir, true)

		v := violation.Violation{ID: "test"}
		incident := violation.Incident{URI: "file://" + testFile}

		result, err := fixer.FixIncident(context.Background(), v, incident)

		require.NoError(t, err)
		assert.True(t, result.Success)

		// Verify file was NOT updated
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, originalContent, string(content))

		mockProvider.AssertExpectations(t)
	})

	t.Run("file read error", func(t *testing.T) {
		mockProvider := new(MockProvider)
		fixer := New(mockProvider, "/nonexistent", false)

		v := violation.Violation{ID: "test"}
		incident := violation.Incident{URI: "file:///nonexistent/file.java"}

		result, err := fixer.FixIncident(context.Background(), v, incident)

		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.NotNil(t, result.Error)
		assert.Contains(t, result.Error.Error(), "failed to read file")
	})

	t.Run("provider error", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main"), 0644)
		require.NoError(t, err)

		mockProvider := new(MockProvider)
		mockProvider.On("FixViolation", mock.Anything, mock.Anything).Return(
			(*provider.FixResponse)(nil),
			assert.AnError,
		)

		fixer := New(mockProvider, tmpDir, false)

		v := violation.Violation{ID: "test"}
		incident := violation.Incident{URI: "file://" + testFile}

		result, err := fixer.FixIncident(context.Background(), v, incident)

		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, assert.AnError, result.Error)

		mockProvider.AssertExpectations(t)
	})
}

func (m *MockProvider) FixBatch(ctx context.Context, req provider.BatchRequest) (*provider.BatchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.BatchResponse), args.Error(1)
}

func TestNewWithConfidence(t *testing.T) {
	t.Run("creates fixer with custom confidence config", func(t *testing.T) {
		mockProvider := new(MockProvider)
		tmpDir := t.TempDir()

		confidenceConf := confidence.Config{
			Enabled: true,
			Thresholds: map[string]float64{
				"medium": 0.85,
			},
			OnLowConfidence: confidence.ActionSkip,
		}

		fixer := NewWithConfidence(mockProvider, tmpDir, false, confidenceConf)

		assert.NotNil(t, fixer)
		assert.Equal(t, mockProvider, fixer.provider)
		assert.Equal(t, tmpDir, fixer.inputDir)
		assert.False(t, fixer.dryRun)
		assert.True(t, fixer.confidenceConf.Enabled)
		assert.Equal(t, confidence.ActionSkip, fixer.confidenceConf.OnLowConfidence)
	})

	t.Run("creates fixer with warn-and-apply action", func(t *testing.T) {
		mockProvider := new(MockProvider)
		tmpDir := t.TempDir()

		confidenceConf := confidence.Config{
			Enabled:         true,
			OnLowConfidence: confidence.ActionWarnAndApply,
		}

		fixer := NewWithConfidence(mockProvider, tmpDir, true, confidenceConf)

		assert.True(t, fixer.dryRun)
		assert.Equal(t, confidence.ActionWarnAndApply, fixer.confidenceConf.OnLowConfidence)
	})

	t.Run("creates fixer with manual-review-file action", func(t *testing.T) {
		mockProvider := new(MockProvider)
		tmpDir := t.TempDir()

		confidenceConf := confidence.Config{
			Enabled:         true,
			OnLowConfidence: confidence.ActionManualReviewFile,
		}

		fixer := NewWithConfidence(mockProvider, tmpDir, false, confidenceConf)

		assert.Equal(t, confidence.ActionManualReviewFile, fixer.confidenceConf.OnLowConfidence)
	})
}

func TestWriteToReviewFile(t *testing.T) {
	t.Run("creates review file with single item", func(t *testing.T) {
		tmpDir := t.TempDir()
		mockProvider := new(MockProvider)

		fixer := New(mockProvider, tmpDir, false)

		v := violation.Violation{
			ID:                  "test-001",
			Description:         "Test violation",
			Category:            "mandatory",
			Effort:              5,
			MigrationComplexity: "medium",
		}
		incident := violation.Incident{
			LineNumber: 42,
		}
		result := &FixResult{
			FilePath: "src/test.java",
		}

		err := fixer.writeToReviewFile(v, incident, result, "Confidence too low", 0.65)
		require.NoError(t, err)

		// Verify review file was created
		reviewPath := filepath.Join(tmpDir, "ReviewFileName")
		data, err := os.ReadFile(reviewPath)
		require.NoError(t, err)

		var reviews []ReviewItem
		err = yaml.Unmarshal(data, &reviews)
		require.NoError(t, err)
		require.Len(t, reviews, 1)

		// Verify review item
		assert.Equal(t, "test-001", reviews[0].ViolationID)
		assert.Equal(t, "src/test.java", reviews[0].FilePath)
		assert.Equal(t, 42, reviews[0].LineNumber)
		assert.Equal(t, "Test violation", reviews[0].Description)
		assert.Equal(t, 0.65, reviews[0].Confidence)
		assert.Equal(t, "Confidence too low", reviews[0].Reason)
		assert.Equal(t, "mandatory", reviews[0].Category)
		assert.Equal(t, 5, reviews[0].Effort)
		assert.Equal(t, "medium", reviews[0].Complexity)
	})

	t.Run("appends to existing review file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mockProvider := new(MockProvider)

		fixer := New(mockProvider, tmpDir, false)

		v1 := violation.Violation{ID: "test-001", Description: "First", Category: "mandatory"}
		incident1 := violation.Incident{LineNumber: 10}
		result1 := &FixResult{FilePath: "file1.java"}

		// Add first review
		err := fixer.writeToReviewFile(v1, incident1, result1, "Low confidence", 0.60)
		require.NoError(t, err)

		v2 := violation.Violation{ID: "test-002", Description: "Second", Category: "optional"}
		incident2 := violation.Incident{LineNumber: 20}
		result2 := &FixResult{FilePath: "file2.java"}

		// Add second review
		err = fixer.writeToReviewFile(v2, incident2, result2, "Very low confidence", 0.50)
		require.NoError(t, err)

		// Verify both reviews are in file
		reviewPath := filepath.Join(tmpDir, "ReviewFileName")
		data, err := os.ReadFile(reviewPath)
		require.NoError(t, err)

		var reviews []ReviewItem
		err = yaml.Unmarshal(data, &reviews)
		require.NoError(t, err)
		require.Len(t, reviews, 2)

		assert.Equal(t, "test-001", reviews[0].ViolationID)
		assert.Equal(t, "test-002", reviews[1].ViolationID)
	})

	t.Run("handles corrupt review file gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		mockProvider := new(MockProvider)

		fixer := New(mockProvider, tmpDir, false)

		// Write corrupt YAML to review file
		reviewPath := filepath.Join(tmpDir, "ReviewFileName")
		err := os.WriteFile(reviewPath, []byte("invalid yaml: [[[ :::"), 0644)
		require.NoError(t, err)

		v := violation.Violation{ID: "test-001", Description: "Test"}
		incident := violation.Incident{LineNumber: 10}
		result := &FixResult{FilePath: "test.java"}

		// Should handle corrupt file and create new review
		err = fixer.writeToReviewFile(v, incident, result, "Test reason", 0.70)
		require.NoError(t, err)

		// Verify file was rewritten with valid data
		data, err := os.ReadFile(reviewPath)
		require.NoError(t, err)

		var reviews []ReviewItem
		err = yaml.Unmarshal(data, &reviews)
		require.NoError(t, err)
		require.Len(t, reviews, 1)
	})

	t.Run("uses atomic write-rename pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		mockProvider := new(MockProvider)

		fixer := New(mockProvider, tmpDir, false)

		v := violation.Violation{ID: "test-001"}
		incident := violation.Incident{LineNumber: 10}
		result := &FixResult{FilePath: "test.java"}

		err := fixer.writeToReviewFile(v, incident, result, "Test", 0.70)
		require.NoError(t, err)

		// Verify temporary file was cleaned up
		tmpPath := filepath.Join(tmpDir, "ReviewFileName.tmp")
		_, err = os.Stat(tmpPath)
		assert.True(t, os.IsNotExist(err), "Temporary file should be cleaned up")

		// Verify final file exists
		reviewPath := filepath.Join(tmpDir, "ReviewFileName")
		_, err = os.Stat(reviewPath)
		assert.NoError(t, err, "Review file should exist")
	})
}

func TestNewBatchFixerWithConfidence(t *testing.T) {
	t.Run("creates batch fixer with custom confidence config", func(t *testing.T) {
		mockProvider := new(MockProvider)
		tmpDir := t.TempDir()

		batchConfig := DefaultBatchConfig()
		confidenceConf := confidence.Config{
			Enabled: true,
			Thresholds: map[string]float64{
				"high": 0.95,
			},
			OnLowConfidence: confidence.ActionSkip,
		}

		batchFixer := NewBatchFixerWithConfidence(mockProvider, tmpDir, false, batchConfig, confidenceConf)

		assert.NotNil(t, batchFixer)
		assert.Equal(t, mockProvider, batchFixer.provider)
		assert.Equal(t, tmpDir, batchFixer.inputDir)
		assert.False(t, batchFixer.dryRun)
		assert.Equal(t, batchConfig, batchFixer.config)
		assert.True(t, batchFixer.confidenceConf.Enabled)
		assert.Equal(t, confidence.ActionSkip, batchFixer.confidenceConf.OnLowConfidence)
	})

	t.Run("creates batch fixer with warn-and-apply action", func(t *testing.T) {
		mockProvider := new(MockProvider)
		tmpDir := t.TempDir()

		batchConfig := BatchConfig{
			MaxBatchSize: 10,
			Parallelism:  2,
		}
		confidenceConf := confidence.Config{
			Enabled:         true,
			OnLowConfidence: confidence.ActionWarnAndApply,
		}

		batchFixer := NewBatchFixerWithConfidence(mockProvider, tmpDir, true, batchConfig, confidenceConf)

		assert.True(t, batchFixer.dryRun)
		assert.Equal(t, 10, batchFixer.config.MaxBatchSize)
		assert.Equal(t, confidence.ActionWarnAndApply, batchFixer.confidenceConf.OnLowConfidence)
	})

	t.Run("creates batch fixer with manual-review-file action", func(t *testing.T) {
		mockProvider := new(MockProvider)
		tmpDir := t.TempDir()

		batchConfig := DefaultBatchConfig()
		confidenceConf := confidence.Config{
			Enabled:         true,
			OnLowConfidence: confidence.ActionManualReviewFile,
			Thresholds: map[string]float64{
				"trivial": 0.70,
				"low":     0.75,
				"medium":  0.80,
				"high":    0.90,
				"expert":  0.95,
			},
		}

		batchFixer := NewBatchFixerWithConfidence(mockProvider, tmpDir, false, batchConfig, confidenceConf)

		assert.Equal(t, confidence.ActionManualReviewFile, batchFixer.confidenceConf.OnLowConfidence)
		assert.Equal(t, 0.95, batchFixer.confidenceConf.Thresholds["expert"])
	})

	t.Run("respects batch config settings", func(t *testing.T) {
		mockProvider := new(MockProvider)
		tmpDir := t.TempDir()

		batchConfig := BatchConfig{
			MaxBatchSize: 5,
			Parallelism:  3,
		}
		confidenceConf := confidence.DefaultConfig()

		batchFixer := NewBatchFixerWithConfidence(mockProvider, tmpDir, false, batchConfig, confidenceConf)

		assert.Equal(t, 5, batchFixer.config.MaxBatchSize)
		assert.Equal(t, 3, batchFixer.config.Parallelism)
	})
}
