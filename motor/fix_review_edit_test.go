package motor

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestReviewedNullableProposalKeepsJSONNullAndCanBeEdited(t *testing.T) {
	id := "nullable-enum-contains-null"
	execution := &RuleSetExecution{Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Null","version":"1"},"paths":{},"components":{"schemas":{"Value":{"type":["string","null"],"enum":["one"]}}}}`), SpecFileName: "null.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	options := review.Options()
	require.Len(t, options, 1)
	option := options[0]
	require.True(t, option.Available)
	require.False(t, option.Selected)
	require.True(t, option.Editable)
	require.Equal(t, "null", option.Value)
	preview, err := review.Preview([]FixChoice{{Index: option.Index}})
	require.NoError(t, err)
	defer preview.Close()
	require.Empty(t, preview.Results)
	bad := `"null"`
	_, err = review.Preview([]FixChoice{{Index: option.Index, Value: &bad}})
	require.Error(t, err)
	require.Contains(t, string(preview.Candidate), "null")
}

func TestReviewedExampleEditsMustSatisfySchema(t *testing.T) {
	for _, tc := range []struct{ name, id, source string }{
		{"async", "asyncapi-message-examples", `{"asyncapi":"3.0.0","info":{"title":"Samples","version":"1"},"channels":{},"components":{"messages":{"Event":{"payload":{"type":"object","required":["id"],"properties":{"id":{"type":"integer"}}},"examples":[{"summary":"sample"}]}}}}`},
		{"missing", "oas3-missing-example", `{"openapi":"3.1.0","info":{"title":"Samples","version":"1"},"paths":{},"components":{"schemas":{"Value":{"type":"array","items":{"type":"integer"},"minItems":1}}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := rulesets.GetAllBuiltInRules()[tc.id]
			if tc.name == "async" {
				rule = rulesets.GetAllAsyncAPIRules()[tc.id]
			}
			review, err := NewFixReview(&RuleSetExecution{Spec: []byte(tc.source), SpecFileName: "sample.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.id: rule}}}, nil)
			require.NoError(t, err)
			defer review.Close()
			options := review.Options()
			require.NotEmpty(t, options)
			option := options[0]
			require.True(t, option.Editable)
			bad := `"not a valid sample"`
			preview, err := review.Preview([]FixChoice{{Index: option.Index, Value: &bad}})
			require.ErrorContains(t, err, "does not satisfy")
			require.Nil(t, preview)
		})
	}
}

func TestReviewedCompoundIdentifierCannotEditOneSide(t *testing.T) {
	id := "operation-operationId-valid-in-url"
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Links","version":"1"},"paths":{"/pets":{"get":{"operationId":"get pets","responses":{"200":{"description":"OK","links":{"self":{"operationId":"get pets"}}}}}}}}`)
	review, err := NewFixReview(&RuleSetExecution{Spec: source, SpecFileName: "links.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}, nil)
	require.NoError(t, err)
	defer review.Close()
	option := review.Options()[0]
	require.True(t, option.Available)
	require.False(t, option.Editable)
	changed := "customPets"
	preview, err := review.Preview([]FixChoice{{Index: option.Index, Value: &changed}})
	require.Error(t, err)
	require.Nil(t, preview)
	preview, err = review.Preview([]FixChoice{{Index: option.Index}})
	require.NoError(t, err)
	defer preview.Close()
	require.NotContains(t, string(preview.Candidate), "get pets")
}
