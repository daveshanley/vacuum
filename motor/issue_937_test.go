// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

const issue937Root = "test_data/issue_937/openapi.yaml"

type issue937Finding struct {
	message string
	path    string
	origin  string
}

func TestRuleSetExecution_Issue937_ExternalResponseExamplesAtSameCoordinates(t *testing.T) {
	firstResponse, err := os.ReadFile("test_data/issue_937/responses/first.yaml")
	require.NoError(t, err)
	secondResponse, err := os.ReadFile("test_data/issue_937/responses/second.yaml")
	require.NoError(t, err)
	require.Equal(t, firstResponse, secondResponse,
		"whitespace and ordering in the external responses are part of the regression setup")

	split := runIssue937ExampleValidation(t, issue937Root, true)
	require.Len(t, split, 2)
	require.Equal(t, []string{"responses/first.yaml", "responses/second.yaml"}, issue937Origins(split))
	require.Equal(t, []string{
		"$.paths['/first'].get.responses['200'].content['application/json'].example",
		"$.paths['/second'].get.responses['200'].content['application/json'].example",
	}, issue937Paths(split))

	bundled := runIssue937ExampleValidation(t, "test_data/issue_937/openapi-bundled.yaml", false)
	require.Len(t, bundled, 2)
	require.Equal(t, issue937Messages(split), issue937Messages(bundled))
	require.Equal(t, issue937Paths(split), issue937Paths(bundled))
}

func runIssue937ExampleValidation(t *testing.T, specPath string, allowLookup bool) []issue937Finding {
	t.Helper()

	spec, err := os.ReadFile(specPath)
	require.NoError(t, err)
	exampleRule := rulesets.GetOAS3ExamplesRule()
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{exampleRule.Id: exampleRule},
		},
		Spec:         spec,
		SpecFileName: specPath,
		AllowLookup:  allowLookup,
		SilenceLogs:  true,
	})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)

	findings := make([]issue937Finding, 0, len(result.Results))
	for i := range result.Results {
		ruleResult := &result.Results[i]
		if ruleResult.RuleId != rulesets.Oas3ValidSchemaExample {
			continue
		}
		require.Equal(t, "got string, want integer", ruleResult.Message)
		if allowLookup {
			require.NotNil(t, ruleResult.Origin)
		}
		var origin string
		if ruleResult.Origin != nil {
			origin = ruleResult.Origin.AbsoluteLocation
			if origin == "" {
				origin = ruleResult.Origin.AbsoluteLocationValue
			}
		}
		findings = append(findings, issue937Finding{
			message: ruleResult.Message,
			path:    ruleResult.Path,
			origin:  filepath.ToSlash(origin),
		})
	}
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].path < findings[j].path
	})
	return findings
}

func issue937Messages(findings []issue937Finding) []string {
	messages := make([]string, len(findings))
	for i := range findings {
		messages[i] = findings[i].message
	}
	return messages
}

func issue937Paths(findings []issue937Finding) []string {
	paths := make([]string, len(findings))
	for i := range findings {
		paths[i] = findings[i].path
	}
	return paths
}

func issue937Origins(findings []issue937Finding) []string {
	origins := make([]string, len(findings))
	for i := range findings {
		switch {
		case strings.HasSuffix(findings[i].origin, "responses/first.yaml"):
			origins[i] = "responses/first.yaml"
		case strings.HasSuffix(findings[i].origin, "responses/second.yaml"):
			origins[i] = "responses/second.yaml"
		default:
			origins[i] = findings[i].origin
		}
	}
	sort.Strings(origins)
	return origins
}
