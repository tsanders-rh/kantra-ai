package gitutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FormatPerViolationMessage formats a detailed commit message for a violation
func FormatPerViolationMessage(violationID, description, category string, effort int,
	fixes []FixRecord, providerName string) string {

	var sb strings.Builder

	// First line: short summary
	shortDesc := description
	if len(shortDesc) > 50 {
		shortDesc = shortDesc[:47] + "..."
	}
	sb.WriteString(fmt.Sprintf("fix(konveyor): %s - %s\n\n", violationID, shortDesc))

	// Violation details
	sb.WriteString(fmt.Sprintf("Violation: %s\n", violationID))
	sb.WriteString(fmt.Sprintf("Description: %s\n", description))
	sb.WriteString(fmt.Sprintf("Category: %s\n", category))
	sb.WriteString(fmt.Sprintf("Effort: %d\n\n", effort))

	// Fixed files
	sb.WriteString("Fixed Files:\n")
	totalCost := 0.0
	totalTokens := 0
	for _, fix := range fixes {
		sb.WriteString(fmt.Sprintf("- %s:%d\n", fix.Result.FilePath, fix.Incident.LineNumber))
		totalCost += fix.Result.Cost
		totalTokens += fix.Result.TokensUsed
	}

	// Summary stats
	sb.WriteString(fmt.Sprintf("\nProvider: %s\n", providerName))
	sb.WriteString(fmt.Sprintf("Incidents Fixed: %d\n", len(fixes)))
	sb.WriteString(fmt.Sprintf("Total Cost: $%.4f\n", totalCost))
	sb.WriteString(fmt.Sprintf("Total Tokens: %d\n", totalTokens))

	return sb.String()
}

// FormatPerIncidentMessage formats a detailed commit message for a single incident
func FormatPerIncidentMessage(violationID, description, filePath string, lineNumber int,
	cost float64, tokens int, providerName string) string {

	var sb strings.Builder

	// First line: short summary with filename
	filename := filepath.Base(filePath)
	sb.WriteString(fmt.Sprintf("fix(konveyor): %s in %s\n\n", violationID, filename))

	// Incident details
	sb.WriteString(fmt.Sprintf("Violation: %s\n", violationID))
	sb.WriteString(fmt.Sprintf("Description: %s\n", description))
	sb.WriteString(fmt.Sprintf("File: %s\n", filePath))
	sb.WriteString(fmt.Sprintf("Line: %d\n\n", lineNumber))

	// Stats
	sb.WriteString(fmt.Sprintf("Provider: %s\n", providerName))
	sb.WriteString(fmt.Sprintf("Cost: $%.4f\n", cost))
	sb.WriteString(fmt.Sprintf("Tokens: %d\n", tokens))

	return sb.String()
}

// FormatAtEndMessage formats a summary commit message for all fixes
func FormatAtEndMessage(fixesByViolation map[string][]FixRecord, providerName string) string {
	var sb strings.Builder

	// Count total incidents
	totalIncidents := 0
	for _, fixes := range fixesByViolation {
		totalIncidents += len(fixes)
	}

	// First line: summary
	sb.WriteString(fmt.Sprintf("fix(konveyor): Batch remediation of %d violations\n\n", len(fixesByViolation)))

	// List violations
	sb.WriteString("Violations Fixed:\n")
	totalCost := 0.0
	totalTokens := 0
	filesModified := make(map[string]bool)

	for violationID, fixes := range fixesByViolation {
		if len(fixes) == 0 {
			continue
		}

		// Get category from first fix
		category := fixes[0].Violation.Category
		sb.WriteString(fmt.Sprintf("- %s (%s): %d incidents\n", violationID, category, len(fixes)))

		// Accumulate stats
		for _, fix := range fixes {
			totalCost += fix.Result.Cost
			totalTokens += fix.Result.TokensUsed
			filesModified[fix.Result.FilePath] = true
		}
	}

	// Summary stats
	sb.WriteString(fmt.Sprintf("\nTotal Files Modified: %d\n", len(filesModified)))
	sb.WriteString(fmt.Sprintf("Provider: %s\n", providerName))
	sb.WriteString(fmt.Sprintf("Total Cost: $%.4f\n", totalCost))
	sb.WriteString(fmt.Sprintf("Total Tokens: %d\n", totalTokens))

	return sb.String()
}
