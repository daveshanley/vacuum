// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestOpenLintJavaScriptDocumentAndMissingFields(t *testing.T) {
	cases := []struct {
		name, document, parserOptions, field, message string
		parserCount                                   int
	}{
		{"duplicate-off", "info:\n  title: first\n  title: second\n", "duplicateKeys: off", "", "object:second:[\"info\"]", 0},
		{"duplicate-warn", "info:\n  title: first\n  title: second\n", "duplicateKeys: warn", "", "object:second:[\"info\"]", 1},
		{"missing-field", "info: {title: known}\n", "", "description", "undefined:known:[\"info\"]", 0},
		{"missing-wildcard", "info: {title: known}\n", "", "$.missing[*]", "undefined:known:[\"info\"]", 0},
		{"missing-bracket", "info: {title: known}\n", "", "'$[\"missing\"]'", "undefined:known:[\"info\"]", 0},
		{"missing-index", "info: {title: known, values: []}\n", "", "$.values[1]", "undefined:known:[\"info\"]", 0},
		{"present-null", "info: {title: known, description: null}\n", "", "description", "object:known:[\"info\",\"description\"]", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "functions"), 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "functions", "check.js"), []byte(`export default (value, options, context) => [{message: typeof value + ':' + context.document.data.info.title + ':' + JSON.stringify(context.path)}];`), 0600))
			rules := fmt.Sprintf("parserOptions: {%s}\nfunctions: [check]\nrules:\n  check:\n    given: $.info\n    resolved: false\n    then:\n      function: check\n", tc.parserOptions)
			if tc.field != "" {
				rules += "      field: " + tc.field + "\n"
			}
			path := filepath.Join(dir, "rules.yaml")
			require.NoError(t, os.WriteFile(path, []byte(rules), 0600))
			raw, err := rulesets.LoadLocalRuleSetWithContract(context.Background(), path, model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			require.NoError(t, rs.LoadError())
			result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte(tc.document), SpecFileName: filepath.Join(dir, "input.yaml"), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft, RunTimeout: 5 * time.Second})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1+tc.parserCount)
			var checks, parsers int
			for _, finding := range result.Results {
				if finding.RuleId == "parser" {
					parsers++
					require.Equal(t, model.SeverityWarn, finding.RuleSeverity)
					continue
				}
				checks++
				require.Equal(t, tc.message, finding.Message)
				path := "$.info"
				if tc.name == "present-null" {
					path += ".description"
				}
				require.Equal(t, path, finding.Path)
			}
			require.Equal(t, 1, checks)
			require.Equal(t, tc.parserCount, parsers)
		})
	}
}

func TestOpenLintJavaScriptResolvedExplicitSourcePath(t *testing.T) {
	for _, tc := range []struct{ name, path, want string }{
		{"reference", `["Use","description"]`, "$.Target.description"},
		{"container", `["Use"]`, "$.Target"},
		{"missing", `["Use","missing"]`, "$.Target"},
		{"other", `["Other","description"]`, "$.Other.description"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "functions"), 0700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "functions", "check.js"), []byte(`export default () => [{message: "check",path:`+tc.path+`}];`), 0600))
			path := filepath.Join(dir, "rules.yaml")
			require.NoError(t, os.WriteFile(path, []byte("functions: [check]\nrules:\n  check: {given: $.Use, then: {function: check}}\n"), 0600))
			raw, err := rulesets.LoadLocalRuleSetWithContract(context.Background(), path, model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("Target: {description: target}\nUse: {$ref: '#/Target'}\nOther: {description: other}\n"), SpecFileName: filepath.Join(dir, "api.yaml"), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			require.Equal(t, tc.want, result.Results[0].Path)
		})
	}
}
