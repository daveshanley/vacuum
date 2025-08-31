// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/daveshanley/vacuum/model"
	"golang.org/x/term"
)

// FilterState represents the current filter mode for cycling through severities
type FilterState int

const (
	FilterAll      FilterState = iota // Show all results
	FilterErrors                      // Show only errors
	FilterWarnings                    // Show only warnings
	FilterInfo                        // Show only info messages
)

const (
	// TableWidthAdjustment is the amount to reduce table width to ensure it fits properly
	// and shows the right border. Adjust this value if the table extends off screen.
	TableWidthAdjustment = 2
)

// TableLintModel holds the state for the interactive table view
type TableLintModel struct {
	table           table.Model
	allResults      []*model.RuleFunctionResult
	filteredResults []*model.RuleFunctionResult
	rows            []table.Row
	fileName        string
	quitting        bool
	width           int
	height          int
	filterState     FilterState
	categories      []string // Unique categories from results
	categoryIndex   int      // Current category filter index (-1 = all)
	categoryFilter  string   // Current category filter (empty = all)
	rules           []string // Unique rule IDs from results
	ruleIndex       int      // Current rule filter index (-1 = all)
	ruleFilter      string   // Current rule filter (empty = all)
}

// applyTableStyles configures the table with neon pink theme
func applyTableStyles(t *table.Model) {
	neonPink := lipgloss.Color("#f83aff")
	s := table.DefaultStyles()

	// Header with pink text and separators
	s.Header = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(neonPink).
		BorderBottom(true).
		BorderLeft(false).
		BorderRight(false). // Remove right border to avoid doubles
		BorderTop(false).
		Foreground(neonPink).
		Bold(true).
		Padding(0, 1) // Add padding for readability

	// Selected row style with primary blue background and black text
	s.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")). // Black text
		Background(lipgloss.Color("#62c4ff")). // Primary blue background
		Padding(0, 0)

	// Regular cells with padding for readability
	s.Cell = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(neonPink).
		BorderRight(false). // Remove to avoid double borders
		Padding(0, 1)       // Add padding for readability

	t.SetStyles(s)
}

