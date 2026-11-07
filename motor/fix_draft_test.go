// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestFixDraftGeneratedValuesRequireSelectionAndStayEditable(t *testing.T) {
	execution := reviewFixture()
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	var generated []FixChoice
	for _, option := range review.Options() {
		if !option.Editable {
			continue
		}
		require.True(t, option.Generated)
		require.False(t, option.Selected)
		require.NotEmpty(t, option.Value)
		draft, err := review.Draft(FixChoice{Index: option.Index})
		require.NoError(t, err)
		require.True(t, draft.Generated)
		require.Contains(t, draft.After, option.Value)
		require.NotContains(t, draft.Before, option.Value)
		generated = append(generated, FixChoice{Index: option.Index})
		custom := "Reviewed description for " + option.Result.RuleId
		edited, err := review.Draft(FixChoice{Index: option.Index, Value: &custom})
		require.NoError(t, err)
		require.False(t, edited.Generated)
		require.Contains(t, edited.After, custom)
		require.Equal(t, option.Value, review.Options()[option.Index].Value, "draft must not change the option's default")
	}
	require.Len(t, generated, 2)
	preview, err := review.Preview(generated)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 2, preview.Fixed)
	require.Len(t, preview.Results, 1)
	require.NotContains(t, string(execution.Spec), `"description":"Review API."`)
}

func TestFixDraftDoesNotLintAndMechanicalBeforeAfterUsesSource(t *testing.T) {
	execution := reviewFixture()
	execution.Spec = []byte("openapi: 3.1.0\r\ninfo: {title: Review, version: '1'}\r\npaths: {}\r\ncomponents:\r\n  schemas:\r\n    Pet:\r\n      type: string\r\n      enum: [cat, cat, dog]\r\n")
	counter := &countingRuleFunction{}
	execution.RuleSet.Rules["count"] = &model.Rule{Id: "count", Given: "$", Then: &model.RuleAction{Function: "controlled"}}
	execution.CustomFunctions = map[string]model.RuleFunction{"controlled": counter}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	calls := counter.calls.Load()
	require.Positive(t, calls)
	for _, option := range review.Options() {
		if !option.Available || option.Editable {
			continue
		}
		draft, err := review.Draft(FixChoice{Index: option.Index})
		require.NoError(t, err)
		require.False(t, draft.Generated)
		require.Equal(t, "/components/schemas/Pet/enum", draft.Pointer)
		require.Contains(t, draft.Before, "enum: [cat, cat, dog]")
		require.Contains(t, draft.After, "enum: [cat, dog]")
		require.NotContains(t, draft.Before, "�")
		require.NotContains(t, draft.After, "\r")
	}
	require.Equal(t, calls, counter.calls.Load(), "opening details must not run lint again")
}

func TestFixDraftContactAndLicenseHaveEditableNeutralProposals(t *testing.T) {
	execution := reviewFixture()
	execution.RuleSet.Rules = map[string]*model.Rule{}
	for _, id := range []string{"info-contact", "info-license"} {
		execution.RuleSet.Rules[id] = rulesets.GetAllBuiltInRules()[id]
	}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 2)
	var choices []FixChoice
	for _, option := range review.Options() {
		require.True(t, option.Editable)
		require.True(t, option.Generated)
		require.False(t, option.Selected)
		var value map[string]string
		require.NoError(t, json.Unmarshal([]byte(option.Value), &value))
		require.Len(t, value, 1, "do not invent contact addresses or license grants")
		if option.Result.RuleId == "info-contact" {
			require.Equal(t, "Review support", value["name"])
		} else {
			require.Equal(t, "Unspecified", value["name"])
		}
		draft, err := review.Draft(FixChoice{Index: option.Index})
		require.NoError(t, err)
		require.False(t, draft.NeedsValue)
		require.True(t, draft.Generated)
		require.Contains(t, draft.After, value["name"])
		choices = append(choices, FixChoice{Index: option.Index})
	}
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 2, preview.Fixed)
	require.Empty(t, preview.Results)
}

