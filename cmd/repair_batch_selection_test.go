// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/motor/autofix"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/testify/require"
)

type batchReviewProbe struct {
	calls   [][]autofix.FixChoice
	preview func([]autofix.FixChoice) (*autofix.FixPreview, error)
}

func (p *batchReviewProbe) Options() []autofix.FixOption {
	options := make([]autofix.FixOption, 4)
	for i := range options {
		options[i] = autofix.FixOption{Index: i, Result: &model.RuleFunctionResult{RuleId: "test", Path: "$.value"}}
	}
	return options
}
func (p *batchReviewProbe) Preview(choices []autofix.FixChoice) (*autofix.FixPreview, error) {
	p.calls = append(p.calls, append([]autofix.FixChoice(nil), choices...))
	return p.preview(choices)
}

func TestRepairBatchRetriesOnlySurvivorsOfRemainingTargets(t *testing.T) {
	probe := &batchReviewProbe{}
	probe.preview = func(choices []autofix.FixChoice) (*autofix.FixPreview, error) {
		if len(probe.calls) == 1 {
			return nil, &autofix.FixSelectionError{Indexes: []int{1, 3}, Err: errors.New("targets remain")}
		}
		return &autofix.FixPreview{Fixed: len(choices)}, nil
	}
	preview, warnings, err := previewCLIRepairs(probe, []autofix.FixChoice{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}}, nil)
	require.NoError(t, err)
	defer preview.Close()
	require.Len(t, probe.calls, 2, "all unresolved targets must be removed after one failed lint")
	require.Equal(t, []autofix.FixChoice{{Index: 0}, {Index: 2}}, probe.calls[1])
	require.Equal(t, 2, preview.Fixed)
	require.Len(t, warnings, 2)
}

func TestRepairBatchRetainsAcceptedPrefix(t *testing.T) {
	probe := &batchReviewProbe{}
	probe.preview = func(choices []autofix.FixChoice) (*autofix.FixPreview, error) {
		switch len(probe.calls) {
		case 1:
			return nil, errors.New("cannot combine")
		case 2:
			return &autofix.FixPreview{Fixed: 2}, nil
		default:
			return nil, &autofix.FixSelectionError{Indexes: []int{0}, Err: errors.New("accepted target remains")}
		}
	}
	preview, warnings, err := previewCLIRepairs(probe, []autofix.FixChoice{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}}, nil)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 2, preview.Fixed)
	require.Len(t, warnings, 2)
	require.Len(t, probe.calls, 5)
	for _, call := range probe.calls[2:] {
		require.Equal(t, []autofix.FixChoice{{Index: 0}, {Index: 1}}, call[:2])
	}
}

func TestRepairBatchCancellationDoesNotRetryOrDecline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	probe := &batchReviewProbe{}
	probe.preview = func([]autofix.FixChoice) (*autofix.FixPreview, error) {
		cancel()
		return nil, &autofix.FixSelectionError{Indexes: []int{1}, Err: errors.New("target remains")}
	}
	preview, warnings, err := previewCLIRepairs(probe, []autofix.FixChoice{{Index: 0}, {Index: 1}}, &motor.ExecutionOptions{Context: ctx})
	require.Nil(t, preview)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, warnings)
	require.Len(t, probe.calls, 1)
}

func TestRepairBatchStopsOnReviewDeadline(t *testing.T) {
	probe := &batchReviewProbe{}
	probe.preview = func([]autofix.FixChoice) (*autofix.FixPreview, error) { return nil, context.DeadlineExceeded }
	preview, warnings, err := previewCLIRepairs(probe, []autofix.FixChoice{{Index: 0}, {Index: 1}}, nil)
	require.Nil(t, preview)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Empty(t, warnings)
	require.Len(t, probe.calls, 1)
}

func TestRepairBatchRechecksSurvivorsAfterTypedRejection(t *testing.T) {
	for _, remainingError := range []error{
		&autofix.FixSelectionError{Indexes: []int{0, 2}, Err: errors.New("survivors remain unresolved")},
		errors.New("survivors introduce a diagnostic"),
	} {
		t.Run(remainingError.Error(), func(t *testing.T) {
			probe := &batchReviewProbe{}
			probe.preview = func([]autofix.FixChoice) (*autofix.FixPreview, error) {
				if len(probe.calls) == 1 {
					return nil, &autofix.FixSelectionError{Indexes: []int{1}, Err: errors.New("first target remains")}
				}
				return nil, remainingError
			}
			preview, warnings, err := previewCLIRepairs(probe, []autofix.FixChoice{{Index: 0}, {Index: 1}, {Index: 2}}, nil)
			require.Nil(t, preview, "a failed survivor candidate must never be returned")
			require.ErrorContains(t, err, "no selected fixes passed verification")
			require.Len(t, warnings, 3)
			require.Equal(t, []autofix.FixChoice{{Index: 0}, {Index: 2}}, probe.calls[1])
			var rejected *autofix.FixSelectionError
			if errors.As(remainingError, &rejected) {
				require.Len(t, probe.calls, 2)
			} else {
				require.Len(t, probe.calls, 4, "unattributed survivor failure must still bisect")
			}
		})
	}
}
