// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestRenderExecutionInputErrors(t *testing.T) {
	assert.NoError(t, renderExecutionInputErrors("spec.yaml", nil))
	assert.NoError(t, renderExecutionInputErrors("spec.yaml", &motor.RuleSetExecutionResult{}))
	assert.NoError(t, renderExecutionInputErrors("spec.yaml", &motor.RuleSetExecutionResult{Errors: []error{nil}}))

	err := renderExecutionInputErrors("spec.yaml", &motor.RuleSetExecutionResult{
		Errors: []error{
			&motor.UnknownRuleFunctionError{RuleID: "check-title", Function: "patternd"},
			nil,
			errors.New("another execution error"),
		},
	})
	require.Error(t, err)
	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, ExitCodeInputError, exitErr.Code)
	assert.Equal(t, "linting failed due to 2 issues", exitErr.Error())
}

func TestRenderPreParseExecutionErrors(t *testing.T) {
	functionErr := &motor.UnknownRuleFunctionError{RuleID: "check-title", Function: "patternd"}
	assert.NoError(t, renderPreParseExecutionErrors("spec.yaml", nil))
	assert.NoError(t, renderPreParseExecutionErrors("spec.yaml", &motor.RuleSetExecutionResult{
		SpecInfo: &datamodel.SpecInfo{},
		Errors:   []error{functionErr},
	}))
	require.Error(t, renderPreParseExecutionErrors("spec.yaml", &motor.RuleSetExecutionResult{
		Errors: []error{functionErr},
	}))
}
