// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"testing"

	"github.com/daveshanley/vacuum/motor"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestFixDescriptionsKeepParameterContextAndAvoidAuthoredDuplicates(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info:
  title: Parameters
  version: "1"
  description: id path parameter for /pets/{id}.
paths:
  /pets/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: string}}
    get:
      parameters:
        - {name: limit, in: query, schema: {type: integer}}
      responses:
        '200': {description: Retrieved pets}
    post:
      parameters:
        - {name: limit, in: query, schema: {type: integer}}
      responses:
        '200': {description: Created pets}
  /users/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: string}}
    get:
      responses:
        '200': {description: Retrieved users}
`)
	rules := map[string]*model.Rule{}
	for _, id := range []string{"oas3-parameter-description", "description-duplication"} {
		rules[id] = rulesets.GetAllBuiltInRules()[id]
		require.NotNil(t, rules[id])
	}
	review, err := NewFixReview(&motor.RuleSetExecution{SpecFileName: "/tmp/parameter-context.yaml", Spec: source, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: rules}}, nil)
	require.NoError(t, err)
	defer review.Close()
	var choices []FixChoice
	values := map[string]bool{}
	for _, option := range review.Options() {
		require.Equal(t, "oas3-parameter-description", option.Result.RuleId)
		require.True(t, option.Available)
		require.True(t, option.Generated)
		require.False(t, values[option.Value], "generated descriptions must be distinct")
		values[option.Value] = true
		choices = append(choices, FixChoice{Index: option.Index})
	}
	require.Len(t, choices, 4)
	require.True(t, values["limit query parameter for GET /pets/{id}."])
	require.True(t, values["limit query parameter for POST /pets/{id}."])
	require.True(t, values["id path parameter for /users/{id}."])
	require.True(t, values["Description of id path parameter for /pets/{id}."])
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 4, preview.Fixed)
	require.Empty(t, preview.Results)
}

func TestFixDescriptionsAvoidSummaryAndGeneratedTextCollisions(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("summary: GET /pets.\ndescription: Description of GET /pets.\n"), &root))
	used := fixDescriptionValues(&root)
	first := uniqueFixDescription("GET /pets.", "description", used)
	require.Equal(t, "Description of GET /pets (2).", first)
	second := uniqueFixDescription("GET /pets.", "summary", used)
	require.Equal(t, "Summary of GET /pets.", second)
	require.NotEqual(t, first, uniqueFixDescription("GET /pets.", "description", used))
}
