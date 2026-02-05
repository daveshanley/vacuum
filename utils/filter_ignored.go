// Copyright 2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import "github.com/daveshanley/vacuum/model"

// FilterIgnoredResultsPtr filters the given results slice, taking out any (RuleID, Path) combos that were listed in the
// ignore file
func FilterIgnoredResultsPtr(results []*model.RuleFunctionResult, ignored model.IgnoredItems) []*model.RuleFunctionResult {
	var filteredResults []*model.RuleFunctionResult

	for _, r := range results {

		var found bool
		for _, i := range ignored[r.Rule.Id] {
			// Check if the single Path matches
			if r.Path == i {
				found = true
				break
			}
			// Check if any of the Paths array matches
			if !found && len(r.Paths) > 0 {
				for _, p := range r.Paths {
					if p == i {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if !found {
			filteredResults = append(filteredResults, r)
		}
	}

	return filteredResults
}

// FilterIgnoredResults does the filtering of ignored results on non-pointer result elements
func FilterIgnoredResults(results []model.RuleFunctionResult, ignored model.IgnoredItems) []model.RuleFunctionResult {
	resultsPtrs := make([]*model.RuleFunctionResult, 0, len(results))
	for _, r := range results {
		r := r // prevent loop memory aliasing
		resultsPtrs = append(resultsPtrs, &r)
	}
	resultsFiltered := make([]model.RuleFunctionResult, 0, len(results))
	for _, r := range FilterIgnoredResultsPtr(resultsPtrs, ignored) {
		resultsFiltered = append(resultsFiltered, *r)
	}
	return resultsFiltered
}
