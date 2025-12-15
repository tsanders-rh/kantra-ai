// Package prompt provides configurable AI prompt templates for code remediation.
package prompt

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// TemplateType identifies different prompt templates
type TemplateType string

const (
	// SingleFixTemplate is for fixing a single incident
	SingleFixTemplate TemplateType = "single-fix"
	// BatchFixTemplate is for fixing multiple incidents together
	BatchFixTemplate TemplateType = "batch-fix"
)

// Template holds a prompt template and can render it with data
type Template struct {
	Name     string
	Content  string
	compiled *template.Template
}

// Templates holds all prompt templates for a provider
type Templates struct {
	SingleFix *Template
	BatchFix  *Template
}

// Config configures prompt template loading
type Config struct {
	// Provider name (used for loading provider-specific defaults)
	Provider string
	// Custom template paths (optional)
	SingleFixPath string
	BatchFixPath  string
}

// SingleFixData contains all data needed to render a single fix prompt
type SingleFixData struct {
	Category       string
	Description    string
	RuleID         string
	RuleMessage    string
	File           string
	Line           int
	CodeSnippet    string
	FileContent    string
	Language       string
	IncidentMessage string
}

// BatchFixData contains all data needed to render a batch fix prompt
type BatchFixData struct {
	ViolationID    string
	Description    string
	IncidentCount  int
	Incidents      []BatchIncident
	Language       string
}

// BatchIncident represents a single incident in batch processing
type BatchIncident struct {
	Index       int    // 1-based index
	File        string
	Line        int
	Message     string
	CodeContext string
}

// Load loads templates based on the configuration
func Load(cfg Config) (*Templates, error) {
	templates := &Templates{}

	// Load single fix template
	if cfg.SingleFixPath != "" {
		tmpl, err := loadFromFile(cfg.SingleFixPath, "single-fix")
		if err != nil {
			return nil, fmt.Errorf("failed to load single-fix template: %w", err)
		}
		templates.SingleFix = tmpl
	} else {
		templates.SingleFix = getDefaultSingleFixTemplate(cfg.Provider)
	}

	// Load batch fix template
	if cfg.BatchFixPath != "" {
		tmpl, err := loadFromFile(cfg.BatchFixPath, "batch-fix")
		if err != nil {
			return nil, fmt.Errorf("failed to load batch-fix template: %w", err)
		}
		templates.BatchFix = tmpl
	} else {
		templates.BatchFix = getDefaultBatchFixTemplate(cfg.Provider)
	}

	// Compile templates
	if err := templates.SingleFix.compile(); err != nil {
		return nil, fmt.Errorf("failed to compile single-fix template: %w", err)
	}
	if err := templates.BatchFix.compile(); err != nil {
		return nil, fmt.Errorf("failed to compile batch-fix template: %w", err)
	}

	return templates, nil
}

// loadFromFile loads a template from a file
func loadFromFile(path string, name string) (*Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	return &Template{
		Name:    name,
		Content: string(content),
	}, nil
}

// compile compiles the template for rendering
func (t *Template) compile() error {
	tmpl, err := template.New(t.Name).Parse(t.Content)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	t.compiled = tmpl
	return nil
}

// RenderSingleFix renders a single fix prompt with the given data
func (t *Template) RenderSingleFix(data SingleFixData) (string, error) {
	if t.compiled == nil {
		return "", fmt.Errorf("template not compiled")
	}

	var buf bytes.Buffer
	if err := t.compiled.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderBatchFix renders a batch fix prompt with the given data
func (t *Template) RenderBatchFix(data BatchFixData) (string, error) {
	if t.compiled == nil {
		return "", fmt.Errorf("template not compiled")
	}

	var buf bytes.Buffer
	if err := t.compiled.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// BuildSingleFixData constructs template data from a FixRequest
func BuildSingleFixData(req provider.FixRequest) SingleFixData {
	return SingleFixData{
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
func BuildBatchFixData(req provider.BatchRequest) BatchFixData {
	incidents := make([]BatchIncident, len(req.Incidents))

	for i, incident := range req.Incidents {
		filePath := incident.GetFilePath()

		// Build code context (5 lines before/after)
		codeContext := ""
		if content, ok := req.FileContents[filePath]; ok {
			codeContext = buildCodeContext(content, incident.LineNumber, req.Language)
		}

		incidents[i] = BatchIncident{
			Index:       i + 1,
			File:        filePath,
			Line:        incident.LineNumber,
			Message:     incident.Message,
			CodeContext: codeContext,
		}
	}

	return BatchFixData{
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

// Helper to convert violation.Incident to work with existing code
func incidentGetFilePath(incident violation.Incident) string {
	return incident.GetFilePath()
}