func TestFixDraftNeutralMetadataCannotIntroduceDiagnostics(t *testing.T) {
	for _, pair := range [][2]string{{"info-contact", "contact-properties"}, {"info-license", "license-url"}} {
		t.Run(pair[0], func(t *testing.T) {
			execution := reviewFixture()
			original := string(execution.Spec)
			execution.RuleSet.Rules = map[string]*model.Rule{}
			for _, id := range pair {
				execution.RuleSet.Rules[id] = rulesets.GetAllBuiltInRules()[id]
			}
			review, err := NewFixReview(execution, nil)
			require.NoError(t, err)
			defer review.Close()
			var selected []FixChoice
			for _, option := range review.Options() {
				if option.Result.RuleId == pair[0] {
					require.True(t, option.Generated)
					selected = append(selected, FixChoice{Index: option.Index})
				}
			}
			require.Len(t, selected, 1)
			preview, err := review.Preview(selected)
			require.ErrorContains(t, err, "introduces diagnostic "+pair[1])
			require.Nil(t, preview)
			require.Equal(t, original, string(execution.Spec))
		})
	}
}

func TestFixDraftOperationIDSuggestionsResolveExistingAndGeneratedCollisions(t *testing.T) {
	execution := reviewFixture()
	execution.Spec = []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"responses":{}}},"/pets/{id}":{"get":{"responses":{}}},"/pets/id":{"get":{"responses":{}}},"/existing":{"get":{"operationId":"getPets","responses":{}}},"/users":{"post":{"responses":{}}}}}`)
	execution.RuleSet.Rules = map[string]*model.Rule{"operation-operationId": rulesets.GetAllBuiltInRules()["operation-operationId"]}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	ids := map[string]bool{"getPets": true}
	var choices []FixChoice
	for _, option := range review.Options() {
		require.True(t, option.Generated)
		require.False(t, option.Selected)
		require.NotEmpty(t, option.Value)
		require.False(t, ids[option.Value], "generated ID %q must be unique", option.Value)
		ids[option.Value] = true
		choices = append(choices, FixChoice{Index: option.Index})
	}
	require.Len(t, choices, 4)
	require.True(t, ids["getPets2"])
	require.True(t, ids["getPetsId"])
	require.True(t, ids["getPetsId2"])
	require.True(t, ids["postUsers"])
	repeated, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer repeated.Close()
	byPath := make(map[string]string)
	for _, option := range review.Options() {
		byPath[option.Result.Path] = option.Value
	}
	for _, option := range repeated.Options() {
		require.Equal(t, byPath[option.Result.Path], option.Value)
	}
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 4, preview.Fixed)
	require.Empty(t, preview.Results)
}

func TestFixDraftRequestBodyStartsWithUsableProposal(t *testing.T) {
	for _, source := range []string{
		`{"openapi":"3.1.0","info":{"title":"Images","version":"1"},"paths":{"/pets/{id}/uploadImage":{"post":{"description":"Upload an image.","requestBody":{"content":{"application/octet-stream":{"schema":{"type":"string","format":"binary"}}}},"responses":{"200":{"description":"Image received."}}}}}}`,
		"openapi: 3.1.0\ninfo: {title: Images, version: '1'}\npaths:\n  /pets/{id}/uploadImage:\n    post:\n      description: Upload an image.\n      requestBody:\n        content:\n          application/octet-stream:\n            schema:\n              type: string\n              format: binary\n      responses:\n        '200':\n          description: Image received.\n",
	} {
		t.Run(fmt.Sprintf("json=%t", json.Valid([]byte(source))), func(t *testing.T) {
			execution := reviewFixture()
			execution.Spec = []byte(source)
			execution.RuleSet.Rules = map[string]*model.Rule{"operation-description": rulesets.GetAllBuiltInRules()["operation-description"]}
			review, err := NewFixReview(execution, nil)
			require.NoError(t, err)
			defer review.Close()
			require.Len(t, review.Options(), 1)
			option := review.Options()[0]
			require.True(t, option.Generated)
			require.False(t, option.Selected)
			require.Equal(t, "Binary request body for POST /pets/{id}/uploadImage.", option.Value)
			draft, err := review.Draft(FixChoice{Index: option.Index})
			require.NoError(t, err)
			require.False(t, draft.NeedsValue)
			require.Contains(t, draft.Before, "requestBody")
			require.NotContains(t, draft.Before, option.Value)
			require.Contains(t, draft.After, option.Value)
			require.Contains(t, draft.After, "application/octet-stream")
			preview, err := review.Preview([]FixChoice{{Index: option.Index}})
			require.NoError(t, err)
			defer preview.Close()
			require.Equal(t, 1, preview.Fixed)
			require.Empty(t, preview.Results)
			require.Contains(t, string(preview.Candidate), option.Value)
			require.Equal(t, source, string(execution.Spec))
			custom := "The image data submitted by the caller."
			edited, err := review.Draft(FixChoice{Index: option.Index, Value: &custom})
			require.NoError(t, err)
			require.Contains(t, edited.After, custom)
			require.False(t, edited.Generated)
			require.Equal(t, option.Value, review.Options()[0].Value)
		})
	}
}

func TestFixDraftExcerptsAreBoundedAndValidUTF8(t *testing.T) {
	before := []byte("a\n" + strings.Repeat("界", 2000) + "\n" + strings.Repeat("old\n", 100))
	after := []byte("a\n" + strings.Repeat("界", 2000) + "\n" + strings.Repeat("new\n", 100))
	a, b := fixDraftExcerpts(before, after, 0)
	for _, part := range []string{a, b} {
		require.True(t, utf8.ValidString(part))
		require.NotContains(t, part, "�")
		require.Contains(t, part, "line truncated")
		require.Contains(t, part, "more lines omitted")
		require.Less(t, len(part), 35000)
	}
}

func TestFixDraftRejectsClosedOrInvalidChoice(t *testing.T) {
	review, err := NewFixReview(reviewFixture(), nil)
	require.NoError(t, err)
	_, err = review.Draft(FixChoice{Index: -1})
	require.ErrorContains(t, err, "invalid")
	review.Close()
	_, err = review.Draft(FixChoice{Index: 0})
	require.ErrorContains(t, err, "closed")
}

func TestFixReviewCountsUniqueRuleLocations(t *testing.T) {
	execution := reviewFixture()
	execution.Spec = []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a","b","b"]}}}}`)
	execution.RuleSet.Rules = map[string]*model.Rule{"duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"]}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	var choices []FixChoice
	for _, option := range review.Options() {
		if option.Available && !option.Editable {
			choices = append(choices, FixChoice{Index: option.Index})
		}
	}
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 1, preview.Fixed, "multiple duplicate entries at one rule location count as one fix")
	require.Empty(t, preview.Results)
	require.Contains(t, string(preview.Candidate), `"enum":["a","b"]`)
}

