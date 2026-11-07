package cmd

import (
	"fmt"

	"github.com/daveshanley/vacuum/motor"
)

// previewCLIRepairs keeps the largest verified prefix found by splitting failed
// batches. Each accepted batch is checked together with all earlier survivors.
// Dashboard selections continue to use the strict all-or-nothing Preview API.
func previewCLIRepairs(review *motor.FixReview, choices []motor.FixChoice, options *motor.ExecutionOptions) (*motor.FixPreview, []string, error) {
	var accepted []motor.FixChoice
	var warnings []string
	var preview *motor.FixPreview
	records := review.Options()
	var visit func([]motor.FixChoice) error
	visit = func(batch []motor.FixChoice) error {
		if options != nil && options.Context != nil && options.Context.Err() != nil {
			return options.Context.Err()
		}
		combined := append(append([]motor.FixChoice(nil), accepted...), batch...)
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
		return nil, warnings, fmt.Errorf("no selected fixes passed verification: %v", warnings)
	}
	return preview, warnings, nil
}
