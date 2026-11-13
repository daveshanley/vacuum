// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"context"
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/motor"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestFixSelectionErrorCollectsRemainingTargets(t *testing.T) {
	source := []byte(`{"openapi":"3.0.3","info":{"title":"Test","version":"1"},"paths":{"/a":{"get":{"responses":{"200":{"description":"<script>"}}}},"/b":{"get":{"responses":{"200":{"description":"<script>"}}}},"/c":{"get":{"responses":{"200":{"description":"<script>"}}}}}}`)
	builtins := rulesets.GetAllBuiltInRules()
	execution := &motor.RuleSetExecution{Spec: source, SpecFileName: "remaining.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"no-script-tags-in-markdown": builtins["no-script-tags-in-markdown"], "info-description": builtins["info-description"]}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	review, err := NewFixReview(execution, &motor.ExecutionOptions{Context: ctx})
	require.NoError(t, err)
	defer review.Close()
	var choices, survivors []FixChoice
	var failedIndexes []int
	unchanged := "<script>"
	for _, option := range review.Options() {
		require.True(t, option.Available, option.Reason)
		choice := FixChoice{Index: option.Index}
		if option.Result.RuleId == "no-script-tags-in-markdown" {
			require.True(t, option.Editable)
			choice.Value = &unchanged
			failedIndexes = append(failedIndexes, option.Index)
		} else {
			survivors = append(survivors, choice)
		}
		choices = append(choices, choice)
	}
	require.Len(t, failedIndexes, 3)
	require.Len(t, survivors, 1)
	preview, err := review.Preview(choices)
	require.Nil(t, preview)
	var selected *FixSelectionError
	require.ErrorAs(t, err, &selected)
	require.Equal(t, failedIndexes, selected.Indexes)
	var remaining *remainingRepairTargets
	require.ErrorAs(t, err, &remaining)
	require.Len(t, remaining.targets, 3)
	require.ErrorContains(t, err, "target remains")
	// A rejected candidate releases its resources but retains the baseline review.
	preview, err = review.Preview(survivors)
	require.NoError(t, err)
	require.Equal(t, 1, preview.Fixed)
	preview.Close()
	preview.Close()
	cancel()
	preview, err = review.Preview(choices)
	require.Nil(t, preview)
	require.ErrorIs(t, err, context.Canceled)
	selected = nil
	require.False(t, errors.As(err, &selected), "cancellation must not discard a selection")
}
