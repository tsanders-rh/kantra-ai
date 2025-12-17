package gitutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FormatPRTitleForViolation creates a PR title for a violation
func FormatPRTitleForViolation(violationID, description string) string {
	// Keep title concise with just the violation ID
	// Full description is included in the PR body
	return fmt.Sprintf("fix: Konveyor violation %s", violationID)
}

// FormatPRBodyForViolation creates a PR body for a violation
func FormatPRBodyForViolation(violationID, description, category string, effort int,
	fixes []FixRecord, providerName string) string {

	var sb strings.Builder

	// Calculate statistics
	totalCost := 0.0
	totalTokens := 0
	highConfidence := 0
	mediumConfidence := 0
	lowConfidence := 0
	avgConfidence := 0.0

	for _, fix := range fixes {
		totalCost += fix.Result.Cost
		totalTokens += fix.Result.TokensUsed
		avgConfidence += fix.Result.Confidence

		// Categorize confidence levels
		if fix.Result.Confidence >= 0.85 {
			highConfidence++
		} else if fix.Result.Confidence >= 0.70 {
			mediumConfidence++
		} else {
			lowConfidence++
		}
	}
	if len(fixes) > 0 {
		avgConfidence /= float64(len(fixes))
	}

	// Count unique files
	filesModified := make(map[string][]int) // map[filepath][]lineNumbers
	for _, fix := range fixes {
		filesModified[fix.Result.FilePath] = append(
			filesModified[fix.Result.FilePath],
			fix.Incident.LineNumber,
		)
	}

	// Estimate effort saved (rough: 15 min per incident of effort 1-3, 30 min for 4-5, 1hr for 6+)
	effortMinutes := 15
	if effort >= 6 {
		effortMinutes = 60
	} else if effort >= 4 {
		effortMinutes = 30
	}
	totalMinutes := effortMinutes * len(fixes)
	hoursStr := fmt.Sprintf("~%.1fh", float64(totalMinutes)/60.0)
	if totalMinutes < 60 {
		hoursStr = fmt.Sprintf("~%dm", totalMinutes)
	}

	// Header
	sb.WriteString("## 🤖 AI-Generated Migration Fixes\n\n")

	// Summary section
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("This PR remediates the Konveyor violation: **%s**\n\n", violationID))
	sb.WriteString(fmt.Sprintf("**Violation:** %s\n", violationID))
	sb.WriteString(fmt.Sprintf("**Category:** %s\n", category))
	sb.WriteString(fmt.Sprintf("**Effort:** %d\n", effort))
	sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", description))

	// Quick stats
	sb.WriteString("### Changes Summary\n\n")
	sb.WriteString(fmt.Sprintf("- 📝 **Files Modified:** %d\n", len(filesModified)))
	sb.WriteString(fmt.Sprintf("- 🔧 **Incidents Fixed:** %d\n", len(fixes)))
	sb.WriteString(fmt.Sprintf("- ⏱️  **Estimated Effort Saved:** %s\n", hoursStr))
	sb.WriteString(fmt.Sprintf("- 💰 **AI Cost:** $%.4f\n", totalCost))
	sb.WriteString(fmt.Sprintf("- 🎫 **Tokens Used:** %s\n\n", formatTokens(totalTokens)))

	// Confidence section
	sb.WriteString("### Confidence Assessment\n\n")
	sb.WriteString(fmt.Sprintf("- **Average Confidence:** %.0f%%\n", avgConfidence*100))
	sb.WriteString(fmt.Sprintf("- ✅ **High Confidence** (≥85%%): %d fix(es)\n", highConfidence))
	if mediumConfidence > 0 {
		sb.WriteString(fmt.Sprintf("- ⚠️ **Medium Confidence** (70-84%%): %d fix(es)\n", mediumConfidence))
	}
	if lowConfidence > 0 {
		sb.WriteString(fmt.Sprintf("- 🔴 **Low Confidence** (<70%%): %d fix(es) - **Review Carefully**\n", lowConfidence))
	}
	sb.WriteString("\n")

	// Detailed breakdown
	sb.WriteString("<details>\n")
	sb.WriteString("<summary>📊 Detailed Fix Breakdown</summary>\n\n")
	sb.WriteString("| File | Line(s) | Confidence | Status |\n")
	sb.WriteString("|------|---------|------------|--------|\n")

	for file, lines := range filesModified {
		// Get confidence for this file (use first incident's confidence if multiple)
		fileConfidence := 0.0
		for _, fix := range fixes {
			if fix.Result.FilePath == file {
				fileConfidence = fix.Result.Confidence
				break
			}
		}

		linesStr := ""
		if len(lines) == 1 {
			linesStr = fmt.Sprintf("%d", lines[0])
		} else {
			// Show first few lines
			if len(lines) <= 3 {
				lineStrs := make([]string, len(lines))
				for i, line := range lines {
					lineStrs[i] = fmt.Sprintf("%d", line)
				}
				linesStr = strings.Join(lineStrs, ", ")
			} else {
				linesStr = fmt.Sprintf("%d, %d, ... +%d more", lines[0], lines[1], len(lines)-2)
			}
		}

		confidenceStr := fmt.Sprintf("%.0f%%", fileConfidence*100)
		statusIcon := "✅"
		if fileConfidence < 0.70 {
			statusIcon = "🔴"
		} else if fileConfidence < 0.85 {
			statusIcon = "⚠️"
		}

		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n", file, linesStr, confidenceStr, statusIcon))
	}

	sb.WriteString("\n</details>\n\n")

	// Review checklist
	sb.WriteString("### Review Checklist\n\n")
	sb.WriteString("- [ ] Verify fixes are semantically correct\n")
	sb.WriteString("- [ ] Check for any unintended side effects\n")
	if lowConfidence > 0 {
		sb.WriteString(fmt.Sprintf("- [ ] **Carefully review %d low-confidence fix(es)**\n", lowConfidence))
	}
	sb.WriteString("- [ ] Confirm tests pass after merge\n")
	sb.WriteString("- [ ] Update documentation if needed\n\n")

	// AI details section
	sb.WriteString("---\n\n")
	sb.WriteString("<details>\n")
	sb.WriteString("<summary>🔧 AI Remediation Details</summary>\n\n")
	sb.WriteString(fmt.Sprintf("- **Provider:** %s\n", providerName))
	sb.WriteString(fmt.Sprintf("- **Total Cost:** $%.4f\n", totalCost))
	sb.WriteString(fmt.Sprintf("- **Total Tokens:** %s\n", formatTokens(totalTokens)))
	sb.WriteString(fmt.Sprintf("- **Average Confidence:** %.2f\n", avgConfidence))
	sb.WriteString("\n</details>\n\n")

	// Footer
	sb.WriteString("---\n")
	sb.WriteString("*🤖 Generated by [kantra-ai](https://github.com/tsanders-rh/kantra-ai)*\n")

	return sb.String()
}

