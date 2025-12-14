package confidence

import (
	"fmt"
)

// Complexity levels based on Konveyor enhancement proposal
// https://github.com/konveyor/enhancements/pull/255
const (
	ComplexityTrivial = "trivial" // 95%+ AI success - mechanical find/replace
	ComplexityLow     = "low"     // 80%+ AI success - straightforward API equivalents
	ComplexityMedium  = "medium"  // 60%+ AI success - requires context understanding
	ComplexityHigh    = "high"    // 30-50% AI success - architectural changes
	ComplexityExpert  = "expert"  // <30% AI success - domain expertise required
)

// Action defines what to do with low-confidence fixes
type Action string

const (
	ActionSkip            Action = "skip"              // Skip low-confidence fixes (default, safest)
	ActionWarnAndApply    Action = "warn-and-apply"    // Apply but warn about low confidence
	ActionManualReviewFile Action = "manual-review-file" // Write to review file for manual processing
)

// Config holds confidence threshold configuration
type Config struct {
	Enabled bool // Enable confidence filtering

	// Complexity-based thresholds
	Thresholds map[string]float64

	// Default threshold when no migration_complexity metadata
	Default float64

	// Use effort level as fallback for complexity
	UseEffortFallback bool

	// What to do with low-confidence fixes
	OnLowConfidence Action
}

// DefaultConfig returns the default confidence configuration
func DefaultConfig() Config {
	return Config{
		Enabled:           false, // Disabled by default for backward compatibility
		Thresholds:        DefaultComplexityThresholds(),
		Default:           0.80,
		UseEffortFallback: true,
		OnLowConfidence:   ActionSkip,
	}
}

// DefaultComplexityThresholds returns default thresholds based on expected AI success rates
// Aligned with Konveyor enhancement proposal expectations
func DefaultComplexityThresholds() map[string]float64 {
	return map[string]float64{
		ComplexityTrivial: 0.70, // 95%+ AI success - accept lower confidence
		ComplexityLow:     0.75, // 80%+ AI success
		ComplexityMedium:  0.80, // 60%+ AI success - default
		ComplexityHigh:    0.90, // 30-50% AI success - need high confidence
		ComplexityExpert:  0.95, // <30% AI success - very high confidence required
	}
}

// EffortToComplexity maps effort levels (0-10) to complexity levels
// Used as fallback when migration_complexity metadata is missing
func EffortToComplexity(effort int) string {
	switch {
	case effort <= 2:
		return ComplexityTrivial
	case effort <= 4:
		return ComplexityLow
	case effort <= 6:
		return ComplexityMedium
	case effort <= 8:
		return ComplexityHigh
	default:
		return ComplexityExpert
	}
}

// GetThreshold returns the confidence threshold for a given complexity level
func (c *Config) GetThreshold(complexity string) float64 {
	if threshold, ok := c.Thresholds[complexity]; ok {
		return threshold
	}
	return c.Default
}

// ShouldApplyFix determines whether a fix should be applied based on confidence
func (c *Config) ShouldApplyFix(confidence float64, complexity string, effort int) (bool, string) {
	if !c.Enabled {
		return true, "" // Confidence filtering disabled
	}

	// Validate confidence range
	if confidence < 0.0 || confidence > 1.0 {
		return false, fmt.Sprintf("invalid confidence value %.2f (must be 0.0-1.0)", confidence)
	}

	// Determine effective complexity
	effectiveComplexity := complexity
	if effectiveComplexity == "" && c.UseEffortFallback {
		effectiveComplexity = EffortToComplexity(effort)
	}
	if effectiveComplexity == "" {
		effectiveComplexity = ComplexityMedium // Ultimate fallback
	}

	threshold := c.GetThreshold(effectiveComplexity)

	if confidence >= threshold {
		return true, ""
	}

	// Below threshold
	reason := fmt.Sprintf("confidence %.2f below threshold %.2f (complexity: %s)",
		confidence, threshold, effectiveComplexity)

	return false, reason
}

// IsHighComplexity returns true if the complexity is high or expert level
// Used for marking violations that need manual review
func IsHighComplexity(complexity string, effort int, useEffortFallback bool) bool {
	effectiveComplexity := complexity
	if effectiveComplexity == "" && useEffortFallback {
		effectiveComplexity = EffortToComplexity(effort)
	}

	return effectiveComplexity == ComplexityHigh || effectiveComplexity == ComplexityExpert
}

// IsValidComplexity checks if a string is a valid complexity level
func IsValidComplexity(level string) bool {
	switch level {
	case ComplexityTrivial, ComplexityLow, ComplexityMedium, ComplexityHigh, ComplexityExpert:
		return true
	default:
		return false
	}
}

// ValidComplexityLevels returns all valid complexity level strings
func ValidComplexityLevels() []string {
	return []string{ComplexityTrivial, ComplexityLow, ComplexityMedium, ComplexityHigh, ComplexityExpert}
}

// ComplexityDescription returns a human-readable description of a complexity level
func ComplexityDescription(complexity string) string {
	descriptions := map[string]string{
		ComplexityTrivial: "Mechanical find/replace (95%+ AI success)",
		ComplexityLow:     "Straightforward API equivalents (80%+ AI success)",
		ComplexityMedium:  "Requires context understanding (60%+ AI success)",
		ComplexityHigh:    "Architectural changes (30-50% AI success) - Manual review recommended",
		ComplexityExpert:  "Domain expertise required (<30% AI success) - Manual review required",
	}

	if desc, ok := descriptions[complexity]; ok {
		return desc
	}
	return "Unknown complexity"
}

// Stats tracks confidence-based filtering statistics
type Stats struct {
	TotalFixes       int
	AppliedFixes     int
	SkippedFixes     int
	ByComplexity     map[string]*ComplexityStats
}

// ComplexityStats tracks statistics per complexity level
type ComplexityStats struct {
	Total   int
	Applied int
	Skipped int
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		ByComplexity: make(map[string]*ComplexityStats),
	}
}

// RecordFix records a fix attempt
func (s *Stats) RecordFix(complexity string, applied bool) {
	s.TotalFixes++
	if applied {
		s.AppliedFixes++
	} else {
		s.SkippedFixes++
	}

	if _, ok := s.ByComplexity[complexity]; !ok {
		s.ByComplexity[complexity] = &ComplexityStats{}
	}

	s.ByComplexity[complexity].Total++
	if applied {
		s.ByComplexity[complexity].Applied++
	} else {
		s.ByComplexity[complexity].Skipped++
	}
}

// Summary returns a formatted summary of the stats
func (s *Stats) Summary() string {
	if s.TotalFixes == 0 {
		return "No fixes attempted"
	}

	summary := fmt.Sprintf("Applied: %d/%d (%.1f%%)",
		s.AppliedFixes, s.TotalFixes, float64(s.AppliedFixes)/float64(s.TotalFixes)*100)

	if s.SkippedFixes > 0 {
		summary += fmt.Sprintf(", Skipped: %d (low confidence)", s.SkippedFixes)
	}

	return summary
}
