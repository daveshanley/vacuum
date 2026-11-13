// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/daveshanley/vacuum/motor/autofix"

	"github.com/daveshanley/vacuum/motor"
)

type repairBatchReviewer interface {
	Options() []autofix.FixOption
	Preview([]autofix.FixChoice) (*autofix.FixPreview, error)
}

const maxRepairPreviewAttempts = 16

// previewCLIRepairs excludes unresolved targets and splits other failed batches.
// Each accepted batch is checked together with all earlier survivors.
// Dashboard selections continue to use the strict all-or-nothing Preview API.
func previewCLIRepairs(review repairBatchReviewer, choices []autofix.FixChoice, options *motor.ExecutionOptions, progress io.Writer) (*autofix.FixPreview, []string, error) {
	var accepted []autofix.FixChoice
	var warnings []string
	var preview *autofix.FixPreview
	records := review.Options()
	attempts := 0
	var visit func([]autofix.FixChoice) error
	visit = func(batch []autofix.FixChoice) error {
		if options != nil && options.Context != nil && options.Context.Err() != nil {
			return options.Context.Err()
		}
		if attempts >= maxRepairPreviewAttempts {
			for _, choice := range batch {
				o := records[choice.Index]
				warnings = append(warnings, fmt.Sprintf("fix declined: %s at %s: verification budget reached; select fewer fixes", o.Result.RuleId, o.Result.Path))
			}
			return nil
		}
		attempts++
		if _, err := fmt.Fprintf(progress, " Verifying %d fixes (pass %d/%d)…\n", len(accepted)+len(batch), attempts, maxRepairPreviewAttempts); err != nil {
			return err
		}
		combined := append(append([]autofix.FixChoice(nil), accepted...), batch...)
		next, err := review.Preview(combined)
		if err == nil {
			if preview != nil {
				preview.Close()
			}
			preview = next
			accepted = combined
			return nil
		}
		if options != nil && options.Context != nil && options.Context.Err() != nil {
			return options.Context.Err()
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		var rejected *autofix.FixSelectionError
		if errors.As(err, &rejected) && len(rejected.Indexes) > 0 {
			inBatch := make(map[int]bool, len(batch))
			for _, choice := range batch {
				inBatch[choice.Index] = true
			}
			failed := make(map[int]bool, len(rejected.Indexes))
			for _, index := range rejected.Indexes {
				if !inBatch[index] {
					failed = nil
					break
				}
				failed[index] = true
			}
			// Never remove an accepted choice to accommodate a later batch.
			// Unknown indexes also retain the conservative bisection path.
			if len(failed) > 0 {
				survivors := make([]autofix.FixChoice, 0, len(batch)-len(failed))
				for _, choice := range batch {
					if failed[choice.Index] {
						option := records[choice.Index]
						warnings = append(warnings, fmt.Sprintf("fix declined: %s at %s: selected target remains unresolved", option.Result.RuleId, option.Result.Path))
					} else {
						survivors = append(survivors, choice)
					}
				}
				if len(survivors) == 0 {
					return nil
				}
				return visit(survivors)
			}
		}
		if len(batch) == 1 {
			option := records[batch[0].Index]
			warnings = append(warnings, fmt.Sprintf("fix declined: %s at %s: %v", option.Result.RuleId, option.Result.Path, err))
			return nil
		}
		middle := len(batch) / 2
		if err := visit(batch[:middle]); err != nil {
			return err
		}
		return visit(batch[middle:])
	}
	if len(choices) == 0 {
		return nil, nil, fmt.Errorf("no fixes selected")
	}
	if err := visit(choices); err != nil {
		if preview != nil {
			preview.Close()
		}
		return nil, warnings, err
	}
	if preview == nil {
		return nil, warnings, fmt.Errorf("no selected fixes passed verification")
	}
	return preview, warnings, nil
}