// formatTokens formats token count with thousands separator
func formatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	return fmt.Sprintf("%d,%03d", tokens/1000, tokens%1000)
}

// FormatPRTitleForIncident creates a PR title for a single incident
func FormatPRTitleForIncident(violationID, description, filename string) string {
	return fmt.Sprintf("fix: %s in %s", violationID, filename)
}

// FormatPRBodyForIncident creates a PR body for a single incident
func FormatPRBodyForIncident(violationID, description, filePath string, lineNumber int,
	cost float64, tokens int, providerName string) string {

	var sb strings.Builder

	// Summary section
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("This PR fixes a Konveyor violation in `%s`\n\n", filepath.Base(filePath)))

	// Incident details
	sb.WriteString(fmt.Sprintf("**Violation:** %s\n", violationID))
	sb.WriteString(fmt.Sprintf("**Description:** %s\n", description))
	sb.WriteString(fmt.Sprintf("**File:** `%s`\n", filePath))
	sb.WriteString(fmt.Sprintf("**Line:** %d\n\n", lineNumber))

	// AI details section
	sb.WriteString("## AI Remediation Details\n\n")
	sb.WriteString(fmt.Sprintf("- **Provider:** %s\n", providerName))
	sb.WriteString(fmt.Sprintf("- **Cost:** $%.4f\n", cost))
	sb.WriteString(fmt.Sprintf("- **Tokens:** %d\n\n", tokens))

	// Footer
	sb.WriteString("---\n")
	sb.WriteString("*This PR was automatically generated by [kantra-ai](https://github.com/tsanders-rh/kantra-ai)*\n")

	return sb.String()
}

