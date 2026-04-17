// Copyright 2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import "github.com/daveshanley/vacuum/model"

// FilterIgnoredResultsPtr filters the given results slice, taking out any (RuleID, Path) combos that were listed in the
// ignore file
func FilterIgnoredResultsPtr(results []*model.RuleFunctionResult, ignored model.IgnoredItems) []*model.RuleFunctionResult {
	return FilterIgnoredResultsPtrWithOptions(results, ignored, IgnoreMatcherOptions{})
}

// FilterIgnoredResultsPtrWithOptions filters result pointers using exact path ignores
// and, when a document root is available, JSONPath ignore expressions.
func FilterIgnoredResultsPtrWithOptions(
	results []*model.RuleFunctionResult,
	ignored model.IgnoredItems,
	options IgnoreMatcherOptions,
) []*model.RuleFunctionResult {
	matcher := NewIgnoreMatcher(ignored, options)
	if len(matcher.literalByRule) == 0 && len(matcher.resolvedByRule) == 0 {
		return results
	}

	filteredResults := make([]*model.RuleFunctionResult, 0, len(results))
	for _, result := range results {
		if matcher.Matches(result) {
			continue
		}
		filteredResults = append(filteredResults, result)
	}
	return filteredResults
}

// FilterIgnoredResults does the filtering of ignored results on non-pointer result elements
func FilterIgnoredResults(results []model.RuleFunctionResult, ignored model.IgnoredItems) []model.RuleFunctionResult {
	return FilterIgnoredResultsWithOptions(results, ignored, IgnoreMatcherOptions{})
}

// FilterIgnoredResultsWithOptions filters non-pointer results using exact path ignores
// and, when a document root is available, JSONPath ignore expressions.
func FilterIgnoredResultsWithOptions(
	results []model.RuleFunctionResult,
	ignored model.IgnoredItems,
	options IgnoreMatcherOptions,
) []model.RuleFunctionResult {
	if len(ignored) == 0 {
		return results
	}
	resultsPtrs := make([]*model.RuleFunctionResult, 0, len(results))
	for _, r := range results {
		r := r // prevent loop memory aliasing
		resultsPtrs = append(resultsPtrs, &r)
	}
	resultsFiltered := make([]model.RuleFunctionResult, 0, len(results))
	for _, r := range FilterIgnoredResultsPtrWithOptions(resultsPtrs, ignored, options) {
		resultsFiltered = append(resultsFiltered, *r)
	}
	return resultsFiltered
}