func TestFixDraftIncludesShortOwnerContext(t *testing.T) {
	execution := reviewFixture()
	execution.Spec = []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    User:\n      type: object\n      properties:\n        role:\n          type: string\n          enum: [admin, reader]\n")
	execution.RuleSet.Rules = map[string]*model.Rule{"component-description": rulesets.GetAllBuiltInRules()["component-description"]}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 1)
	draft, err := review.Draft(FixChoice{Index: review.Options()[0].Index})
	require.NoError(t, err)
	before := "User:\n  type: object\n  properties:\n    role:\n      type: string\n      enum: [admin, reader]"
	require.Equal(t, before, draft.Before)
	require.Equal(t, before+"\n  description: User schema.", draft.After)
}

func TestFixDraftLargeOwnerKeepsChangeInBoundedContext(t *testing.T) {
	before := []byte("Owner:\n" + strings.Repeat("  property: unchanged\n", 100) + "  enum: [a, a]\n")
	after := []byte("Owner:\n" + strings.Repeat("  property: unchanged\n", 100) + "  enum: [a]\n")
	a, b := fixDraftExcerpts(before, after, 1)
	require.NotContains(t, a, "Owner:")
	require.Contains(t, a, "enum: [a, a]")
	require.Contains(t, b, "enum: [a]")
	require.LessOrEqual(t, strings.Count(a, "\n")+1, 40)
	require.LessOrEqual(t, strings.Count(b, "\n")+1, 40)
}

