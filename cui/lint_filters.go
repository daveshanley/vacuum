// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import "github.com/daveshanley/vacuum/model"

func (m *ViolationResultTableModel) ApplyFilter() {
	// pre-allocate with estimated capacity to reduce allocations
	estimatedCapacity := len(m.allResults)
	if m.filterState != FilterAll {
		estimatedCapacity = len(m.allResults) / 3 // rough estimate for filtered results
	}
	
	filtered := make([]*model.RuleFunctionResult, 0, estimatedCapacity)

	// single-pass filtering - check all conditions in one loop
	for _, r := range m.allResults {
		if r.Rule == nil {
			continue
		}

		// check severity filter
		if m.filterState != FilterAll {
			switch m.filterState {
			case FilterErrors:
				if r.Rule.Severity != model.SeverityError {
					continue
				}
			case FilterWarnings:
				if r.Rule.Severity != model.SeverityWarn {
					continue
				}
			case FilterInfo:
				if r.Rule.Severity != model.SeverityInfo {
					continue
				}
			}
		}

		// check category filter
		if m.categoryFilter != "" {
			if r.Rule.RuleCategory == nil || r.Rule.RuleCategory.Name != m.categoryFilter {
				continue
			}
		}

		// check rule filter
		if m.ruleFilter != "" {
			if r.Rule.Id != m.ruleFilter {
				continue
			}
		}

		// all filters passed, add to results
		filtered = append(filtered, r)
	}

	m.filteredResults = filtered

	// rebuild table data with filtered results - recalculate column widths
	columns, rows := BuildResultTableData(m.filteredResults, m.fileName, m.width, m.showPath)
	m.rows = rows
	m.table.SetRows(rows)
	m.table.SetColumns(columns)

	ApplyLintDetailsTableStyles(&m.table)

	// reset cursor.
	m.table.SetCursor(0)
}
