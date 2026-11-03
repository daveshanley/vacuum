package motor

import (
	"encoding/json"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func reviewFixture() *RuleSetExecution {
	builtins := rulesets.GetAllBuiltInRules()
	rules := make(map[string]*model.Rule)
	for _, id := range []string{"info-description", "operation-description", "duplicated-entry-in-enum"} {
		rules[id] = builtins[id]
	}
	return &RuleSetExecution{SpecFileName: "review.json", SilenceLogs: true, Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a"]}}}}`), RuleSet: &rulesets.RuleSet{Rules: rules}}
}

func TestFixReviewEditTwoValuesAndSkipMechanical(t *testing.T) {
	execution := reviewFixture()
	original := string(execution.Spec)
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	var choices []FixChoice
	for _, option := range review.Options() {
		if !option.Editable {
			continue
		}
		value := "Edited " + option.Result.RuleId
		choices = append(choices, FixChoice{Index: option.Index, Value: &value})
	}
	require.Len(t, choices, 2)
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 2, preview.Fixed)
	require.Len(t, preview.Results, 1)
	require.Equal(t, "duplicated-entry-in-enum", preview.Results[0].RuleId)
	require.Equal(t, original, string(execution.Spec), "preview must not alter source")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(preview.Candidate, &parsed))
	require.Equal(t, "Edited info-description", parsed["info"].(map[string]any)["description"])
	get := parsed["paths"].(map[string]any)["/pets"].(map[string]any)["get"].(map[string]any)
	require.Equal(t, "Edited operation-description", get["description"])
	// A new value produces a new candidate, not a mutation of the prior preview.
	changed := "A different description"
	choices[0].Value = &changed
	next, err := review.Preview(choices)
	require.NoError(t, err)
	defer next.Close()
	require.NotEqual(t, string(preview.Candidate), string(next.Candidate))
	require.NotContains(t, string(preview.Candidate), changed)
}

func TestFixReviewMechanicalSubsetAndRepeat(t *testing.T) {
	execution := reviewFixture()
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	var choices []FixChoice
	for _, option := range review.Options() {
		if option.Selected {
			choices = append(choices, FixChoice{Index: option.Index})
		}
	}
	require.Len(t, choices, 1)
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 1, preview.Fixed)
	require.Len(t, preview.Results, 2)
	execution.Spec = preview.Candidate
	next, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer next.Close()
	for _, option := range next.Options() {
		require.False(t, option.Selected)
	}
	review.Close()
	_, err = review.Preview(choices)
	require.ErrorContains(t, err, "closed")
	review.Close()
}

func TestFixReviewRejectsInvalidSelectionAndValues(t *testing.T) {
	review, err := NewFixReview(reviewFixture(), nil)
	require.NoError(t, err)
	defer review.Close()
	_, err = review.Preview(nil)
	require.ErrorContains(t, err, "select at least")
	_, err = review.Preview([]FixChoice{{Index: -1}})
	require.ErrorContains(t, err, "invalid")
	for _, option := range review.Options() {
		_, err = review.Preview([]FixChoice{{Index: option.Index}, {Index: option.Index}})
		require.Error(t, err)
		if option.Editable {
			empty := " "
			_, err = review.Preview([]FixChoice{{Index: option.Index, Value: &empty}})
			require.ErrorContains(t, err, "nonempty")
		} else if option.Selected {
			value := "override"
			_, err = review.Preview([]FixChoice{{Index: option.Index, Value: &value}})
			require.ErrorContains(t, err, "cannot be edited")
		}
	}
}

func TestFixReviewMetadataObjects(t *testing.T) {
	for _, tc := range []struct {
		field, value string
		valid        bool
	}{
		{"contact", `{"email":"team@example.com"}`, true},
		{"license", `{"name":"MIT"}`, true},
		{"contact", `"team@example.com"`, false},
		{"contact", `{"email":true}`, false},
		{"contact", `{"email":" "}`, false},
		{"contact", `{"$ref":"https://example.com"}`, false},
		{"license", `{"url":"https://example.com"}`, false},
		{"license", `{}`, false},
	} {
		t.Run(tc.field+tc.value, func(t *testing.T) {
			_, err := fixValue(tc.field, tc.value)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestFixReviewRejectsNewDiagnosticAndPreservesChoices(t *testing.T) {
	execution := reviewFixture()
	execution.RuleSet.Rules["description-duplication"] = rulesets.GetAllBuiltInRules()["description-duplication"]
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	var choices []FixChoice
	duplicate := "Identical description"
	for _, option := range review.Options() {
		if option.Editable {
			choices = append(choices, FixChoice{Index: option.Index, Value: &duplicate})
		}
	}
	preview, err := review.Preview(choices)
	require.ErrorContains(t, err, "introduces diagnostic")
	require.Nil(t, preview)
	require.Len(t, review.Options(), 3)
	for i := range choices {
		value := "Independent description " + string(rune('A'+i))
		choices[i].Value = &value
	}
	preview, err = review.Preview(choices)
	require.NoError(t, err)
	preview.Close()
}

func TestFixReviewRetainsOriginalSnapshot(t *testing.T) {
	execution := reviewFixture()
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	for i := range execution.Spec {
		execution.Spec[i] = ' '
	}
	var choices []FixChoice
	for _, option := range review.Options() {
		if option.Selected {
			choices = append(choices, FixChoice{Index: option.Index})
		}
	}
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.True(t, json.Valid(preview.Candidate))
	require.Contains(t, string(preview.Candidate), "Review")
}
