package fixer

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveAndValidateFilePath_PathTraversal tests security edge cases
func TestResolveAndValidateFilePath_PathTraversal(t *testing.T) {
	tests := []struct {
		name          string
		filePath      string
		inputDir      string
		wantErr       bool
		errContains   string
		skipOnWindows bool
	}{
		{
			name:     "normal relative path",
			filePath: "src/Main.java",
			inputDir: "/workspace/project",
			wantErr:  false,
		},
		{
			name:     "path with dot dot attempting traversal",
			filePath: "../../../etc/passwd",
			inputDir: "/workspace/project",
			wantErr:  true,
			errContains: "outside input directory",
		},
		{
			name:     "path with dot dot in middle",
			filePath: "src/../../etc/passwd",
			inputDir: "/workspace/project",
			wantErr:  true,
			errContains: "outside input directory",
		},
		{
			name:          "local absolute path outside input dir returns error",
			filePath:      "/Users/other/project/file.java",
			inputDir:      "/workspace/project",
			wantErr:       true,
			errContains:   "does not match input directory",
			skipOnWindows: true, // Unix-style absolute paths don't work the same on Windows
		},
		{
			name:     "container path /opt/input is made relative",
			filePath: "/opt/input/source/Main.java",
			inputDir: "/workspace/project",
			wantErr:  false,
		},
		{
			name:     "container path without system prefix is made relative",
			filePath: "/src/Main.java",
			inputDir: "/workspace/project",
			wantErr:  false,
		},
		{
			name:     "path with multiple slashes",
			filePath: "src////Main.java",
			inputDir: "/workspace/project",
			wantErr:  false,
		},
		{
			name:     "path starting with dot slash",
			filePath: "./src/Main.java",
			inputDir: "/workspace/project",
			wantErr:  false,
		},
		{
			name:     "symlink-style traversal attempt",
			filePath: "src/../../../root/file.txt",
			inputDir: "/workspace/project",
			wantErr:  true,
			errContains: "outside input directory",
		},
		{
			name:     "empty path",
			filePath: "",
			inputDir: "/workspace/project",
			wantErr:  false, // Empty path resolves to current directory
		},
		{
			name:     "path with spaces",
			filePath: "src/My File.java",
			inputDir: "/workspace/project",
			wantErr:  false,
		},
		{
			name:     "valid absolute path within input dir",
			filePath: "/workspace/project/src/Main.java",
			inputDir: "/workspace/project",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping on Windows: Unix-style absolute paths behave differently")
			}

			result, err := resolveAndValidateFilePath(tt.filePath, tt.inputDir)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// TestResolveAndValidateFilePath_EdgeCases tests additional edge cases
func TestResolveAndValidateFilePath_EdgeCases(t *testing.T) {
	t.Run("null byte injection attempt", func(t *testing.T) {
		_, err := resolveAndValidateFilePath("src/file\x00.java", "/workspace/project")
		// filepath.Clean will handle this, just ensure it doesn't panic
		assert.NotPanics(t, func() {
			_, _ = resolveAndValidateFilePath("src/file\x00.java", "/workspace/project")
		})
		_ = err // May or may not error depending on OS
	})

	t.Run("very long path", func(t *testing.T) {
		longPath := "src/"
		for i := 0; i < 100; i++ {
			longPath += "very_long_directory_name/"
		}
		longPath += "file.java"

		_, err := resolveAndValidateFilePath(longPath, "/workspace/project")
		// Should not panic or cause issues
		assert.NotPanics(t, func() {
			_, _ = resolveAndValidateFilePath(longPath, "/workspace/project")
		})
		_ = err // May succeed or fail depending on OS path limits
	})

	t.Run("unicode characters in path", func(t *testing.T) {
		result, err := resolveAndValidateFilePath("src/文件.java", "/workspace/project")
		require.NoError(t, err)
		assert.Contains(t, result, "文件.java")
	})
}
