package gitutil

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tsanders/kantra-ai/pkg/fixer"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

func TestFormatPerViolationMessage(t *testing.T) {
	v := violation.Violation{
		ID:          "test-violation-001",
		Description: "Replace javax with jakarta imports",
		Category:    "mandatory",
		Effort:      1,
	}

	fixes := []FixRecord{
		{
			Violation: v,
			Incident: violation.Incident{
				URI:        "file:///src/File1.java",
				LineNumber: 10,
			},
			Result: &fixer.FixResult{
				FilePath:   "src/File1.java",
				Cost:       0.01,
				TokensUsed: 100,
			},
			Timestamp: time.Now(),
		},
		{
			Violation: v,
			Incident: violation.Incident{
				URI:        "file:///src/File2.java",
				LineNumber: 25,
			},
			Result: &fixer.FixResult{
				FilePath:   "src/File2.java",
				Cost:       0.02,
				TokensUsed: 150,
			},
			Timestamp: time.Now(),
		},
	}

	message := FormatPerViolationMessage(
		v.ID,
		v.Description,
		v.Category,
		v.Effort,
		fixes,
		"claude",
	)

	// Verify message contains all key information
	assert.Contains(t, message, "fix(konveyor): test-violation-001")
	assert.Contains(t, message, "Violation: test-violation-001")
	assert.Contains(t, message, "Description: Replace javax with jakarta imports")
	assert.Contains(t, message, "Category: mandatory")
	assert.Contains(t, message, "Effort: 1")
	assert.Contains(t, message, "Fixed Files:")
	assert.Contains(t, message, "src/File1.java:10")
	assert.Contains(t, message, "src/File2.java:25")
	assert.Contains(t, message, "Provider: claude")
	assert.Contains(t, message, "Incidents Fixed: 2")
	assert.Contains(t, message, "Total Cost: $0.0300")
	assert.Contains(t, message, "Total Tokens: 250")
}

func TestFormatPerIncidentMessage(t *testing.T) {
	message := FormatPerIncidentMessage(
		"test-violation-002",
		"Update deprecated API usage",
		"src/Service.java",
		42,
		0.015,
		125,
		"openai",
	)

	// Verify message contains all key information
	assert.Contains(t, message, "fix(konveyor): test-violation-002 in Service.java")
	assert.Contains(t, message, "Violation: test-violation-002")
	assert.Contains(t, message, "Description: Update deprecated API usage")
	assert.Contains(t, message, "File: src/Service.java")
	assert.Contains(t, message, "Line: 42")
	assert.Contains(t, message, "Provider: openai")
	assert.Contains(t, message, "Cost: $0.0150")
	assert.Contains(t, message, "Tokens: 125")
}

func TestFormatAtEndMessage(t *testing.T) {
	v1 := violation.Violation{
		ID:       "violation-001",
		Category: "mandatory",
	}
	v2 := violation.Violation{
		ID:       "violation-002",
		Category: "optional",
	}

	fixesByViolation := map[string][]FixRecord{
		"violation-001": {
			{
				Violation: v1,
				Result: &fixer.FixResult{
					FilePath:   "src/File1.java",
					Cost:       0.01,
					TokensUsed: 100,
				},
			},
			{
				Violation: v1,
				Result: &fixer.FixResult{
					FilePath:   "src/File2.java",
					Cost:       0.02,
					TokensUsed: 150,
				},
			},
		},
		"violation-002": {
			{
				Violation: v2,
				Result: &fixer.FixResult{
					FilePath:   "src/File3.java",
					Cost:       0.03,
					TokensUsed: 200,
				},
			},
		},
	}

	message := FormatAtEndMessage(fixesByViolation, "claude")

	// Verify message contains summary information
	assert.Contains(t, message, "fix(konveyor): Batch remediation of 2 violations")
	assert.Contains(t, message, "Violations Fixed:")
	assert.Contains(t, message, "violation-001 (mandatory): 2 incidents")
	assert.Contains(t, message, "violation-002 (optional): 1 incidents")
	assert.Contains(t, message, "Total Files Modified: 3")
	assert.Contains(t, message, "Provider: claude")
	assert.Contains(t, message, "Total Cost: $0.0600")
	assert.Contains(t, message, "Total Tokens: 450")
}

func TestFormatPerViolationMessage_LongDescription(t *testing.T) {
	v := violation.Violation{
		ID:          "violation-long",
		Description: strings.Repeat("A", 100), // Very long description
		Category:    "mandatory",
		Effort:      2,
	}

	fixes := []FixRecord{
		{
			Violation: v,
			Result: &fixer.FixResult{
				FilePath:   "test.java",
				Cost:       0.01,
				TokensUsed: 50,
			},
		},
	}

	message := FormatPerViolationMessage(
		v.ID,
		v.Description,
		v.Category,
		v.Effort,
		fixes,
		"claude",
	)

	// First line should truncate long description
	lines := strings.Split(message, "\n")
	assert.Contains(t, lines[0], "...")
}

func TestFormatPerViolationMessage_ZeroCostAndTokens(t *testing.T) {
	v := violation.Violation{
		ID:       "test",
		Category: "mandatory",
		Effort:   1,
	}

	fixes := []FixRecord{
		{
			Violation: v,
			Result: &fixer.FixResult{
				FilePath:   "test.java",
				Cost:       0.0,
				TokensUsed: 0,
			},
		},
	}

	message := FormatPerViolationMessage(
		v.ID,
		v.Description,
		v.Category,
		v.Effort,
		fixes,
		"claude",
	)

	// Should still format with zero values
	assert.Contains(t, message, "Total Cost: $0.0000")
	assert.Contains(t, message, "Total Tokens: 0")
}
