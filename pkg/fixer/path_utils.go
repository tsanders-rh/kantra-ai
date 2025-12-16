package fixer

import (
	"fmt"
	"path/filepath"
	"strings"
)

// resolveAndValidateFilePath resolves a file path to be relative to the input directory
// and validates that it doesn't escape the input directory boundary.
//
// This function:
//   - Cleans the path to normalize it and remove ".." components
//   - Converts absolute paths within inputDir to relative paths
//   - Handles URIs like file:///src/file.java by stripping leading slashes
//   - Validates the resolved path stays within inputDir (prevents path traversal)
//
// Returns the clean relative path or an error if validation fails.
func resolveAndValidateFilePath(filePath, inputDir string) (string, error) {
	// Clean the path to normalize it and remove any ".." components
	cleanPath := filepath.Clean(filePath)

	// Make it relative to input directory if it's absolute
	if filepath.IsAbs(cleanPath) {
		// Try to make it relative to inputDir
		absInputDir, _ := filepath.Abs(inputDir)
		if strings.HasPrefix(cleanPath, absInputDir) {
			cleanPath = strings.TrimPrefix(cleanPath, absInputDir)
			cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
		} else {
			// Path looks absolute but doesn't match input dir
			// This happens with URIs like file:///src/file.java
			// Strip leading slash(es) to make it relative
			cleanPath = strings.TrimLeft(cleanPath, string(filepath.Separator))
		}
	}

	// Build the full path and validate it's within inputDir (prevent path traversal)
	fullPath := filepath.Join(inputDir, cleanPath)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	absInputDir, err := filepath.Abs(inputDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve input directory: %w", err)
	}

	// Security check: ensure the resolved path is within the input directory
	if !strings.HasPrefix(absFullPath, absInputDir+string(filepath.Separator)) &&
		absFullPath != absInputDir {
		return "", fmt.Errorf("security: file path '%s' resolves outside input directory '%s'",
			cleanPath, inputDir)
	}

	return cleanPath, nil
}
