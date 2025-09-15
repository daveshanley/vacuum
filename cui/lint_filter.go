// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import "github.com/daveshanley/vacuum/model"

// UpdateFilterState updates filter state
func (m *ViolationResultTableModel) UpdateFilterState(filter FilterState) {
	m.uiState.FilterState = filter
}

// UpdateCategoryFilter updates category filter
func (m *ViolationResultTableModel) UpdateCategoryFilter(category string) {
	m.uiState.CategoryFilter = category
}

// UpdateRuleFilter updates rule filter
func (m *ViolationResultTableModel) UpdateRuleFilter(rule string) {
	m.uiState.RuleFilter = rule
}

// filterResults applies current filters to allResults and updates filteredResults
func (m *ViolationResultTableModel) filterResults() {
	filtered := m.allResults

	// severity filter
	if m.uiState.FilterState != FilterAll {
		var severityFiltered []*model.RuleFunctionResult
		for _, result := range filtered {
			switch m.uiState.FilterState {
			case FilterErrors:
				if result.Rule.Severity == "error" {
					severityFiltered = append(severityFiltered, result)
				}
			case FilterWarnings:
				if result.Rule.Severity == "warn" {
					severityFiltered = append(severityFiltered, result)
				}
			case FilterInfo:
				if result.Rule.Severity == "info" {
					severityFiltered = append(severityFiltered, result)
				}
			}
		}
		filtered = severityFiltered
	}

	// category filter
	if m.uiState.CategoryFilter != "" {
		var categoryFiltered []*model.RuleFunctionResult
		for _, result := range filtered {
			if result.Rule.Formats != nil {
				for _, format := range result.Rule.Formats {
					if format == m.uiState.CategoryFilter {
						categoryFiltered = append(categoryFiltered, result)
						break
					}
				}
			}
		}
		filtered = categoryFiltered
	}

	// rule filter
	if m.uiState.RuleFilter != "" {
		var ruleFiltered []*model.RuleFunctionResult
		for _, result := range filtered {
			if result.Rule.Id == m.uiState.RuleFilter {
				ruleFiltered = append(ruleFiltered, result)
			}
		}
		filtered = ruleFiltered
	}

	m.filteredResults = filtered
}
