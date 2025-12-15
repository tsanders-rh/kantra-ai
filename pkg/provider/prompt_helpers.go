package provider

import (
	"bytes"

	"github.com/tsanders/kantra-ai/pkg/prompt"
)

// BuildSingleFixData constructs template data from a FixRequest
func BuildSingleFixData(req FixRequest) prompt.SingleFixData {
	return prompt.SingleFixData{
		Category:        req.Violation.Category,
		Description:     req.Violation.Description,
		RuleID:          req.Violation.Rule.ID,
		RuleMessage:     req.Violation.Rule.Message,
		File:            req.Incident.URI,
		Line:            req.Incident.LineNumber,
		CodeSnippet:     req.Incident.CodeSnip,
		FileContent:     req.FileContent,
		Language:        req.Language,
		IncidentMessage: req.Incident.Message,
	}
}

// BuildBatchFixData constructs template data from a BatchRequest
func BuildBatchFixData(req BatchRequest) prompt.BatchFixData {
	incidents := make([]prompt.BatchIncident, len(req.Incidents))

	for i, incident := range req.Incidents {
		filePath := incident.GetFilePath()

		// Build code context (5 lines before/after)
		codeContext := ""
		if content, ok := req.FileContents[filePath]; ok {
			codeContext = buildCodeContext(content, incident.LineNumber, req.Language)
		}

		incidents[i] = prompt.BatchIncident{
			Index:       i + 1,
			File:        filePath,
			Line:        incident.LineNumber,
			Message:     incident.Message,
			CodeContext: codeContext,
		}
	}

	return prompt.BatchFixData{
		ViolationID:   req.Violation.ID,
		Description:   req.Violation.Description,
		IncidentCount: len(req.Incidents),
		Incidents:     incidents,
		Language:      req.Language,
	}
}

// buildCodeContext extracts code context around a line number
func buildCodeContext(content string, lineNumber int, language string) string {
	lines := splitLines(content)
	start := max(0, lineNumber-5)
	end := min(len(lines), lineNumber+5)

	var buf bytes.Buffer
	buf.WriteString("```")
	buf.WriteString(language)
	buf.WriteString("\n")

	for i := start; i < end; i++ {
		if i+1 == lineNumber {
			buf.WriteString(">>> ") // Mark the problematic line
		}
		buf.WriteString(lines[i])
		buf.WriteString("\n")
	}

	buf.WriteString("```")
	return buf.String()
}

// splitLines splits content into lines
func splitLines(content string) []string {
	var lines []string
	start := 0

	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}

	if start < len(content) {
		lines = append(lines, content[start:])
	}

	return lines
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
