package verifier

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// VerificationType defines what to verify after applying fixes
type VerificationType int

const (
	// VerificationNone skips verification
	VerificationNone VerificationType = iota
	// VerificationBuild checks if code compiles
	VerificationBuild
	// VerificationTest runs the test suite
	VerificationTest
)

// VerificationStrategy defines when to run verification
type VerificationStrategy int

const (
	// StrategyPerFix runs verification after each individual fix
	StrategyPerFix VerificationStrategy = iota
	// StrategyPerViolation runs verification after all fixes for a violation
	StrategyPerViolation
	// StrategyAtEnd runs verification once after all fixes
	StrategyAtEnd
)

// Config holds verification configuration
type Config struct {
	Type           VerificationType
	Strategy       VerificationStrategy
	WorkingDir     string
	CustomCommand  string // Optional custom verification command
	Timeout        time.Duration
	FailFast       bool // Stop on first verification failure
	SkipOnDryRun   bool // Skip verification in dry-run mode
}

// Result represents the outcome of a verification run
type Result struct {
	Success   bool
	Output    string
	Error     error
	Duration  time.Duration
	Command   string
	Timestamp time.Time
}

// Verifier runs build/test verification after fixes
type Verifier struct {
	config      Config
	projectType ProjectType
}

// ProjectType represents the type of project being verified
type ProjectType int

const (
	ProjectUnknown ProjectType = iota
	ProjectGo
	ProjectMaven
	ProjectGradle
	ProjectNpm
)

// NewVerifier creates a new verifier with the given configuration
func NewVerifier(config Config) (*Verifier, error) {
	if config.WorkingDir == "" {
		return nil, fmt.Errorf("working directory is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute // Default timeout
	}

	projectType := detectProjectType(config.WorkingDir)

	return &Verifier{
		config:      config,
		projectType: projectType,
	}, nil
}

// Verify runs the configured verification
func (v *Verifier) Verify() (*Result, error) {
	start := time.Now()

	command := v.getVerificationCommand()
	if command == "" {
		return nil, fmt.Errorf("no verification command available for project type")
	}

	result := &Result{
		Command:   command,
		Timestamp: start,
	}

	// Parse and execute command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid verification command: %s", command)
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = v.config.WorkingDir

	// Capture output
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("verification failed: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// getVerificationCommand returns the appropriate verification command
func (v *Verifier) getVerificationCommand() string {
	// Use custom command if provided
	if v.config.CustomCommand != "" {
		return v.config.CustomCommand
	}

	// Determine command based on project type and verification type
	switch v.projectType {
	case ProjectGo:
		return v.getGoCommand()
	case ProjectMaven:
		return v.getMavenCommand()
	case ProjectGradle:
		return v.getGradleCommand()
	case ProjectNpm:
		return v.getNpmCommand()
	default:
		return ""
	}
}

// getGoCommand returns the appropriate Go verification command
func (v *Verifier) getGoCommand() string {
	switch v.config.Type {
	case VerificationBuild:
		return "go build ./..."
	case VerificationTest:
		return "go test ./..."
	default:
		return ""
	}
}

// getMavenCommand returns the appropriate Maven verification command
func (v *Verifier) getMavenCommand() string {
	switch v.config.Type {
	case VerificationBuild:
		return "mvn compile"
	case VerificationTest:
		return "mvn test"
	default:
		return ""
	}
}

// getGradleCommand returns the appropriate Gradle verification command
func (v *Verifier) getGradleCommand() string {
	switch v.config.Type {
	case VerificationBuild:
		return "gradle build -x test"
	case VerificationTest:
		return "gradle test"
	default:
		return ""
	}
}

// getNpmCommand returns the appropriate npm verification command
func (v *Verifier) getNpmCommand() string {
	switch v.config.Type {
	case VerificationBuild:
		return "npm run build"
	case VerificationTest:
		return "npm test"
	default:
		return ""
	}
}

// detectProjectType attempts to detect the project type from files in the directory
func detectProjectType(dir string) ProjectType {
	// Check for Go
	if fileExists(filepath.Join(dir, "go.mod")) {
		return ProjectGo
	}

	// Check for Maven
	if fileExists(filepath.Join(dir, "pom.xml")) {
		return ProjectMaven
	}

	// Check for Gradle
	if fileExists(filepath.Join(dir, "build.gradle")) || fileExists(filepath.Join(dir, "build.gradle.kts")) {
		return ProjectGradle
	}

	// Check for npm
	if fileExists(filepath.Join(dir, "package.json")) {
		return ProjectNpm
	}

	return ProjectUnknown
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ProjectTypeName returns a human-readable name for the project type
func (pt ProjectType) String() string {
	switch pt {
	case ProjectGo:
		return "Go"
	case ProjectMaven:
		return "Maven"
	case ProjectGradle:
		return "Gradle"
	case ProjectNpm:
		return "npm"
	default:
		return "Unknown"
	}
}

// ParseVerificationType parses a string into a VerificationType
func ParseVerificationType(s string) (VerificationType, error) {
	switch strings.ToLower(s) {
	case "build":
		return VerificationBuild, nil
	case "test", "tests":
		return VerificationTest, nil
	case "none", "":
		return VerificationNone, nil
	default:
		return VerificationNone, fmt.Errorf("invalid verification type: %s (valid: build, test, none)", s)
	}
}

// ParseVerificationStrategy parses a string into a VerificationStrategy
func ParseVerificationStrategy(s string) (VerificationStrategy, error) {
	switch s {
	case "per-fix":
		return StrategyPerFix, nil
	case "per-violation":
		return StrategyPerViolation, nil
	case "at-end", "":
		return StrategyAtEnd, nil
	default:
		return StrategyAtEnd, fmt.Errorf("invalid verification strategy: %s (valid: per-fix, per-violation, at-end)", s)
	}
}
