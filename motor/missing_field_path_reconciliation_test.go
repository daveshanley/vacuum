// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestCombinedDefinedAndSchemaActionsPreserveMissingFieldPath(t *testing.T) {
	spec := []byte(`openapi: 3.0.0
info:
  title: Example
  version: 1.0.0
paths: {}
`)

	rules := []byte(`extends: [[vacuum:oas, off]]
rules:
  missing-field-path:
    description: Missing field path test.
    message: Missing x-test.
    severity: warn
    given: $.info
    then:
      - field: x-test
        function: defined
      - field: x-test
        function: schema
        functionOptions:
          schema:
            type: boolean
`)

	supplied, err := rulesets.CreateRuleSetFromData(rules)
	require.NoError(t, err)

	selected := rulesets.BuildDefaultRuleSets().
		GenerateRuleSetFromSuppliedRuleSet(supplied)
	require.NoError(t, selected.LoadError())

	outcome := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: selected,
		Spec:    spec,
	})
	defer outcome.Release()

	require.Empty(t, outcome.Errors)
	require.Len(t, outcome.Results, 1)
	require.Equal(t, "missing-field-path", outcome.Results[0].RuleId)
	require.Equal(t, "$.info", outcome.Results[0].Path)
	require.Equal(t, "Missing x-test.", outcome.Results[0].Message)
}
