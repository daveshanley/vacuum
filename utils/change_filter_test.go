// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
)

func TestNewChangeFilter_NilChanges(t *testing.T) {
	cf := NewChangeFilter(nil, nil)
	assert.NotNil(t, cf)
	assert.False(t, cf.HasChanges())
	assert.Equal(t, 0, cf.GetChangedLineCount())
	assert.Equal(t, 0, cf.GetChangedModelCount())
}

func TestNewChangeFilter_EmptyChanges(t *testing.T) {
	changes := &wcModel.DocumentChanges{}
	cf := NewChangeFilter(changes, nil)
	assert.NotNil(t, cf)
	assert.False(t, cf.HasChanges())
}

func TestChangeFilter_IsInChangedArea_NoChanges(t *testing.T) {
	cf := NewChangeFilter(nil, nil)

	// With no changes, everything should be included
	result := &model.RuleFunctionResult{
		Range: reports.Range{
			Start: reports.RangeItem{Line: 10},
		},
	}
	assert.True(t, cf.IsInChangedArea(result))
}

func TestChangeFilter_IsInChangedArea_WithChanges(t *testing.T) {
	// Create a mock change that points to line 10
	line := 10
	changes := &wcModel.DocumentChanges{
		PropertyChanges: &wcModel.PropertyChanges{
			Changes: []*wcModel.Change{
				{
					ChangeType: wcModel.Modified,
					Context: &wcModel.ChangeContext{
						NewLine: &line,
					},
				},
			},
		},
	}

	cf := NewChangeFilter(changes, nil)
	assert.True(t, cf.HasChanges())
	assert.Equal(t, 1, cf.GetChangedLineCount())

	// Result on changed line should be included
	resultOnLine := &model.RuleFunctionResult{
		Range: reports.Range{
			Start: reports.RangeItem{Line: 10},
		},
	}
	assert.True(t, cf.IsInChangedArea(resultOnLine))

	// Result on different line should NOT be included
	resultOffLine := &model.RuleFunctionResult{
		Range: reports.Range{
			Start: reports.RangeItem{Line: 20},
		},
	}
	assert.False(t, cf.IsInChangedArea(resultOffLine))
}

func TestChangeFilter_FilterResults(t *testing.T) {
	line1 := 10
	line2 := 20
	changes := &wcModel.DocumentChanges{
		PropertyChanges: &wcModel.PropertyChanges{
			Changes: []*wcModel.Change{
				{
					ChangeType: wcModel.Modified,
					Context: &wcModel.ChangeContext{
						NewLine: &line1,
					},
				},
				{
					ChangeType: wcModel.PropertyAdded,
					Context: &wcModel.ChangeContext{
						NewLine: &line2,
					},
				},
			},
		},
	}

	cf := NewChangeFilter(changes, nil)

	results := []*model.RuleFunctionResult{
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 10}},
			Rule:  &model.Rule{Id: "rule-1"},
		},
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 15}}, // Not on changed line
			Rule:  &model.Rule{Id: "rule-2"},
		},
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 20}},
			Rule:  &model.Rule{Id: "rule-3"},
		},
	}

	filtered := cf.FilterResults(results)
	assert.Len(t, filtered, 2)
	assert.Equal(t, 10, filtered[0].Range.Start.Line)
	assert.Equal(t, 20, filtered[1].Range.Start.Line)
}

func TestChangeFilter_FilterResultsWithStats(t *testing.T) {
	line := 10
	changes := &wcModel.DocumentChanges{
		PropertyChanges: &wcModel.PropertyChanges{
			Changes: []*wcModel.Change{
				{
					ChangeType: wcModel.Modified,
					Context: &wcModel.ChangeContext{
						NewLine: &line,
					},
				},
			},
		},
	}

	cf := NewChangeFilter(changes, nil)

	results := []*model.RuleFunctionResult{
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 10}},
			Rule:  &model.Rule{Id: "rule-1"},
		},
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 15}},
			Rule:  &model.Rule{Id: "rule-1"},
		},
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 20}},
			Rule:  &model.Rule{Id: "rule-2"},
		},
	}

	filtered, stats := cf.FilterResultsWithStats(results)

	assert.Len(t, filtered, 1)
	assert.Equal(t, 3, stats.TotalResultsBefore)
	assert.Equal(t, 1, stats.TotalResultsAfter)
	assert.Equal(t, 2, stats.ResultsDropped)

	// rule-1 has 1 result filtered, 1 remaining
	assert.Equal(t, 1, stats.RulesPartialFiltered["rule-1"])

	// rule-2 has all results filtered
	assert.Contains(t, stats.RulesFullyFiltered, "rule-2")
}

func TestChangeFilter_IgnoresRemovedChanges(t *testing.T) {
	addedLine := 10
	removedLine := 20

	changes := &wcModel.DocumentChanges{
		PropertyChanges: &wcModel.PropertyChanges{
			Changes: []*wcModel.Change{
				{
					ChangeType: wcModel.PropertyAdded,
					Context: &wcModel.ChangeContext{
						NewLine: &addedLine,
					},
				},
				{
					ChangeType: wcModel.PropertyRemoved, // Should be ignored
					Context: &wcModel.ChangeContext{
						NewLine: &removedLine,
					},
				},
			},
		},
	}

	cf := NewChangeFilter(changes, nil)

	// Only the added line should be in changed lines
	assert.True(t, cf.IsLineChanged(10))
	assert.False(t, cf.IsLineChanged(20)) // Removed line not tracked
}

func TestChangeFilter_NilResult(t *testing.T) {
	line := 10
	changes := &wcModel.DocumentChanges{
		PropertyChanges: &wcModel.PropertyChanges{
			Changes: []*wcModel.Change{
				{
					ChangeType: wcModel.Modified,
					Context: &wcModel.ChangeContext{
						NewLine: &line,
					},
				},
			},
		},
	}

	cf := NewChangeFilter(changes, nil)
	assert.False(t, cf.IsInChangedArea(nil))
}

func TestChangeFilter_FilterResultsValues(t *testing.T) {
	line := 10
	changes := &wcModel.DocumentChanges{
		PropertyChanges: &wcModel.PropertyChanges{
			Changes: []*wcModel.Change{
				{
					ChangeType: wcModel.Modified,
					Context: &wcModel.ChangeContext{
						NewLine: &line,
					},
				},
			},
		},
	}

	cf := NewChangeFilter(changes, nil)

	// use value slice directly (no pointer conversion)
	results := []model.RuleFunctionResult{
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 10}},
			Rule:  &model.Rule{Id: "rule-1"},
		},
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 15}}, // not on changed line
			Rule:  &model.Rule{Id: "rule-1"},
		},
		{
			Range: reports.Range{Start: reports.RangeItem{Line: 20}}, // not on changed line
			Rule:  &model.Rule{Id: "rule-2"},
		},
	}

	filtered, stats := cf.FilterResultsValues(results)

	assert.Len(t, filtered, 1)
	assert.Equal(t, 10, filtered[0].Range.Start.Line)
	assert.Equal(t, 3, stats.TotalResultsBefore)
	assert.Equal(t, 1, stats.TotalResultsAfter)
	assert.Equal(t, 2, stats.ResultsDropped)
	assert.Equal(t, 1, stats.RulesPartialFiltered["rule-1"])
	assert.Contains(t, stats.RulesFullyFiltered, "rule-2")
}
