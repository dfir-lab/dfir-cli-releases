package commands

import "fmt"

// ExitError wraps an error with a specific exit code. Commands return this
// when the exit code should differ from the default (1 for errors).
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

// NewExitError creates an ExitError with the given code and message.
func NewExitError(code int, msg string) *ExitError {
	return &ExitError{Code: code, Message: msg}
}

// NewExitErrorf creates an ExitError with a formatted message.
func NewExitErrorf(code int, format string, args ...interface{}) *ExitError {
	return &ExitError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// SilentExitError is returned when the command has already printed its output
// and just needs to exit with a specific code (no error message to print).
type SilentExitError struct {
	Code int
}

func (e *SilentExitError) Error() string {
	return ""
}
