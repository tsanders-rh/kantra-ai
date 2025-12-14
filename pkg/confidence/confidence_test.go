package confidence

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, 0.80, config.Default)
	assert.True(t, config.UseEffortFallback)
	assert.Equal(t, ActionSkip, config.OnLowConfidence)
	assert.NotNil(t, config.Thresholds)
	assert.Equal(t, 0.70, config.Thresholds[ComplexityTrivial])
	assert.Equal(t, 0.95, config.Thresholds[ComplexityExpert])
}

func TestEffortToComplexity(t *testing.T) {
	tests := []struct {
		effort   int
		expected string
	}{
		// Normal range (0-10)
		{0, ComplexityTrivial},
		{2, ComplexityTrivial},
		{3, ComplexityLow},
		{4, ComplexityLow},
		{5, ComplexityMedium},
		{6, ComplexityMedium},
		{7, ComplexityHigh},
		{8, ComplexityHigh},
		{9, ComplexityExpert},
		{10, ComplexityExpert},
		// Edge cases - negative (should clamp to 0 -> Trivial)
		{-1, ComplexityTrivial},
		{-10, ComplexityTrivial},
		// Edge cases - over 10 (should clamp to 10 -> Expert)
		{11, ComplexityExpert},
		{15, ComplexityExpert},
		{100, ComplexityExpert},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("effort_%d", tt.effort), func(t *testing.T) {
			result := EffortToComplexity(tt.effort)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetThreshold(t *testing.T) {
	config := DefaultConfig()

	tests := []struct {
		name       string
		complexity string
		expected   float64
	}{
		{"trivial", ComplexityTrivial, 0.70},
		{"low", ComplexityLow, 0.75},
		{"medium", ComplexityMedium, 0.80},
		{"high", ComplexityHigh, 0.90},
		{"expert", ComplexityExpert, 0.95},
		{"unknown", "unknown", 0.80}, // Falls back to Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := config.GetThreshold(tt.complexity)
			assert.Equal(t, tt.expected, threshold)
		})
	}
}

func TestConfig_ShouldApplyFix(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true

	tests := []struct {
		name         string
		confidence   float64
		complexity   string
		effort       int
		shouldApply  bool
		hasReason    bool
	}{
		{
			name:        "high confidence trivial - should apply",
			confidence:  0.85,
			complexity:  ComplexityTrivial,
			effort:      1,
			shouldApply: true,
		},
		{
			name:        "low confidence trivial - should apply (0.70 threshold)",
			confidence:  0.72,
			complexity:  ComplexityTrivial,
			effort:      1,
			shouldApply: true,
		},
		{
			name:        "very low confidence trivial - should skip",
			confidence:  0.65,
			complexity:  ComplexityTrivial,
			effort:      1,
			shouldApply: false,
			hasReason:   true,
		},
		{
			name:        "medium confidence expert - should skip (0.95 threshold)",
			confidence:  0.88,
			complexity:  ComplexityExpert,
			effort:      10,
			shouldApply: false,
			hasReason:   true,
		},
		{
			name:        "high confidence expert - should apply",
			confidence:  0.97,
			complexity:  ComplexityExpert,
			effort:      10,
			shouldApply: true,
		},
		{
			name:        "no complexity with effort 5 (medium) - should apply",
			confidence:  0.82,
			complexity:  "",
			effort:      5,
			shouldApply: true,
		},
		{
			name:        "no complexity with effort 9 (expert) - should skip",
			confidence:  0.90,
			complexity:  "",
			effort:      9,
			shouldApply: false,
			hasReason:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldApply, reason := config.ShouldApplyFix(tt.confidence, tt.complexity, tt.effort)
			assert.Equal(t, tt.shouldApply, shouldApply)
			if tt.hasReason {
				assert.NotEmpty(t, reason)
				assert.Contains(t, reason, "confidence")
			} else {
				assert.Empty(t, reason)
			}
		})
	}
}

func TestConfig_ShouldApplyFix_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	// When disabled, should always apply regardless of confidence
	shouldApply, reason := config.ShouldApplyFix(0.10, ComplexityExpert, 10)
	assert.True(t, shouldApply)
	assert.Empty(t, reason)
}

