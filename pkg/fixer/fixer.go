package fixer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsanders/kantra-ai/pkg/provider"
	"github.com/tsanders/kantra-ai/pkg/violation"
)

// Fixer applies AI-generated fixes to files
type Fixer struct {
	provider provider.Provider
	inputDir string
	dryRun   bool
}

// New creates a new Fixer
func New(provider provider.Provider, inputDir string, dryRun bool) *Fixer {
	return &Fixer{
		provider: provider,
		inputDir: inputDir,
		dryRun:   dryRun,
	}
}

// FixResult contains the result of fixing a single incident
type FixResult struct {
	ViolationID string
	IncidentURI string
	FilePath    string // Relative file path for git tracking
	Success     bool
	Cost        float64
	TokensUsed  int
	Error       error
	Explanation string
}

// FixIncident fixes a single incident of a violation
func (f *Fixer) FixIncident(ctx context.Context, v violation.Violation, incident violation.Incident) (*FixResult, error) {
	result := &FixResult{
		ViolationID: v.ID,
		IncidentURI: incident.URI,
	}

	// Get the file path
	filePath := incident.GetFilePath()

	// Make it relative to input directory if it's absolute
	if filepath.IsAbs(filePath) {
		// Try to make it relative to inputDir
		absInputDir, _ := filepath.Abs(f.inputDir)
		if strings.HasPrefix(filePath, absInputDir) {
			filePath = strings.TrimPrefix(filePath, absInputDir)
			filePath = strings.TrimPrefix(filePath, string(filepath.Separator))
		} else {
			// Path looks absolute but doesn't match input dir
			// This happens with URIs like file:///src/file.java
			// Strip leading slash(es) to make it relative
			filePath = strings.TrimLeft(filePath, string(filepath.Separator))
		}
	}

	// Store the relative file path for git tracking
	result.FilePath = filePath

	fullPath := filepath.Join(f.inputDir, filePath)

	// Read the current file content
	fileContent, err := os.ReadFile(fullPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to read file '%s': %w\n\n"+
			"Possible causes:\n"+
			"  - File does not exist at the specified path\n"+
			"  - Insufficient read permissions\n"+
			"  - File path is relative but --input directory is incorrect\n\n"+
			"Please verify:\n"+
			"  1. The file exists: ls -la %s\n"+
			"  2. You have read permissions: chmod +r %s\n"+
			"  3. The --input path points to the correct directory",
			fullPath, err, fullPath, fullPath)
		return result, err
	}

	// Detect language from file extension
	language := detectLanguage(filePath)

	// Build the fix request
	req := provider.FixRequest{
		Violation:   v,
		Incident:    incident,
		FileContent: string(fileContent),
		Language:    language,
	}

	// Get the fix from AI provider
	resp, err := f.provider.FixViolation(ctx, req)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.Success = resp.Success
	result.Cost = resp.Cost
	result.TokensUsed = resp.TokensUsed
	result.Explanation = resp.Explanation

	if !resp.Success {
		result.Error = resp.Error
		return result, resp.Error
	}

	// Clean up the response (remove markdown code blocks if present)
	fixedContent := cleanResponse(resp.FixedContent)

	// Apply the fix (or just log if dry-run)
	if f.dryRun {
		fmt.Printf("  [DRY-RUN] Would write %d bytes to %s\n", len(fixedContent), fullPath)
	} else {
		if err := os.WriteFile(fullPath, []byte(fixedContent), 0644); err != nil {
			result.Error = fmt.Errorf("failed to write file '%s': %w\n\n"+
				"Possible causes:\n"+
				"  - Insufficient write permissions\n"+
				"  - Disk is full or read-only filesystem\n"+
				"  - File is locked by another process\n"+
				"  - Parent directory does not exist\n\n"+
				"Please verify:\n"+
				"  1. You have write permissions: chmod +w %s\n"+
				"  2. Sufficient disk space: df -h %s\n"+
				"  3. File is not locked: lsof %s",
				fullPath, err, fullPath, filepath.Dir(fullPath), fullPath)
			return result, err
		}
		fmt.Printf("  ✓ Fixed: %s (cost: $%.4f, %d tokens)\n", fullPath, result.Cost, result.TokensUsed)
	}

	return result, nil
}

// detectLanguage detects programming language from file extension
func detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".java":
		return "java"
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rb":
		return "ruby"
	case ".xml":
		return "xml"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "unknown"
	}
}

// cleanResponse removes markdown code blocks and extra formatting
func cleanResponse(content string) string {
	// Remove markdown code blocks
	content = strings.TrimPrefix(content, "```java\n")
	content = strings.TrimPrefix(content, "```python\n")
	content = strings.TrimPrefix(content, "```go\n")
	content = strings.TrimPrefix(content, "```javascript\n")
	content = strings.TrimPrefix(content, "```typescript\n")
	content = strings.TrimPrefix(content, "```\n")
	content = strings.TrimSuffix(content, "\n```")
	content = strings.TrimSuffix(content, "```")

	return content
}
