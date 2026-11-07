// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/testify/require"
)

type batchReviewProbe struct {
	calls   [][]motor.FixChoice
	preview func([]motor.FixChoice) (*motor.FixPreview, error)
}

func (p *batchReviewProbe) Options() []motor.FixOption {
	options := make([]motor.FixOption, 4)
	for i := range options {
		options[i] = motor.FixOption{Index: i, Result: &model.RuleFunctionResult{RuleId: "test", Path: "$.value"}}
	}
	return options
}
func (p *batchReviewProbe) Preview(choices []motor.FixChoice) (*motor.FixPreview, error) {
	p.calls = append(p.calls, append([]motor.FixChoice(nil), choices...))
	return p.preview(choices)
}

func TestRepairBatchRetriesOnlySurvivorsOfRemainingTargets(t *testing.T) {
	probe := &batchReviewProbe{}
	probe.preview = func(choices []motor.FixChoice) (*motor.FixPreview, error) {
		if len(probe.calls) == 1 {
			return nil, &motor.FixSelectionError{Indexes: []int{1, 3}, Err: errors.New("targets remain")}
		}
		return &motor.FixPreview{Fixed: len(choices)}, nil
	}
	preview, warnings, err := previewCLIRepairs(probe, []motor.FixChoice{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}}, nil)
	require.NoError(t, err)
	defer preview.Close()
	require.Len(t, probe.calls, 2, "all unresolved targets must be removed after one failed lint")
	require.Equal(t, []motor.FixChoice{{Index: 0}, {Index: 2}}, probe.calls[1])
	require.Equal(t, 2, preview.Fixed)
	require.Len(t, warnings, 2)
}

func TestRepairBatchRetainsAcceptedPrefix(t *testing.T) {
	probe := &batchReviewProbe{}
	probe.preview = func(choices []motor.FixChoice) (*motor.FixPreview, error) {
		switch len(probe.calls) {
		case 1:
			return nil, errors.New("cannot combine")
		case 2:
			return &motor.FixPreview{Fixed: 2}, nil
		default:
			return nil, &motor.FixSelectionError{Indexes: []int{0}, Err: errors.New("accepted target remains")}
		}
	}
	preview, warnings, err := previewCLIRepairs(probe, []motor.FixChoice{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}}, nil)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 2, preview.Fixed)
	require.Len(t, warnings, 2)
	require.Len(t, probe.calls, 5)
	for _, call := range probe.calls[2:] {
		require.Equal(t, []motor.FixChoice{{Index: 0}, {Index: 1}}, call[:2])
	}
}

func TestRepairBatchCancellationDoesNotRetryOrDecline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	probe := &batchReviewProbe{}
	probe.preview = func([]motor.FixChoice) (*motor.FixPreview, error) {
		cancel()
		return nil, &motor.FixSelectionError{Indexes: []int{1}, Err: errors.New("target remains")}
	}
	preview, warnings, err := previewCLIRepairs(probe, []motor.FixChoice{{Index: 0}, {Index: 1}}, &motor.ExecutionOptions{Context: ctx})
	require.Nil(t, preview)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, warnings)
	require.Len(t, probe.calls, 1)
}

func TestRepairBatchStopsOnReviewDeadline(t *testing.T) {
	probe := &batchReviewProbe{}
	probe.preview = func([]motor.FixChoice) (*motor.FixPreview, error) { return nil, context.DeadlineExceeded }
	preview, warnings, err := previewCLIRepairs(probe, []motor.FixChoice{{Index: 0}, {Index: 1}}, nil)
	require.Nil(t, preview)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Empty(t, warnings)
	require.Len(t, probe.calls, 1)
}

func TestRepairBatchRechecksSurvivorsAfterTypedRejection(t *testing.T) {
	for _, remainingError := range []error{
		&motor.FixSelectionError{Indexes: []int{0, 2}, Err: errors.New("survivors remain unresolved")},
		errors.New("survivors introduce a diagnostic"),
	} {
		t.Run(remainingError.Error(), func(t *testing.T) {
			probe := &batchReviewProbe{}
			probe.preview = func([]motor.FixChoice) (*motor.FixPreview, error) {
				if len(probe.calls) == 1 {
					return nil, &motor.FixSelectionError{Indexes: []int{1}, Err: errors.New("first target remains")}
				}
				return nil, remainingError
			}
			preview, warnings, err := previewCLIRepairs(probe, []motor.FixChoice{{Index: 0}, {Index: 1}, {Index: 2}}, nil)
			require.Nil(t, preview, "a failed survivor candidate must never be returned")
			require.ErrorContains(t, err, "no selected fixes passed verification")
			require.Len(t, warnings, 3)
			require.Equal(t, []motor.FixChoice{{Index: 0}, {Index: 2}}, probe.calls[1])
			var rejected *motor.FixSelectionError
			if errors.As(remainingError, &rejected) {
				require.Len(t, probe.calls, 2)
			} else {
				require.Len(t, probe.calls, 4, "unattributed survivor failure must still bisect")
			}
		})
	}
}
