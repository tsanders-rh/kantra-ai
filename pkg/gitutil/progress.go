package gitutil

import "fmt"

// ProgressWriter is an interface for reporting progress during PR creation.
// It allows PR creation operations to provide real-time feedback to users
// without being tightly coupled to a specific output mechanism.
//
// Implementations should be safe for concurrent use if needed, though
// the current PR creation flow is sequential.
type ProgressWriter interface {
	// Printf formats and writes a progress message using fmt.Printf-style formatting.
	// Implementations should handle newlines and formatting consistently.
	//
	// Parameters:
	//   format - A format string as used by fmt.Printf
	//   args   - Arguments to be formatted according to the format string
	Printf(format string, args ...interface{})
}

// StdoutProgressWriter writes progress messages to standard output.
// This is the default implementation used by the CLI to provide
// real-time feedback during PR creation.
//
// Example usage:
//
//	progress := &gitutil.StdoutProgressWriter{}
//	tracker, err := gitutil.NewPRTracker(config, workingDir, provider, progress)
type StdoutProgressWriter struct{}

// Printf writes a formatted progress message to stdout.
// Messages are written immediately without buffering.
func (w *StdoutProgressWriter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// NoOpProgressWriter discards all progress messages.
// This implementation is useful for tests or scenarios where
// progress output is not desired.
//
// Example usage:
//
//	progress := &gitutil.NoOpProgressWriter{}
//	tracker, err := gitutil.NewPRTracker(config, workingDir, provider, progress)
type NoOpProgressWriter struct{}

// Printf discards the formatted message without writing it anywhere.
// This is a no-op implementation.
func (w *NoOpProgressWriter) Printf(format string, args ...interface{}) {
	// Intentionally empty - discard all progress messages
}