func TestIsHighComplexity(t *testing.T) {
	tests := []struct {
		name               string
		complexity         string
		effort             int
		useEffortFallback  bool
		expected           bool
	}{
		{"trivial complexity", ComplexityTrivial, 1, true, false},
		{"low complexity", ComplexityLow, 3, true, false},
		{"medium complexity", ComplexityMedium, 5, true, false},
		{"high complexity", ComplexityHigh, 7, true, true},
		{"expert complexity", ComplexityExpert, 10, true, true},
		{"no complexity high effort with fallback", "", 8, true, true},
		{"no complexity high effort without fallback", "", 8, false, false},
		{"no complexity low effort", "", 3, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHighComplexity(tt.complexity, tt.effort, tt.useEffortFallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComplexityDescription(t *testing.T) {
	tests := []struct {
		complexity string
		contains   string
	}{
		{ComplexityTrivial, "95%+"},
		{ComplexityLow, "80%+"},
		{ComplexityMedium, "60%+"},
		{ComplexityHigh, "Manual review recommended"},
		{ComplexityExpert, "Manual review required"},
		{"unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.complexity, func(t *testing.T) {
			desc := ComplexityDescription(tt.complexity)
			assert.Contains(t, desc, tt.contains)
		})
	}
}

func TestStats(t *testing.T) {
	stats := NewStats()

	// Record some fixes
	stats.RecordFix(ComplexityTrivial, true)
	stats.RecordFix(ComplexityTrivial, true)
	stats.RecordFix(ComplexityMedium, true)
	stats.RecordFix(ComplexityHigh, false)
	stats.RecordFix(ComplexityExpert, false)

	// Verify totals
	assert.Equal(t, 5, stats.TotalFixes)
	assert.Equal(t, 3, stats.AppliedFixes)
	assert.Equal(t, 2, stats.SkippedFixes)

	// Verify by complexity
	assert.Equal(t, 2, stats.ByComplexity[ComplexityTrivial].Total)
	assert.Equal(t, 2, stats.ByComplexity[ComplexityTrivial].Applied)
	assert.Equal(t, 0, stats.ByComplexity[ComplexityTrivial].Skipped)

	assert.Equal(t, 1, stats.ByComplexity[ComplexityHigh].Total)
	assert.Equal(t, 0, stats.ByComplexity[ComplexityHigh].Applied)
	assert.Equal(t, 1, stats.ByComplexity[ComplexityHigh].Skipped)

	// Verify summary
	summary := stats.Summary()
	assert.Contains(t, summary, "3/5")
	assert.Contains(t, summary, "60.0%")
	assert.Contains(t, summary, "Skipped: 2")
}

func TestStats_NoFixes(t *testing.T) {
	stats := NewStats()
	summary := stats.Summary()
	assert.Equal(t, "No fixes attempted", summary)
}

func TestStats_AllApplied(t *testing.T) {
	stats := NewStats()
	stats.RecordFix(ComplexityTrivial, true)
	stats.RecordFix(ComplexityLow, true)

	summary := stats.Summary()
	assert.Contains(t, summary, "2/2")
	assert.NotContains(t, summary, "Skipped")
}

func TestStats_ComplexityBreakdown(t *testing.T) {
	stats := NewStats()

	// Add fixes across multiple complexity levels
	stats.RecordFix(ComplexityTrivial, true)
	stats.RecordFix(ComplexityTrivial, true)
	stats.RecordFix(ComplexityLow, true)
	stats.RecordFix(ComplexityMedium, false)
	stats.RecordFix(ComplexityHigh, false)
	stats.RecordFix(ComplexityExpert, false)

	summary := stats.Summary()

	// Verify overall stats
	assert.Contains(t, summary, "3/6")
	assert.Contains(t, summary, "50.0%")
	assert.Contains(t, summary, "Skipped: 3")

	// Verify complexity breakdown is included
	assert.Contains(t, summary, "By complexity:")
	assert.Contains(t, summary, "trivial: 2 applied, 0 skipped")
	assert.Contains(t, summary, "low: 1 applied, 0 skipped")
	assert.Contains(t, summary, "medium: 0 applied, 1 skipped")
	assert.Contains(t, summary, "high: 0 applied, 1 skipped")
	assert.Contains(t, summary, "expert: 0 applied, 1 skipped")
}

func TestStats_InvalidConfidence(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true

	tests := []struct {
		name       string
		confidence float64
		shouldFail bool
	}{
		{"valid 0.0", 0.0, false},
		{"valid 0.5", 0.5, false},
		{"valid 1.0", 1.0, false},
		{"invalid -0.1", -0.1, true},
		{"invalid 1.1", 1.1, true},
		{"invalid 2.0", 2.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldApply, reason := config.ShouldApplyFix(tt.confidence, ComplexityMedium, 5)
			if tt.shouldFail {
				assert.False(t, shouldApply)
				assert.Contains(t, reason, "invalid confidence")
			}
		})
	}
}

func TestShouldApplyFix_ReasonIncludesAction(t *testing.T) {
	tests := []struct {
		name           string
		action         Action
		expectedAction string
	}{
		{"skip action", ActionSkip, "skip"},
		{"warn-and-apply action", ActionWarnAndApply, "warn-and-apply"},
		{"manual-review action", ActionManualReviewFile, "manual-review"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Enabled = true
			config.OnLowConfidence = tt.action

			// Use low confidence to trigger reason generation
			shouldApply, reason := config.ShouldApplyFix(0.60, ComplexityMedium, 5)

			assert.False(t, shouldApply)
			assert.NotEmpty(t, reason)
			assert.Contains(t, reason, "action: "+tt.expectedAction)
		})
	}
}

func TestShouldApplyFix_EmptyComplexityNoFallback(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.UseEffortFallback = false // Disable effort fallback

	// With empty complexity and no fallback, should use default threshold
	shouldApply, reason := config.ShouldApplyFix(0.82, "", 9)

	// Default is 0.80, so 0.82 should pass
	assert.True(t, shouldApply)
	assert.Empty(t, reason)

	// Test below default threshold
	shouldApply, reason = config.ShouldApplyFix(0.75, "", 9)
	assert.False(t, shouldApply)
	assert.NotEmpty(t, reason)
	assert.Contains(t, reason, "medium") // Should fall back to medium
}

func TestStats_ConcurrentAccess(t *testing.T) {
	stats := NewStats()

	// Simulate concurrent access from multiple goroutines
	const numGoroutines = 10
	const fixesPerGoroutine = 100

	done := make(chan bool)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			complexity := ComplexityMedium
			if id%2 == 0 {
				complexity = ComplexityHigh
			}

			for j := 0; j < fixesPerGoroutine; j++ {
				applied := j%3 != 0 // Apply 2/3, skip 1/3
				stats.RecordFix(complexity, applied)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify totals - with mutex protection, these should be accurate
	expectedTotal := numGoroutines * fixesPerGoroutine
	// j%3 != 0 gives: skip on 0,3,6,9... = 34 skips, 66 applies (for 0-99)
	expectedApplied := numGoroutines * 66
	expectedSkipped := numGoroutines * 34

	assert.Equal(t, expectedTotal, stats.TotalFixes, "TotalFixes should match")
	assert.Equal(t, expectedApplied, stats.AppliedFixes, "AppliedFixes should match")
	assert.Equal(t, expectedSkipped, stats.SkippedFixes, "SkippedFixes should match")

	// Verify complexity breakdown
	// 5 goroutines used Medium, 5 used High (id%2 == 0)
	assert.Equal(t, fixesPerGoroutine*5, stats.ByComplexity[ComplexityMedium].Total)
	assert.Equal(t, fixesPerGoroutine*5, stats.ByComplexity[ComplexityHigh].Total)

	// Verify Summary doesn't panic during concurrent access
	summary := stats.Summary()
	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, fmt.Sprintf("%d/%d", expectedApplied, expectedTotal))
}
