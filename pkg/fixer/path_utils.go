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

	// Ensure inputDir is absolute for proper comparison
	absInputDir, err := filepath.Abs(inputDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve input directory '%s': %w", inputDir, err)
	}

	// Make it relative to input directory if it's absolute
	if filepath.IsAbs(cleanPath) {
		// Try to make it relative to inputDir
		if strings.HasPrefix(cleanPath, absInputDir) {
			cleanPath = strings.TrimPrefix(cleanPath, absInputDir)
			cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
		} else {
			// Path looks absolute but doesn't match input dir
			// This could be either:
			// 1. Container paths like file:///src/file.java (should be made relative)
			// 2. Local absolute paths that don't match inputDir (configuration error)

			// Check if this looks like a local filesystem absolute path (known system prefixes)
			// These indicate a real absolute path that should match inputDir
			isLocalAbsolutePath := strings.HasPrefix(cleanPath, "/Users/") ||
				strings.HasPrefix(cleanPath, "/home/") ||
				strings.HasPrefix(cleanPath, "/root/") ||
				strings.HasPrefix(cleanPath, "/mnt/") ||
				strings.HasPrefix(cleanPath, "/media/") ||
				strings.HasPrefix(cleanPath, "/var/") ||
				strings.HasPrefix(cleanPath, "/tmp/") ||
				strings.HasPrefix(cleanPath, "/etc/") ||
				strings.HasPrefix(cleanPath, "/usr/") ||
				strings.HasPrefix(cleanPath, "/opt/") && !strings.HasPrefix(cleanPath, "/opt/input") || // /opt/input is a common container path
				strings.HasPrefix(cleanPath, "C:\\") ||
				strings.HasPrefix(cleanPath, "D:\\")

			if isLocalAbsolutePath {
				// Local absolute path that doesn't match inputDir - this is likely a configuration error
				return "", fmt.Errorf("file path '%s' does not match input directory '%s'\n\n"+
					"This usually means:\n"+
					"  1. The --input directory path is incorrect\n"+
					"  2. The analysis was run on a different directory than specified in --input\n\n"+
					"Please verify:\n"+
					"  • Analysis source directory: %s\n"+
					"  • Your --input flag: %s\n\n"+
					"These paths should match. Update your --input flag to point to the correct source directory.",
					cleanPath, absInputDir, filepath.Dir(cleanPath), absInputDir)
			} else {
				// Looks like a container path (e.g., /src/file.java, /workspace/file.java)
				// Strip leading slash(es) to make it relative
				cleanPath = strings.TrimLeft(cleanPath, string(filepath.Separator))
			}
		}
	}

	// Build the full path and validate it's within inputDir (prevent path traversal)
	fullPath := filepath.Join(absInputDir, cleanPath)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Security check: ensure the resolved path is within the input directory
	if !strings.HasPrefix(absFullPath, absInputDir+string(filepath.Separator)) &&
		absFullPath != absInputDir {
		return "", fmt.Errorf("security: file path '%s' resolves outside input directory '%s'",
			cleanPath, inputDir)
	}

	return cleanPath, nil
}
