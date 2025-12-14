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
