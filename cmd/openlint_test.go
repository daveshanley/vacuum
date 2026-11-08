// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenLintIncompleteExecutionIsNotClean(t *testing.T) {
	for _, err := range []error{
		&motor.RuleExecutionError{Kind: motor.RuleErrorFunction, Err: errors.New("function failed")},
		&motor.RuleExecutionError{Kind: motor.RuleErrorPanic, Err: errors.New("panic")},
		&motor.RuleTimeoutError{RuleId: "r"}, context.DeadlineExceeded, context.Canceled, errors.New("invalid field selector"),
	} {
		result := &motor.RuleSetExecutionResult{RuleSetExecution: &motor.RuleSetExecution{RuleSet: &rulesets.RuleSet{Contract: model.OpenLintV1Draft}}, Errors: []error{err}}
		require.ErrorIs(t, ruleSetConfigurationError(result), err)
	}
}

func TestOpenLintSpectralReportRejectsMalformedSelectors(t *testing.T) {
	for _, rule := range []string{
		`rules: {r: {given: '$[', then: {function: undefined}}}`,
		`rules: {r: {given: '$', then: {field: '$[', function: undefined}}}`,
	} {
		dir := t.TempDir()
		input := filepath.Join(dir, "input.yaml")
		rules := filepath.Join(dir, "rules.yaml")
		output := filepath.Join(dir, "report.json")
		require.NoError(t, os.WriteFile(input, []byte("x: hello\n"), 0600))
		require.NoError(t, os.WriteFile(rules, []byte(rule), 0600))
		cmd := GetSpectralReportCommand()
		cmd.PersistentFlags().Bool("openlint", false, "")
		cmd.PersistentFlags().StringP("ruleset", "r", "", "")
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"--openlint", "-r", rules, input, output})
		require.Error(t, cmd.Execute())
		_, err := os.Stat(output)
		require.ErrorIs(t, err, os.ErrNotExist)
	}
}
