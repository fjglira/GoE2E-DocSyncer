package domain

import "fmt"

// DocSyncerError is the base error type with context.
type DocSyncerError struct {
	Phase      string // "config", "scan", "parse", "convert", "template", "write"
	File       string
	LineNumber int
	Message    string
	Cause      error
}

func (e *DocSyncerError) Error() string {
	s := fmt.Sprintf("[%s]", e.Phase)
	if e.File != "" {
		s += fmt.Sprintf(" %s", e.File)
	}
	if e.LineNumber > 0 {
		s += fmt.Sprintf(":%d", e.LineNumber)
	}
	s += fmt.Sprintf(": %s", e.Message)
	if e.Cause != nil {
		s += fmt.Sprintf(": %v", e.Cause)
	}
	return s
}

func (e *DocSyncerError) Unwrap() error {
	return e.Cause
}

// NewError creates a new DocSyncerError.
func NewError(phase, file string, line int, message string, cause error) *DocSyncerError {
	return &DocSyncerError{
		Phase:      phase,
		File:       file,
		LineNumber: line,
		Message:    message,
		Cause:      cause,
	}
}
