package verifier

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected ProjectType
	}{
		{
			name:     "Go project",
			files:    []string{"go.mod"},
			expected: ProjectGo,
		},
		{
			name:     "Maven project",
			files:    []string{"pom.xml"},
			expected: ProjectMaven,
		},
		{
			name:     "Gradle project with .gradle",
			files:    []string{"build.gradle"},
			expected: ProjectGradle,
		},
		{
			name:     "Gradle project with .kts",
			files:    []string{"build.gradle.kts"},
			expected: ProjectGradle,
		},
		{
			name:     "npm project",
			files:    []string{"package.json"},
			expected: ProjectNpm,
		},
		{
			name:     "Unknown project",
			files:    []string{"README.md"},
			expected: ProjectUnknown,
		},
		{
			name:     "Go takes precedence",
			files:    []string{"go.mod", "pom.xml"},
			expected: ProjectGo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				err := os.WriteFile(path, []byte("test"), 0644)
				require.NoError(t, err)
			}

			got := detectProjectType(tmpDir)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseVerificationType(t *testing.T) {
	tests := []struct {
		input   string
		want    VerificationType
		wantErr bool
	}{
		{"build", VerificationBuild, false},
		{"Build", VerificationBuild, false},
		{"test", VerificationTest, false},
		{"tests", VerificationTest, false},
		{"none", VerificationNone, false},
		{"", VerificationNone, false},
		{"invalid", VerificationNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVerificationType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseVerificationStrategy(t *testing.T) {
	tests := []struct {
		input   string
		want    VerificationStrategy
		wantErr bool
	}{
		{"per-fix", StrategyPerFix, false},
		{"per-violation", StrategyPerViolation, false},
		{"at-end", StrategyAtEnd, false},
		{"", StrategyAtEnd, false},
		{"invalid", StrategyAtEnd, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVerificationStrategy(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewVerifier(t *testing.T) {
	t.Run("requires working directory", func(t *testing.T) {
		config := Config{
			Type: VerificationTest,
		}
		_, err := NewVerifier(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "working directory is required")
	})

	t.Run("sets default timeout", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := Config{
			Type:       VerificationTest,
			WorkingDir: tmpDir,
		}
		verifier, err := NewVerifier(config)
		require.NoError(t, err)
		assert.Equal(t, 10*time.Minute, verifier.config.Timeout)
	})

	t.Run("detects project type", func(t *testing.T) {
		tmpDir := t.TempDir()
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test"), 0644)
		require.NoError(t, err)

		config := Config{
			Type:       VerificationTest,
			WorkingDir: tmpDir,
		}
		verifier, err := NewVerifier(config)
		require.NoError(t, err)
		assert.Equal(t, ProjectGo, verifier.projectType)
	})
}

func TestVerifier_GetGoCommand(t *testing.T) {
	tmpDir := t.TempDir()
	goModPath := filepath.Join(tmpDir, "go.mod")
	err := os.WriteFile(goModPath, []byte("module test"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name    string
		vType   VerificationType
		want    string
	}{
		{"build", VerificationBuild, "go build ./..."},
		{"test", VerificationTest, "go test ./..."},
		{"none", VerificationNone, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Type:       tt.vType,
				WorkingDir: tmpDir,
			}
			verifier, err := NewVerifier(config)
			require.NoError(t, err)

			got := verifier.getVerificationCommand()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerifier_GetMavenCommand(t *testing.T) {
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	err := os.WriteFile(pomPath, []byte("<project></project>"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name  string
		vType VerificationType
		want  string
	}{
		{"build", VerificationBuild, "mvn compile"},
		{"test", VerificationTest, "mvn test"},
		{"none", VerificationNone, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Type:       tt.vType,
				WorkingDir: tmpDir,
			}
			verifier, err := NewVerifier(config)
			require.NoError(t, err)

			got := verifier.getVerificationCommand()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerifier_CustomCommand(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		Type:          VerificationTest,
		WorkingDir:    tmpDir,
		CustomCommand: "make test",
	}

	verifier, err := NewVerifier(config)
	require.NoError(t, err)

	got := verifier.getVerificationCommand()
	assert.Equal(t, "make test", got)
}

func TestVerifier_Verify(t *testing.T) {
	t.Run("successful verification", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := Config{
			Type:          VerificationBuild,
			WorkingDir:    tmpDir,
			CustomCommand: "echo 'success'",
		}

		verifier, err := NewVerifier(config)
		require.NoError(t, err)

		result, err := verifier.Verify()
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "success")
		assert.Greater(t, result.Duration, time.Duration(0))
	})

	t.Run("failed verification", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := Config{
			Type:          VerificationTest,
			WorkingDir:    tmpDir,
			CustomCommand: "false", // Command that always fails
		}

		verifier, err := NewVerifier(config)
		require.NoError(t, err)

		result, err := verifier.Verify()
		require.NoError(t, err) // No error returned, failure is in result
		assert.False(t, result.Success)
		assert.NotNil(t, result.Error)
	})

	t.Run("invalid command", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := Config{
			Type:          VerificationTest,
			WorkingDir:    tmpDir,
			CustomCommand: "", // Will try to auto-detect, but dir has no project files
		}

		verifier, err := NewVerifier(config)
		require.NoError(t, err)

		_, err = verifier.Verify()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no verification command available")
	})
}

func TestProjectTypeString(t *testing.T) {
	tests := []struct {
		pt   ProjectType
		want string
	}{
		{ProjectGo, "Go"},
		{ProjectMaven, "Maven"},
		{ProjectGradle, "Gradle"},
		{ProjectNpm, "npm"},
		{ProjectUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.pt.String()
			assert.Equal(t, tt.want, got)
		})
	}
}
