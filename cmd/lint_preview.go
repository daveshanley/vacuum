// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/table"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
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
	TableWidthAdjustment = 0 // Use full terminal width
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
	modalContent    *model.RuleFunctionResult // Current result being shown in modal/split view
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
func ShowTableLintView(results []*model.RuleFunctionResult, fileName string, specContent []byte) error {
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
	columns, rows := buildTableData(results, fileName, width, true) // Default to showing path

	// Create table
	// Account for the border that will be added by addTableBorders (2 chars)
	tableActualWidth := width - 2
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height-5), // Title (2 lines with blank), table border (2), status (1)
		table.WithWidth(tableActualWidth),     // Account for border wrapper
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

func buildTableData(results []*model.RuleFunctionResult, fileName string, terminalWidth int, showPath bool) ([]table.Column, []table.Row) {
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

		// Build row based on whether we're showing path column
		if showPath {
			rows = append(rows, table.Row{
				location,
				severity,
				r.Message,
				ruleID,
				category,
				":||:" + r.Path + ":||:", // Add delimiters to mark the path column
			})
		} else {
			// No path column
			rows = append(rows, table.Row{
				location,
				severity,
				r.Message,
				ruleID,
				category,
			})
		}
	}

	// Calculate column widths with natural sizes
	locWidth := maxLocWidth
	sevWidth := 9 // Fixed for "warning"
	ruleWidth := maxRuleWidth
	catWidth := maxCatWidth
	
	// Store natural widths for restoration
	naturalRuleWidth := ruleWidth
	naturalCatWidth := catWidth

	// Account for table borders AND internal padding
	columnCount := 5
	if showPath {
		columnCount = 6
	}
	// Account for the border wrapper (2 chars) that will be added
	actualTableWidth := terminalWidth - 2
	// Each column gets 2 chars of padding from the table component
	columnPadding := columnCount * 2
	// Available width for columns = table width - column padding
	availableWidth := actualTableWidth - columnPadding

	// Define minimum widths
	minMsgWidth := 40  // Message should be readable
	minPathWidth := 20 // Minimum for path
	minRuleWidth := 20 // Minimum for rule
	minCatWidth := 20  // Minimum for category

	var msgWidth, pathWidth int

	if showPath {
		// Start with natural message width (approx 60% of typical remaining space)
		// This is our "natural" message width target
		naturalMsgWidth := 80
		naturalPathWidth := 50
		
		// Calculate total natural width
		totalNaturalWidth := locWidth + sevWidth + naturalMsgWidth + naturalRuleWidth + naturalCatWidth + naturalPathWidth
		
		if totalNaturalWidth <= availableWidth {
			// We have enough space for natural widths or more
			msgWidth = naturalMsgWidth
			pathWidth = naturalPathWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth
			
			// Distribute extra space 50/50 between message and path
			extraSpace := availableWidth - totalNaturalWidth
			if extraSpace > 0 {
				msgWidth += extraSpace / 2
				pathWidth += extraSpace - (extraSpace / 2) // Use remainder to avoid rounding issues
			}
		} else {
			// Need to compress - use hierarchical compression
			// Start with natural widths
			msgWidth = naturalMsgWidth
			pathWidth = naturalPathWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth
			
			// Calculate how much we need to save
			needToSave := totalNaturalWidth - availableWidth
			
			// Phase 1: Compress path first
			if needToSave > 0 && pathWidth > minPathWidth {
				canSave := pathWidth - minPathWidth
				if canSave >= needToSave {
					pathWidth -= needToSave
					needToSave = 0
				} else {
					pathWidth = minPathWidth
					needToSave -= canSave
				}
			}
			
			// Phase 2: Compress category
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
			
			// Phase 3: Compress rule
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
			
			// Phase 4: Last resort - compress message
			if needToSave > 0 && msgWidth > minMsgWidth {
				canSave := msgWidth - minMsgWidth
				if canSave >= needToSave {
					msgWidth -= needToSave
				} else {
					msgWidth = minMsgWidth
				}
			}
		}
	} else {
		// No path column - simpler calculation
		naturalMsgWidth := 100
		totalNaturalWidth := locWidth + sevWidth + naturalMsgWidth + naturalRuleWidth + naturalCatWidth
		
		if totalNaturalWidth <= availableWidth {
			// We have enough space
			msgWidth = naturalMsgWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth
			
			// Give all extra space to message
			extraSpace := availableWidth - totalNaturalWidth
			if extraSpace > 0 {
				msgWidth += extraSpace
			}
		} else {
			// Need to compress
			msgWidth = naturalMsgWidth
			ruleWidth = naturalRuleWidth
			catWidth = naturalCatWidth
			
			needToSave := totalNaturalWidth - availableWidth
			
			// Phase 1: Compress category
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
			
			// Phase 2: Compress rule
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
			
			// Phase 3: Last resort - compress message
			if needToSave > 0 && msgWidth > minMsgWidth {
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

	// CRITICAL: Ensure columns sum to EXACTLY match available width
	// The table component doesn't stretch rows, so we need exact match
	var totalColWidth int
	if showPath {
		totalColWidth = locWidth + sevWidth + msgWidth + ruleWidth + catWidth + pathWidth
	} else {
		totalColWidth = locWidth + sevWidth + msgWidth + ruleWidth + catWidth
	}

	targetWidth := availableWidth // Match calculated available width
	widthDiff := targetWidth - totalColWidth

	// Add any difference to the message column (or path if shown)
	if widthDiff > 0 {
		if showPath {
			pathWidth += widthDiff
		} else {
			msgWidth += widthDiff
		}
	} else if widthDiff < 0 {
		// If we're over, reduce appropriate column
		if showPath {
			pathWidth += widthDiff // widthDiff is negative, so this reduces
			if pathWidth < 35 {
				// If path becomes too small, reduce message instead
				msgWidth += widthDiff
				pathWidth = 35
			}
		} else {
			msgWidth += widthDiff
		}
	}

	// Build columns array based on showPath
	columns := []table.Column{
		{Title: "Location", Width: locWidth},
		{Title: "Severity", Width: sevWidth},
		{Title: "Message", Width: msgWidth},
		{Title: "Rule", Width: ruleWidth},
		{Title: "Category", Width: catWidth},
	}

	if showPath {
		columns = append(columns, table.Column{
			Title: ":||:Path:||:", Width: pathWidth,
		})
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
	columns, rows := buildTableData(m.filteredResults, m.fileName, m.width, m.showPath)
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
		columns, rows := buildTableData(m.filteredResults, m.fileName, msg.Width, m.showPath)
		m.table.SetColumns(columns)
		m.table.SetRows(rows)
		m.table.SetWidth(msg.Width - 2) // Account for border wrapper

		// Adjust table height based on split view state
		if m.showSplitView {
			// When split view is open, table gets remaining space after fixed split view
			tableHeight := m.height - 15 - 5 // terminal height - split view height - margins
			if tableHeight < 10 {
				tableHeight = 10 // Minimum height
			}
			m.table.SetHeight(tableHeight)
		} else {
			m.table.SetHeight(msg.Height - 5)
		}

		// Reapply styles after resize
		applyTableStyles(&m.table)

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
				m.table.SetHeight(m.height - 5)
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
				tableHeight := m.height - 15 - 5 // terminal height - split view - margins
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
			applyTableStyles(&m.table)

			// Restore cursor position and viewport
			if currentCursor < len(rows) {
				// First go to top to reset viewport
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
func (m TableLintModel) buildTableView() string {
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
	}

	// Remove extra newline - no status bar needed

	return builder.String()
}

// wrapText wraps text to fit within a specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) <= width {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// extractCodeSnippet extracts lines around the issue with context
func (m TableLintModel) extractCodeSnippet(result *model.RuleFunctionResult, contextLines int) (string, int) {
	if m.specContent == nil || result == nil {
		return "", 0
	}

	// Get the line number from the result
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

	// Split content into lines
	lines := bytes.Split(m.specContent, []byte("\n"))

	// Calculate start and end lines with context
	startLine := line - contextLines - 1 // -1 because line numbers are 1-based
	if startLine < 0 {
		startLine = 0
	}

	endLine := line + contextLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Build the snippet
	var snippet strings.Builder
	for i := startLine; i < endLine; i++ {
		snippet.Write(lines[i])
		if i < endLine-1 {
			snippet.WriteString("\n")
		}
	}

	return snippet.String(), startLine + 1 // Return 1-based line number for display
}

// buildSplitView builds the split view content with three columns
func (m TableLintModel) buildSplitView() string {
	if m.modalContent == nil {
		return ""
	}

	// Calculate dimensions - FIXED height regardless of terminal size
	splitHeight := 15 // Fixed compact height with proper padding
	if m.height < 20 {
		return "" // Don't show split view if terminal too small
	}

	splitWidth := m.width // Match terminal width for consistency

	// Content height for the three columns (must be exact same for all)
	contentHeight := 11 // Fixed content height for all columns

	// Column widths: details (25%), how-to-fix (35%), code (40%)
	// Adjust for container padding
	innerWidth := splitWidth - 4 // Account for container borders and padding
	detailsWidth := int(float64(innerWidth) * 0.25)
	howToFixWidth := int(float64(innerWidth) * 0.35)
	codeWidth := innerWidth - detailsWidth - howToFixWidth

	// Styles
	neonPink := lipgloss.Color("#f83aff")
	blue := lipgloss.Color("#62c4ff")
	gray := lipgloss.Color("#909090")

	// Extract code snippet
	codeSnippet, startLine := m.extractCodeSnippet(m.modalContent, 4)

	// BUILD PATH BAR
	pathBarStyle := lipgloss.NewStyle().
		Width(splitWidth-2).
		Padding(0, 1). // Simple horizontal padding only
		Foreground(gray)

	path := m.modalContent.Path
	if path == "" && m.modalContent.Paths != nil && len(m.modalContent.Paths) > 0 {
		path = m.modalContent.Paths[0]
	}

	// Truncate path if too long (at the end as requested)
	maxPathWidth := splitWidth - 6 // Account for container borders and padding
	if len(path) > maxPathWidth && maxPathWidth > 3 {
		path = path[:maxPathWidth-3] + "..."
	}

	pathBar := pathBarStyle.Render(path)

	// BUILD DETAILS COLUMN
	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)

	var detailsContent strings.Builder

	// Get severity for emoji
	severity := getSeverity(m.modalContent)
	var emoji string
	var emojiStyle lipgloss.Style
	switch severity {
	case "error":
		emoji = "✗" // Red cross
		emojiStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)
	case "warning":
		emoji = "▲" // Yellow triangle
		emojiStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	case "info":
		emoji = "●" // Blue filled dot
		emojiStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00aaff")).Bold(true)
	default:
		emoji = "●"
		emojiStyle = lipgloss.NewStyle().Foreground(gray).Bold(true)
	}

	ruleName := "Issue"
	if m.modalContent.Rule != nil && m.modalContent.Rule.Id != "" {
		ruleName = m.modalContent.Rule.Id
	}

	// Title line with emoji and rule name
	titleStyle := lipgloss.NewStyle().Foreground(neonPink).Bold(true)
	detailsContent.WriteString(fmt.Sprintf("%s %s", emojiStyle.Render(emoji), titleStyle.Render(ruleName)))
	detailsContent.WriteString("\n")

	// Location line
	location := formatLocation(m.modalContent, m.fileName)
	detailsContent.WriteString(lipgloss.NewStyle().Foreground(blue).Render(location))
	detailsContent.WriteString("\n\n")

	// Message (wrap to fit width)
	msgStyle := lipgloss.NewStyle().Foreground(neonPink).Width(detailsWidth - 2)
	detailsContent.WriteString(msgStyle.Render(m.modalContent.Message))

	detailsPanel := detailsStyle.Render(detailsContent.String())

	// BUILD HOW-TO-FIX COLUMN
	howToFixStyle := lipgloss.NewStyle().
		Width(howToFixWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)

	var howToFixContent strings.Builder

	// Start at same line as details title
	//howToFixContent.WriteString("\n")

	// How to fix content
	if m.modalContent.Rule != nil && m.modalContent.Rule.HowToFix != "" {
		// Split how-to-fix into lines and format
		fixLines := strings.Split(m.modalContent.Rule.HowToFix, "\n")
		for i, line := range fixLines {
			if i > 0 {
				howToFixContent.WriteString("\n")
			}
			// Wrap long lines
			wrapped := wrapText(line, howToFixWidth-4)
			howToFixContent.WriteString(wrapped)
		}
	} else {
		howToFixContent.WriteString(lipgloss.NewStyle().Foreground(gray).Italic(true).
			Render("No fix suggestions available"))
	}

	howToFixPanel := howToFixStyle.Render(howToFixContent.String())

	// BUILD CODE COLUMN
	codeStyle := lipgloss.NewStyle().
		Width(codeWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)

	var codeContent strings.Builder

	// Start at same line as details title
	//codeContent.WriteString("\n")

	// Code snippet with line numbers
	if codeSnippet != "" {
		codeLines := strings.Split(codeSnippet, "\n")
		lineNumStyle := lipgloss.NewStyle().Foreground(gray).Bold(true)
		codeTextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
		highlightStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#5a0000")).
			Foreground(lipgloss.Color("#ffaaaa"))

		for i, codeLine := range codeLines {
			actualLineNum := startLine + i
			isHighlighted := false

			// Check if this is the error line
			if m.modalContent.StartNode != nil && actualLineNum == m.modalContent.StartNode.Line {
				isHighlighted = true
			} else if m.modalContent.Origin != nil && actualLineNum == m.modalContent.Origin.Line {
				isHighlighted = true
			}

			// Format line number with padding
			lineNumStr := fmt.Sprintf("%4d ", actualLineNum)
			codeContent.WriteString(lineNumStyle.Render(lineNumStr))

			// Apply highlighting if needed
			if isHighlighted {
				codeContent.WriteString(highlightStyle.Render(codeLine))
			} else {
				codeContent.WriteString(codeTextStyle.Render(codeLine))
			}

			if i < len(codeLines)-1 {
				codeContent.WriteString("\n")
			}
		}
	} else {
		codeContent.WriteString(lipgloss.NewStyle().Foreground(gray).Italic(true).
			Render("No code context available"))
	}

	codePanel := codeStyle.Render(codeContent.String())

	// Combine all three columns horizontally
	combinedPanels := lipgloss.JoinHorizontal(lipgloss.Top,
		detailsPanel,
		howToFixPanel,
		codePanel,
	)

	// Add a blank line between path and panels for spacing
	spacer := lipgloss.NewStyle().Height(1).Render(" ")

	// Combine path bar and panels vertically with spacer
	combinedContent := lipgloss.JoinVertical(lipgloss.Left,
		pathBar,
		spacer,
		combinedPanels,
	)

	// Wrap in a container with blue border
	containerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(blue).
		Width(splitWidth).
		Height(splitHeight)

	return containerStyle.Render(combinedContent)
}

// buildModalView builds the DOCS modal placeholder
func (m TableLintModel) buildModalView() string {
	// Calculate modal dimensions
	modalWidth := 40
	modalHeight := 10

	if m.modalContent == nil {
		return ""
	}

	// Styles
	neonPink := lipgloss.Color("#f83aff")
	blue := lipgloss.Color("#62c4ff")

	// Modal style
	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(neonPink).
		Background(lipgloss.Color("#1a1a1a"))

	// Content
	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(blue).
		Bold(true).
		Align(lipgloss.Center).
		Width(modalWidth - 4)

	content.WriteString(titleStyle.Render("📚 DOCS"))
	content.WriteString("\n\n")

	// Placeholder message
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#909090")).
		Align(lipgloss.Center).
		Width(modalWidth - 4)

	content.WriteString(msgStyle.Render("Documentation coming soon..."))

	return modalStyle.Render(content.String())
}

// buildModalView_old builds the enhanced modal content with panels
func (m TableLintModel) buildModalView_old() string {
	// Calculate modal dimensions - fixed height, responsive width
	modalWidth := int(float64(m.width) * 0.75)
	modalHeight := 35 // Fixed height for consistent appearance

	if m.modalContent == nil {
		return ""
	}

	// Styles
	neonPink := lipgloss.Color("#f83aff")
	blue := lipgloss.Color("#62c4ff")
	yellow := lipgloss.Color("#fddb00")
	gray := lipgloss.Color("#4B5263")

	// Calculate panel dimensions with fixed heights
	topPanelHeight := 20 // Fixed height for top panels
	leftPanelWidth := int(float64(modalWidth) * 0.4)
	rightPanelWidth := int(float64(modalWidth) * 0.6)

	// Extract code snippet
	codeSnippet, startLine := m.extractCodeSnippet(m.modalContent, 4)

	// Build LEFT PANEL - Details
	leftPanelStyle := lipgloss.NewStyle().
		Width(leftPanelWidth-4).
		Height(topPanelHeight).
		Padding(1, 2)

	var leftContent strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Foreground(neonPink).Bold(true)
	leftContent.WriteString(titleStyle.Render("📍 Issue Details"))
	leftContent.WriteString("\n\n")

	// Location
	location := formatLocation(m.modalContent, m.fileName)
	leftContent.WriteString(lipgloss.NewStyle().Foreground(blue).Render("Location: "))
	leftContent.WriteString(location)
	leftContent.WriteString("\n\n")

	// Rule
	if m.modalContent.Rule != nil {
		leftContent.WriteString(lipgloss.NewStyle().Foreground(blue).Render("Rule: "))
		leftContent.WriteString(m.modalContent.Rule.Id)
		leftContent.WriteString("\n\n")

		// Severity
		severity := getSeverity(m.modalContent)
		sevColor := gray
		switch severity {
		case "error":
			sevColor = lipgloss.Color("#ff0000")
		case "warning":
			sevColor = lipgloss.Color("#ffaa00")
		case "info":
			sevColor = lipgloss.Color("#00aaff")
		}
		leftContent.WriteString(lipgloss.NewStyle().Foreground(blue).Render("Severity: "))
		leftContent.WriteString(lipgloss.NewStyle().Foreground(sevColor).Bold(true).Render(severity))
		leftContent.WriteString("\n\n")
	}

	// Message
	leftContent.WriteString(lipgloss.NewStyle().Foreground(blue).Render("Message:"))
	leftContent.WriteString("\n")
	leftContent.WriteString(lipgloss.NewStyle().Foreground(yellow).Render(m.modalContent.Message))
	leftContent.WriteString("\n\n")

	// Description (if available from rule)
	if m.modalContent.Rule != nil && m.modalContent.Rule.Description != "" {
		leftContent.WriteString(lipgloss.NewStyle().Foreground(blue).Render("Description:"))
		leftContent.WriteString("\n")
		// Create a viewport for scrollable description
		descLines := strings.Split(m.modalContent.Rule.Description, "\n")
		for i, line := range descLines {
			if i > 5 { // Limit to first 5 lines in this view
				leftContent.WriteString("...")
				break
			}
			leftContent.WriteString(line)
			if i < len(descLines)-1 {
				leftContent.WriteString("\n")
			}
		}
	}

	leftPanel := leftPanelStyle.Render(leftContent.String())

	// Build RIGHT PANEL - Code Snippet
	var rightContent strings.Builder

	// Code header
	codeHeaderStyle := lipgloss.NewStyle().Foreground(neonPink).Bold(true)
	rightContent.WriteString(codeHeaderStyle.Render("📝 Code"))
	rightContent.WriteString("\n")

	// Count actual lines for dynamic height
	codeLines := 1 // Start with header line

	// Create a read-only textarea for code display
	if codeSnippet != "" {
		// Add line numbers to the snippet
		snippetLines := strings.Split(codeSnippet, "\n")
		codeLines += len(snippetLines)

		for i, line := range snippetLines {
			lineNum := startLine + i
			lineNumStr := fmt.Sprintf("%4d │ ", lineNum)

			// Highlight the error line
			if m.modalContent.StartNode != nil && lineNum == m.modalContent.StartNode.Line {
				rightContent.WriteString(lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render(lineNumStr))
				rightContent.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#3a1a1a")).Render(line))
			} else if m.modalContent.Origin != nil && lineNum == m.modalContent.Origin.Line {
				rightContent.WriteString(lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render(lineNumStr))
				rightContent.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#3a1a1a")).Render(line))
			} else {
				rightContent.WriteString(lipgloss.NewStyle().Foreground(gray).Render(lineNumStr))
				rightContent.WriteString(line)
			}

			if i < len(snippetLines)-1 {
				rightContent.WriteString("\n")
			}
		}
	} else {
		rightContent.WriteString("\nNo code snippet available")
		codeLines += 1
	}

	// Make the right panel height fit content but not exceed available space
	rightPanelHeight := codeLines + 2 // Add padding for border
	if rightPanelHeight > topPanelHeight {
		rightPanelHeight = topPanelHeight
	}

	rightPanelStyle := lipgloss.NewStyle().
		Width(rightPanelWidth-4).
		Height(rightPanelHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(gray).
		Padding(0, 1)

	rightPanel := rightPanelStyle.Render(rightContent.String())

	// Join panels side by side
	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Build BOTTOM SECTION - How to Fix
	howToFixHeight := 8 // Fixed height for how-to-fix section
	howToFixStyle := lipgloss.NewStyle().
		Width(modalWidth-6).
		MaxHeight(howToFixHeight).
		Padding(1, 2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(gray).
		BorderTop(true)

	var howToFixContent strings.Builder

	fixHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true)
	howToFixContent.WriteString(fixHeaderStyle.Render("💡 How to Fix"))
	howToFixContent.WriteString("\n")

	if m.modalContent.Rule != nil && m.modalContent.Rule.HowToFix != "" {
		// Truncate if too long - keep it very compact
		fixText := m.modalContent.Rule.HowToFix
		lines := strings.Split(fixText, "\n")
		maxLines := 3 // Even fewer lines for compact view
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			lines = append(lines, "...")
		}
		howToFixContent.WriteString(strings.Join(lines, "\n"))
	} else {
		howToFixContent.WriteString("No fix suggestions available for this rule.")
	}

	howToFixSection := howToFixStyle.Render(howToFixContent.String())

	// Combine all sections
	fullContent := lipgloss.JoinVertical(lipgloss.Left, topSection, howToFixSection)

	// Wrap in modal border - don't duplicate the header
	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		MaxHeight(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonPink).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(1, 2)

	// Build the modal without redundant header
	var modalBuilder strings.Builder
	modalBuilder.WriteString(fullContent)
	modalBuilder.WriteString("\n\n")

	footerStyle := lipgloss.NewStyle().
		Foreground(gray).
		Width(modalWidth - 8).
		Align(lipgloss.Center)
	modalBuilder.WriteString(footerStyle.Render("Press ESC, Q, or Enter to close"))

	return modalStyle.Render(modalBuilder.String())
}

// calculateModalPosition calculates the position for the modal (right-aligned)
func (m TableLintModel) calculateModalPosition() (int, int) {
	modalWidth := int(float64(m.width) * 0.75)
	modalHeight := 35 // Fixed height matching buildModalView

	// Position on the right side with good padding
	rightPadding := 8 // Good distance from right edge
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

func (m TableLintModel) View() string {
	if m.quitting {
		return ""
	}

	// Build the base table view
	tableView := m.buildTableView()

	// Build navigation bar (always at bottom)
	navStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5263")).
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
		splitView := m.buildSplitView()
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
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5263"))
	return textStyle.Render(strings.Join(resultLines, "\n"))
}

func addTableBorders(tableView string) string {
	neonPink := lipgloss.Color("#f83aff")
	// Just wrap in a simple border for now
	tableStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(neonPink).
		PaddingTop(0)

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
