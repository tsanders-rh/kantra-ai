// Package violation provides types and utilities for working with Konveyor analysis results.
// It handles loading and filtering violations from output.yaml files produced by Konveyor static analysis.
package violation

// Analysis represents the root structure of Konveyor's output.yaml file.
// It contains all violations found during static analysis of an application.
type Analysis struct {
	Violations []Violation `yaml:"violations"`
}

// NativeKantraRuleset represents the native format output by 'kantra analyze'.
// The native format is an array of rulesets, each containing a map of violations.
type NativeKantraRuleset struct {
	Name        string                           `yaml:"name"`
	Description string                           `yaml:"description"`
	Violations  map[string]NativeKantraViolation `yaml:"violations"`
	Skipped     []string                         `yaml:"skipped,omitempty"`
}

// NativeKantraViolation represents a violation in the native Kantra format.
// In native format, violations are stored as a map (keyed by violation ID).
type NativeKantraViolation struct {
	Description string           `yaml:"description"`
	Category    string           `yaml:"category"`
	Effort      int              `yaml:"effort"`
	Labels      []string         `yaml:"labels,omitempty"`
	Incidents   []Incident       `yaml:"incidents"`
}

// Violation represents a specific rule that was violated in the analyzed codebase.
// Each violation can have multiple incidents (specific occurrences across different files/lines).
// Violations are categorized by severity (mandatory/optional/potential) and assigned
// effort levels (0-10) and migration complexity (trivial/low/medium/high/expert).
type Violation struct {
	ID                  string            `yaml:"id"`                                 // Unique identifier for the rule
	Description         string            `yaml:"description"`                        // Human-readable description of the violation
	Category            string            `yaml:"category"`                           // mandatory, optional, or potential
	Effort              int               `yaml:"effort"`                             // Estimated effort to fix (0-10 scale)
	MigrationComplexity string            `yaml:"migration_complexity,omitempty"`     // trivial, low, medium, high, or expert
	Incidents           []Incident        `yaml:"incidents"`                          // Specific occurrences of this violation
	RuleSet             string            `yaml:"ruleSet"`                            // Ruleset that detected this violation
	Rule                Rule              `yaml:"rule"`                               // Detailed rule information
	Labels              map[string]string `yaml:"labels,omitempty"`                   // Additional metadata labels
}

// Incident represents a specific occurrence of a violation in the codebase.
// Each incident points to a file and line number where the violation was detected,
// along with context (code snippet, variables) to aid in understanding and fixing.
type Incident struct {
	URI        string                 `yaml:"uri"`                 // File URI (e.g., file:///path/to/file.java)
	Message    string                 `yaml:"message"`             // Specific message for this incident
	CodeSnip   string                 `yaml:"codeSnip"`            // Code snippet showing context around the violation
	LineNumber int                    `yaml:"lineNumber"`          // Line number where violation occurs
	Variables  map[string]interface{} `yaml:"variables,omitempty"` // Template variables for this incident
}

// Rule contains metadata about the rule that was violated
type Rule struct {
	ID          string            `yaml:"id"`
	Message     string            `yaml:"message"`     // Explanation of what needs to change
	RuleSet     string            `yaml:"ruleSet"`
	Labels      []string          `yaml:"labels,omitempty"`
	Links       []string          `yaml:"links,omitempty"`
	Category    string            `yaml:"category,omitempty"`
}

// GetFilePath extracts the file path from a file:// URI
func (i *Incident) GetFilePath() string {
	// Remove file:// prefix
	path := i.URI
	if len(path) > 7 && path[:7] == "file://" {
		path = path[7:]
	}
	return path
}
