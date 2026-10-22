// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"sort"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// TestIssue526DeepGraphSharedSchemaPathIsStable pins vacuum issue 526: with
// BuildDeepGraph the schema cache is off and a worker pool can attach a shared
// schema to whichever parent finishes first. A constraint on a component schema
// $ref'd from two operations must report the same path set every run.
func TestIssue526DeepGraphSharedSchemaPathIsStable(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: shared schema
  version: "1.0.0"
paths:
  /widgets:
    get:
      operationId: getWidgets
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Shared"
    post:
      operationId: createWidget
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Shared"
components:
  schemas:
    Shared:
      type: object
      properties:
        name:
          type: string
          minLength: -1
`)

	rule := rulesets.GetSchemaTypeCheckRule()
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	var first []string
	for i := 0; i < 2; i++ {
		results := ApplyRulesToRuleSet(&RuleSetExecution{
			RuleSet:        ruleSet,
			Spec:           spec,
			BuildDeepGraph: true,
			SilenceLogs:    true,
		})
		require.NotNil(t, results)

		paths := minLengthFindingPaths(results)
		require.NotEmpty(t, paths, "iteration %d", i)
		if i == 0 {
			first = paths
			continue
		}
		assert.Equal(t, first, paths)
	}
}

func minLengthFindingPaths(results *RuleSetExecutionResult) []string {
	seen := map[string]struct{}{}
	var paths []string
	add := func(path string) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for _, result := range results.Results {
		if !strings.Contains(result.Message, "minLength") {
			continue
		}
		add(result.Path)
		for _, path := range result.Paths {
			add(path)
		}
	}
	sort.Strings(paths)
	return paths
}
