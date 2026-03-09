// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import "fmt"

// Exit code constants for CI/CD integration.
//
//	0 = success (spec processed, no violations at threshold)
//	1 = violations found at or above the configured --fail-severity
//	2 = tool / input error (spec could not be parsed, file not found, invalid flags, etc.)
const (
	ExitCodeSuccess    = 0
	ExitCodeViolations = 1
	ExitCodeInputError = 2
)

// ExitError is an error that carries a specific process exit code.
// When returned from a cobra RunE function, Execute() will call
// os.Exit with the embedded code instead of the default 1.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

// NewInputError creates an ExitError with exit code 2 (tool/input error).
func NewInputError(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitCodeInputError, Message: fmt.Sprintf(format, args...)}
}

// NewViolationError creates an ExitError with exit code 1 (violations found).
func NewViolationError(format string, args ...any) *ExitError {
	return &ExitError{Code: ExitCodeViolations, Message: fmt.Sprintf(format, args...)}
}
