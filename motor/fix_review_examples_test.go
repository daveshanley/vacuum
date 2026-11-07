// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestFixReviewMissingExamplesIncludesSiblingFindings(t *testing.T) {
	for _, schemaLocation := range []string{"media", "component"} {
		t.Run(schemaLocation, func(t *testing.T) {
			schema := `{"type":"object","properties":{"amount":{"type":"integer"},"card":{"type":"string"}}}`
			data := `{"openapi":"3.0.3","info":{"title":"Examples","version":"1"},"paths":{},"components":{"schemas":{"Payment":` + schema + `}}}`
			if schemaLocation == "media" {
				data = `{"openapi":"3.0.3","info":{"title":"Examples","version":"1"},"paths":{"/payment":{"post":{"requestBody":{"content":{"application/json":{"schema":` + schema + `}}},"responses":{"200":{"description":"ok"}}}}}}`
			}
			execution := &RuleSetExecution{SpecFileName: "examples.json", Spec: []byte(data), SilenceLogs: true,
				RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"oas3-missing-example": rulesets.GetAllBuiltInRules()["oas3-missing-example"]}}}
			plain := ApplyRulesToRuleSet(execution.CopyConfig())
			defer plain.ReleaseOwnedResources()
			require.Empty(t, plain.Errors)
			// Plain lint keeps its grouped property message; review must reveal each
			// sibling before applying any fix so validation sees a stable baseline.
			plainPropertyMessages := 0
			for _, result := range plain.Results {
				if strings.Contains(result.Message, "schema property") {
					plainPropertyMessages++
				}
			}
			require.Equal(t, 1, plainPropertyMessages)
			review, err := NewFixReview(execution, nil)
			require.NoError(t, err)
			defer review.Close()
			var amount, card *FixChoice
			for _, option := range review.Options() {
				if !strings.Contains(option.Result.Message, "schema property") {
					continue
				}
				require.True(t, option.Available, option.Reason)
				choice := FixChoice{Index: option.Index}
				if strings.Contains(option.Result.Message, "`amount`") {
					amount = &choice
				}
				if strings.Contains(option.Result.Message, "`card`") {
					card = &choice
				}
			}
			require.NotNil(t, amount)
			require.NotNil(t, card)
			preview, err := review.Preview([]FixChoice{*amount})
			require.NoError(t, err)
			defer preview.Close()
			require.Equal(t, 1, preview.Fixed)
			remainingCard := false
			for _, result := range preview.Results {
				if strings.Contains(result.Message, "schema property `card`") {
					remainingCard = true
				}
			}
			require.True(t, remainingCard, "unchanged sibling is retained as an existing diagnostic")
			both, err := review.Preview([]FixChoice{*amount, *card})
			require.NoError(t, err)
			defer both.Close()
			require.Equal(t, 2, both.Fixed)
		})
	}
}
