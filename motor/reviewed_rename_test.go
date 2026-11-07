// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestReviewedPropertyRenameDeclinesUnmigratedData(t *testing.T) {
	object := `{"type":"object","additionalProperties":false,"required":["first_name"],"properties":{"first_name":{"type":"string"}}%s}`
	for _, tc := range []struct{ name, schema string }{
		{"const", fmt.Sprintf(object, `,"const":{"first_name":"Ada"},"example":{"first_name":"Ada"}`)},
		{"enum", fmt.Sprintf(object, `,"enum":[{"first_name":"Ada"}],"example":{"first_name":"Ada"}`)},
		{"parent example", fmt.Sprintf(`{"type":"object","properties":{"user":%s},"example":{"user":{"first_name":"Ada"}}}`, fmt.Sprintf(object, ""))},
		{"parent default", fmt.Sprintf(`{"type":"object","properties":{"user":%s},"default":{"user":{"first_name":"Ada"}}}`, fmt.Sprintf(object, ""))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(reviewedRecipeSchema(tc.schema))
			rule := rulesets.GetAllBuiltInRules()["camel-case-properties"]
			review, err := NewFixReview(&RuleSetExecution{Spec: source, SpecFileName: "rename.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}}, nil)
			require.NoError(t, err)
			defer review.Close()
			options := review.Options()
			require.Len(t, options, 1)
			require.False(t, options[0].Available)
			require.Contains(t, options[0].Reason, "coordinated migration")
			preview, err := review.Preview([]FixChoice{{Index: options[0].Index}})
			require.Nil(t, preview)
			require.ErrorContains(t, err, "unavailable")
			require.Equal(t, source, review.execution.Spec)
		})
	}
}

func TestReviewedPropertyRenameDeclinesMediaExample(t *testing.T) {
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"OK","content":{"application/json":{"schema":{"type":"object","required":["first_name"],"properties":{"first_name":{"type":"string"}}},"example":{"first_name":"Ada"}}}}}}}}}`)
	id := "camel-case-properties"
	review, err := NewFixReview(&RuleSetExecution{Spec: source, SpecFileName: "media.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}, nil)
	require.NoError(t, err)
	defer review.Close()
	options := review.Options()
	require.Len(t, options, 1)
	require.False(t, options[0].Available)
	require.Contains(t, options[0].Reason, "coordinated migration")
}
