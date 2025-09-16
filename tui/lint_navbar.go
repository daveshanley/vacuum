// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
)

// buildNavBar builds the navigation bar at the bottom
func (m *ViolationResultTableModel) buildNavBar() string {
	navStyle := lipgloss.NewStyle().
		Foreground(color.RGBGrey).
		Width(m.width)

	rowText := ""
	if len(m.filteredResults) > 0 {
		rowText = fmt.Sprintf(" %d/%d", m.table.Cursor()+1, len(m.filteredResults))
	}

	// Add watch status indicator
	watchIndicator := ""
	if m.watchConfig != nil && m.watchConfig.Enabled {
		switch m.watchState {
		case WatchStateIdle:
			// No indicator when idle
		case WatchStateProcessing:
			watchIndicator = " ●" // Green filled circle for processing
		case WatchStateError:
			watchIndicator = " ●" // Red filled circle for error
		}
	}

	// Add watch error message if present
	watchErrorText := ""
	if m.watchState == WatchStateError && m.watchError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)
		watchErrorText = fmt.Sprintf(" | %s", errorStyle.Render(fmt.Sprintf("the specification '%s' is invalid: %s", m.fileName, m.watchError)))
	}

	baseNavText := fmt.Sprintf("%s | pgup/pgdn/↑↓/jk: nav | tab: severity | c: category | r: rule | p: path | enter: details | d: docs | x: code | q: quit", rowText)

	if watchIndicator != "" {
		var indicatorColor string
		if m.watchState == WatchStateProcessing {
			indicatorColor = "#00ff00" // Green
		} else if m.watchState == WatchStateError {
			indicatorColor = "#ff0000" // Red
		}
		coloredIndicator := lipgloss.NewStyle().Foreground(lipgloss.Color(indicatorColor)).Render(watchIndicator)
		return navStyle.Render(baseNavText + coloredIndicator + watchErrorText)
	}

	return navStyle.Render(baseNavText + watchErrorText)
}
