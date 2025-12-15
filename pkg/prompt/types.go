// Package prompt provides configurable AI prompt templates for code remediation.
package prompt

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
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
	SingleFix *Template // Base single-fix template (fallback)
	BatchFix  *Template // Base batch-fix template (fallback)
	// Language-specific template overrides
	languageTemplates map[string]*LanguageTemplates
}

// LanguageTemplates holds templates for a specific language
type LanguageTemplates struct {
	SingleFix *Template
	BatchFix  *Template
}

// Config configures prompt template loading
type Config struct {
	// Provider name (used for loading provider-specific defaults)
	Provider string
	// Custom base template paths (optional, used as fallback)
	SingleFixPath string
	BatchFixPath  string
	// Language-specific template overrides (optional)
	LanguageTemplates map[string]LanguagePaths
}

// LanguagePaths holds template paths for a specific language
type LanguagePaths struct {
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
	templates := &Templates{
		languageTemplates: make(map[string]*LanguageTemplates),
	}

	// Load base single fix template (fallback)
	if cfg.SingleFixPath != "" {
		tmpl, err := loadFromFile(cfg.SingleFixPath, "single-fix")
		if err != nil {
			return nil, fmt.Errorf("failed to load single-fix template: %w", err)
		}
		templates.SingleFix = tmpl
	} else {
		templates.SingleFix = getDefaultSingleFixTemplate(cfg.Provider)
	}

	// Load base batch fix template (fallback)
	if cfg.BatchFixPath != "" {
		tmpl, err := loadFromFile(cfg.BatchFixPath, "batch-fix")
		if err != nil {
			return nil, fmt.Errorf("failed to load batch-fix template: %w", err)
		}
		templates.BatchFix = tmpl
	} else {
		templates.BatchFix = getDefaultBatchFixTemplate(cfg.Provider)
	}

	// Compile base templates
	if err := templates.SingleFix.compile(); err != nil {
		return nil, fmt.Errorf("failed to compile single-fix template: %w", err)
	}
	if err := templates.BatchFix.compile(); err != nil {
		return nil, fmt.Errorf("failed to compile batch-fix template: %w", err)
	}

	// Load language-specific templates
	for lang, paths := range cfg.LanguageTemplates {
		langTemplates := &LanguageTemplates{}

		// Load language-specific single-fix template
		if paths.SingleFixPath != "" {
			tmpl, err := loadFromFile(paths.SingleFixPath, fmt.Sprintf("single-fix-%s", lang))
			if err != nil {
				return nil, fmt.Errorf("failed to load %s single-fix template: %w", lang, err)
			}
			if err := tmpl.compile(); err != nil {
				return nil, fmt.Errorf("failed to compile %s single-fix template: %w", lang, err)
			}
			langTemplates.SingleFix = tmpl
		}

		// Load language-specific batch-fix template
		if paths.BatchFixPath != "" {
			tmpl, err := loadFromFile(paths.BatchFixPath, fmt.Sprintf("batch-fix-%s", lang))
			if err != nil {
				return nil, fmt.Errorf("failed to load %s batch-fix template: %w", lang, err)
			}
			if err := tmpl.compile(); err != nil {
				return nil, fmt.Errorf("failed to compile %s batch-fix template: %w", lang, err)
			}
			langTemplates.BatchFix = tmpl
		}

		// Only store if at least one template was loaded
		if langTemplates.SingleFix != nil || langTemplates.BatchFix != nil {
			templates.languageTemplates[lang] = langTemplates
		}
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

// GetSingleFixTemplate returns the appropriate single-fix template for the given language
// Falls back to the base template if no language-specific template exists
func (t *Templates) GetSingleFixTemplate(language string) *Template {
	// Check for language-specific template
	if langTemplates, ok := t.languageTemplates[language]; ok {
		if langTemplates.SingleFix != nil {
			return langTemplates.SingleFix
		}
	}
	// Fall back to base template
	return t.SingleFix
}

// GetBatchFixTemplate returns the appropriate batch-fix template for the given language
// Falls back to the base template if no language-specific template exists
func (t *Templates) GetBatchFixTemplate(language string) *Template {
	// Check for language-specific template
	if langTemplates, ok := t.languageTemplates[language]; ok {
		if langTemplates.BatchFix != nil {
			return langTemplates.BatchFix
		}
	}
	// Fall back to base template
	return t.BatchFix
}