// ShowTableLintView displays results in an interactive table
func ShowTableLintView(results []*model.RuleFunctionResult, fileName string) error {
	if len(results) == 0 {
		return nil
	}

	// Get terminal size
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 180
	}
	if height == 0 {
		height = 40
	}

	// Calculate column widths
	columns, rows := buildTableData(results, fileName, width-TableWidthAdjustment)

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height-5),                  // Title (2 lines with blank), table border (2), status (1)
		table.WithWidth(width-TableWidthAdjustment), // Account for borders
	)

	// Apply table styles
	applyTableStyles(&t)

	// Extract unique categories and rules
	categories := extractCategories(results)
	rules := extractRules(results)

	// Create and run model
	m := TableLintModel{
		table:           t,
		allResults:      results,
		filteredResults: results,
		rows:            rows,
		fileName:        fileName,
		width:           width,
		height:          height,
		filterState:     FilterAll,
		categories:      categories,
		categoryIndex:   -1, // -1 means "All"
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

func buildTableData(results []*model.RuleFunctionResult, fileName string, width int) ([]table.Column, []table.Row) {
	rows := []table.Row{}
	maxLocWidth := len("Location") // Start with header width
	maxRuleWidth := len("Rule")
	maxCatWidth := len("Category")

	// First pass: build rows and find max widths
	for _, r := range results {
		location := formatLocation(r, fileName)
		severity := getSeverity(r)
		category := ""
		if r.Rule != nil && r.Rule.RuleCategory != nil {
			category = r.Rule.RuleCategory.Name
		}
		ruleID := ""
		if r.Rule != nil {
			ruleID = r.Rule.Id
		}

		// Track max widths
		if len(location) > maxLocWidth {
			maxLocWidth = len(location)
		}
		if len(ruleID) > maxRuleWidth {
			maxRuleWidth = len(ruleID)
		}
		if len(category) > maxCatWidth {
			maxCatWidth = len(category)
		}

		// Don't add spaces - let the table handle content as-is
		rows = append(rows, table.Row{
			location,
			severity,
			r.Message,
			ruleID,
			category,
			":||:" + r.Path + ":||:", // Add delimiters to mark the path column
		})
	}

	// Calculate column widths
	locWidth := maxLocWidth
	sevWidth := 9 // Fixed for "warning"
	ruleWidth := maxRuleWidth
	catWidth := maxCatWidth

	// Calculate fixed width (all columns except message and path)
	fixedWidth := locWidth + sevWidth + ruleWidth + catWidth

	// IMPORTANT: The table component adds padding via Cell style (Padding(0,1))
	// This adds 2 chars per column (1 left, 1 right) = 12 chars total for 6 columns
	// So the actual rendered width = sum(columnWidths) + 12
	// We need: sum(columnWidths) = width - 12
	totalPadding := 12
	availableWidth := width - totalPadding
	remainingWidth := availableWidth - fixedWidth

	// Ensure we have positive remaining width
	if remainingWidth < 100 {
		remainingWidth = 100
	}

	// Split remaining space between message (60%) and path (40%)
	msgWidth := (remainingWidth * 60) / 100
	pathWidth := remainingWidth - msgWidth // Use all remaining width to avoid rounding issues

	// Ensure minimum widths
	if msgWidth < 50 {
		msgWidth = 50
	}
	if pathWidth < 35 { // Minimum path width
		pathWidth = 35
		// Recalculate message width with minimum path
		msgWidth = availableWidth - fixedWidth - pathWidth
	}

	// CRITICAL: Ensure columns sum to EXACTLY (width - totalPadding)
	// The table component doesn't stretch rows, so we need exact match
	totalColWidth := locWidth + sevWidth + msgWidth + ruleWidth + catWidth + pathWidth
	targetWidth := width - totalPadding // Account for the padding the Cell style adds
	widthDiff := targetWidth - totalColWidth

	// Add any difference to the path column (last column)
	if widthDiff > 0 {
		pathWidth += widthDiff
	} else if widthDiff < 0 {
		// If we're over, reduce path width
		pathWidth += widthDiff // widthDiff is negative, so this reduces
		if pathWidth < 35 {
			// If path becomes too small, reduce message instead
			msgWidth += widthDiff
			pathWidth = 35
		}
	}

	columns := []table.Column{
		{Title: "Location", Width: locWidth},
		{Title: "Severity", Width: sevWidth},
		{Title: "Message", Width: msgWidth},
		{Title: "Rule", Width: ruleWidth},
		{Title: "Category", Width: catWidth},
		{Title: ":||:Path:||:", Width: pathWidth}, // Add delimiters to header too
	}

	return columns, rows
}

func formatLocation(r *model.RuleFunctionResult, fileName string) string {
	startLine := 0
	startCol := 0
	f := fileName

	if r.StartNode != nil {
		startLine = r.StartNode.Line
		startCol = r.StartNode.Column
	}

	if r.Origin != nil {
		f = r.Origin.AbsoluteLocation
		startLine = r.Origin.Line
		startCol = r.Origin.Column
	}

	// Make path relative
	if absPath, err := filepath.Abs(f); err == nil {
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, absPath); err == nil {
				f = relPath
			}
		}
	}

	return fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
}

func getSeverity(r *model.RuleFunctionResult) string {
	if r.Rule != nil {
		switch r.Rule.Severity {
		case model.SeverityError:
			return "error"
		case model.SeverityWarn:
			return "warning"
		default:
			return "info"
		}
	}
	return "info"
}

func (m *TableLintModel) applyFilter() {
	// Start with severity filter
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

	// Apply category filter on top of severity filter
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

	// Apply rule filter on top of other filters
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

	// Rebuild table data with filtered results - recalculate column widths
	columns, rows := buildTableData(m.filteredResults, m.fileName, m.width-TableWidthAdjustment)
	m.rows = rows

	// Update table with new rows and columns for optimal width
	m.table.SetRows(rows)
	m.table.SetColumns(columns)

	// Reapply styles after updating columns (to ensure borders persist)
	applyTableStyles(&m.table)

	// Reset cursor to top
	m.table.SetCursor(0)
}

func getFilterName(state FilterState) string {
	switch state {
	case FilterAll:
		return "All"
	case FilterErrors:
		return "Errors"
	case FilterWarnings:
		return "Warnings"
	case FilterInfo:
		return "Info"
	default:
		return "All"
	}
}

