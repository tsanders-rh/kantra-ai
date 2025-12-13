package violation

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadAnalysis loads and parses a Konveyor output.yaml file
func LoadAnalysis(analysisPath string) (*Analysis, error) {
	// Check if path is a directory (contains output.yaml) or direct file path
	path := analysisPath
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		path = filepath.Join(path, "output.yaml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read analysis file: %w", err)
	}

	var analysis Analysis
	if err := yaml.Unmarshal(data, &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis YAML: %w", err)
	}

	return &analysis, nil
}

// FilterViolations filters violations based on criteria
func (a *Analysis) FilterViolations(violationIDs []string, categories []string, maxEffort int) []Violation {
	if len(violationIDs) == 0 && len(categories) == 0 && maxEffort == 0 {
		return a.Violations
	}

	var filtered []Violation

	for _, v := range a.Violations {
		// Filter by ID if specified
		if len(violationIDs) > 0 {
			found := false
			for _, id := range violationIDs {
				if v.ID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by category if specified
		if len(categories) > 0 {
			found := false
			for _, cat := range categories {
				if v.Category == cat {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by effort if specified
		if maxEffort > 0 && v.Effort > maxEffort {
			continue
		}

		filtered = append(filtered, v)
	}

	return filtered
}
