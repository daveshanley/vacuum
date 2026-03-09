// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
)

// violationKey identifies a violation across spec versions.
// Includes Message to distinguish rules that emit multiple distinct violations
// at the same path (e.g., info-contact-properties emits separate name/url/email
// failures all at $.info.contact).
type violationKey struct {
	RuleId  string
	Path    string
	Message string
}

// extractPath returns the JSONPath from a result, falling back to Paths[0] then "".
func extractPath(path string, paths []string) string {
	if path != "" {
		return path
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

// diffCore builds a count map from original keys and returns which new indices survive filtering.
func diffCore(originalKeys []violationKey, newKeys []violationKey) (kept []int, stats *ChangeFilterStats) {
	stats = &ChangeFilterStats{
		TotalResultsBefore:   len(newKeys),
		RulesPartialFiltered: make(map[string]int),
	}

	// Build occurrence count map from original results
	origCounts := make(map[violationKey]int, len(originalKeys))
	for _, k := range originalKeys {
		origCounts[k]++
	}

	// Track per-rule before/after counts
	ruleResultsBefore := make(map[string]int)
	ruleResultsAfter := make(map[string]int)

	kept = make([]int, 0, len(newKeys))
	for i, k := range newKeys {
		ruleResultsBefore[k.RuleId]++
		if origCounts[k] > 0 {
			// This violation existed in the original — suppress it
			origCounts[k]--
		} else {
			// New violation — keep it
			kept = append(kept, i)
			ruleResultsAfter[k.RuleId]++
		}
	}

	stats.TotalResultsAfter = len(kept)
	stats.ResultsDropped = stats.TotalResultsBefore - stats.TotalResultsAfter

	for ruleId, beforeCount := range ruleResultsBefore {
		afterCount := ruleResultsAfter[ruleId]
		dropped := beforeCount - afterCount
		if dropped > 0 {
			if afterCount == 0 {
				stats.RulesFullyFiltered = append(stats.RulesFullyFiltered, ruleId)
			} else {
				stats.RulesPartialFiltered[ruleId] = dropped
			}
		}
	}

	return kept, stats
}

// DiffViolationsValues compares original (value slice) and new (value slice) violations,
// returning only violations in new that don't exist in the original.
// Used by lint_cmd.go where both slices are values.
func DiffViolationsValues(original, new []model.RuleFunctionResult) ([]model.RuleFunctionResult, *ChangeFilterStats) {
	originalKeys := make([]violationKey, len(original))
	for i := range original {
		originalKeys[i] = violationKey{
			RuleId:  original[i].RuleId,
			Path:    extractPath(original[i].Path, original[i].Paths),
			Message: original[i].Message,
		}
	}

	newKeys := make([]violationKey, len(new))
	for i := range new {
		newKeys[i] = violationKey{
			RuleId:  new[i].RuleId,
			Path:    extractPath(new[i].Path, new[i].Paths),
			Message: new[i].Message,
		}
	}

	kept, stats := diffCore(originalKeys, newKeys)

	result := make([]model.RuleFunctionResult, len(kept))
	for i, idx := range kept {
		result[i] = new[idx]
	}
	return result, stats
}

// DiffViolationsMixed compares original (value slice from LintOriginalSpec) and new (pointer slice
// from report commands), returning only violations in new that don't exist in the original.
// Used by spectral-report, html-report, vacuum-report, and dashboard.
func DiffViolationsMixed(original []model.RuleFunctionResult, new []*model.RuleFunctionResult) ([]*model.RuleFunctionResult, *ChangeFilterStats) {
	originalKeys := make([]violationKey, len(original))
	for i := range original {
		originalKeys[i] = violationKey{
			RuleId:  original[i].RuleId,
			Path:    extractPath(original[i].Path, original[i].Paths),
			Message: original[i].Message,
		}
	}

	newKeys := make([]violationKey, len(new))
	for i := range new {
		if new[i] == nil {
			continue
		}
		newKeys[i] = violationKey{
			RuleId:  new[i].RuleId,
			Path:    extractPath(new[i].Path, new[i].Paths),
			Message: new[i].Message,
		}
	}

	kept, stats := diffCore(originalKeys, newKeys)

	result := make([]*model.RuleFunctionResult, len(kept))
	for i, idx := range kept {
		result[i] = new[idx]
	}
	return result, stats
}