func extractCategories(results []*model.RuleFunctionResult) []string {
	categoryMap := make(map[string]bool)
	for _, r := range results {
		if r.Rule != nil && r.Rule.RuleCategory != nil {
			categoryMap[r.Rule.RuleCategory.Name] = true
		}
	}

	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}

	// Sort categories for consistent ordering
	for i := 0; i < len(categories); i++ {
		for j := i + 1; j < len(categories); j++ {
			if categories[i] > categories[j] {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}

	return categories
}

func extractRules(results []*model.RuleFunctionResult) []string {
	ruleMap := make(map[string]bool)
	for _, r := range results {
		if r.Rule != nil && r.Rule.Id != "" {
			ruleMap[r.Rule.Id] = true
		}
	}

	rules := make([]string, 0, len(ruleMap))
	for rule := range ruleMap {
		rules = append(rules, rule)
	}

	// Sort rules for consistent ordering
	for i := 0; i < len(rules); i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i] > rules[j] {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	return rules
}

func (m TableLintModel) Init() tea.Cmd {
	return nil
}

func (m TableLintModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Rebuild table with new dimensions
		columns, rows := buildTableData(m.filteredResults, m.fileName, msg.Width-TableWidthAdjustment)
		m.table.SetColumns(columns)
		m.table.SetRows(rows)
		m.table.SetWidth(msg.Width - TableWidthAdjustment) // Account for borders
		m.table.SetHeight(msg.Height - 5)

		// Reapply styles after resize
		applyTableStyles(&m.table)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
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
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m TableLintModel) View() string {
	if m.quitting {
		return ""
	}

	var builder strings.Builder

	// Title with filter state
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f83aff")).
		Bold(true)

	title := "📋 Linting Results (Interactive View)"
	builder.WriteString(titleStyle.Render(title))

	// Show filter indicators
	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#62c4ff")).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 1).
		Bold(true)

	// Show severity filter
	if m.filterState != FilterAll {
		builder.WriteString("  ")
		builder.WriteString(filterStyle.Render("🔍 Severity: " + getFilterName(m.filterState)))
	}

	// Show category filter
	if m.categoryFilter != "" {
		builder.WriteString("  ")
		categoryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fddb00")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1).
			Bold(true)
		builder.WriteString(categoryStyle.Render("📂 Category: " + m.categoryFilter))
	}

	// Show rule filter
	if m.ruleFilter != "" {
		builder.WriteString("  ")
		ruleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff00")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1).
			Bold(true)
		builder.WriteString(ruleStyle.Render("📏 Rule: " + m.ruleFilter))
	}

	builder.WriteString("\n\n")

	// Main content area - use consistent height for both states
	contentHeight := m.height - 2 // Reserve space for title (1), blank line (1), and status bar (1)

	if len(m.filteredResults) == 0 {
		// Show empty state with ASCII art
		emptyView := renderEmptyState(m.width, contentHeight)
		builder.WriteString(emptyView)
	} else {
		// Apply colors to table output
		tableView := colorizeTableOutput(m.table.View(), m.table.Cursor(), m.rows)

		// Add borders and separators to the table
		borderedTable := addTableBorders(tableView)

		// Just write the bordered table
		builder.WriteString(borderedTable)
		//builder.WriteString(tableView)
	}

	builder.WriteString("\n")

	// Status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5263"))

	// Show filtered count vs total when any filter is active
	resultsText := fmt.Sprintf("%d results", len(m.filteredResults))
	if (m.filterState != FilterAll || m.categoryFilter != "" || m.ruleFilter != "") && len(m.allResults) > 0 {
		resultsText = fmt.Sprintf("%d/%d results", len(m.filteredResults), len(m.allResults))
	}

	rowText := ""
	if len(m.filteredResults) > 0 {
		rowText = fmt.Sprintf(" • Row %d/%d", m.table.Cursor()+1, len(m.filteredResults))
	}

	status := statusStyle.Render(fmt.Sprintf(
		" %s%s • ↑↓/jk: nav • tab: severity • c: category • r: rule • pgup/pgdn: page • q: quit",
		resultsText,
		rowText))

	builder.WriteString(status)

	return builder.String()
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
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5263"))
	return textStyle.Render(strings.Join(resultLines, "\n"))
}

func addTableBorders(tableView string) string {
	neonPink := lipgloss.Color("#f83aff")
	// Just wrap in a simple border for now
	tableStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(neonPink).PaddingTop(0)

	return tableStyle.Render(tableView)
}

func colorizeTableOutput(tableView string, cursor int, rows []table.Row) string {
	lines := strings.Split(tableView, "\n")

	// Get selected row's location to identify it
	var selectedLocation string
	if cursor >= 0 && cursor < len(rows) {
		selectedLocation = rows[cursor][0]
	}

	// Define tertiary color - lighter gray options:
	// tertiaryColor := "\033[38;2;128;128;128m" // Medium gray #808080
	// tertiaryColor := "\033[38;2;160;160;160m" // Light gray #a0a0a0
	// tertiaryColor := "\033[38;2;192;192;192m" // Silver #c0c0c0
	tertiaryColor := "\033[38;2;144;144;144m" // Nice readable gray #909090
	reset := "\033[0m"

	var result strings.Builder
	for i, line := range lines {
		// Skip coloring for headers and selected row
		isSelectedLine := selectedLocation != "" && strings.Contains(line, selectedLocation)

		// First, handle the path column delimiters for all lines (including header)
		if strings.Contains(line, ":||:") {
			// Find content between delimiters and color it (unless it's the selected line)
			start := strings.Index(line, ":||:")
			if start != -1 {
				// Look for the closing delimiter
				end := strings.Index(line[start+4:], ":||:")
				if end != -1 {
					// Found both delimiters - extract content between them
					end = start + 4 + end
					pathContent := line[start+4 : end]

					coloredPath := pathContent

					// remove any truncated delimiter characters
					if len(coloredPath) > 5 {

						if strings.Contains(coloredPath, ":||") {
							coloredPath = strings.Replace(coloredPath, ":||", "", 1)
							coloredPath = strings.Replace(coloredPath, ":||...", "", 1)
						}
						if strings.Contains(coloredPath, ":|") {
							coloredPath = strings.Replace(coloredPath, ":|", "", 1)
							coloredPath = strings.Replace(coloredPath, ":|...", "", 1)
						}
						if strings.Contains(coloredPath, ":") {
							coloredPath = strings.Replace(coloredPath, ":", "", 1)
							coloredPath = strings.Replace(coloredPath, ":...", "", 1)
						}
					}

					// Color the path content if not selected
					if !isSelectedLine && i > 0 { // Don't color header or selected rows
						coloredPath = tertiaryColor + coloredPath + reset
					}

					// Replace the delimited content with colored version (removing delimiters)
					line = line[:start] + coloredPath + line[end+4:]
				} else {
					// No closing delimiter found (likely truncated)
					// Just remove the opening delimiter and color the rest
					pathContent := line[start+4:]

					// Color the path content if not selected
					coloredPath := pathContent

					// remove any truncated delimiter characters
					if len(coloredPath) > 5 {

						if strings.Contains(coloredPath, ":||") {
							coloredPath = strings.Replace(coloredPath, ":||", "", 1)
							coloredPath = strings.Replace(coloredPath, ":||...", "", 1)
						}
						if strings.Contains(coloredPath, ":|") {
							coloredPath = strings.Replace(coloredPath, ":|", "", 1)
							coloredPath = strings.Replace(coloredPath, ":|...", "", 1)
						}
						if strings.Contains(coloredPath, ":") {
							coloredPath = strings.Replace(coloredPath, ":", "", 1)
							coloredPath = strings.Replace(coloredPath, ":...", "", 1)
						}
					}

					if !isSelectedLine && i > 0 { // Don't color header or selected rows
						coloredPath = tertiaryColor + coloredPath + reset
					}

					// Replace from delimiter to end of line
					line = line[:start] + coloredPath
				}
			}
		}

		if i >= 1 && !isSelectedLine { // Start from line 1 (skip header row at 0)
			// Apply severity colors
			line = strings.Replace(line, " error ", " \033[31merror\033[0m ", -1)
			line = strings.Replace(line, " warning ", " \033[33mwarning\033[0m ", -1)
			line = strings.Replace(line, " info ", " \033[36minfo\033[0m ", -1)
		}

		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
