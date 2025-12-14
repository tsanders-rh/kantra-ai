package gitutil

import "fmt"

// ProgressWriter is an interface for reporting progress during PR creation
type ProgressWriter interface {
	// Printf formats and writes progress messages
	Printf(format string, args ...interface{})
}

// StdoutProgressWriter writes progress to stdout
type StdoutProgressWriter struct{}

// Printf writes a progress message to stdout
func (w *StdoutProgressWriter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// NoOpProgressWriter discards progress messages (for tests)
type NoOpProgressWriter struct{}

// Printf discards the message
func (w *NoOpProgressWriter) Printf(format string, args ...interface{}) {
	// Do nothing
}
