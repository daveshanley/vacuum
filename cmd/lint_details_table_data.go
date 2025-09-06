// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/daveshanley/vacuum/model"
)

// BuildResultTableData builds the violation table data, calculating column widths based on terminal size and content.
func BuildResultTableData(results []*model.RuleFunctionResult, fileName string, terminalWidth int, showPath bool) ([]table.Column, []table.Row) {
	var rows []table.Row
	maxLocWidth := len("Location") // Start with header width
	maxRuleWidth := len("Rule")
	maxCatWidth := len("Category")

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

		if len(location) > maxLocWidth {
			maxLocWidth = len(location)
		}
		if len(ruleID) > maxRuleWidth {
			maxRuleWidth = len(ruleID)
		}
		if len(category) > maxCatWidth {
			maxCatWidth = len(category)
		}

		// pressing 'p' toggles the path column
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

	locWidth := maxLocWidth
	sevWidth := severityColumnWidth + 1 // fixed severity width for consistency (+1 for icon space)
	ruleWidth := maxRuleWidth
	catWidth := maxCatWidth

	naturalRuleWidth := ruleWidth
	naturalCatWidth := catWidth

	// account for borders and padding
	columnCount := 5
	if showPath {
		columnCount = 6
	}
	// the border has 2 chars at the end.
	actualTableWidth := terminalWidth - 2
	columnPadding := columnCount * 2
	availableWidth := actualTableWidth - columnPadding

	// minimum widths for various columns
	minMsgWidth := minMessageWidth // Message should be readable
	minPathWidth := minPathWidth   // Minimum for path
	minRuleWidth := 20             // Minimum for rule
	minCatWidth := 20              // Minimum for category

	var msgWidth, pathWidth int

	if showPath {

		// start with natural message width
		// this is our "natural" message width target
		naturalMsgWidth := 80
		naturalPathWidth := 50

		// calculate total natural width
		totalNaturalWidth := locWidth + sevWidth + naturalMsgWidth + naturalRuleWidth + naturalCatWidth + naturalPathWidth

		if totalNaturalWidth <= availableWidth {
			// we have enough space for natural widths or more
			msgWidth = naturalMsgWidth
			pathWidth = naturalPathWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth

			// distribute extra space 50/50 between message and path
			extraSpace := availableWidth - totalNaturalWidth
			if extraSpace > 0 {
				msgWidth += extraSpace / 2
				pathWidth += extraSpace - (extraSpace / 2) // use the remainder to avoid rounding issues
			}
		} else {
			// need to compress - use hierarchical compression
			// start with natural widths
			msgWidth = naturalMsgWidth
			pathWidth = naturalPathWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth

			// calculate how much we need to save
			needToSave := totalNaturalWidth - availableWidth

			// compress path first
			if needToSave > 0 {
				canSave := pathWidth - minPathWidth
				if canSave >= needToSave {
					pathWidth -= needToSave
					needToSave = 0
				} else {
					pathWidth = minPathWidth
					needToSave -= canSave
				}
			}

			// compress category
			if needToSave > 0 && catWidth > minCatWidth {
				canSave := catWidth - minCatWidth
				if canSave >= needToSave {
					catWidth -= needToSave
					needToSave = 0
				} else {
					catWidth = minCatWidth
					needToSave -= canSave
				}
			}

			// compress rule
			if needToSave > 0 && ruleWidth > minRuleWidth {
				canSave := ruleWidth - minRuleWidth
				if canSave >= needToSave {
					ruleWidth -= needToSave
					needToSave = 0
				} else {
					ruleWidth = minRuleWidth
					needToSave -= canSave
				}
			}

			// compress message
			if needToSave > 0 {
				canSave := msgWidth - minMsgWidth
				if canSave >= needToSave {
					msgWidth -= needToSave
				} else {
					msgWidth = minMsgWidth
				}
			}

		}
	} else {
		// no path column - simpler calculation
		naturalMsgWidth := 100
		totalNaturalWidth := locWidth + sevWidth + naturalMsgWidth + naturalRuleWidth + naturalCatWidth

		if totalNaturalWidth <= availableWidth {
			// we have enough space
			msgWidth = naturalMsgWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth

			// give all extra space to the message
			extraSpace := availableWidth - totalNaturalWidth
			if extraSpace > 0 {
				msgWidth += extraSpace
			}
		} else {
			// need to compress
			msgWidth = naturalMsgWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth

			needToSave := totalNaturalWidth - availableWidth

			// compress category
			if needToSave > 0 && catWidth > minCatWidth {
				canSave := catWidth - minCatWidth
				if canSave >= needToSave {
					catWidth -= needToSave
					needToSave = 0
				} else {
					catWidth = minCatWidth
					needToSave -= canSave
				}
			}

			// compress rule
			if needToSave > 0 && ruleWidth > minRuleWidth {
				canSave := ruleWidth - minRuleWidth
				if canSave >= needToSave {
					ruleWidth -= needToSave
					needToSave = 0
				} else {
					ruleWidth = minRuleWidth
					needToSave -= canSave
				}
			}

			// compress message
			if needToSave > 0 {
				canSave := msgWidth - minMsgWidth
				if canSave >= needToSave {
					msgWidth -= needToSave
				} else {
					msgWidth = minMsgWidth
				}
			}
		}
		pathWidth = 0
	}

	// ensure columns sum to EXACTLY match the available width. The table component doesn't stretch rows.
	var totalColWidth int
	if showPath {
		totalColWidth = locWidth + sevWidth + msgWidth + ruleWidth + catWidth + pathWidth
	} else {
		totalColWidth = locWidth + sevWidth + msgWidth + ruleWidth + catWidth
	}

	targetWidth := availableWidth // match calculated available width
	widthDiff := targetWidth - totalColWidth

	// add any difference to the message column (or path if shown)
	if widthDiff > 0 {
		if showPath {
			pathWidth += widthDiff
		} else {
			msgWidth += widthDiff
		}
	} else if widthDiff < 0 {
		// if we're over, reduce appropriate column
		if showPath {
			pathWidth += widthDiff // (widthDiff is negative, so this reduces)
			if pathWidth < minPathWidthCompressed {
				// if the path becomes too small, reduce the message instead
				msgWidth += widthDiff
				pathWidth = minPathWidthCompressed
			}
		} else {
			msgWidth += widthDiff
		}
	}

	columns := []table.Column{
		{Title: "Location", Width: locWidth},
		{Title: "Severity", Width: sevWidth},
		{Title: "Message", Width: msgWidth},
		{Title: "Rule", Width: ruleWidth},
		{Title: "Category", Width: catWidth},
	}

	if showPath {
		columns = append(columns, table.Column{
			Title: "Path", Width: pathWidth,
		})
	}

	return columns, rows
}
