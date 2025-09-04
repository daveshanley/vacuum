// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/model"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// regular expressions.
var locationRegex = regexp.MustCompile(`((?:[a-zA-Z]:)?[^\s│]*?[/\\]?[^\s│/\\]+\.[a-zA-Z]+):(\d+):(\d+)`)
var jsonPathRegex = regexp.MustCompile(`\$\.\S+`)
var circularRefRegex = regexp.MustCompile(`\b[a-zA-Z0-9_-]+(?:\s*->\s*[a-zA-Z0-9_-]+)+\b`)
var partRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)|(\s*->\s*)`)

// Pre-compiled regex patterns for syntax highlighting (performance optimization)
var (
	yamlKeyValueRegex = regexp.MustCompile(`^(\s*)([a-zA-Z0-9_-]+)(\s*:\s*)(.*)`)
	yamlListItemRegex = regexp.MustCompile(`^(\s*)(- )(.*)`)
	numberValueRegex  = regexp.MustCompile(`^-?\d+\.?\d*$`)
	jsonKeyRegex      = regexp.MustCompile(`"([^"]+)"\s*:`)
	jsonStringRegex   = regexp.MustCompile(`:\s*"[^"]*"`)
)

// Pre-created styles for syntax highlighting (created on first use)
var (
	syntaxKeyStyle     lipgloss.Style
	syntaxStringStyle  lipgloss.Style
	syntaxNumberStyle  lipgloss.Style
	syntaxBoolStyle    lipgloss.Style
	syntaxCommentStyle lipgloss.Style
	syntaxDashStyle    lipgloss.Style
	syntaxStylesInit   bool
)

// FilterState represents the current filter mode for cycling through severities
type FilterState int

const (
	FilterAll      FilterState = iota // Show all results
	FilterErrors                      // Show only errors
	FilterWarnings                    // Show only warnings
	FilterInfo                        // Show only info messages
)

// DocsState represents the state of documentation fetching
type DocsState int

const (
	DocsStateLoading DocsState = iota
	DocsStateLoaded
	DocsStateError
)

// docsLoadedMsg is sent when documentation is successfully loaded
type docsLoadedMsg struct {
	ruleID  string
	content string
}

// docsErrorMsg is sent when documentation loading fails
type docsErrorMsg struct {
	ruleID string
	err    string
	is404  bool
}

// ViolationResultTableModel holds the state for the interactive table view
type ViolationResultTableModel struct {
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
	showSplitView   bool                      // Whether to show the split view (details)
	showCodeView    bool                      // Whether to show the expanded code view modal
	modalContent    *model.RuleFunctionResult // The current result being shown in the splitview
	docsState       DocsState                 // State of documentation loading
	docsContent     string                    // Loaded documentation content
	docsError       string                    // Error message if docs failed to load
	docsCache       map[string]string         // Cache of loaded documentation by rule ID
	docsSpinner     spinner.Model             // Spinner for loading state
	docsViewport    viewport.Model            // Viewport for scrollable docs content
	codeViewport    viewport.Model            // Viewport for expanded code view
}

