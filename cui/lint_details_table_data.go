// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/daveshanley/vacuum/model"
)

// contentWidths holds the natural widths of content in each column
type contentWidths struct {
	location int
	rule     int
	category int
}

// columnWidths holds the calculated widths for each column
type columnWidths struct {
	location int
	severity int
	message  int
	rule     int
	category int
	path     int
}

// BuildResultTableData builds the violation table data, calculating column widths based on terminal size and content.
func BuildResultTableData(results []*model.RuleFunctionResult, fileName string, terminalWidth int, showPath bool) ([]table.Column, []table.Row) {
	rows := buildTableRows(results, fileName, showPath)
	contentWidths := calculateContentWidths(results, fileName)
	columnWidths := calculateColumnWidths(terminalWidth, contentWidths, showPath)
	columns := buildTableColumns(columnWidths, showPath)
	
	return columns, rows
}

// buildTableRows creates the table rows from results
func buildTableRows(results []*model.RuleFunctionResult, fileName string, showPath bool) []table.Row {
	var rows []table.Row
	
	for _, r := range results {
		location := formatFileLocation(r, fileName)
		severity := getRuleSeverity(r)
		
		category := ""
		if r.Rule != nil && r.Rule.RuleCategory != nil {
			category = r.Rule.RuleCategory.Name
		}
		
		ruleID := ""
		if r.Rule != nil {
			ruleID = r.Rule.Id
		}
		
		if showPath {
			rows = append(rows, table.Row{
				location,
				severity,
				r.Message,
				ruleID,
				category,
				r.Path,
			})
		} else {
			rows = append(rows, table.Row{
				location,
				severity,
				r.Message,
				ruleID,
				category,
			})
		}
	}
	
	return rows
}

// calculateContentWidths finds the maximum natural width of content in each column
func calculateContentWidths(results []*model.RuleFunctionResult, fileName string) contentWidths {
	widths := contentWidths{
		location: len("Location"),
		rule:     len("Rule"),
		category: len("Category"),
	}
	
	for _, r := range results {
		location := formatFileLocation(r, fileName)
		if len(location) > widths.location {
			widths.location = len(location)
		}
		
		if r.Rule != nil {
			if len(r.Rule.Id) > widths.rule {
				widths.rule = len(r.Rule.Id)
			}
			if r.Rule.RuleCategory != nil && len(r.Rule.RuleCategory.Name) > widths.category {
				widths.category = len(r.Rule.RuleCategory.Name)
			}
		}
	}
	
	return widths
}

// calculateColumnWidths calculates responsive column widths based on terminal size
func calculateColumnWidths(terminalWidth int, content contentWidths, showPath bool) columnWidths {
	// the border has 2 chars at the end.
	actualTableWidth := terminalWidth - 2
	columnCount := 5
	if showPath {
		columnCount = 6
	}
	columnPadding := columnCount * 2
	availableWidth := actualTableWidth - columnPadding
	
	widths := columnWidths{
		location: content.location,
		severity: SeverityColumnWidth + 1, // +1 for icon space
		rule:     content.rule,
		category: content.category,
	}
	
	if showPath {
		calculateWithPathColumn(availableWidth, &widths, content)
	} else {
		calculateWithoutPathColumn(availableWidth, &widths, content)
	}
	
	// ensure exact width match
	totalColWidth := widths.location + widths.severity + widths.message + widths.rule + widths.category
	if showPath {
		totalColWidth += widths.path
	}
	
	widthDiff := availableWidth - totalColWidth
	if widthDiff > 0 {
		if showPath {
			widths.path += widthDiff
		} else {
			widths.message += widthDiff
		}
	} else if widthDiff < 0 {
		if showPath && widths.path > 35 {
			widths.path += widthDiff
		} else {
			widths.message += widthDiff
		}
	}
	
	return widths
}

// calculateWithPathColumn calculates widths when path column is shown
func calculateWithPathColumn(availableWidth int, widths *columnWidths, content contentWidths) {
	const (
		naturalMsgWidth  = 80
		naturalPathWidth = 50
		minMsgWidth      = 40
		minPathWidth     = 20
		minRuleWidth     = 20
		minCatWidth      = 20
	)
	
	totalNaturalWidth := widths.location + widths.severity + naturalMsgWidth + 
	                    content.rule + content.category + naturalPathWidth
	
	if totalNaturalWidth <= availableWidth {
		// enough space - use natural widths and distribute extra
		widths.message = naturalMsgWidth
		widths.path = naturalPathWidth
		
		extraSpace := availableWidth - totalNaturalWidth
		if extraSpace > 0 {
			widths.message += extraSpace / 2
			widths.path += extraSpace - (extraSpace / 2)
		}
	} else {
		// need compression - apply hierarchically
		widths.message = naturalMsgWidth
		widths.path = naturalPathWidth
		
		needToSave := totalNaturalWidth - availableWidth
		
		// compress path first
		needToSave = compressColumn(&widths.path, minPathWidth, needToSave)
		
		// then category
		needToSave = compressColumn(&widths.category, minCatWidth, needToSave)
		
		// then rule
		needToSave = compressColumn(&widths.rule, minRuleWidth, needToSave)
		
		// finally message
		compressColumn(&widths.message, minMsgWidth, needToSave)
	}
}

// calculateWithoutPathColumn calculates widths without path column
func calculateWithoutPathColumn(availableWidth int, widths *columnWidths, content contentWidths) {
	const (
		naturalMsgWidth = 100
		minMsgWidth     = 40
		minRuleWidth    = 20
		minCatWidth     = 20
	)
	
	totalNaturalWidth := widths.location + widths.severity + naturalMsgWidth + 
	                    content.rule + content.category
	
	if totalNaturalWidth <= availableWidth {
		// enough space - give extra to message
		widths.message = naturalMsgWidth + (availableWidth - totalNaturalWidth)
	} else {
		// need compression
		widths.message = naturalMsgWidth
		
		needToSave := totalNaturalWidth - availableWidth
		
		// compress category first
		needToSave = compressColumn(&widths.category, minCatWidth, needToSave)
		
		// then rule
		needToSave = compressColumn(&widths.rule, minRuleWidth, needToSave)
		
		// finally message
		compressColumn(&widths.message, minMsgWidth, needToSave)
	}
}

// compressColumn reduces a column width down to minimum if needed
func compressColumn(width *int, minWidth int, needToSave int) int {
	if needToSave <= 0 || *width <= minWidth {
		return needToSave
	}
	
	canSave := *width - minWidth
	if canSave >= needToSave {
		*width -= needToSave
		return 0
	}
	
	*width = minWidth
	return needToSave - canSave
}

// buildTableColumns creates the table column definitions
func buildTableColumns(widths columnWidths, showPath bool) []table.Column {
	columns := []table.Column{
		{Title: "Location", Width: widths.location},
		{Title: "Severity", Width: widths.severity},
		{Title: "Message", Width: widths.message},
		{Title: "Rule", Width: widths.rule},
		{Title: "Category", Width: widths.category},
	}
	
	if showPath {
		columns = append(columns, table.Column{
			Title: "Path", 
			Width: widths.path,
		})
	}
	
	return columns
}