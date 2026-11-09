// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestOpenLintRequiresExplicitRuleset(t *testing.T) {
	command := &cobra.Command{Use: "lint"}
	command.Flags().Bool("openlint", true, "")
	command.Flags().String("ruleset", "", "")
	require.ErrorContains(t, validateOpenLintCommand(command), "requires --ruleset")
	require.NoError(t, command.Flags().Set("ruleset", "rules.yaml"))
	require.NoError(t, validateOpenLintCommand(command))
	require.NoError(t, command.Flags().Set("ruleset", ""))
	require.NoError(t, command.Flags().Set("openlint", "false"))
	require.NoError(t, validateOpenLintCommand(command))
}

// Existing consumers can use loader functions as callbacks without opting in.
var (
	_ func([]byte) (*rulesets.RuleSet, error)                                                = rulesets.CreateRuleSetFromData
	_ func([]byte) (*rulesets.RuleSet, error)                                                = rulesets.CreateRuleSetUsingJSON
	_ func(context.Context, string) (*rulesets.RuleSet, error)                               = rulesets.LoadLocalRuleSet
	_ func(context.Context, string, func(string) ([]byte, error)) (*rulesets.RuleSet, error) = rulesets.LoadLocalRuleSetWithReader
	_ func(context.Context, string, *http.Client) (*rulesets.RuleSet, error)                 = rulesets.DownloadRemoteRuleSet
	_ func(string, rulesets.RuleSets, bool, *http.Client) (*rulesets.RuleSet, error)         = BuildRuleSetFromUserSuppliedLocation
)

func TestNativeBuildResultsKeepsConfigurationErrorsInExecution(t *testing.T) {
	dir := t.TempDir()
	rules := filepath.Join(dir, "rules.mjs")
	require.NoError(t, os.WriteFile(rules, []byte(`export default {rules:{r:{given:'',then:{function:'truthy'}}}};`), 0600))
	spec := []byte("openapi: 3.0.3\ninfo: {title: test, version: '1'}\npaths: {}\n")
	results, execution, err := BuildResults(true, false, rules, spec, nil, "", false, 5*time.Second, 5*time.Second, utils.HTTPClientConfig{}, nil, model.IgnoredItems{}, nil)
	require.NoError(t, err)
	require.NotNil(t, results)
	require.NotNil(t, execution)
	defer execution.ReleaseOwnedResources()
	require.NotEmpty(t, execution.Errors)
	require.Contains(t, execution.Errors[0].Error(), "empty selector")
}