// FormatPRTitleAtEnd creates a PR title for batch remediation
func FormatPRTitleAtEnd(violationCount int) string {
	if violationCount == 1 {
		return "fix: Konveyor violation remediation"
	}
	return fmt.Sprintf("fix: Konveyor batch remediation (%d violations)", violationCount)
}

// FormatPRBodyAtEnd creates a PR body for batch remediation
func FormatPRBodyAtEnd(fixesByViolation map[string][]FixRecord, providerName string) string {
	var sb strings.Builder

	// Calculate statistics
	totalIncidents := 0
	totalCost := 0.0
	totalTokens := 0
	allFilesModified := make(map[string]bool)
	highConfidence := 0
	mediumConfidence := 0
	lowConfidence := 0
	totalConfidence := 0.0

	for _, fixes := range fixesByViolation {
		for _, fix := range fixes {
			totalIncidents++
			totalCost += fix.Result.Cost
			totalTokens += fix.Result.TokensUsed
			allFilesModified[fix.Result.FilePath] = true
			totalConfidence += fix.Result.Confidence

			if fix.Result.Confidence >= 0.85 {
				highConfidence++
			} else if fix.Result.Confidence >= 0.70 {
				mediumConfidence++
			} else {
				lowConfidence++
			}
		}
	}

	avgConfidence := 0.0
	if totalIncidents > 0 {
		avgConfidence = totalConfidence / float64(totalIncidents)
	}

	// Estimate effort saved
	totalMinutes := totalIncidents * 20 // Rough average: 20 min per incident
	hoursStr := fmt.Sprintf("~%.1fh", float64(totalMinutes)/60.0)
	if totalMinutes < 60 {
		hoursStr = fmt.Sprintf("~%dm", totalMinutes)
	}

	// Header
	sb.WriteString("## 🤖 AI-Generated Migration Fixes\n\n")

	// Summary section
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("This PR remediates **%d** Konveyor violation(s) with **%d** total incident(s) fixed.\n\n",
		len(fixesByViolation), totalIncidents))

	// Quick stats
	sb.WriteString("### Changes Summary\n\n")
	sb.WriteString(fmt.Sprintf("- 📝 **Files Modified:** %d\n", len(allFilesModified)))
	sb.WriteString(fmt.Sprintf("- 🔧 **Incidents Fixed:** %d\n", totalIncidents))
	sb.WriteString(fmt.Sprintf("- 🎯 **Violations Remediated:** %d\n", len(fixesByViolation)))
	sb.WriteString(fmt.Sprintf("- ⏱️  **Estimated Effort Saved:** %s\n", hoursStr))
	sb.WriteString(fmt.Sprintf("- 💰 **AI Cost:** $%.4f\n", totalCost))
	sb.WriteString(fmt.Sprintf("- 🎫 **Tokens Used:** %s\n\n", formatTokens(totalTokens)))

	// Confidence section
	sb.WriteString("### Confidence Assessment\n\n")
	sb.WriteString(fmt.Sprintf("- **Average Confidence:** %.0f%%\n", avgConfidence*100))
	sb.WriteString(fmt.Sprintf("- ✅ **High Confidence** (≥85%%): %d fix(es)\n", highConfidence))
	if mediumConfidence > 0 {
		sb.WriteString(fmt.Sprintf("- ⚠️ **Medium Confidence** (70-84%%): %d fix(es)\n", mediumConfidence))
	}
	if lowConfidence > 0 {
		sb.WriteString(fmt.Sprintf("- 🔴 **Low Confidence** (<70%%): %d fix(es) - **Review Carefully**\n", lowConfidence))
	}
	sb.WriteString("\n")

	// Violations section
	sb.WriteString("### Violations Fixed\n\n")

	for violationID, fixes := range fixesByViolation {
		if len(fixes) == 0 {
			continue
		}

		// Get category from first fix
		category := fixes[0].Violation.Category
		description := fixes[0].Violation.Description
		effort := fixes[0].Violation.Effort

		// Calculate violation-specific confidence
		violationConfidence := 0.0
		for _, fix := range fixes {
			violationConfidence += fix.Result.Confidence
		}
		violationConfidence /= float64(len(fixes))

		// Truncate long descriptions for list view
		shortDesc := description
		if len(shortDesc) > 80 {
			shortDesc = shortDesc[:77] + "..."
		}

		sb.WriteString(fmt.Sprintf("#### %s\n\n", violationID))
		sb.WriteString(fmt.Sprintf("- **Category:** %s | **Effort:** %d | **Confidence:** %.0f%%\n", category, effort, violationConfidence*100))
		sb.WriteString(fmt.Sprintf("- **Description:** %s\n", shortDesc))
		sb.WriteString(fmt.Sprintf("- **Incidents Fixed:** %d\n", len(fixes)))

		// List affected files for this violation
		filesForViolation := make(map[string]bool)
		for _, fix := range fixes {
			filesForViolation[fix.Result.FilePath] = true
		}

		if len(filesForViolation) <= 5 {
			sb.WriteString("- **Files:**\n")
			for file := range filesForViolation {
				sb.WriteString(fmt.Sprintf("  - `%s`\n", file))
			}
		} else {
			sb.WriteString(fmt.Sprintf("- **Files:** %d files modified\n", len(filesForViolation)))
		}
		sb.WriteString("\n")
	}

	// Review checklist
	sb.WriteString("### Review Checklist\n\n")
	sb.WriteString("- [ ] Verify fixes are semantically correct across all files\n")
	sb.WriteString("- [ ] Check for any unintended side effects\n")
	if lowConfidence > 0 {
		sb.WriteString(fmt.Sprintf("- [ ] **Carefully review %d low-confidence fix(es)**\n", lowConfidence))
	}
	sb.WriteString("- [ ] Run full test suite to verify nothing breaks\n")
	sb.WriteString("- [ ] Update documentation to reflect migration changes\n\n")

	// AI details section
	sb.WriteString("---\n\n")
	sb.WriteString("<details>\n")
	sb.WriteString("<summary>🔧 AI Remediation Details</summary>\n\n")
	sb.WriteString(fmt.Sprintf("- **Provider:** %s\n", providerName))
	sb.WriteString(fmt.Sprintf("- **Total Files Modified:** %d\n", len(allFilesModified)))
	sb.WriteString(fmt.Sprintf("- **Total Cost:** $%.4f\n", totalCost))
	sb.WriteString(fmt.Sprintf("- **Total Tokens:** %s\n", formatTokens(totalTokens)))
	sb.WriteString(fmt.Sprintf("- **Average Confidence:** %.2f\n", avgConfidence))
	sb.WriteString("\n</details>\n\n")

	// Footer
	sb.WriteString("---\n")
	sb.WriteString("*🤖 Generated by [kantra-ai](https://github.com/tsanders-rh/kantra-ai)*\n")

	return sb.String()
}
