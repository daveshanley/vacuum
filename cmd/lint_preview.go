// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/v2/table"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/model"
	"golang.org/x/term"
)

// regular expressions.
var locationRegex = regexp.MustCompile(`((?:[a-zA-Z]:)?[^\s│]*?[/\\]?[^\s│/\\]+\.[a-zA-Z]+):(\d+):(\d+)`)
var jsonPathRegex = regexp.MustCompile(`\$\.\S+`)
var circularRefRegex = regexp.MustCompile(`\b[a-zA-Z0-9_-]+(?:\s*->\s*[a-zA-Z0-9_-]+)+\b`)
var partRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)|(\s*->\s*)`)

// FilterState represents the current filter mode for cycling through severities
type FilterState int

const (
	FilterAll      FilterState = iota // Show all results
	FilterErrors                      // Show only errors
	FilterWarnings                    // Show only warnings
	FilterInfo                        // Show only info messages
)

// TableLintModel holds the state for the interactive table view
type TableLintModel struct {
	table           table.Model
	allResults      []*model.RuleFunctionResult
	filteredResults []*model.RuleFunctionResult
	rows            []table.Row
	fileName        string
	specContent     []byte // Raw spec content for code snippets
	quitting        bool
	width           int
	height          int
	filterState     FilterState
	categories      []string                  // Unique categories from results
	categoryIndex   int                       // Current category filter index (-1 = all)
	categoryFilter  string                    // Current category filter (empty = all)
	rules           []string                  // Unique rule IDs from results
	ruleIndex       int                       // Current rule filter index (-1 = all)
	ruleFilter      string                    // Current rule filter (empty = all)
	showPath        bool                      // Toggle for showing/hiding path column
	showModal       bool                      // Whether to show the DOCS modal
	showSplitView   bool                      // Whether to show the split view
	modalContent    *model.RuleFunctionResult // The current result being shown in the splitview
}

// ShowTableLintView displays results in an interactive table
func ShowTableLintView(results []*model.RuleFunctionResult, fileName string, specContent []byte) error {
	if len(results) == 0 {
		return nil
	}

	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 180
	}
	if height == 0 {
		height = 40
	}

	columns, rows := buildTableData(results, fileName, width, true) // Default to showing path

	// account for the border that will be added by addTableBorders (2 chars)
	tableActualWidth := width - 2
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height-5),        // Title (2 lines with blank), table border (2), status (1)
		table.WithWidth(tableActualWidth), // account for border wrapper
	)

	applyLintDetailsTableStyles(&t)

	categories := extractCategories(results)
	rules := extractRules(results)

	m := &TableLintModel{
		table:           t,
		allResults:      results,
		filteredResults: results,
		rows:            rows,
		fileName:        fileName,
		specContent:     specContent,
		width:           width,
		height:          height,
		filterState:     FilterAll,
		categories:      categories,
		categoryIndex:   -1,   // -1 means "All"
		showPath:        true, // Default to showing path column
		categoryFilter:  "",
		rules:           rules,
		ruleIndex:       -1, // -1 means "All"
		ruleFilter:      "",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func (m *TableLintModel) applyFilter() {
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
	columns, rows := buildTableData(m.filteredResults, m.fileName, m.width, m.showPath)
	m.rows = rows
	m.table.SetRows(rows)
	m.table.SetColumns(columns)

	applyLintDetailsTableStyles(&m.table)

	// reset cursor.
	m.table.SetCursor(0)
}

func (m *TableLintModel) Init() tea.Cmd {
	return nil
}

func (m *TableLintModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Rebuild table with new dimensions
		columns, rows := buildTableData(m.filteredResults, m.fileName, msg.Width, m.showPath)
		m.table.SetColumns(columns)
		m.table.SetRows(rows)
		m.table.SetWidth(msg.Width - 2) // Account for border wrapper

		// Adjust table height based on split view state
		if m.showSplitView {
			// When split view is open, table gets remaining space after fixed split view
			tableHeight := m.height - 15 - 4 // terminal height - split view height - margins
			if tableHeight < 10 {
				tableHeight = 10 // Minimum height
			}
			m.table.SetHeight(tableHeight)
		} else {
			m.table.SetHeight(msg.Height - 4)
		}

		// Reapply styles after resize
		applyLintDetailsTableStyles(&m.table)

		return m, nil

	case tea.KeyPressMsg:
		// Handle modal-specific keys first
		if m.showModal {
			switch msg.String() {
			case "esc", "q", "enter":
				m.showModal = false
				m.modalContent = nil
				return m, nil
			}
			// Don't process other keys when modal is open
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			// ESC closes split view if open, otherwise quits
			if m.showSplitView {
				m.showSplitView = false
				m.modalContent = nil
				// Rebuild table to full height
				m.table.SetHeight(m.height - 4)
			} else {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		case "enter":
			// Toggle split view
			m.showSplitView = !m.showSplitView
			if m.showSplitView {
				// Set content to currently selected result
				if m.table.Cursor() < len(m.filteredResults) {
					m.modalContent = m.filteredResults[m.table.Cursor()]
				}
				// Resize table to leave room for fixed-height split view
				tableHeight := m.height - 15 - 4 // terminal height - split view - margins
				if tableHeight < 10 {
					tableHeight = 10
				}
				m.table.SetHeight(tableHeight)
			} else {
				m.modalContent = nil
				// Restore table to full height
				m.table.SetHeight(m.height - 5)
			}
			return m, nil
		case "d":
			// Show DOCS modal with selected result
			if m.table.Cursor() < len(m.filteredResults) {
				m.modalContent = m.filteredResults[m.table.Cursor()]
				m.showModal = true
			}
			return m, nil
		case "tab":
			// Cycle through severity filter states
			m.filterState = (m.filterState + 1) % 4
			m.applyFilter()
			return m, nil
		case "c":
			// Cycle through category filters
			m.categoryIndex = (m.categoryIndex + 1) % (len(m.categories) + 1)
			if m.categoryIndex == -1 || m.categoryIndex == len(m.categories) {
				m.categoryIndex = -1
				m.categoryFilter = ""
			} else {
				m.categoryFilter = m.categories[m.categoryIndex]
			}
			m.applyFilter()
			return m, nil
		case "r":
			// Cycle through rule filters
			m.ruleIndex = (m.ruleIndex + 1) % (len(m.rules) + 1)
			if m.ruleIndex == -1 || m.ruleIndex == len(m.rules) {
				m.ruleIndex = -1
				m.ruleFilter = ""
			} else {
				m.ruleFilter = m.rules[m.ruleIndex]
			}
			m.applyFilter()
			return m, nil
		case "p":
			// Toggle path column visibility
			m.showPath = !m.showPath

			// Store current cursor position
			currentCursor := m.table.Cursor()

			// Calculate the cursor's position within the viewport
			// We'll try to maintain this relative position
			viewportHeight := m.table.Height()

			// Estimate where the viewport starts based on cursor position
			// The table tries to keep the cursor in the middle third of the viewport
			viewportStart := 0
			if currentCursor > viewportHeight/2 {
				viewportStart = currentCursor - viewportHeight/2
			}
			cursorOffsetInViewport := currentCursor - viewportStart

			// Rebuild table with new column configuration
			columns, rows := buildTableData(m.filteredResults, m.fileName, m.width, m.showPath)
			m.rows = rows

			// Update the existing table with new columns and rows
			// First clear the rows to avoid index issues
			m.table.SetRows([]table.Row{})
			m.table.SetColumns(columns)
			m.table.SetRows(rows)

			// Reapply styles
			applyLintDetailsTableStyles(&m.table)

			// Restore cursor position and viewport
			if currentCursor < len(rows) {
				// First go to top to ASCIIReset viewport
				m.table.GotoTop()

				// Move to where we want the viewport to start
				targetCursor := currentCursor

				// If we were scrolled down, overshoot and come back to position cursor correctly
				if viewportStart > 0 {
					// Move past the target
					overshoot := cursorOffsetInViewport
					for i := 0; i < targetCursor+overshoot && i < len(rows)-1; i++ {
						m.table.MoveDown(1)
					}
					// Then move back up to get cursor in right viewport position
					for i := 0; i < overshoot; i++ {
						m.table.MoveUp(1)
					}
				} else {
					// Just move to cursor position
					for i := 0; i < targetCursor; i++ {
						m.table.MoveDown(1)
					}
				}
			} else if len(rows) > 0 {
				m.table.SetCursor(0)
			}

			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)

	// Update split view content if it's open and cursor has changed
	if m.showSplitView {
		if m.table.Cursor() < len(m.filteredResults) {
			newContent := m.filteredResults[m.table.Cursor()]
			if m.modalContent != newContent {
				m.modalContent = newContent
			}
		}
	}

	return m, cmd
}

// buildTableView builds the complete table view with title, filters, and status bar
func (m *TableLintModel) buildTableView() string {
	var builder strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(RGBPink).
		Bold(true)

	title := "📋 Linting Results (Interactive View)"
	builder.WriteString(titleStyle.Render(title))

	filterStyle := lipgloss.NewStyle().
		Foreground(RGBBlue).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 1).
		Bold(true)

	if m.filterState != FilterAll {
		builder.WriteString("  ")
		builder.WriteString(filterStyle.Render("🔍 Severity: " + getLintingFilterName(m.filterState)))
	}

	if m.categoryFilter != "" {
		builder.WriteString("  ")
		categoryStyle := lipgloss.NewStyle().
			Foreground(RBGYellow).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1).
			Bold(true)
		builder.WriteString(categoryStyle.Render("📂 Category: " + m.categoryFilter))
	}

	if m.ruleFilter != "" {
		builder.WriteString("  ")
		ruleStyle := lipgloss.NewStyle().
			Foreground(RGBGreen).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1).
			Bold(true)
		builder.WriteString(ruleStyle.Render("📏 Rule: " + m.ruleFilter))
	}

	builder.WriteString("\n")

	contentHeight := m.height - 1 // Reserve space for title (1), blank line (1), and status bar (1)

	if len(m.filteredResults) == 0 {
		// Show empty state with ASCII art
		emptyView := renderEmptyState(m.width, contentHeight)
		builder.WriteString(emptyView)
	} else {

		tableView := ColorizeTableOutput(m.table.View(), m.table.Cursor(), m.rows)
		borderedTable := addTableBorders(tableView)
		builder.WriteString(borderedTable)
	}

	return builder.String()
}

// extractCodeSnippet extracts lines around the issue with context
func (m *TableLintModel) extractCodeSnippet(result *model.RuleFunctionResult, contextLines int) (string, int) {
	if m.specContent == nil || result == nil {
		return "", 0
	}

	line := 0
	if result.StartNode != nil {
		line = result.StartNode.Line
	}
	if result.Origin != nil {
		line = result.Origin.Line
	}

	if line == 0 {
		return "", 0
	}

	lines := bytes.Split(m.specContent, []byte("\n"))

	startLine := line - contextLines - 1 // -1 because line numbers are 1-based
	if startLine < 0 {
		startLine = 0
	}

	endLine := line + contextLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var snippet strings.Builder
	for i := startLine; i < endLine; i++ {
		snippet.Write(lines[i])
		if i < endLine-1 {
			snippet.WriteString("\n")
		}
	}

	return snippet.String(), startLine + 1
}

// buildModalView builds the DOCS modal placeholder
func (m *TableLintModel) buildModalView() string {

	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - 10

	if m.modalContent == nil {
		return ""
	}

	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(0).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink)

	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(RGBBlue).
		Bold(true).
		Align(lipgloss.Left).
		Width(modalWidth - 4)

	content.WriteString(titleStyle.Render("vacuum documentation"))
	content.WriteString("\n\n")

	msgStyle := lipgloss.NewStyle().
		Foreground(RGBGrey).
		Align(lipgloss.Left).
		Width(modalWidth - 4)

	content.WriteString(msgStyle.Render("coming soon bro!"))

	return modalStyle.Render(content.String())
}

// calculateModalPosition calculates the position for the modal (right-aligned)
func (m *TableLintModel) calculateModalPosition() (int, int) {
	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - 10

	// Position on the right side with good padding
	rightPadding := 6 // Good distance from right edge
	x := m.width - modalWidth - rightPadding

	// Center vertically
	y := (m.height - modalHeight) / 2

	// Ensure positive values
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return x, y
}

func (m *TableLintModel) View() string {
	if m.quitting {
		return ""
	}

	// Build the base table view
	tableView := m.buildTableView()

	// Build navigation bar (always at bottom)
	navStyle := lipgloss.NewStyle().
		Foreground(RGBDarkGrey).
		Width(m.width)
	// Add results count to nav bar
	resultsText := fmt.Sprintf("%d results", len(m.filteredResults))
	if (m.filterState != FilterAll || m.categoryFilter != "" || m.ruleFilter != "") && len(m.allResults) > 0 {
		resultsText = fmt.Sprintf("%d/%d results", len(m.filteredResults), len(m.allResults))
	}
	rowText := ""
	if len(m.filteredResults) > 0 {
		rowText = fmt.Sprintf(" • Row %d/%d", m.table.Cursor()+1, len(m.filteredResults))
	}

	navBar := navStyle.Render(fmt.Sprintf(" %s%s • ↑↓/jk: nav • tab: severity • c: category • r: rule • p: path • pgup/pgdn: page • enter: split • d: docs • q: quit", resultsText, rowText))

	// If split view is active, combine table with split panel
	if m.showSplitView {
		splitView := m.BuildDetailsView()
		// Join vertically: table on top, split view in middle, nav at bottom
		combined := lipgloss.JoinVertical(lipgloss.Left, tableView, splitView, navBar)

		// Create layers with the combined view
		layers := []*lipgloss.Layer{
			lipgloss.NewLayer(combined), // Base layer with split
		}

		// Add modal layer if shown (DOCS modal can appear over split view)
		if m.showModal {
			modal := m.buildModalView()
			x, y := m.calculateModalPosition()

			// Add modal as an overlay layer
			layers = append(layers,
				lipgloss.NewLayer(modal).X(x).Y(y).Z(1))
		}

		// Render the canvas with all layers
		canvas := lipgloss.NewCanvas(layers...)
		return canvas.Render()
	}

	// Normal view without split - nav at bottom
	combined := lipgloss.JoinVertical(lipgloss.Left, tableView, navBar)
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(combined), // Base layer with nav
	}

	// Add modal layer if shown
	if m.showModal {
		modal := m.buildModalView()
		x, y := m.calculateModalPosition()

		// Add modal as an overlay layer
		layers = append(layers,
			lipgloss.NewLayer(modal).X(x).Y(y).Z(1))
	}

	// Render the canvas with all layers
	canvas := lipgloss.NewCanvas(layers...)
	return canvas.Render()
}

func renderEmptyState(width, height int) string {
	// ASCII art for empty state
	art := []string{
		"",
		"",
		" _|      _|     _|_|     _|_|_|_|_|   _|    _|   _|_|_|   _|      _|     _|_|_|  ",
		" _|_|    _|   _|    _|       _|       _|    _|     _|     _|_|    _|   _|        ",
		" _|  _|  _|   _|    _|       _|       _|_|_|_|     _|     _|  _|  _|   _|  _|_|  ",
		" _|    _|_|   _|    _|       _|       _|    _|     _|     _|    _|_|   _|    _|  ",
		" _|      _|     _|_|         _|       _|    _|   _|_|_|   _|      _|     _|_|_|  ",
		"",
		"",
		" _|    _|   _|_|_|_|   _|_|_|     _|_|_|_|  ",
		" _|    _|   _|         _|    _|   _|        ",
		" _|_|_|_|   _|_|_|     _|_|_|     _|_|_|    ",
		" _|    _|   _|         _|    _|   _|        ",
		" _|    _|   _|_|_|_|   _|    _|   _|_|_|_|  ",
		"",
		"",
		" Nothing to vacuum, the filters are too strict.",
		"",
		" To adjust them:",
		"",
		" > tab - cycle severity",
		" > c   - cycle categories",
		" > r   - cycle rules",
		"",
	}

	// Join the art lines with preserved formatting
	artStr := strings.Join(art, "\n")

	// Calculate padding to center the block horizontally
	maxLineWidth := 82 // Width of the longest ASCII art line
	leftPadding := (width - maxLineWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	// Add left padding to each line to center the entire block
	artLines := strings.Split(artStr, "\n")
	paddedLines := make([]string, len(artLines))
	padding := strings.Repeat(" ", leftPadding)
	for i, line := range artLines {
		if line != "" {
			paddedLines[i] = padding + line
		} else {
			paddedLines[i] = ""
		}
	}

	// Calculate vertical centering
	totalLines := len(paddedLines)
	topPadding := (height - totalLines) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Build the result to exactly fill the height
	var resultLines []string

	// Add top padding
	for i := 0; i < topPadding; i++ {
		resultLines = append(resultLines, "")
	}

	// Add the content
	resultLines = append(resultLines, paddedLines...)

	// Add bottom padding to exactly fill the height
	for len(resultLines) < height {
		resultLines = append(resultLines, "")
	}

	// Ensure we don't exceed the height
	if len(resultLines) > height {
		resultLines = resultLines[:height]
	}

	// Apply color styling
	textStyle := lipgloss.NewStyle().Foreground(RGBDarkGrey)
	return textStyle.Render(strings.Join(resultLines, "\n"))
}

func addTableBorders(tableView string) string {
	// Just wrap in a simple border for now
	tableStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink).
		PaddingTop(0)

	return tableStyle.Render(tableView)
}
