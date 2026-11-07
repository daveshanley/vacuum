// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"encoding/json"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestReviewedExampleGeneration(t *testing.T) {
	for _, tc := range []struct{ name, rule, example string }{
		{"invalid", "oas3-valid-schema-example", `,"example":{"id":"wrong","status":"unknown"}`},
		{"missing", "oas3-missing-example", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`{"openapi":"3.1.0","info":{"title":"Example","version":"1"},"paths":{},"components":{"schemas":{"Sample":{"type":"object","required":["id","status"],"properties":{"id":{"type":"integer","minimum":1,"maximum":10},"status":{"type":"string","enum":["active","inactive"]}}` + tc.example + `}}}}`)
			execution := &motor.RuleSetExecution{Spec: source, SpecFileName: "example.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule: rulesets.GetAllBuiltInRules()[tc.rule]}}}
			review, err := NewFixReview(execution, nil)
			require.NoError(t, err)
			defer review.Close()
			var choices []FixChoice
			for _, o := range review.Options() {
				require.False(t, o.Selected)
				if o.Available {
					choices = append(choices, FixChoice{Index: o.Index})
				}
			}
			require.NotEmpty(t, choices)
			preview, err := review.Preview(choices)
			require.NoError(t, err)
			defer preview.Close()
			require.Empty(t, preview.Results)
			again, err := review.Preview(choices)
			require.NoError(t, err)
			defer again.Close()
			require.Equal(t, preview.Candidate, again.Candidate)
			require.Equal(t, source, execution.Spec)
		})
	}
}

func TestReviewedAsyncMessageExample(t *testing.T) {
	source := []byte(`asyncapi: 3.0.0
info:
  title: Samples
  version: '1'
channels: {}
components:
  messages:
    Event:
      payload:
        type: object
        required: [id]
        properties:
          id: {type: integer, minimum: 1, maximum: 10}
      examples:
        - summary: Generated sample
`)
	id := "asyncapi-message-examples"
	execution := &motor.RuleSetExecution{Spec: source, SpecFileName: "message.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllAsyncAPIRules()[id]}}}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 1)
	o := review.Options()[0]
	require.True(t, o.Available, o.Reason)
	p, err := review.Preview([]FixChoice{{Index: o.Index}})
	require.NoError(t, err)
	defer p.Close()
	require.Empty(t, p.Results)
	require.Contains(t, string(p.Candidate), "summary: Generated sample")
	require.Contains(t, string(p.Candidate), "payload:")
	require.Equal(t, RepairImpactUnknown, p.Impact.Status)
}

func TestReviewedJSONSchemaExamplePreservesDefault(t *testing.T) {
	id := "json-schema-examples-valid"
	source := []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"integer","minimum":1,"maximum":10,"default":2,"examples":["wrong"]}`)
	execution := &motor.RuleSetExecution{Spec: source, SpecFormat: model.JSONSchemaDraft2020, SkipDocumentCheck: true, SpecFileName: "sample.schema.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetJSONSchemaRecommendedRules()[id]}}}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	options := review.Options()
	require.Len(t, options, 1)
	require.True(t, options[0].Available, options[0].Reason)
	p, err := review.Preview([]FixChoice{{Index: options[0].Index}})
	require.NoError(t, err)
	defer p.Close()
	require.Empty(t, p.Results)
	var document map[string]any
	require.NoError(t, json.Unmarshal(p.Candidate, &document))
	require.Equal(t, float64(2), document["default"])
}

func TestReviewedOpenAPI30ExampleUsesNativeDialect(t *testing.T) {
	id := "oas3-valid-schema-example"
	source := []byte(`{"openapi":"3.0.3","info":{"title":"Examples","version":"1"},"paths":{},"components":{"schemas":{"Value":{"type":"integer","minimum":1,"maximum":10,"example":"wrong"}}}}`)
	review, err := NewFixReview(&motor.RuleSetExecution{Spec: source, SpecFileName: "sample.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}, nil)
	require.NoError(t, err)
	defer review.Close()
	options := review.Options()
	require.NotEmpty(t, options)
	require.True(t, options[0].Available, options[0].Reason)
	p, err := review.Preview([]FixChoice{{Index: options[0].Index}})
	require.NoError(t, err)
	defer p.Close()
	require.Empty(t, p.Results)
}
