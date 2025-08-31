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
		BorderRight(true).
		Foreground(neonPink).
		Bold(true).
		Padding(0, 1)
	
	// Selected row style with proper highlighting
	s.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#3a3a3a")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(neonPink).
		BorderRight(true).
		Padding(0, 1)
	
	// Regular cells with column separators
	s.Cell = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(neonPink).
		BorderRight(true).
		Padding(0, 1)
	
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
	columns, rows := buildTableData(results, fileName, width)

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height-3), // Title (1), status (1), spacing (1)
		table.WithWidth(width-3), // Account for borders
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

		rows = append(rows, table.Row{
			location,
			severity,
			r.Message,
			ruleID,
			category,
			r.Path,
		})
	}

	// Calculate column widths with minimal padding
	locWidth := maxLocWidth + 1  // Minimal padding
	sevWidth := 9                // Fixed for "warning" 
	ruleWidth := maxRuleWidth + 1
	catWidth := maxCatWidth + 1
	
	// Calculate remaining space for message and path columns
	fixedWidth := locWidth + sevWidth + ruleWidth + catWidth
	availableWidth := width - 12 // Account for borders and column separators
	
	// Split remaining space between message (70%) and path (30%)
	remainingWidth := availableWidth - fixedWidth
	msgWidth := (remainingWidth * 70) / 100
	pathWidth := remainingWidth - msgWidth
	
	// Ensure minimum widths
	if msgWidth < 50 {
		msgWidth = 50
	}
	if pathWidth < 25 {
		pathWidth = 25
		// Recalculate message width with minimum path
		msgWidth = availableWidth - fixedWidth - pathWidth
	}

	columns := []table.Column{
		{Title: "Location", Width: locWidth},
		{Title: "Severity", Width: sevWidth},
		{Title: "Message", Width: msgWidth},
		{Title: "Rule", Width: ruleWidth},
		{Title: "Category", Width: catWidth},
		{Title: "Path", Width: pathWidth},
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
	columns, rows := buildTableData(m.filteredResults, m.fileName, m.width)
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
		columns, rows := buildTableData(m.filteredResults, m.fileName, msg.Width)
		m.table.SetColumns(columns)
		m.table.SetRows(rows)
		m.table.SetWidth(msg.Width - 3) // Account for borders
		m.table.SetHeight(msg.Height - 3)
		
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
	
	builder.WriteString("\n")

	// Main content area - use consistent height for both states
	contentHeight := m.height - 1 // Reserve space for status bar only (title already handled)
	
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
		BorderForeground(neonPink)
	
	return tableStyle.Render(tableView)
}

func colorizeTableOutput(tableView string, cursor int, rows []table.Row) string {
	lines := strings.Split(tableView, "\n")
	
	// Get selected row's location to identify it
	var selectedLocation string
	if cursor >= 0 && cursor < len(rows) {
		selectedLocation = rows[cursor][0]
	}
	
	var result strings.Builder
	for i, line := range lines {
		// Skip coloring for headers and selected row
		isSelectedLine := selectedLocation != "" && strings.Contains(line, selectedLocation)
		
		if i >= 2 && !isSelectedLine {
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