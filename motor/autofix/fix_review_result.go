// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
)

// reportResults preserves the ordinary lint result set while the repair engine
// can use every sibling diagnostic to check a proposed edit.
func reportResults(results []model.RuleFunctionResult) []model.RuleFunctionResult {
	for i, result := range results {
		if result.RepairOnly {
			visible := make([]model.RuleFunctionResult, i, len(results)-1)
			copy(visible, results[:i])
			for _, next := range results[i+1:] {
				if !next.RepairOnly {
					visible = append(visible, next)
				}
			}
			return visible
		}
	}
	return results
}

func reportResult(result *motor.RuleSetExecutionResult) *motor.RuleSetExecutionResult {
	if result != nil {
		result.Results = reportResults(result.Results)
		result.IgnoredResults = reportResults(result.IgnoredResults)
	}
	return result
}

// TakeResult transfers a verified preview's lint resources to the caller.
// It grants no write authority; callers must first obtain acceptance and check
// the original source at the write boundary. The preview can no longer be used.
func (p *FixPreview) TakeResult() *motor.RuleSetExecutionResult {
	if p == nil || p.final == nil {
		return nil
	}
	result := p.final
	p.final = nil
	result.ModifiedSpec = append([]byte(nil), p.Candidate...)
	result.FixedResults = p.fixed
	result.RuleSetExecution.Spec = append([]byte(nil), p.original...)
	p.Candidate = nil
	return result
}

// TakeBaselineResult transfers the original lint result when no fixes are used.
// The review is closed after transfer; the caller must release the result.
func (r *FixReview) TakeBaselineResult() *motor.RuleSetExecutionResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := r.baseline
	r.baseline = nil
	return reportResult(result)
}
