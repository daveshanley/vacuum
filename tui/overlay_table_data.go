// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/table"
	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
	"github.com/pb33f/libopenapi/overlay"
)

// overlayColumnWidths holds the calculated widths for each column
type overlayColumnWidths struct {
	position    int
	target      int
	action      int
	status      int
	description int
}

// BuildOverlayActionsTable builds the overlay actions table with responsive column widths
func BuildOverlayActionsTable(actions []*highoverlay.Action, warnings []*overlay.Warning, overlayPath string, terminalWidth int) ([]table.Column, []table.Row) {
	rows := buildOverlayTableRows(actions, warnings, overlayPath)
	widths := calculateOverlayColumnWidths(actions, terminalWidth)
	columns := buildOverlayTableColumns(widths)

	return columns, rows
}

// buildOverlayTableRows creates the table rows from actions
func buildOverlayTableRows(actions []*highoverlay.Action, warnings []*overlay.Warning, overlayPath string) []table.Row {
	var rows []table.Row

	// Build a map of warnings by target for quick lookup
	warningMap := make(map[string]*overlay.Warning)
	for _, w := range warnings {
		warningMap[w.Target] = w
	}

	for _, action := range actions {
		// Get position from low-level action's RootNode - format as file:line:column for IDE clickability
		position := "-"
		if lowAction := action.GoLow(); lowAction != nil && lowAction.RootNode != nil {
			position = fmt.Sprintf("%s:%d:%d", overlayPath, lowAction.RootNode.Line, lowAction.RootNode.Column)
		}

		actionType := "[~] update"
		if action.Remove {
			actionType = "[-] remove"
		}

		status := "OK"
		// Check if this action had a warning
		if _, hasWarning := warningMap[action.Target]; hasWarning {
			status = "WARN"
		}

		description := action.Description
		if description == "" {
			description = "-"
		}

		rows = append(rows, table.Row{
			position,
			action.Target,
			actionType,
			status,
			description,
		})
	}

	return rows
}

// calculateOverlayColumnWidths calculates responsive column widths based on terminal size
func calculateOverlayColumnWidths(actions []*highoverlay.Action, terminalWidth int) overlayColumnWidths {
	const (
		positionWidth      = 28 // Fixed: "file.yaml:XXX:YYY" format
		actionWidth        = 12 // Fixed: "[~] update" or "[-] remove"
		statusWidth        = 6  // Fixed: "OK" or "WARN"
		naturalTargetWidth = 40
		naturalDescWidth   = 50
		minTargetWidth     = 15
		minDescWidth       = 20
	)

	// Calculate available width
	columnCount := 5
	columnPadding := columnCount * 2
	availableWidth := terminalWidth - 2 - columnPadding

	widths := overlayColumnWidths{
		position: positionWidth,
		action:   actionWidth,
		status:   statusWidth,
	}

	// Calculate natural widths based on content
	maxTargetWidth := len("Target")
	maxDescWidth := len("Description")
	for _, action := range actions {
		if len(action.Target) > maxTargetWidth {
			maxTargetWidth = len(action.Target)
		}
		if len(action.Description) > maxDescWidth {
			maxDescWidth = len(action.Description)
		}
	}

	// Cap at natural widths
	if maxTargetWidth > naturalTargetWidth {
		maxTargetWidth = naturalTargetWidth
	}
	if maxDescWidth > naturalDescWidth {
		maxDescWidth = naturalDescWidth
	}

	// Start with natural widths
	widths.target = maxTargetWidth
	widths.description = maxDescWidth

	// Calculate total and check if compression needed
	fixedWidth := widths.position + widths.action + widths.status
	flexibleWidth := widths.target + widths.description
	totalWidth := fixedWidth + flexibleWidth

	if totalWidth > availableWidth {
		// Need compression - apply hierarchically
		needToSave := totalWidth - availableWidth

		// Compress description first
		needToSave = compressOverlayColumn(&widths.description, minDescWidth, needToSave)

		// Finally target
		compressOverlayColumn(&widths.target, minTargetWidth, needToSave)
	} else if totalWidth < availableWidth {
		// Extra space - give to description column
		widths.description += availableWidth - totalWidth
	}

	return widths
}

// compressOverlayColumn reduces a column width down to minimum if needed
func compressOverlayColumn(width *int, minWidth int, needToSave int) int {
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

// buildOverlayTableColumns creates the table column definitions
func buildOverlayTableColumns(widths overlayColumnWidths) []table.Column {
	return []table.Column{
		{Title: "Position", Width: widths.position},
		{Title: "Target", Width: widths.target},
		{Title: "Action", Width: widths.action},
		{Title: "Status", Width: widths.status},
		{Title: "Description", Width: widths.description},
	}
}

// CountOverlayActionTypes counts the number of update and remove actions
func CountOverlayActionTypes(actions []*highoverlay.Action) (updates, removes int) {
	for _, action := range actions {
		if action.Remove {
			removes++
		} else {
			updates++
		}
	}
	return
}

// FormatOverlaySummary formats the summary line for overlay actions
func FormatOverlaySummary(actions []*highoverlay.Action, warningCount int) string {
	updates, removes := CountOverlayActionTypes(actions)

	summary := fmt.Sprintf("%d actions applied (%d updates, %d removals)", len(actions), updates, removes)

	if warningCount > 0 {
		summary += fmt.Sprintf(" - %d warning(s)", warningCount)
	}

	return summary
}
