// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/charmbracelet/bubbles/v2/viewport"
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

// pre-compiled regex patterns for syntax highlighting
var (
	yamlKeyValueRegex = regexp.MustCompile(`^(\s*)([a-zA-Z0-9_-]+)(\s*:\s*)(.*)`)
	yamlListItemRegex = regexp.MustCompile(`^(\s*)(- )(.*)`)
	numberValueRegex  = regexp.MustCompile(`^-?\d+\.?\d*$`)
	jsonKeyRegex      = regexp.MustCompile(`"([^"]+)"\s*:`)
	jsonStringRegex   = regexp.MustCompile(`:\s*"[^"]*"`)
)

// pre-created styles for syntax highlighting
var (
	syntaxKeyStyle         lipgloss.Style
	syntaxStringStyle      lipgloss.Style
	syntaxNumberStyle      lipgloss.Style
	syntaxBoolStyle        lipgloss.Style
	syntaxCommentStyle     lipgloss.Style
	syntaxDashStyle        lipgloss.Style
	syntaxRefStyle         lipgloss.Style // For $ref values
	syntaxDefaultStyle     lipgloss.Style // Default pink for unmatched text
	syntaxSingleQuoteStyle lipgloss.Style // Pink italic for single-quoted strings
	syntaxStylesInit       bool
)

// layout constants
const (
	// terminal dimensions
	defaultTerminalWidth  = 180
	defaultTerminalHeight = 40
	minTableHeight        = 10

	// modal dimensions
	modalWidthReduction = 40 // How much to reduce width for modal
	modalHeightMargin   = 5  // Margin from bottom for modal

	// split view dimensions
	splitViewHeight    = 15 // Fixed height for split view
	splitViewMargin    = 4  // Margin for split view
	splitContentHeight = 11 // Fixed content height inside split view

	// Column width percentages (for split view)
	detailsColumnPercent  = 30 // 30% for details column
	howToFixColumnPercent = 30 // 30% for how-to-fix column
	// codeColumnPercent gets the remainder (40%)

	// Table column percentages
	locationColumnPercent = 25
	messageColumnPercent  = 35
	ruleColumnPercent     = 15

	// Fixed column widths
	severityColumnWidth = 10
	categoryColumnWidth = 12

	// Minimum column widths
	minLocationWidth       = 25
	minMessageWidth        = 40
	minRuleWidth           = 15
	minPathWidth           = 20
	minPathWidthCompressed = 35

	// Code view settings
	codeWindowSize = 3000 // Max lines to show above/below target line

	// Other layout constants
	tableSeparatorWidth = 10
	viewportPadding     = 4
	contentHeightMargin = 4
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
		width = defaultTerminalWidth
	}
	if height == 0 {
		height = defaultTerminalHeight
	}

	columns, rows := BuildResultTableData(results, fileName, width, true)

	tableActualWidth := width - 2
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height-5),
		table.WithWidth(tableActualWidth),
	)

	ApplyLintDetailsTableStyles(&t)

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

func (m *ViolationResultTableModel) Init() tea.Cmd {
	return m.docsSpinner.Tick
}

func (m *ViolationResultTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// viewport updates when modal is open and loaded
	if m.showModal && m.docsState == DocsStateLoaded {
		m.docsViewport, cmd = m.docsViewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// spinner updates when loading
	if m.showModal && m.docsState == DocsStateLoading {
		m.docsSpinner, cmd = m.docsSpinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// documentation messages
	if handled, msgCmd := m.HandleDocsMessages(msg); handled {
		if msgCmd != nil {
			return m, msgCmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := m.HandleWindowResize(msg)
		return m, cmd

	case tea.KeyPressMsg:
		key := msg.String()

		// code view keys
		if handled, cmd := m.HandleCodeViewKeys(key); handled {
			return m, cmd
		}

		// modal keys
		if handled, cmd := m.HandleDocsModalKeys(key); handled {
			return m, cmd
		}

		switch key {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			return m.HandleEscapeKey()
		default:
			// filter keys
			if handled, cmd := m.HandleFilterKeys(key); handled {
				return m, cmd
			}

			// toggle keys
			if handled, cmd := m.HandleToggleKeys(key); handled {
				return m, cmd
			}
		}
	}

	m.table, cmd = m.table.Update(msg)

	// update split view content based on cursor
	m.UpdateDetailsViewContent()

	// combine any commands
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
			filterStyle = GetSeverityInfo(model.SeverityError).TextStyle
		case FilterWarnings:
			severityText = "▲ warnings"
			filterStyle = GetSeverityInfo(model.SeverityWarn).TextStyle
		case FilterInfo:
			severityText = "● info"
			filterStyle = GetSeverityInfo(model.SeverityInfo).TextStyle
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

	contentHeight := m.height - contentHeightMargin
	if contentHeight < minTableHeight {
		contentHeight = minTableHeight
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

// buildModalView builds the documentation modal
func (m *ViolationResultTableModel) buildModalView() string {
	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - modalHeightMargin

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

		errorMsg := fmt.Sprintf("❌ oh dear, failed to load documentation.\n\n%s\n\n"+
			"This is a mistake. It should not have happened, sorry!", m.docsError)
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
	modalHeight := m.height - modalHeightMargin

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
		// join vertically: table on top, split view in the middle, nav at the bottom
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
			modal := m.BuildCodeView()
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
		modal := m.BuildCodeView()
		x, y := m.calculateModalPosition()

		// code view modal as overlay layer (higher z-index than docs)
		layers = append(layers, lipgloss.NewLayer(modal).X(x).Y(y).Z(2))
	}

	canvas := lipgloss.NewCanvas(layers...)
	return canvas.Render()
}
