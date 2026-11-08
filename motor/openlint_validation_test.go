// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestOpenLintValidatesDisabledDefinitions(t *testing.T) {
	for _, state := range []string{"severity: off", "recommended: false"} {
		t.Run(state, func(t *testing.T) {
			raw, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  r:\n    "+state+"\n    given: $\n    then: {function: pattern, functionOptions: {match: 42}}\n"), model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			require.NoError(t, rs.LoadError())
			require.Empty(t, rs.Rules)
			errs := ValidateRuleSet(rs, nil)
			require.NotEmpty(t, errs)
			require.ErrorContains(t, errs[0], "must be a string")
			require.Empty(t, rs.Rules, "preflight must not activate a disabled definition")
		})
	}
}

func TestOpenLintValidatesInheritedOffDefinitions(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte("rules: {r: {given: '$', then: {function: pattern, functionOptions: {match: 42}}}}"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rules.yaml"), []byte("extends: [[./base.yaml, off]]\nrules: {}"), 0600))
	raw, err := rulesets.LoadLocalRuleSet(context.Background(), filepath.Join(dir, "rules.yaml"), model.OpenLintV1Draft)
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	require.NoError(t, rs.LoadError())
	require.Empty(t, rs.Rules)
	errs := ValidateRuleSet(rs, nil)
	require.NotEmpty(t, errs)
	require.ErrorContains(t, errs[0], "must be a string")
}

func TestOpenLintValidDisabledRuleDoesNotExecute(t *testing.T) {
	raw, err := rulesets.CreateRuleSetFromData([]byte("rules: {r: {severity: off, given: '$', then: {function: falsy}}}"), model.OpenLintV1Draft)
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	require.NoError(t, rs.LoadError())
	require.Empty(t, ValidateRuleSet(rs, nil))
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("{nonempty: true}"), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Empty(t, result.Results)
}
