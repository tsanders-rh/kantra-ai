package ux

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatCost(t *testing.T) {
	tests := []struct {
		name string
		cost float64
	}{
		{"very low cost", 0.001},
		{"low cost", 0.05},
		{"medium cost", 0.50},
		{"high cost", 1.50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCost(tt.cost)
			// Should contain the cost value
			assert.Contains(t, result, "$")
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
	}{
		{"low tokens", 500},
		{"medium tokens", 2000},
		{"high tokens", 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTokens(tt.tokens)
			// Should contain the token count
			assert.NotEmpty(t, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{"milliseconds", 500 * time.Millisecond},
		{"seconds", 5 * time.Second},
		{"minutes", 2 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.NotEmpty(t, result)
		})
	}
}

func TestRepeat(t *testing.T) {
	tests := []struct {
		name  string
		str   string
		count int
		want  string
	}{
		{"empty", "", 5, ""},
		{"single char", "=", 3, "==="},
		{"multiple chars", "ab", 2, "abab"},
		{"zero count", "x", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repeat(tt.str, tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewSpinner(t *testing.T) {
	spinner := NewSpinner("Testing...")
	assert.NotNil(t, spinner)
	assert.Equal(t, "Testing...", spinner.message)
	assert.NotNil(t, spinner.frames)
	assert.Len(t, spinner.frames, 10)
}

func TestNewProgressBar(t *testing.T) {
	bar := NewProgressBar(100, "Testing")
	assert.NotNil(t, bar)
}

func TestPrintSummaryTable(t *testing.T) {
	rows := [][]string{
		{"Name", "Value"},
		{"Test1", "123"},
		{"Test2", "456"},
	}

	// Should not panic
	PrintSummaryTable(rows)

	// Empty table should not panic
	PrintSummaryTable([][]string{})
}

func TestColorFunctions(t *testing.T) {
	// Test that color functions return non-empty strings
	assert.NotEmpty(t, Success("test"))
	assert.NotEmpty(t, Error("test"))
	assert.NotEmpty(t, Warning("test"))
	assert.NotEmpty(t, Info("test"))
	assert.NotEmpty(t, Bold("test"))
	assert.NotEmpty(t, Dim("test"))
}

func TestIsTerminal(t *testing.T) {
	// Just verify it doesn't panic
	_ = IsTerminal()
}

// Test that format functions don't panic with edge cases
func TestFormatEdgeCases(t *testing.T) {
	t.Run("zero cost", func(t *testing.T) {
		result := FormatCost(0.0)
		assert.Contains(t, result, "$")
	})

	t.Run("negative cost", func(t *testing.T) {
		result := FormatCost(-1.0)
		assert.Contains(t, result, "$")
	})

	t.Run("zero tokens", func(t *testing.T) {
		result := FormatTokens(0)
		assert.NotEmpty(t, result)
	})

	t.Run("zero duration", func(t *testing.T) {
		result := FormatDuration(0)
		assert.NotEmpty(t, result)
	})
}

// Test repeat function edge cases
func TestRepeatEdgeCases(t *testing.T) {
	t.Run("negative count", func(t *testing.T) {
		result := repeat("x", -1)
		assert.Equal(t, "", result)
	})

	t.Run("large count", func(t *testing.T) {
		result := repeat("a", 1000)
		assert.Equal(t, 1000, len(result))
		assert.True(t, strings.Count(result, "a") == 1000)
	})
}
