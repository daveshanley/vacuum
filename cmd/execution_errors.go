// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/tui"
)

func renderPreParseExecutionErrors(source string, result *motor.RuleSetExecutionResult) error {
	if result == nil || result.SpecInfo != nil {
		return nil
	}
	return renderExecutionInputErrors(source, result)
}

func renderExecutionInputErrors(source string, result *motor.RuleSetExecutionResult) error {
	if result == nil || len(result.Errors) == 0 {
		return nil
	}
	errorCount := 0
	for _, executionErr := range result.Errors {
		if executionErr != nil {
			errorCount++
			tui.RenderErrorString("Unable to process spec '%s': %s", source, executionErr.Error())
		}
	}
	if errorCount == 0 {
		return nil
	}
	return NewInputError("linting failed due to %d issues", errorCount)
}
