package violation

// Analysis represents the root structure of kantra output.yaml
type Analysis struct {
	Violations []Violation `yaml:"violations"`
}

// Violation represents a rule that was broken
type Violation struct {
	ID                  string            `yaml:"id"`
	Description         string            `yaml:"description"`
	Category            string            `yaml:"category"` // mandatory, optional, potential
	Effort              int               `yaml:"effort"`
	MigrationComplexity string            `yaml:"migration_complexity,omitempty"` // trivial, low, medium, high, expert
	Incidents           []Incident        `yaml:"incidents"`
	RuleSet             string            `yaml:"ruleSet"`
	Rule                Rule              `yaml:"rule"`
	Labels              map[string]string `yaml:"labels,omitempty"`
}

// Incident represents a specific occurrence of a violation
type Incident struct {
	URI        string `yaml:"uri"`        // file:///path/to/file.java
	Message    string `yaml:"message"`    // Specific message for this incident
	CodeSnip   string `yaml:"codeSnip"`   // Code snippet showing context
	LineNumber int    `yaml:"lineNumber"` // Line where violation occurs
	Variables  map[string]interface{} `yaml:"variables,omitempty"`
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
