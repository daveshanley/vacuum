// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExitError_Error(t *testing.T) {
	e := &ExitError{Code: 2, Message: "something broke"}
	assert.Equal(t, "something broke", e.Error())
}

func TestNewInputError(t *testing.T) {
	e := NewInputError("file '%s' not found", "api.yaml")
	assert.Equal(t, ExitCodeInputError, e.Code)
	assert.Equal(t, "file 'api.yaml' not found", e.Message)
}

func TestNewViolationError(t *testing.T) {
	e := NewViolationError("failed with %d errors", 5)
	assert.Equal(t, ExitCodeViolations, e.Code)
	assert.Equal(t, "failed with 5 errors", e.Message)
}

func TestExitError_ErrorsAs(t *testing.T) {
	err := NewInputError("bad input")
	var exitErr *ExitError
	assert.True(t, errors.As(err, &exitErr))
	assert.Equal(t, ExitCodeInputError, exitErr.Code)
}

func TestExitCodeConstants(t *testing.T) {
	assert.Equal(t, 0, ExitCodeSuccess)
	assert.Equal(t, 1, ExitCodeViolations)
	assert.Equal(t, 2, ExitCodeInputError)
}

func TestCheckFailureSeverity_ReturnsExitError(t *testing.T) {
	err := CheckFailureSeverity("error", 3, 0, 0)
	assert.Error(t, err)
	var exitErr *ExitError
	assert.ErrorAs(t, err, &exitErr)
	assert.Equal(t, ExitCodeViolations, exitErr.Code)
}

func TestCheckFailureSeverity_NoErrors(t *testing.T) {
	err := CheckFailureSeverity("error", 0, 5, 3)
	assert.NoError(t, err)
}

func TestCheckFailureSeverity_None(t *testing.T) {
	err := CheckFailureSeverity("none", 10, 20, 30)
	assert.NoError(t, err)
}
