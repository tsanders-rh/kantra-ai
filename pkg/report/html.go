// Package report provides HTML report generation for migration plans.
package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsanders/kantra-ai/pkg/planfile"
)

// GenerateHTML creates an HTML report from a migration plan.
// The HTML file is written to the same directory as the plan file as plan.html.
func GenerateHTML(plan *planfile.Plan, planPath string) (string, error) {
	// Determine output path - save as plan.html in the same directory
	dir := filepath.Dir(planPath)
	htmlPath := filepath.Join(dir, "plan.html")

	// Create HTML file
	f, err := os.Create(htmlPath)
	if err != nil {
		return "", fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer f.Close()

	// Prepare template data
	data := prepareTemplateData(plan)

	// Execute template
	tmpl, err := template.New("plan").Funcs(templateFuncs()).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return htmlPath, nil
}

// TemplateData holds all data needed for the HTML template
type TemplateData struct {
	Plan            *planfile.Plan
	TotalIncidents  int
	TotalCost       float64
	TotalDuration   int
	CategoryCounts  map[string]int
	RiskCounts      map[string]int
	EffortDistribution map[int]int
}

// prepareTemplateData extracts summary statistics from the plan
func prepareTemplateData(plan *planfile.Plan) *TemplateData {
	data := &TemplateData{
		Plan:               plan,
		CategoryCounts:     make(map[string]int),
		RiskCounts:         make(map[string]int),
		EffortDistribution: make(map[int]int),
	}

	for _, phase := range plan.Phases {
		data.TotalCost += phase.EstimatedCost
		data.TotalDuration += phase.EstimatedDurationMinutes
		data.RiskCounts[string(phase.Risk)]++

		for _, violation := range phase.Violations {
			data.TotalIncidents += violation.IncidentCount
			data.CategoryCounts[violation.Category]++
			data.EffortDistribution[violation.Effort]++
		}
	}

	return data
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"riskColor": func(risk planfile.RiskLevel) string {
			switch risk {
			case planfile.RiskLow:
				return "#3E8635" // green
			case planfile.RiskMedium:
				return "#F0AB00" // yellow
			case planfile.RiskHigh:
				return "#C9190B" // red
			default:
				return "#6A6E73" // gray
			}
		},
		"categoryColor": func(category string) string {
			switch category {
			case "mandatory":
				return "#C9190B" // red
			case "optional":
				return "#F0AB00" // yellow
			case "potential":
				return "#2B9AF3" // blue
			default:
				return "#6A6E73" // gray
			}
		},
		"formatDiff": func(message string) template.HTML {
			return template.HTML(formatMessageAsDiff(message))
		},
		"highlightLine": func(codeSnip string, lineNumber int) template.HTML {
			return template.HTML(highlightLineInCode(codeSnip, lineNumber))
		},
	}
}

// formatMessageAsDiff parses incident messages and formats Before/After sections as colored diffs
func formatMessageAsDiff(message string) string {
	// Check if message contains Before/After pattern
	if !strings.Contains(message, "Before:") || !strings.Contains(message, "After:") {
		// No diff sections, return as plain text
		return fmt.Sprintf("<div>%s</div>", template.HTMLEscapeString(message))
	}

	// Split into parts
	parts := strings.Split(message, "Before:")
	if len(parts) < 2 {
		return fmt.Sprintf("<div>%s</div>", template.HTMLEscapeString(message))
	}

	description := strings.TrimSpace(parts[0])
	remainder := parts[1]

	// Split remainder into Before and After
	beforeAfter := strings.Split(remainder, "After:")
	if len(beforeAfter) < 2 {
		return fmt.Sprintf("<div>%s</div>", template.HTMLEscapeString(message))
	}

	beforeCode := strings.TrimSpace(beforeAfter[0])
	afterCode := strings.TrimSpace(beforeAfter[1])

	// Remove the code block markers if present
	beforeCode = strings.TrimPrefix(beforeCode, "```")
	beforeCode = strings.TrimSuffix(beforeCode, "```")
	beforeCode = strings.TrimSpace(beforeCode)

	afterCode = strings.TrimPrefix(afterCode, "```")
	afterCode = strings.TrimSuffix(afterCode, "```")
	afterCode = strings.TrimSpace(afterCode)

	// Build HTML with diff styling matching web UI
	var html strings.Builder

	if description != "" {
		html.WriteString(fmt.Sprintf("<div class='diff-description' style='margin-bottom: 12px; color: #555;'>%s</div>", template.HTMLEscapeString(description)))
	}

	html.WriteString("<div class='diff-container'>")

	// Before pane (removal - red)
	html.WriteString("<div class='diff-pane before-pane'>")
	html.WriteString("<div class='diff-header'><i class='fas fa-minus-circle'></i> Before</div>")
	html.WriteString(fmt.Sprintf("<pre><code>%s</code></pre>", template.HTMLEscapeString(beforeCode)))
	html.WriteString("</div>")

	// After pane (addition - green)
	html.WriteString("<div class='diff-pane after-pane'>")
	html.WriteString("<div class='diff-header'><i class='fas fa-plus-circle'></i> After</div>")
	html.WriteString(fmt.Sprintf("<pre><code>%s</code></pre>", template.HTMLEscapeString(afterCode)))
	html.WriteString("</div>")

	html.WriteString("</div>")

	return html.String()
}

// highlightLineInCode highlights the specific line number in a code snippet
func highlightLineInCode(codeSnip string, targetLine int) string {
	if codeSnip == "" {
		return ""
	}

	lines := strings.Split(codeSnip, "\n")
	var html strings.Builder

	html.WriteString("<pre class='code-snippet'><code>")

	for _, line := range lines {
		// Parse line number from the format " 123    content"
		// Line numbers are at the start, followed by spaces
		var lineNum int

		// Trim leading spaces and try to parse the line number
		trimmed := strings.TrimLeft(line, " ")
		if trimmed != "" {
			// Try to scan the first integer (ignore error - line may not have a number)
			_, _ = fmt.Sscanf(trimmed, "%d", &lineNum)
		}

		// Highlight if this is the target line
		if lineNum == targetLine && lineNum > 0 {
			html.WriteString(fmt.Sprintf("<span class='highlighted-line'>%s</span>\n", template.HTMLEscapeString(line)))
		} else {
			html.WriteString(fmt.Sprintf("%s\n", template.HTMLEscapeString(line)))
		}
	}

	html.WriteString("</code></pre>")
	return html.String()
}