func TestFixDraftScopesLateSchemaDespiteEarlierFormattingChanges(t *testing.T) {
	execution := reviewFixture()
	var source strings.Builder
	source.WriteString("openapi: 3.1.0\ninfo:\n  title: " + strings.Repeat("long early title ", 30) + "\n  version: '1'\npaths: {}\ncomponents:\n  schemas:\n")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&source, "    Other%d:\n      type: string\n      description: Existing%d\n", i, i)
	}
	source.WriteString("    Order:\n      type: object\n      properties:\n        id:\n          type: integer\n")
	execution.Spec = []byte(source.String())
	execution.RuleSet.Rules = map[string]*model.Rule{"component-description": rulesets.GetAllBuiltInRules()["component-description"]}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 1)
	draft, err := review.Draft(FixChoice{Index: review.Options()[0].Index})
	require.NoError(t, err)
	require.Equal(t, "/components/schemas/Order", draft.Pointer)
	require.Contains(t, draft.Before, "Order:")
	require.Contains(t, draft.After, "Order schema.")
	for _, pane := range []string{draft.Before, draft.After} {
		require.NotContains(t, pane, "title")
		require.NotContains(t, pane, "info:")
		require.NotContains(t, pane, "Other79")
	}
	preview, err := review.Preview([]FixChoice{{Index: review.Options()[0].Index}})
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, strings.SplitN(source.String(), "components:", 2)[0], strings.SplitN(string(preview.Candidate), "components:", 2)[0], "unrelated earlier formatting stays unchanged")
	require.Equal(t, source.String(), string(execution.Spec))
}

func TestFixDraftPreservesTagSequenceShape(t *testing.T) {
	execution := reviewFixture()
	execution.Spec = []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ntags:\n  - name: pets\n")
	execution.RuleSet.Rules = map[string]*model.Rule{"tag-description": rulesets.GetAllBuiltInRules()["tag-description"]}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 1)
	draft, err := review.Draft(FixChoice{Index: review.Options()[0].Index})
	require.NoError(t, err)
	require.Equal(t, "/tags", draft.Pointer)
	require.Contains(t, draft.Before, "tags:")
	require.Contains(t, draft.Before, "- name: pets")
	require.NotContains(t, draft.Before, "0:")
	require.Contains(t, draft.After, "description:")
}

func TestFixDraftShowsDuplicateJSONPathMembers(t *testing.T) {
	execution := reviewFixture()
	execution.Spec = []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok"}}}},"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}}}`)
	execution.RuleSet.Rules = map[string]*model.Rule{"duplicate-paths": rulesets.GetAllBuiltInRules()["duplicate-paths"]}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	require.NotEmpty(t, review.Options())
	for _, option := range review.Options() {
		if !option.Available || option.Editable {
			continue
		}
		draft, err := review.Draft(FixChoice{Index: option.Index})
		require.NoError(t, err)
		require.Equal(t, "/paths", draft.Pointer)
		require.Equal(t, 2, strings.Count(draft.Before, `"/pets"`))
		require.Equal(t, 1, strings.Count(draft.After, `"/pets"`))
		require.NotContains(t, draft.Before, `"info"`)
		return
	}
	t.Fatal("expected a duplicate-path repair")
}
