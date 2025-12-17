package gitutil

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestFormatPRTitleForViolation(t *testing.T) {
	tests := []struct {
		name        string
		violationID string
		description string
		want        string
	}{
		{
			name:        "short description",
			violationID: "test-001",
			description: "Short desc",
			want:        "fix: Konveyor violation test-001",
		},
		{
			name:        "long description",
			violationID: "javax-to-jakarta-001",
			description: "This is a very long description that should be truncated but actually isn't used",
			want:        "fix: Konveyor violation javax-to-jakarta-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPRTitleForViolation(tt.violationID, tt.description)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPRBodyForViolation(t *testing.T) {
	t.Run("single incident", func(t *testing.T) {
		fixes := []FixRecord{
			{
				Violation: violation.Violation{
					ID:          "test-001",
					Description: "Test violation",
					Category:    "mandatory",
					Effort:      1,
				},
				Incident: violation.Incident{
					LineNumber: 10,
				},
				Result: &fixer.FixResult{
					FilePath:   "src/Test.java",
					Cost:       0.05,
					TokensUsed: 100,
				},
				Timestamp: time.Now(),
			},
		}

		body := FormatPRBodyForViolation("test-001", "Test violation", "mandatory", 1, fixes, "claude")

		// Verify key sections are present
		assert.Contains(t, body, "### Summary")
		assert.Contains(t, body, "test-001")
		assert.Contains(t, body, "Test violation")
		assert.Contains(t, body, "mandatory")
		assert.Contains(t, body, "Effort:** 1")
		assert.Contains(t, body, "### Changes Summary")
		assert.Contains(t, body, "src/Test.java") // File is now in table format, not "file:line"
		assert.Contains(t, body, "🔧 AI Remediation Details") // Now in collapsed details
		assert.Contains(t, body, "claude")
		assert.Contains(t, body, "$0.0500")
		assert.Contains(t, body, "100")
		assert.Contains(t, body, "kantra-ai")
		assert.Contains(t, body, "### Confidence Assessment") // New section
		assert.Contains(t, body, "### Review Checklist")      // New section
	})

	t.Run("multiple incidents same file", func(t *testing.T) {
		fixes := []FixRecord{
			{
				Violation: violation.Violation{
					ID:          "test-002",
					Description: "Multiple fixes",
					Category:    "optional",
					Effort:      2,
				},
				Incident: violation.Incident{
					LineNumber: 10,
				},
				Result: &fixer.FixResult{
					FilePath:   "src/Main.java",
					Cost:       0.03,
					TokensUsed: 50,
				},
			},
			{
				Violation: violation.Violation{
					ID:          "test-002",
					Description: "Multiple fixes",
					Category:    "optional",
					Effort:      2,
				},
				Incident: violation.Incident{
					LineNumber: 20,
				},
				Result: &fixer.FixResult{
					FilePath:   "src/Main.java",
					Cost:       0.04,
					TokensUsed: 60,
				},
			},
		}

		body := FormatPRBodyForViolation("test-002", "Multiple fixes", "optional", 2, fixes, "openai")

		// Verify aggregation
		assert.Contains(t, body, "**Incidents Fixed:** 2")
		assert.Contains(t, body, "**Files Modified:** 1")
		assert.Contains(t, body, "10, 20") // Lines in table format
		assert.Contains(t, body, "$0.0700") // 0.03 + 0.04
		assert.Contains(t, body, "110")     // 50 + 60
	})

	t.Run("multiple files", func(t *testing.T) {
		fixes := []FixRecord{
			{
				Violation: violation.Violation{ID: "v1"},
				Incident:  violation.Incident{LineNumber: 5},
				Result:    &fixer.FixResult{FilePath: "a.java", Cost: 0.01, TokensUsed: 10},
			},
			{
				Violation: violation.Violation{ID: "v1"},
				Incident:  violation.Incident{LineNumber: 15},
				Result:    &fixer.FixResult{FilePath: "b.java", Cost: 0.02, TokensUsed: 20},
			},
		}

		body := FormatPRBodyForViolation("v1", "desc", "mandatory", 1, fixes, "claude")

		assert.Contains(t, body, "**Incidents Fixed:** 2")
		assert.Contains(t, body, "**Files Modified:** 2")
		assert.Contains(t, body, "a.java") // File in table
		assert.Contains(t, body, "b.java") // File in table
	})
}

func TestFormatPRTitleForIncident(t *testing.T) {
	tests := []struct {
		name        string
		violationID string
		description string
		filename    string
		want        string
	}{
		{
			name:        "basic incident",
			violationID: "test-001",
			description: "Fix this",
			filename:    "Test.java",
			want:        "fix: test-001 in Test.java",
		},
		{
			name:        "long filename",
			violationID: "v-123",
			description: "Description",
			filename:    "VeryLongFileName.java",
			want:        "fix: v-123 in VeryLongFileName.java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPRTitleForIncident(tt.violationID, tt.description, tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPRBodyForIncident(t *testing.T) {
	body := FormatPRBodyForIncident(
		"test-001",
		"Test violation description",
		"src/main/java/Test.java",
		42,
		0.123,
		456,
		"claude",
	)

	// Verify all key elements are present
	assert.Contains(t, body, "## Summary")
	assert.Contains(t, body, "Test.java")
	assert.Contains(t, body, "test-001")
	assert.Contains(t, body, "Test violation description")
	assert.Contains(t, body, "src/main/java/Test.java")
	assert.Contains(t, body, "Line:** 42")
	assert.Contains(t, body, "claude")
	assert.Contains(t, body, "$0.1230")
	assert.Contains(t, body, "456")
	assert.Contains(t, body, "kantra-ai")
}

func TestFormatPRTitleAtEnd(t *testing.T) {
	tests := []struct {
		name           string
		violationCount int
		want           string
	}{
		{
			name:           "single violation",
			violationCount: 1,
			want:           "fix: Konveyor violation remediation",
		},
		{
			name:           "multiple violations",
			violationCount: 5,
			want:           "fix: Konveyor batch remediation (5 violations)",
		},
		{
			name:           "many violations",
			violationCount: 20,
			want:           "fix: Konveyor batch remediation (20 violations)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPRTitleAtEnd(tt.violationCount)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPRBodyAtEnd(t *testing.T) {
	t.Run("single violation", func(t *testing.T) {
		fixesByViolation := map[string][]FixRecord{
			"v1": {
				{
					Violation: violation.Violation{
						ID:          "v1",
						Description: "First violation",
						Category:    "mandatory",
						Effort:      1,
					},
					Incident: violation.Incident{LineNumber: 10},
					Result: &fixer.FixResult{
						FilePath:   "file1.java",
						Cost:       0.05,
						TokensUsed: 100,
					},
				},
			},
		}

		body := FormatPRBodyAtEnd(fixesByViolation, "claude")

		assert.Contains(t, body, "## Summary")
		assert.Contains(t, body, "1** Konveyor violation")
		assert.Contains(t, body, "1** total incident")
		assert.Contains(t, body, "## Violations Fixed")
		assert.Contains(t, body, "### v1")
		assert.Contains(t, body, "First violation")
		assert.Contains(t, body, "mandatory")
		assert.Contains(t, body, "file1.java")
		assert.Contains(t, body, "$0.0500")
		assert.Contains(t, body, "100")
	})

	t.Run("multiple violations", func(t *testing.T) {
		fixesByViolation := map[string][]FixRecord{
			"v1": {
				{
					Violation: violation.Violation{
						ID:          "v1",
						Description: "First violation",
						Category:    "mandatory",
						Effort:      1,
					},
					Result: &fixer.FixResult{
						FilePath:   "file1.java",
						Cost:       0.05,
						TokensUsed: 100,
					},
				},
				{
					Violation: violation.Violation{
						ID:          "v1",
						Description: "First violation",
						Category:    "mandatory",
						Effort:      1,
					},
					Result: &fixer.FixResult{
						FilePath:   "file2.java",
						Cost:       0.03,
						TokensUsed: 50,
					},
				},
			},
			"v2": {
				{
					Violation: violation.Violation{
						ID:          "v2",
						Description: "Second violation",
						Category:    "optional",
						Effort:      2,
					},
					Result: &fixer.FixResult{
						FilePath:   "file3.java",
						Cost:       0.10,
						TokensUsed: 200,
					},
				},
			},
		}

		body := FormatPRBodyAtEnd(fixesByViolation, "openai")

		// Verify summary
		assert.Contains(t, body, "2** Konveyor violation")
		assert.Contains(t, body, "3** total incident")

		// Verify both violations are listed
		assert.Contains(t, body, "### v1")
		assert.Contains(t, body, "### v2")
		assert.Contains(t, body, "First violation")
		assert.Contains(t, body, "Second violation")

		// Verify totals
		assert.Contains(t, body, "3**") // 3 files modified
		assert.Contains(t, body, "$0.1800") // 0.05 + 0.03 + 0.10
		assert.Contains(t, body, "350")     // 100 + 50 + 200
	})

	t.Run("long description truncation", func(t *testing.T) {
		longDesc := strings.Repeat("a", 100)
		fixesByViolation := map[string][]FixRecord{
			"v1": {
				{
					Violation: violation.Violation{
						ID:          "v1",
						Description: longDesc,
						Category:    "mandatory",
					},
					Result: &fixer.FixResult{
						FilePath:   "file.java",
						Cost:       0.01,
						TokensUsed: 10,
					},
				},
			},
		}

		body := FormatPRBodyAtEnd(fixesByViolation, "claude")

		// Verify truncation
		assert.Contains(t, body, "...")
		// Full description should not be present
		assert.NotContains(t, body, longDesc)
	})

	t.Run("many files shows count", func(t *testing.T) {
		fixes := []FixRecord{}
		for i := 0; i < 10; i++ {
			fixes = append(fixes, FixRecord{
				Violation: violation.Violation{
					ID:          "v1",
					Description: "Test",
					Category:    "mandatory",
				},
				Result: &fixer.FixResult{
					FilePath:   "file" + string(rune(i)) + ".java",
					Cost:       0.01,
					TokensUsed: 10,
				},
			})
		}

		fixesByViolation := map[string][]FixRecord{"v1": fixes}
		body := FormatPRBodyAtEnd(fixesByViolation, "claude")

		// Should show count instead of listing all files
		assert.Contains(t, body, "10 files modified")
	})
}

func TestFormatPRTitleForPhase(t *testing.T) {
	tests := []struct {
		name           string
		phaseID        string
		violationCount int
		want           string
	}{
		{
			name:           "single violation",
			phaseID:        "phase-1",
			violationCount: 1,
			want:           "fix: Konveyor phase phase-1",
		},
		{
			name:           "multiple violations",
			phaseID:        "phase-2",
			violationCount: 3,
			want:           "fix: Konveyor phase phase-2 (3 violations)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPRTitleForPhase(tt.phaseID, tt.violationCount)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPRBodyForPhase(t *testing.T) {
	t.Run("single violation", func(t *testing.T) {
		fixesByViolation := map[string][]FixRecord{
			"v1": {
				{
					Violation: violation.Violation{
						ID:          "v1",
						Description: "Test violation",
						Category:    "mandatory",
						Effort:      2,
					},
					Incident: violation.Incident{LineNumber: 10},
					Result: &fixer.FixResult{
						FilePath:   "file1.java",
						Cost:       0.05,
						TokensUsed: 100,
						Confidence: 0.9,
					},
					PhaseID: "phase-1",
				},
			},
		}

		body := FormatPRBodyForPhase("phase-1", fixesByViolation, "claude")

		assert.Contains(t, body, "Phase phase-1")
		assert.Contains(t, body, "1** violation(s)")
		assert.Contains(t, body, "1** total incident(s)")
		assert.Contains(t, body, "v1")
		assert.Contains(t, body, "Test violation")
		assert.Contains(t, body, "mandatory")
		assert.Contains(t, body, "file1.java")
		assert.Contains(t, body, "$0.0500")
		assert.Contains(t, body, "100")
		assert.Contains(t, body, "### Confidence Assessment")
		assert.Contains(t, body, "### Review Checklist")
	})

	t.Run("multiple violations", func(t *testing.T) {
		fixesByViolation := map[string][]FixRecord{
			"v1": {
				{
					Violation: violation.Violation{
						ID:          "v1",
						Description: "First violation",
						Category:    "mandatory",
						Effort:      1,
					},
					Result: &fixer.FixResult{
						FilePath:   "file1.java",
						Cost:       0.05,
						TokensUsed: 100,
						Confidence: 0.95,
					},
					PhaseID: "phase-1",
				},
				{
					Violation: violation.Violation{
						ID:          "v1",
						Description: "First violation",
						Category:    "mandatory",
						Effort:      1,
					},
					Result: &fixer.FixResult{
						FilePath:   "file2.java",
						Cost:       0.03,
						TokensUsed: 50,
						Confidence: 0.92,
					},
					PhaseID: "phase-1",
				},
			},
			"v2": {
				{
					Violation: violation.Violation{
						ID:          "v2",
						Description: "Second violation",
						Category:    "optional",
						Effort:      3,
					},
					Result: &fixer.FixResult{
						FilePath:   "file3.java",
						Cost:       0.10,
						TokensUsed: 200,
						Confidence: 0.85,
					},
					PhaseID: "phase-1",
				},
			},
		}

		body := FormatPRBodyForPhase("phase-1", fixesByViolation, "openai")

		// Verify summary
		assert.Contains(t, body, "Phase phase-1")
		assert.Contains(t, body, "2** violation(s)")
		assert.Contains(t, body, "3** total incident(s)")

		// Verify both violations are listed
		assert.Contains(t, body, "#### v1")
		assert.Contains(t, body, "#### v2")
		assert.Contains(t, body, "First violation")
		assert.Contains(t, body, "Second violation")

		// Verify totals
		assert.Contains(t, body, "3**") // 3 files modified
		assert.Contains(t, body, "$0.1800") // 0.05 + 0.03 + 0.10
		assert.Contains(t, body, "350")     // 100 + 50 + 200

		// Verify phase ID in details
		assert.Contains(t, body, "- **Phase:** phase-1")
	})
}