// ShowViolationTableView displays results in an interactive console table
func ShowViolationTableView(results []*model.RuleFunctionResult, fileName string, specContent []byte) error {
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
		table.WithHeight(height-5), // title (2 lines with blank), table border (2), status (1)
		table.WithWidth(tableActualWidth),
	)

	applyLintDetailsTableStyles(&t)

	categories := extractCategories(results)
	rules := extractRules(results)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(RGBPink)

	// initialize viewport (will be sized when modal opens)
	vp := viewport.New()

	m := &ViolationResultTableModel{
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
		docsCache:       make(map[string]string),
		docsSpinner:     s,
		docsViewport:    vp,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func (m *ViolationResultTableModel) applyFilter() {
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

func (m *ViolationResultTableModel) Init() tea.Cmd {
	return m.docsSpinner.Tick
}

func (m *ViolationResultTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle viewport updates when modal is open and loaded
	if m.showModal && m.docsState == DocsStateLoaded {
		m.docsViewport, cmd = m.docsViewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Handle spinner updates when loading
	if m.showModal && m.docsState == DocsStateLoading {
		m.docsSpinner, cmd = m.docsSpinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	switch msg := msg.(type) {
	case docsLoadedMsg:
		// Cache the content
		m.docsCache[msg.ruleID] = msg.content
		m.docsContent = msg.content
		m.docsState = DocsStateLoaded

		modalWidth := int(float64(m.width) - 40)

		customStyle := CreateVacuumDocsStyle(modalWidth - 4)
		renderer, err := glamour.NewTermRenderer(
			glamour.WithColorProfile(termenv.TrueColor),
			glamour.WithStyles(customStyle),
			glamour.WithWordWrap(modalWidth-4),
		)
		if err == nil {
			rendered, err := renderer.Render(msg.content)
			if err == nil {
				m.docsContent = rendered
			} else {
				// Fallback to raw content if rendering fails
				m.docsContent = msg.content
			}
		} else {
			// Fallback to raw content if renderer creation fails
			m.docsContent = msg.content
		}

		// Update viewport with rendered content
		m.docsViewport.SetContent(m.docsContent)
		m.docsViewport.GotoTop()
		return m, nil

	case docsErrorMsg:
		m.docsState = DocsStateError
		m.docsError = msg.err
		return m, nil
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
		// Handle code view modal keys first
		if m.showCodeView {
			switch msg.String() {
			case "up", "k":
				m.codeViewport.LineUp(1)
				return m, nil
			case "down", "j":
				m.codeViewport.LineDown(1)
				return m, nil
			case "pgup", "pageup", "page up":
				m.codeViewport.ViewUp()
				return m, nil
			case "pgdn", "pagedown", "page down", "pgdown":
				m.codeViewport.ViewDown()
				return m, nil
			case "home", "g":
				m.codeViewport.GotoTop()
				return m, nil
			case "end", "G":
				m.codeViewport.GotoBottom()
				return m, nil
			case " ", "space":
				// Recenter on the highlighted line
				m.recenterCodeView()
				return m, nil
			case "esc", "q", "x":
				m.showCodeView = false
				return m, nil
			}
			// Don't process other keys when code view is open
			return m, nil
		}

		// Handle modal-specific keys 
		if m.showModal {
			// Allow viewport navigation when docs are loaded
			if m.docsState == DocsStateLoaded {
				switch msg.String() {
				case "up", "k":
					m.docsViewport.LineUp(1)
					return m, nil
				case "down", "j":
					m.docsViewport.LineDown(1)
					return m, nil
				case "pgup":
					m.docsViewport.ViewUp()
					return m, nil
				case "pgdn":
					m.docsViewport.ViewDown()
					return m, nil
				case "home", "g":
					m.docsViewport.GotoTop()
					return m, nil
				case "end", "G":
					m.docsViewport.GotoBottom()
					return m, nil
				}
			}

			switch msg.String() {
			case "esc", "q", "enter":
				m.showModal = false
				// Don't clear modalContent if split view is still open
				if !m.showSplitView {
					m.modalContent = nil
				}
				// Reset docs state for next open
				m.docsState = DocsStateLoading
				return m, nil
			case "d":
				// Toggle docs modal off with 'd' key
				m.showModal = false
				// Don't clear modalContent if split view is still open
				if !m.showSplitView {
					m.modalContent = nil
				}
				// Reset docs state for next open
				m.docsState = DocsStateLoading
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
			// If on empty state (no results), clear all filters
			if len(m.filteredResults) == 0 && (m.filterState != FilterAll || m.categoryFilter != "" || m.ruleFilter != "") {
				// Clear all filters
				m.filterState = FilterAll
				m.categoryFilter = ""
				m.ruleFilter = ""
				m.applyFilter()

				// Rebuild the table with all results
				_, rows := buildTableData(m.filteredResults, m.fileName, m.width, m.showPath)
				m.rows = rows
				m.table.SetRows(rows)

				// Reset cursor position
				if len(rows) > 0 {
					m.table.SetCursor(0)
				}
				return m, nil
			}

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
				m.table.SetHeight(m.height - 4)
			}
			return m, nil
		case "x":
			// Toggle expanded code view modal
			if m.table.Cursor() < len(m.filteredResults) {
				// Set the current result for the code view
				if !m.showSplitView {
					m.modalContent = m.filteredResults[m.table.Cursor()]
				}
				m.showCodeView = !m.showCodeView
				
				// If opening code view, prepare the viewport
				if m.showCodeView {
					m.prepareCodeViewport()
				}
			}
			return m, nil
		case "d":
			// Toggle DOCS modal with selected result
			if m.table.Cursor() < len(m.filteredResults) {
				// If split view is open, preserve its modalContent
				if !m.showSplitView {
					m.modalContent = m.filteredResults[m.table.Cursor()]
				}
				m.showModal = !m.showModal

				// If opening modal, fetch documentation
				if m.showModal && m.modalContent != nil && m.modalContent.Rule != nil {
					ruleID := m.modalContent.Rule.Id

					// Check cache first
					if cached, exists := m.docsCache[ruleID]; exists {
						m.docsContent = cached
						m.docsState = DocsStateLoaded

						// Re-render markdown for current terminal size
						modalWidth := int(float64(m.width) - 40)

						// Use custom style with TrueColor profile
						customStyle := CreateVacuumDocsStyle(modalWidth - 4)
						renderer, err := glamour.NewTermRenderer(
							glamour.WithColorProfile(termenv.TrueColor),
							glamour.WithStyles(customStyle),
							glamour.WithWordWrap(modalWidth-4),
						)
						if err == nil {
							rendered, err := renderer.Render(cached)
							if err == nil {
								m.docsContent = rendered
							} else {
								// Fallback to raw content if rendering fails
								m.docsContent = cached
							}
						} else {
							// Fallback to raw content if renderer creation fails
							m.docsContent = cached
						}

						// Update viewport
						m.docsViewport.SetContent(m.docsContent)
						m.docsViewport.SetWidth(modalWidth - 4)
						m.docsViewport.SetHeight(m.height - 14)
						m.docsViewport.GotoTop()
					} else {
						// Start loading
						m.docsState = DocsStateLoading
						m.docsContent = ""
						m.docsError = ""

						// Update viewport size
						modalWidth := int(float64(m.width) - 40)
						m.docsViewport.SetWidth(modalWidth - 4)
						m.docsViewport.SetHeight(m.height - 14)

						// Return both fetch command and spinner tick
						return m, tea.Batch(fetchDocsFromDoctorAPI(ruleID), m.docsSpinner.Tick)
					}
				}
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

	// Combine any commands
	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, cmd
}

// buildTableView builds the complete table view with title, filters, and status bar
func (m *ViolationResultTableModel) buildTableView() string {
	var builder strings.Builder

	// Count violations by severity from all results (not filtered)
	errorCount := 0
	warningCount := 0
	infoCount := 0
	for _, r := range m.allResults {
		if r.Rule != nil {
			switch r.Rule.Severity {
			case model.SeverityError:
				errorCount++
			case model.SeverityWarn:
				warningCount++
			case model.SeverityInfo:
				infoCount++
			}
		}
	}

	// Build the title with total count and severity breakdown
	titleStyle := lipgloss.NewStyle().
		Foreground(RGBPink)

	totalCount := fmt.Sprintf(" %d Violations", len(m.allResults))
	builder.WriteString(titleStyle.Render(totalCount))

	// Add severity breakdown with colored icons
	builder.WriteString("  ")

	// Errors (red cross)
	errorStyle := lipgloss.NewStyle().Foreground(RGBRed)
	builder.WriteString(errorStyle.Render(fmt.Sprintf("✗ %d", errorCount)))

	builder.WriteString("  ")

	// Warnings (yellow triangle)
	warningStyle := lipgloss.NewStyle().Foreground(RBGYellow)
	builder.WriteString(warningStyle.Render(fmt.Sprintf("▲ %d", warningCount)))

	builder.WriteString("  ")

	// Info (blue dot)
	infoStyle := lipgloss.NewStyle().Foreground(RGBBlue)
	builder.WriteString(infoStyle.Render(fmt.Sprintf("● %d", infoCount)))

	// Now add filters if any are active
	if m.filterState != FilterAll {
		builder.WriteString(" | ")

		// "Severity:" in gray, then colored icon and label
		grayStyle := lipgloss.NewStyle().Foreground(RGBGrey)
		builder.WriteString(grayStyle.Render("severity: "))

		// Build severity filter with colored icon
		var severityText string
		var filterStyle lipgloss.Style
		switch m.filterState {
		case FilterErrors:
			severityText = "✗ errors"
			filterStyle = lipgloss.NewStyle().Foreground(RGBRed)
		case FilterWarnings:
			severityText = "▲ warnings"
			filterStyle = lipgloss.NewStyle().Foreground(RBGYellow)
		case FilterInfo:
			severityText = "● info"
			filterStyle = lipgloss.NewStyle().Foreground(RGBBlue)
		}

		builder.WriteString(filterStyle.Render(severityText))
	}

	if m.categoryFilter != "" {
		builder.WriteString(" | ")
		categoryStyle := lipgloss.NewStyle().
			Foreground(RGBGrey)
		builder.WriteString(categoryStyle.Render("category: " + m.categoryFilter))
	}

	if m.ruleFilter != "" {
		builder.WriteString(" | ")
		ruleStyle := lipgloss.NewStyle().
			Foreground(RGBGrey)
		builder.WriteString(ruleStyle.Render("rule: " + m.ruleFilter))
	}

	builder.WriteString("\n")

	contentHeight := m.height - 4
	if contentHeight < 10 {
		contentHeight = 10
	}

	if len(m.filteredResults) == 0 {
		// empty state.
		emptyView := renderEmptyState(m.width-2, contentHeight)
		borderedEmpty := addTableBorders(emptyView)
		builder.WriteString(borderedEmpty)
	} else {

		tableView := ColorizeTableOutput(m.table.View(), m.table.Cursor(), m.rows)
		borderedTable := addTableBorders(tableView)
		builder.WriteString(borderedTable)
	}

	return builder.String()
}

// extractCodeSnippet extracts lines around the issue with context
func (m *ViolationResultTableModel) extractCodeSnippet(result *model.RuleFunctionResult, contextLines int) (string, int) {
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

// buildModalView builds the documentation modal
func (m *ViolationResultTableModel) buildModalView() string {
	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - 5

	if m.modalContent == nil {
		return ""
	}

	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(0, 1, 0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink)

	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(RGBBlue).
		Bold(true).
		Width(modalWidth - 4)

	ruleName := "Documentation"
	if m.modalContent.Rule != nil && m.modalContent.Rule.Id != "" {
		ruleName = fmt.Sprintf("📚 %s", m.modalContent.Rule.Id)
	}
	content.WriteString(titleStyle.Render(ruleName))
	content.WriteString("\n")

	sepStyle := lipgloss.NewStyle().
		Foreground(RGBPink).
		Width(modalWidth - 4)
	content.WriteString(sepStyle.Render(strings.Repeat("-", (modalWidth)-4)))
	content.WriteString("\n\n")

	contentHeight := modalHeight - 4 // account for title, separator, and padding

	switch m.docsState {
	case DocsStateLoading:
		spinnerStyle := lipgloss.NewStyle().
			Width(modalWidth-4).
			Height(contentHeight).
			Align(lipgloss.Left, lipgloss.Top)

		spinnerContent := fmt.Sprintf("%s loading documentation...", m.docsSpinner.View())
		content.WriteString(spinnerStyle.Render(spinnerContent))

	case DocsStateLoaded:
		content.WriteString(m.docsViewport.View())

	case DocsStateError:
		errorStyle := lipgloss.NewStyle().
			Padding(1).
			Width(modalWidth-4).
			Height(contentHeight).
			Align(lipgloss.Left, lipgloss.Top).
			Foreground(RGBRed)

		errorMsg := fmt.Sprintf("❌ oh dear, failed to load documentation.\n\n%s\n\nThis is a mistake. It should not have happened, sorry!", m.docsError)
		content.WriteString(errorStyle.Render(errorMsg))

	default:
		content.WriteString("")
	}

	currentLines := strings.Count(content.String(), "\n")
	neededLines := modalHeight - currentLines - 3
	if neededLines > 0 {
		content.WriteString(strings.Repeat("\n", neededLines))
	}

	// bottom bar with scroll percentage and controls
	var bottomBar string
	if m.docsState == DocsStateLoaded && m.docsViewport.TotalLineCount() > m.docsViewport.Height() {
		scrollPercent := fmt.Sprintf(" %.0f%%", m.docsViewport.ScrollPercent()*100)
		scrollStyle := lipgloss.NewStyle().
			Foreground(RGBBlue)

		controls := "↑↓/jk: scroll | pgup/pgdn: page | esc/d: close "
		controlsStyle := lipgloss.NewStyle().
			Foreground(RGBGrey)

		// calculate spacing to align left and right
		scrollWidth := lipgloss.Width(scrollPercent)
		controlsWidth := lipgloss.Width(controls)
		spacerWidth := (modalWidth - 4) - scrollWidth - controlsWidth
		if spacerWidth < 0 {
			spacerWidth = 1
		}

		// combine with spacing
		bottomBar = scrollStyle.Render(scrollPercent) +
			strings.Repeat(" ", spacerWidth) +
			controlsStyle.Render(controls)
	} else {

		// no scrolling, just show controls centered
		navStyle := lipgloss.NewStyle().
			Foreground(RGBDarkGrey).
			Width(modalWidth - 4).
			Align(lipgloss.Center)
		bottomBar = navStyle.Render("esc/d: close")
	}

	content.WriteString(bottomBar)

	return modalStyle.Render(content.String())
}

// calculateModalPosition calculates the position for the modal (right-aligned)
func (m *ViolationResultTableModel) calculateModalPosition() (int, int) {
	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - 5

	// position on the right side with padding from the edge
	rightPadding := 6
	x := m.width - modalWidth - rightPadding

	// center vertically
	y := (m.height - modalHeight) / 2

	// ensure positive values
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return x, y
}

// formatCodeWithGlamour uses glamour to render the code with syntax highlighting
func (m *ViolationResultTableModel) formatCodeWithGlamour(targetLine int, width int) string {
	// Determine if this is YAML or JSON
	isYAML := strings.HasSuffix(m.fileName, ".yaml") || strings.HasSuffix(m.fileName, ".yml")
	
	// Wrap the content in a code block for glamour
	var codeBlock strings.Builder
	codeBlock.WriteString("```")
	if isYAML {
		codeBlock.WriteString("yaml")
	} else {
		codeBlock.WriteString("json")
	}
	codeBlock.WriteString("\n")
	codeBlock.WriteString(string(m.specContent))
	codeBlock.WriteString("\n```\n")
	
	// Create custom style for code rendering
	customStyle := CreateVacuumDocsStyle(width)
	
	// Create glamour renderer with our custom style
	renderer, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(termenv.TrueColor),
		glamour.WithStyles(customStyle),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		// Fallback to plain text with line numbers
		return m.formatCodeWithLineNumbers(targetLine)
	}
	
	// Render the code block
	rendered, err := renderer.Render(codeBlock.String())
	if err != nil {
		// Fallback to plain text with line numbers
		return m.formatCodeWithLineNumbers(targetLine)
	}
	
	// Add line highlighting if we have a target line
	if targetLine > 0 {
		return m.addLineHighlight(rendered, targetLine)
	}
	
	return rendered
}

// formatCodeWithLineNumbers provides a simple fallback with just line numbers
func (m *ViolationResultTableModel) formatCodeWithLineNumbers(targetLine int) string {
	lines := strings.Split(string(m.specContent), "\n")
	var result strings.Builder
	
	lineNumStyle := lipgloss.NewStyle().Foreground(RGBGrey)
	highlightStyle := lipgloss.NewStyle().
		Background(RGBSubtlePink).
		Foreground(RGBPink).
		Bold(true)
	
	maxLineNum := len(lines)
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum)) + 1
	if lineNumWidth < 5 {
		lineNumWidth = 5
	}
	
	for i, line := range lines {
		lineNum := i + 1
		isHighlighted := lineNum == targetLine
		
		lineNumStr := fmt.Sprintf("%*d ", lineNumWidth-1, lineNum)
		
		if isHighlighted {
			highlightedLineNumStyle := lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
			result.WriteString(highlightedLineNumStyle.Render(lineNumStr))
			result.WriteString(highlightStyle.Render(line))
		} else {
			result.WriteString(lineNumStyle.Render(lineNumStr))
			result.WriteString(line)
		}
		
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// addLineHighlight adds highlighting to a specific line in rendered content
func (m *ViolationResultTableModel) addLineHighlight(content string, targetLine int) string {
	lines := strings.Split(content, "\n")
	highlightStyle := lipgloss.NewStyle().
		Background(RGBSubtlePink).
		Foreground(RGBPink).
		Bold(true)
	
	// Glamour adds some formatting, so we need to be careful about line counting
	// Skip empty lines and formatting lines at the beginning
	codeStarted := false
	actualLine := 0
	
	for i, line := range lines {
		// Look for the start of actual code content
		if !codeStarted && strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "```") {
			codeStarted = true
		}
		
		if codeStarted {
			actualLine++
			if actualLine == targetLine {
				lines[i] = highlightStyle.Render(line)
			}
		}
	}
	
	return strings.Join(lines, "\n")
}

// recenterCodeView recenters the viewport on the highlighted error line
func (m *ViolationResultTableModel) recenterCodeView() {
	if m.modalContent == nil {
		return
	}
	
	// Get the target line number
	targetLine := 0
	if m.modalContent.StartNode != nil {
		targetLine = m.modalContent.StartNode.Line
	} else if m.modalContent.Origin != nil {
		targetLine = m.modalContent.Origin.Line
	}
	
	if targetLine > 0 {
		// Calculate the position of the target line within the rendered content
		allLines := strings.Split(string(m.specContent), "\n")
		totalLines := len(allLines)
		const windowSize = 1000
		
		var targetPositionInWindow int
		if totalLines <= (windowSize*2 + 1) {
			// No windowing, target is at its actual position
			targetPositionInWindow = targetLine
		} else {
			// Windowing is active
			startLine := targetLine - windowSize
			if startLine < 1 {
				startLine = 1
			}
			// Account for the "lines above not shown" notice if present
			if startLine > 1 {
				targetPositionInWindow = targetLine - startLine + 2 // +2 for the notice line
			} else {
				targetPositionInWindow = targetLine - startLine + 1
			}
		}
		
		// Center the target line in the viewport
		scrollTo := targetPositionInWindow - (m.codeViewport.Height() / 2)
		if scrollTo < 0 {
			scrollTo = 0
		}
		m.codeViewport.SetYOffset(scrollTo)
	}
}

// prepareCodeViewport prepares the code viewport with the full spec and highlights the error line
func (m *ViolationResultTableModel) prepareCodeViewport() {
	if m.modalContent == nil || m.specContent == nil {
		return
	}

	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - 5

	// Initialize viewport
	m.codeViewport = viewport.New(viewport.WithWidth(modalWidth-4), viewport.WithHeight(modalHeight-4))

	// Get the line number from the result
	targetLine := 0
	if m.modalContent.StartNode != nil {
		targetLine = m.modalContent.StartNode.Line
	} else if m.modalContent.Origin != nil {
		targetLine = m.modalContent.Origin.Line
	}

	// Use custom syntax highlighting (faster than glamour)
	content := m.formatCodeWithHighlight(targetLine, modalWidth-8)
	
	m.codeViewport.SetContent(content)

	// Scroll to the target line (try to center it in the viewport)
	if targetLine > 0 {
		// For windowed content, we need to calculate the position within the rendered content
		// The target line is always at position 1000 (or less if near the start of file)
		allLines := strings.Split(string(m.specContent), "\n")
		totalLines := len(allLines)
		const windowSize = 1000
		
		// Calculate where the target line appears in our rendered content
		var targetPositionInWindow int
		if totalLines <= (windowSize*2 + 1) {
			// No windowing, target is at its actual position
			targetPositionInWindow = targetLine
		} else {
			// Windowing is active
			startLine := targetLine - windowSize
			if startLine < 1 {
				startLine = 1
			}
			// Account for the "lines above not shown" notice if present
			if startLine > 1 {
				targetPositionInWindow = targetLine - startLine + 2 // +2 for the notice line
			} else {
				targetPositionInWindow = targetLine - startLine + 1
			}
		}
		
		// Now scroll to center the target line in the viewport
		scrollTo := targetPositionInWindow - (m.codeViewport.Height() / 2)
		if scrollTo < 0 {
			scrollTo = 0
		}
		m.codeViewport.SetYOffset(scrollTo)
	}
}

// formatCodeWithHighlight formats the spec content with line numbers and highlights the error line
func (m *ViolationResultTableModel) formatCodeWithHighlight(targetLine int, maxWidth int) string {
	allLines := strings.Split(string(m.specContent), "\n")
	totalLines := len(allLines)
	
	// Window configuration - max 1000 lines above and below target
	const windowSize = 1000
	
	// Calculate the window of lines to render
	startLine := 1
	endLine := totalLines
	actualTargetLine := targetLine // Track the actual line number for highlighting
	
	if totalLines > (windowSize*2 + 1) {
		// Need to limit the window
		if targetLine > 0 {
			// Calculate window centered on target line
			startLine = targetLine - windowSize
			if startLine < 1 {
				startLine = 1
			}
			endLine = targetLine + windowSize
			if endLine > totalLines {
				endLine = totalLines
			}
		} else {
			// No target line, show first 2001 lines
			endLine = windowSize*2 + 1
			if endLine > totalLines {
				endLine = totalLines
			}
		}
	}
	
	// Extract the lines to render (convert to 0-based indexing)
	lines := allLines[startLine-1 : endLine]
	
	var result strings.Builder
	lineNumStyle := lipgloss.NewStyle().Foreground(RGBGrey)
	highlightStyle := lipgloss.NewStyle().
		Background(RGBSubtlePink).
		Foreground(RGBPink).
		Bold(true)
	
	// Determine if this is YAML or JSON
	isYAML := strings.HasSuffix(m.fileName, ".yaml") || strings.HasSuffix(m.fileName, ".yml")
	
	// Calculate line number width based on actual max line number
	lineNumWidth := len(fmt.Sprintf("%d", endLine)) + 1
	if lineNumWidth < 5 {
		lineNumWidth = 5
	}
	
	// Add a notice if we're showing a limited window
	if startLine > 1 {
		noticeStyle := lipgloss.NewStyle().Foreground(RGBGrey).Italic(true)
		result.WriteString(noticeStyle.Render(fmt.Sprintf("    ... (%d lines above not shown) ...", startLine-1)))
		result.WriteString("\n")
	}
	
	for i, line := range lines {
		lineNum := startLine + i // Actual line number in the file
		isHighlighted := lineNum == actualTargetLine
		
		// Format line number
		lineNumStr := fmt.Sprintf("%*d ", lineNumWidth-1, lineNum)
		
		if isHighlighted {
			highlightedLineNumStyle := lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
			result.WriteString(highlightedLineNumStyle.Render(lineNumStr))
		} else {
			result.WriteString(lineNumStyle.Render(lineNumStr))
		}
		
		// Apply simple syntax highlighting
		coloredLine := m.applySyntaxHighlighting(line, isYAML)
		
		if isHighlighted {
			// Pad the line to full width for background color
			displayLine := line
			if len(line) < maxWidth-lineNumWidth {
				displayLine = line + strings.Repeat(" ", maxWidth-lineNumWidth-len(line))
			}
			result.WriteString(highlightStyle.Render(displayLine))
		} else {
			result.WriteString(coloredLine)
		}
		
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	
	// Add a notice if we're cutting off lines at the bottom
	if endLine < totalLines {
		result.WriteString("\n")
		noticeStyle := lipgloss.NewStyle().Foreground(RGBGrey).Italic(true)
		result.WriteString(noticeStyle.Render(fmt.Sprintf("    ... (%d lines below not shown) ...", totalLines-endLine)))
	}
	
	return result.String()
}

// initSyntaxStyles initializes the syntax highlighting styles once
func initSyntaxStyles() {
	if !syntaxStylesInit {
		syntaxKeyStyle = lipgloss.NewStyle().Foreground(RGBBlue)
		syntaxStringStyle = lipgloss.NewStyle().Foreground(RGBGreen)
		syntaxNumberStyle = lipgloss.NewStyle().Foreground(RGBOrange)
		syntaxBoolStyle = lipgloss.NewStyle().Foreground(RGBPurple)
		syntaxCommentStyle = lipgloss.NewStyle().Foreground(RGBGrey)
		syntaxDashStyle = lipgloss.NewStyle().Foreground(RGBPink)
		syntaxStylesInit = true
	}
}

// applySyntaxHighlighting applies basic syntax highlighting for YAML/JSON
func (m *ViolationResultTableModel) applySyntaxHighlighting(line string, isYAML bool) string {
	// Initialize styles once
	initSyntaxStyles()
	
	// Fast path: empty line
	if line == "" {
		return line
	}
	
	// Handle comments for YAML (check for # first as a quick filter)
	if isYAML && strings.IndexByte(line, '#') >= 0 {
		commentIndex := strings.IndexByte(line, '#')
		beforeComment := line[:commentIndex]
		comment := line[commentIndex:]
		return m.applySyntaxHighlighting(beforeComment, isYAML) + syntaxCommentStyle.Render(comment)
	}
	
	if isYAML {
		// YAML key-value pairs (use pre-compiled regex)
		if matches := yamlKeyValueRegex.FindStringSubmatch(line); matches != nil {
			indent := matches[1]
			key := matches[2]
			separator := matches[3]
			value := matches[4]
			
			// Fast checks for common values
			coloredValue := value
			trimmedValue := strings.TrimSpace(value)
			
			// Check boolean values first (most common)
			switch trimmedValue {
			case "true", "false", "null":
				coloredValue = syntaxBoolStyle.Render(value)
			default:
				// Check if it's a number (use pre-compiled regex)
				if numberValueRegex.MatchString(trimmedValue) {
					coloredValue = syntaxNumberStyle.Render(value)
				} else if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
					// Check for quoted strings (fast byte check)
					coloredValue = syntaxStringStyle.Render(value)
				}
			}
			
			return indent + syntaxKeyStyle.Render(key) + separator + coloredValue
		}
		
		// YAML list items (use pre-compiled regex)
		if matches := yamlListItemRegex.FindStringSubmatch(line); matches != nil {
			return matches[1] + syntaxDashStyle.Render(matches[2]) + matches[3]
		}
	} else {
		// JSON - use pre-compiled regexes
		line = jsonKeyRegex.ReplaceAllStringFunc(line, func(match string) string {
			return syntaxKeyStyle.Render(match)
		})
		line = jsonStringRegex.ReplaceAllStringFunc(line, func(match string) string {
			parts := strings.SplitN(match, "\"", 2)
			if len(parts) > 1 {
				return parts[0] + syntaxStringStyle.Render("\""+parts[1])
			}
			return match
		})
	}
	
	return line
}

// buildCodeView builds the expanded code view modal
func (m *ViolationResultTableModel) buildCodeView() string {
	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - 5

	if m.modalContent == nil {
		return ""
	}

	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(0, 1, 0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink)

	var content strings.Builder

	// Title bar with filename and line number
	titleStyle := lipgloss.NewStyle().
		Foreground(RGBBlue).
		Bold(true).
		Width(modalWidth - 4)

	targetLine := 0
	if m.modalContent.StartNode != nil {
		targetLine = m.modalContent.StartNode.Line
	} else if m.modalContent.Origin != nil {
		targetLine = m.modalContent.Origin.Line
	}

	title := fmt.Sprintf("📄 %s - line %d", m.fileName, targetLine)
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")

	// Separator
	sepStyle := lipgloss.NewStyle().
		Foreground(RGBPink).
		Width(modalWidth - 4)
	content.WriteString(sepStyle.Render(strings.Repeat("-", modalWidth-4)))
	content.WriteString("\n\n")

	// Code viewport
	content.WriteString(m.codeViewport.View())

	// Calculate remaining lines for proper modal height
	currentLines := strings.Count(content.String(), "\n")
	neededLines := modalHeight - currentLines - 3
	if neededLines > 0 {
		content.WriteString(strings.Repeat("\n", neededLines))
	}

	// Bottom bar with scroll percentage and controls
	var bottomBar string
	if m.codeViewport.TotalLineCount() > m.codeViewport.Height() {
		scrollPercent := fmt.Sprintf(" %.0f%%", m.codeViewport.ScrollPercent()*100)
		scrollStyle := lipgloss.NewStyle().Foreground(RGBBlue)

		controls := "↑↓/jk: scroll | pgup/pgdn: page | space: recenter | esc/x: close "
		controlsStyle := lipgloss.NewStyle().Foreground(RGBGrey)

		// Calculate spacing
		scrollWidth := lipgloss.Width(scrollPercent)
		controlsWidth := lipgloss.Width(controls)
		spacerWidth := (modalWidth - 4) - scrollWidth - controlsWidth
		if spacerWidth < 0 {
			spacerWidth = 1
		}

		bottomBar = scrollStyle.Render(scrollPercent) +
			strings.Repeat(" ", spacerWidth) +
			controlsStyle.Render(controls)
	} else {
		// No scrolling needed
		navStyle := lipgloss.NewStyle().
			Foreground(RGBDarkGrey).
			Width(modalWidth - 4).
			Align(lipgloss.Center)
		bottomBar = navStyle.Render("esc/x: close")
	}

	content.WriteString(bottomBar)

	return modalStyle.Render(content.String())
}

func (m *ViolationResultTableModel) View() string {
	if m.quitting {
		return ""
	}

	tableView := m.buildTableView()

	navStyle := lipgloss.NewStyle().
		Foreground(RGBGrey).
		Width(m.width)

	rowText := ""
	if len(m.filteredResults) > 0 {
		rowText = fmt.Sprintf(" %d/%d", m.table.Cursor()+1, len(m.filteredResults))
	}

	navBar := navStyle.Render(fmt.Sprintf("%s | pgup/pgdn/↑↓/jk: nav | tab: severity | c: category | r: rule | p: path | enter: details | d: docs | x: code | q: quit", rowText))

	if m.showSplitView {
		detailsView := m.BuildDetailsView()
		// Join vertically: table on top, split view in the middle, nav at the bottom
		combined := lipgloss.JoinVertical(lipgloss.Left, tableView, detailsView, navBar)

		layers := []*lipgloss.Layer{
			lipgloss.NewLayer(combined),
		}

		// docs modal
		if m.showModal {
			modal := m.buildModalView()
			x, y := m.calculateModalPosition()

			// docs modal as overlay layer
			layers = append(layers, lipgloss.NewLayer(modal).X(x).Y(y).Z(1))
		}

		// code view modal
		if m.showCodeView {
			modal := m.buildCodeView()
			x, y := m.calculateModalPosition()

			// code view modal as overlay layer (higher z-index than docs)
			layers = append(layers, lipgloss.NewLayer(modal).X(x).Y(y).Z(2))
		}

		// render canvas with all layers
		canvas := lipgloss.NewCanvas(layers...)
		return canvas.Render()
	}

	// normal view without split - nav at bottom
	combined := lipgloss.JoinVertical(lipgloss.Left, tableView, navBar)
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(combined),
	}

	if m.showModal {
		modal := m.buildModalView()
		x, y := m.calculateModalPosition()

		// docs modal as overlay layer
		layers = append(layers, lipgloss.NewLayer(modal).X(x).Y(y).Z(1))
	}

	// code view modal
	if m.showCodeView {
		modal := m.buildCodeView()
		x, y := m.calculateModalPosition()

		// code view modal as overlay layer (higher z-index than docs)
		layers = append(layers, lipgloss.NewLayer(modal).X(x).Y(y).Z(2))
	}

	canvas := lipgloss.NewCanvas(layers...)
	return canvas.Render()
}
