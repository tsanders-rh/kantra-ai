// Package ux provides user experience utilities for kantra-ai's command-line interface.
// It includes colored output formatting, progress tracking, spinners, and consistent
// message styling for success, error, warning, and informational messages.
package ux

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Color definitions for consistent output
var (
	Success = color.New(color.FgGreen).SprintFunc()
	Error   = color.New(color.FgRed).SprintFunc()
	Warning = color.New(color.FgYellow).SprintFunc()
	Info    = color.New(color.FgCyan).SprintFunc()
	Bold    = color.New(color.Bold).SprintFunc()
	Dim     = color.New(color.Faint).SprintFunc()
)

// PrintSuccess prints a success message with green checkmark
func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Success("✓"), msg)
}

// PrintError prints an error message with red X
func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Error("✗"), msg)
}

// PrintWarning prints a warning message with yellow triangle
func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Warning("⚠"), msg)
}

// PrintInfo prints an info message with cyan dot
func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Info("•"), msg)
}

// PrintHeader prints a bold header
func PrintHeader(text string) {
	fmt.Println(Bold(text))
	fmt.Println(Bold(repeat("=", len(text))))
	fmt.Println()
}

// PrintSection prints a section header
func PrintSection(text string) {
	fmt.Println()
	fmt.Println(Bold(text))
}

// NewProgressBar creates a new progress bar with consistent styling
func NewProgressBar(max int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetElapsedTime(true),
	)
}

// NewSpinner creates a simple text-based spinner
type Spinner struct {
	message string
	frames  []string
	index   int
	done    chan bool
	writer  io.Writer
}

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan bool),
		writer:  os.Stdout,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				// Clear the line
				fmt.Fprintf(s.writer, "\r%s\r", repeat(" ", len(s.message)+5))
				return
			case <-ticker.C:
				frame := s.frames[s.index%len(s.frames)]
				fmt.Fprintf(s.writer, "\r%s %s", Info(frame), s.message)
				s.index++
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.done <- true
	close(s.done)
}

// StopWithSuccess stops the spinner and shows success
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	PrintSuccess(message)
}

// StopWithError stops the spinner and shows error
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	PrintError(message)
}

// FormatCost formats a cost value with color
func FormatCost(cost float64) string {
	if cost < 0.01 {
		return Success(fmt.Sprintf("$%.4f", cost))
	} else if cost < 0.10 {
		return Info(fmt.Sprintf("$%.4f", cost))
	} else if cost < 1.00 {
		return Warning(fmt.Sprintf("$%.4f", cost))
	}
	return Error(fmt.Sprintf("$%.4f", cost))
}

// FormatTokens formats token count with color
func FormatTokens(tokens int) string {
	if tokens < 1000 {
		return Success(fmt.Sprintf("%d", tokens))
	} else if tokens < 5000 {
		return Info(fmt.Sprintf("%d", tokens))
	}
	return Warning(fmt.Sprintf("%d", tokens))
}

// FormatDuration formats a duration nicely
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return Dim(d.Round(time.Millisecond).String())
	}
	return Dim(d.Round(time.Second).String())
}

// FormatWarning returns a warning-colored string
func FormatWarning(s string) string {
	return Warning(s)
}

// ProgressWriter is an interface for reporting execution progress
type ProgressWriter interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	StartPhase(phaseName string)
	EndPhase()
}

// NoOpProgressWriter is a no-op implementation of ProgressWriter
type NoOpProgressWriter struct{}

func (n *NoOpProgressWriter) Info(format string, args ...interface{})       {}
func (n *NoOpProgressWriter) Error(format string, args ...interface{})      {}
func (n *NoOpProgressWriter) StartPhase(phaseName string)                   {}
func (n *NoOpProgressWriter) EndPhase()                                     {}

// ConsoleProgressWriter writes progress to console
type ConsoleProgressWriter struct{}

func (c *ConsoleProgressWriter) Info(format string, args ...interface{}) {
	PrintInfo(format, args...)
}

func (c *ConsoleProgressWriter) Error(format string, args ...interface{}) {
	PrintError(format, args...)
}

func (c *ConsoleProgressWriter) StartPhase(phaseName string) {
	PrintSection(phaseName)
}

func (c *ConsoleProgressWriter) EndPhase() {
	// No-op for now
}

// PrintSummaryTable prints a summary table
func PrintSummaryTable(rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, col := range row {
			// Strip color codes for length calculation
			cleanCol := color.New().Sprint(col)
			if len(cleanCol) > colWidths[i] {
				colWidths[i] = len(cleanCol)
			}
		}
	}

	// Print rows
	for _, row := range rows {
		for i, col := range row {
			fmt.Printf("%-*s  ", colWidths[i], col)
		}
		fmt.Println()
	}
}

// IsTerminal checks if output is going to a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Helper function to repeat a string
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
