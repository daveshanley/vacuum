// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import "github.com/daveshanley/vacuum/model"

func (m *ViolationResultTableModel) ApplyFilter() {
	var filtered []*model.RuleFunctionResult

	switch m.filterState {
	case FilterAll:
		filtered = m.allResults
	case FilterErrors:
		for _, r := range m.allResults {
			if r.Rule != nil && r.Rule.Severity == model.SeverityError {
				filtered = append(filtered, r)
			}
		}
	case FilterWarnings:
		for _, r := range m.allResults {
			if r.Rule != nil && r.Rule.Severity == model.SeverityWarn {
				filtered = append(filtered, r)
			}
		}
	case FilterInfo:
		for _, r := range m.allResults {
			if r.Rule != nil && r.Rule.Severity == model.SeverityInfo {
				filtered = append(filtered, r)
			}
		}
	}

	if m.categoryFilter != "" {
		var categoryFiltered []*model.RuleFunctionResult
		for _, r := range filtered {
			if r.Rule != nil && r.Rule.RuleCategory != nil &&
				r.Rule.RuleCategory.Name == m.categoryFilter {
				categoryFiltered = append(categoryFiltered, r)
			}
		}
		filtered = categoryFiltered
	}

	if m.ruleFilter != "" {
		var ruleFiltered []*model.RuleFunctionResult
		for _, r := range filtered {
			if r.Rule != nil && r.Rule.Id == m.ruleFilter {
				ruleFiltered = append(ruleFiltered, r)
			}
		}
		filtered = ruleFiltered
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
