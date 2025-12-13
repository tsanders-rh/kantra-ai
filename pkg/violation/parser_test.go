package violation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAnalysis(t *testing.T) {
	t.Run("valid YAML file", func(t *testing.T) {
		analysis, err := LoadAnalysis("testdata/valid_analysis.yaml")
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.Len(t, analysis.Violations, 3)

		// Verify first violation
		assert.Equal(t, "violation-001", analysis.Violations[0].ID)
		assert.Equal(t, "Replace javax with jakarta", analysis.Violations[0].Description)
		assert.Equal(t, "mandatory", analysis.Violations[0].Category)
		assert.Equal(t, 1, analysis.Violations[0].Effort)
		assert.Len(t, analysis.Violations[0].Incidents, 2)
	})

	t.Run("directory path with output.yaml", func(t *testing.T) {
		// Create temp directory with output.yaml
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "output.yaml")
		err := os.WriteFile(yamlPath, []byte("violations: []"), 0644)
		require.NoError(t, err)

		analysis, err := LoadAnalysis(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.Len(t, analysis.Violations, 0)
	})

	t.Run("empty violations list", func(t *testing.T) {
		analysis, err := LoadAnalysis("testdata/empty_violations.yaml")
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.Len(t, analysis.Violations, 0)
	})

	t.Run("invalid YAML syntax", func(t *testing.T) {
		_, err := LoadAnalysis("testdata/invalid_syntax.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse analysis YAML")
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadAnalysis("testdata/nonexistent.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read analysis file")
	})
}

func TestAnalysis_FilterViolations(t *testing.T) {
	// Load test data
	analysis, err := LoadAnalysis("testdata/valid_analysis.yaml")
	require.NoError(t, err)

	t.Run("no filters returns all violations", func(t *testing.T) {
		filtered := analysis.FilterViolations(nil, nil, 0)
		assert.Len(t, filtered, 3)
	})

	t.Run("filter by single violation ID", func(t *testing.T) {
		filtered := analysis.FilterViolations([]string{"violation-001"}, nil, 0)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "violation-001", filtered[0].ID)
	})

	t.Run("filter by multiple violation IDs", func(t *testing.T) {
		filtered := analysis.FilterViolations([]string{"violation-001", "violation-003"}, nil, 0)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "violation-001", filtered[0].ID)
		assert.Equal(t, "violation-003", filtered[1].ID)
	})

	t.Run("filter by category mandatory", func(t *testing.T) {
		filtered := analysis.FilterViolations(nil, []string{"mandatory"}, 0)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "mandatory", filtered[0].Category)
	})

	t.Run("filter by category optional", func(t *testing.T) {
		filtered := analysis.FilterViolations(nil, []string{"optional"}, 0)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "optional", filtered[0].Category)
	})

	t.Run("filter by multiple categories", func(t *testing.T) {
		filtered := analysis.FilterViolations(nil, []string{"mandatory", "optional"}, 0)
		assert.Len(t, filtered, 2)
	})

	t.Run("filter by max effort", func(t *testing.T) {
		filtered := analysis.FilterViolations(nil, nil, 2)
		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Effort)
	})

	t.Run("combined filters: ID and category", func(t *testing.T) {
		filtered := analysis.FilterViolations([]string{"violation-001", "violation-002"}, []string{"mandatory"}, 0)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "violation-001", filtered[0].ID)
		assert.Equal(t, "mandatory", filtered[0].Category)
	})

	t.Run("combined filters: category and max effort", func(t *testing.T) {
		filtered := analysis.FilterViolations(nil, []string{"mandatory", "optional"}, 2)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "violation-001", filtered[0].ID)
	})

	t.Run("no matching violations", func(t *testing.T) {
		filtered := analysis.FilterViolations([]string{"nonexistent-id"}, nil, 0)
		assert.Len(t, filtered, 0)
	})
}
