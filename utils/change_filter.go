// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
)

// ChangeFilter filters vacuum results based on what-changed data
type ChangeFilter struct {
	changedLines  map[int]bool    // Lines that have changes
	changedModels map[string]bool // JSONPaths of changed models
	drDoc         *drModel.DrDocument
}

// ChangeFilterStats holds statistics about what was filtered
type ChangeFilterStats struct {
	TotalResultsBefore   int            // Results before filtering
	TotalResultsAfter    int            // Results after filtering
	ResultsDropped       int            // TotalBefore - TotalAfter
	RulesFullyFiltered   []string       // Rules where ALL results were removed
	RulesPartialFiltered map[string]int // Rule -> count of results dropped (partial)
}

// GetDroppedPercentage returns the percentage of results that were filtered out
func (s *ChangeFilterStats) GetDroppedPercentage() int {
	if s.TotalResultsBefore == 0 {
		return 0
	}
	return (s.ResultsDropped * 100) / s.TotalResultsBefore
}

// NewChangeFilter creates a new ChangeFilter from DocumentChanges and a DrDocument
func NewChangeFilter(changes *wcModel.DocumentChanges, drDoc *drModel.DrDocument) *ChangeFilter {
	// estimate capacity from changes to reduce map rehashing
	estimatedCapacity := 64 // reasonable default
	if changes != nil && changes.TotalChanges() > 0 {
		// each change may affect multiple lines, estimate 2x
		estimatedCapacity = changes.TotalChanges() * 2
	}

	cf := &ChangeFilter{
		changedLines:  make(map[int]bool, estimatedCapacity),
		changedModels: make(map[string]bool, estimatedCapacity),
		drDoc:         drDoc,
	}

	if changes != nil {
		cf.extractChangedLines(changes)
		cf.buildChangedModels()
	}

	return cf
}

// extractChangedLines walks the DocumentChanges tree and extracts all NewLine values
func (cf *ChangeFilter) extractChangedLines(changes *wcModel.DocumentChanges) {
	if changes == nil {
		return
	}

	// GetAllChanges may panic in edge cases, so use defensive recovery
	defer func() {
		if r := recover(); r != nil {
			// Silently recover - we'll just have no changed lines to filter by
			// which means all results will be included (safe default)
		}
	}()

	// Skip if no changes at all
	if changes.TotalChanges() == 0 {
		return
	}

	allChanges := changes.GetAllChanges()
	for _, change := range allChanges {
		if change == nil {
			continue
		}

		// Only include changes that are additions or modifications (not removals)
		// because we want to filter results to areas that exist in the NEW spec
		if change.ChangeType == wcModel.ObjectRemoved || change.ChangeType == wcModel.PropertyRemoved {
			continue
		}

		if change.Context != nil && change.Context.NewLine != nil {
			cf.changedLines[*change.Context.NewLine] = true
		}
	}
}

// buildChangedModels uses the DrDocument to find models at changed lines
func (cf *ChangeFilter) buildChangedModels() {
	if cf.drDoc == nil {
		return
	}

	for line := range cf.changedLines {
		models, err := cf.drDoc.LocateModelByLine(line)
		if err != nil || len(models) == 0 {
			continue
		}

		for _, m := range models {
			// Add this model's JSON path
			path := m.GenerateJSONPath()
			cf.changedModels[path] = true

			// Also add all ancestors - if a child changed, the parent contains changes
			cf.addAncestorPaths(m)
		}
	}
}

// addAncestorPaths adds all parent JSON paths to the changed models set
func (cf *ChangeFilter) addAncestorPaths(m drV3.Foundational) {
	parent := m.GetParent()
	for parent != nil {
		cf.changedModels[parent.GenerateJSONPath()] = true
		parent = parent.GetParent()
	}
}

// FilterResults returns only results that affect changed areas
func (cf *ChangeFilter) FilterResults(results []*model.RuleFunctionResult) []*model.RuleFunctionResult {
	filtered, _ := cf.FilterResultsWithStats(results)
	return filtered
}

// FilterResultsWithStats returns filtered results AND statistics about what was filtered
func (cf *ChangeFilter) FilterResultsWithStats(results []*model.RuleFunctionResult) ([]*model.RuleFunctionResult, *ChangeFilterStats) {
	stats := &ChangeFilterStats{
		TotalResultsBefore:   len(results),
		RulesPartialFiltered: make(map[string]int),
	}

	// Track results per rule before and after
	ruleResultsBefore := make(map[string]int)
	ruleResultsAfter := make(map[string]int)

	// pre-allocate with estimated capacity to reduce reallocations
	filtered := make([]*model.RuleFunctionResult, 0, len(results)/2)
	for _, result := range results {
		if result == nil {
			continue
		}

		ruleId := ""
		if result.Rule != nil {
			ruleId = result.Rule.Id
		}
		ruleResultsBefore[ruleId]++

		if cf.IsInChangedArea(result) {
			filtered = append(filtered, result)
			ruleResultsAfter[ruleId]++
		}
	}

	stats.TotalResultsAfter = len(filtered)
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

	return filtered, stats
}

// IsInChangedArea checks if a single result is in a changed area
func (cf *ChangeFilter) IsInChangedArea(result *model.RuleFunctionResult) bool {
	if result == nil {
		return false
	}

	// if no changes were detected at all, include everything
	if len(cf.changedLines) == 0 && len(cf.changedModels) == 0 {
		return true
	}

	line := result.Range.Start.Line

	// try model-based matching first (more accurate)
	// all ancestors are already in changedModels from buildChangedModels()
	if cf.drDoc != nil {
		models, err := cf.drDoc.LocateModelByLine(line)
		if err == nil && len(models) > 0 {
			for _, m := range models {
				if cf.changedModels[m.GenerateJSONPath()] {
					return true
				}
			}
		}
	}

	// fallback: direct line matching
	return cf.changedLines[line]
}

// HasChanges returns true if there are any changes to filter by
func (cf *ChangeFilter) HasChanges() bool {
	return len(cf.changedLines) > 0 || len(cf.changedModels) > 0
}

// GetChangedLineCount returns the number of changed lines
func (cf *ChangeFilter) GetChangedLineCount() int {
	return len(cf.changedLines)
}

// GetChangedModelCount returns the number of changed models
func (cf *ChangeFilter) GetChangedModelCount() int {
	return len(cf.changedModels)
}

// IsLineChanged returns true if the given line number is in the changed lines set
func (cf *ChangeFilter) IsLineChanged(line int) bool {
	return cf.changedLines[line]
}

// FilterResultsValues filters a value slice directly, avoiding pointer conversion overhead.
// Returns the filtered slice and statistics.
func (cf *ChangeFilter) FilterResultsValues(results []model.RuleFunctionResult) ([]model.RuleFunctionResult, *ChangeFilterStats) {
	stats := &ChangeFilterStats{
		TotalResultsBefore:   len(results),
		RulesPartialFiltered: make(map[string]int),
	}

	// track results per rule before and after
	ruleResultsBefore := make(map[string]int)
	ruleResultsAfter := make(map[string]int)

	// pre-allocate with full capacity since we're filtering in place
	filtered := make([]model.RuleFunctionResult, 0, len(results))
	for i := range results {
		result := &results[i]

		ruleId := ""
		if result.Rule != nil {
			ruleId = result.Rule.Id
		}
		ruleResultsBefore[ruleId]++

		if cf.IsInChangedArea(result) {
			filtered = append(filtered, results[i])
			ruleResultsAfter[ruleId]++
		}
	}

	stats.TotalResultsAfter = len(filtered)
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

	return filtered, stats
}
