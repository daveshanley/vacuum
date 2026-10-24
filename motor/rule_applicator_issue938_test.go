// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/pb33f/testify/require"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

func TestIssue938_ConcurrentRootSelectorsRemainMutuallyExclusive(t *testing.T) {
	// The original failure depended on concurrent rule scheduling, so exercise
	// the same ruleset repeatedly instead of relying on a single lucky run.
	const runCount = 50

	spec := []byte(`openapi: 3.0.3
info:
  title: Issue 938
  version: 1.0.0
  x-api-type: pygeoapi
servers:
  - url: http://example.com
paths: {}
`)

	recommended := rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet()
	rules := make(map[string]*model.Rule, len(recommended.Rules)+3)
	for id, rule := range recommended.Rules {
		rules[id] = rule
	}
	rules["may-have-info-x-api-type"] = &model.Rule{
		Id:           "may-have-info-x-api-type",
		Given:        "$.info",
		Resolved:     true,
		Severity:     model.SeverityInfo,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         rulesets.Validation,
		Then: []model.RuleAction{
			{
				Field:    "x-api-type",
				Function: "defined",
			},
			{
				Field:    "x-api-type",
				Function: "schema",
				FunctionOptions: map[string]any{
					"schema": map[string]any{
						"type": "string",
						"enum": []any{"standard", "pygeoapi"},
					},
				},
			},
		},
	}
	rules["must-use-https-protocol-only"] = issue938ProtocolRule(
		"must-use-https-protocol-only",
		`$.servers..[?(@property === "url" && @root.info["x-api-type"] !== "pygeoapi")]`,
		model.SeverityError,
	)
	rules["must-use-https-protocol-only-pygeoapi"] = issue938ProtocolRule(
		"must-use-https-protocol-only-pygeoapi",
		`$.servers..[?(@property === "url" && @root.info["x-api-type"] === "pygeoapi")]`,
		model.SeverityWarn,
	)

	ruleSet := &rulesets.RuleSet{Rules: rules}
	for run := range runCount {
		results := ApplyRulesToRuleSet(&RuleSetExecution{
			RuleSet:     ruleSet,
			Spec:        spec,
			SilenceLogs: true,
		})

		if len(results.Errors) > 0 {
			errs := append([]error(nil), results.Errors...)
			results.Release()
			t.Fatalf("run %d: unexpected execution errors: %v", run, errs)
		}

		var relaxedResults, strictResults int
		for _, result := range results.Results {
			switch result.RuleId {
			case "must-use-https-protocol-only-pygeoapi":
				relaxedResults++
			case "must-use-https-protocol-only":
				strictResults++
			}
		}
		results.Release()

		if strictResults != 0 || relaxedResults != 1 {
			t.Fatalf(
				"run %d: mutually exclusive selectors produced %d strict and %d relaxed results; expected 0 strict and 1 relaxed",
				run, strictResults, relaxedResults,
			)
		}
	}
}

func issue938ProtocolRule(id, given, severity string) *model.Rule {
	return &model.Rule{
		Id:           id,
		Given:        given,
		Resolved:     true,
		Severity:     severity,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         rulesets.Validation,
		Then: model.RuleAction{
			Function: "pattern",
			FunctionOptions: map[string]any{
				"match": `^https:/`,
			},
		},
	}
}

func TestIssue938MergeKeyExclusiveFilters(t *testing.T) {
	supplied, err := rulesets.CreateRuleSetFromData([]byte(`rules:
  must-use-https-protocol-only: &https
    given: $.servers..[?(@property === "url" && @root.info["x-api-type"] !== "pygeoapi")]
    severity: error
    then:
      function: pattern
      functionOptions:
        match: "^https:/"
  must-use-https-protocol-only-pygeoapi:
    <<: *https
    given: $.servers..[?(@property === "url" && @root.info["x-api-type"] === "pygeoapi")]
    severity: warn
`))
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(supplied)
	previous := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(previous)
	for _, procs := range []int{1, 4} {
		runtime.GOMAXPROCS(procs)
		for _, tc := range []struct{ apiType, scheme, rule, severity string }{
			{"pygeoapi", "http", "must-use-https-protocol-only-pygeoapi", model.SeverityWarn},
			{"other", "http", "must-use-https-protocol-only", model.SeverityError},
			{"pygeoapi", "https", "", ""},
			{"other", "https", "", ""},
		} {
			t.Run(fmt.Sprintf("%d/%s/%s", procs, tc.apiType, tc.scheme), func(t *testing.T) {
				for range 5 {
					spec := []byte(fmt.Sprintf("openapi: 3.0.3\ninfo:\n  title: Example\n  version: 1.0.0\n  x-api-type: %s\nservers:\n  - url: %s://example.com\npaths: {}\n", tc.apiType, tc.scheme))
					result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: spec, RuleSet: rs, SilenceLogs: true})
					require.Empty(t, result.Errors)
					if tc.rule == "" {
						require.Empty(t, result.Results)
					} else {
						require.Len(t, result.Results, 1)
						require.Equal(t, tc.rule, result.Results[0].Rule.Id)
						require.Equal(t, tc.severity, result.Results[0].Rule.Severity)
					}
					result.ReleaseOwnedResources()
				}
			})
		}
	}
}
