// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestMetadataReviewKeepsUnchangedNativeSchemaErrors(t *testing.T) {
	source := `openapi: 3.0.3
info: {title: Test, version: '1'}
paths:
  /first:
    get:
      parameters:
        - name: filter
          in: query
          schema: {type: string, pattern: '['}
      responses: {'200': {description: OK}}
  /second:
    get:
      parameters:
        - name: filter
          in: query
          schema: {type: string, pattern: '['}
      responses: {'200': {description: OK}}
components:
  schemas:
    Item: {type: string}
`
	builtins := rulesets.GetAllBuiltInRules()
	for i := 0; i < 8; i++ {
		r, err := NewFixReview(&motor.RuleSetExecution{Spec: []byte(source), SpecFileName: "metadata.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"oas3-schema": builtins["oas3-schema"], "component-description": builtins["component-description"]}}}, nil)
		require.NoError(t, err)
		var choices []FixChoice
		beforeErrors := 0
		for _, o := range r.Options() {
			if o.Result.RuleId == "oas3-schema" {
				beforeErrors++
			}
			if o.Available && o.Result.RuleId == "component-description" {
				choices = append(choices, FixChoice{Index: o.Index})
			}
		}
		require.Positive(t, beforeErrors)
		require.Len(t, choices, 1)
		p, err := r.Preview(choices)
		require.NoError(t, err)
		require.Equal(t, 1, p.Fixed)
		afterErrors := 0
		for _, f := range p.Results {
			if f.RuleId == "oas3-schema" {
				afterErrors++
			}
		}
		require.Equal(t, beforeErrors, afterErrors)
		p.Close()
		r.Close()
	}
}

func TestMetadataReviewDeclinesOAS30ReferenceOwner(t *testing.T) {
	source := `openapi: 3.0.3
info: {title: Test, version: '1'}
paths: {}
components:
  schemas:
    Item: {type: string}
    Alias: {$ref: '#/components/schemas/Item'}
`
	builtins := rulesets.GetAllBuiltInRules()
	r, err := NewFixReview(&motor.RuleSetExecution{Spec: []byte(source), SpecFileName: "metadata.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"component-description": builtins["component-description"], "no-$ref-siblings": builtins["no-$ref-siblings"]}}}, nil)
	require.NoError(t, err)
	defer r.Close()
	found := false
	for _, o := range r.Options() {
		if strings.Contains(o.Result.Path, "Alias") {
			found = true
			require.False(t, o.Available)
			require.False(t, o.Editable)
			require.Contains(t, o.Reason, "referenced source")
		}
	}
	require.True(t, found)
}
