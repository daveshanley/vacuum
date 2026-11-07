// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestFixSelectionErrorCollectsRemainingTargets(t *testing.T) {
	source := []byte(`{"openapi":"3.0.3","info":{"title":"Test","version":"1"},"paths":{"/a":{"get":{"responses":{"200":{"description":"same"}}}},"/b":{"get":{"responses":{"200":{"description":"same"}}}},"/c":{"get":{"responses":{"200":{"description":"same"}}}}}}`)
	builtins := rulesets.GetAllBuiltInRules()
	execution := &RuleSetExecution{Spec: source, SpecFileName: "remaining.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"description-duplication": builtins["description-duplication"], "info-description": builtins["info-description"]}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	review, err := NewFixReview(execution, &ExecutionOptions{Context: ctx})
	require.NoError(t, err)
	defer review.Close()
	var choices, survivors []FixChoice
	var failedIndexes []int
	unchanged := "same"
	for _, option := range review.Options() {
		require.True(t, option.Available, option.Reason)
		choice := FixChoice{Index: option.Index}
		if option.Result.RuleId == "description-duplication" {
			require.True(t, option.Editable)
			choice.Value = &unchanged
			failedIndexes = append(failedIndexes, option.Index)
		} else {
			survivors = append(survivors, choice)
		}
		choices = append(choices, choice)
	}
	require.Len(t, failedIndexes, 2)
	require.Len(t, survivors, 1)
	preview, err := review.Preview(choices)
	require.Nil(t, preview)
	var selected *FixSelectionError
	require.ErrorAs(t, err, &selected)
	require.Equal(t, failedIndexes, selected.Indexes)
	var remaining *remainingRepairTargets
	require.ErrorAs(t, err, &remaining)
	require.Len(t, remaining.targets, 2)
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
