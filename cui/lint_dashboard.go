// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
	"golang.org/x/term"
)

const (
	DocsStateLoading DocsState = iota
	DocsStateLoaded
	DocsStateError
)

const (
	FilterAll      FilterState = iota // Show all results
	FilterErrors                      // Show only errors
	FilterWarnings                    // Show only warnings
	FilterInfo                        // Show only info messages
)

// watch constants
const (
	WatchDebounceDelay = 200 * time.Millisecond
)

type WatchState int

const (
	WatchStateIdle WatchState = iota
	WatchStateProcessing
	WatchStateError
)

// layout constants
const (
	DefaultTerminalWidth   = 180
	DefaultTerminalHeight  = 40
	MinTableHeight         = 10
	ModalWidthReduction    = 40 // How much to reduce width for modal
	ModalHeightMargin      = 5  // Margin from bottom for modal
	SplitViewHeight        = 15 // Fixed height for detail view
	SplitViewMargin        = 4  // Margin for split view
	SplitContentHeight     = 11 // Fixed content height inside detail view
	DetailsColumnPercent   = 30 // 30% for details column
	HowToFixColumnPercent  = 30 // 30% for how-to-fix column
	SeverityColumnWidth    = 10
	MinMessageWidth        = 40
	MinPathWidth           = 20
	MinPathWidthCompressed = 35
	CodeWindowSize         = 3000 // Max lines to show above/below target line
	ViewportPadding        = 4
	ContentHeightMargin    = 4
)

var LocationRegex = regexp.MustCompile(`((?:[a-zA-Z]:)?[^\s│]*?[/\\]?[^\s│/\\]+\.[a-zA-Z]+):(\d+):(\d+)`)
var JsonPathRegex = regexp.MustCompile(`\$\.\S+`)
var CircularRefRegex = regexp.MustCompile(`\b[a-zA-Z0-9_-]+(?:\s*->\s*[a-zA-Z0-9_-]+)+\b`)
var PartRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)|(\s*->\s*)`)

// pre-compiled regex patterns for syntax highlighting
var (
	YamlKeyValueRegex = regexp.MustCompile(`^(\s*)([$a-zA-Z0-9_-]+)(\s*:\s*)(.*)`)
	YamlListItemRegex = regexp.MustCompile(`^(\s*)(- )(.*)`)
	NumberValueRegex  = regexp.MustCompile(`^-?\d+\.?\d*$`)
	JsonKeyRegex      = regexp.MustCompile(`"([^"]+)"\s*:`)
	JsonStringRegex   = regexp.MustCompile(`:\s*"[^"]*"`)
	BacktickRegex     = regexp.MustCompile("`([^`]+)`")
	SingleQuoteRegex  = regexp.MustCompile(`'([^']+)'`)
	LogPrefixRegex    = regexp.MustCompile(`\[([^\]]+)\]`)
)

// syntax highlighting styles are now centralized in styles.go
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

// FilterState represents the current filter mode for cycling through severities
type FilterState int

const (
	AllSeverity FilterState = iota
	ErrorSeverity
	WarningSeverity
	InfoSeverity
)

// String returns the string representation of the FilterState
func (f FilterState) String() string {
	switch f {
	case AllSeverity:
		return "All"
	case ErrorSeverity:
		return "Errors"
	case WarningSeverity:
		return "Warnings"
	case InfoSeverity:
		return "Info"
	default:
		return "Unknown"
	}
}

// ViewMode represents the primary view state
type ViewMode int

const (
	ViewModeTable ViewMode = iota
	ViewModeTableWithSplit
)

// ModalType represents which modal is currently open
type ModalType int

const (
	ModalNone ModalType = iota
	ModalDocs
	ModalCode
)

// UIState encapsulates all UI state
type UIState struct {
	ViewMode       ViewMode
	ActiveModal    ModalType
	ShowPath       bool
	FilterState    FilterState
	CategoryFilter string
	RuleFilter     string
}

// DocsState represents the state of documentation fetching
type DocsState int

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

// file watcher message types
type fileChangeMsg struct {
	fileName string
}

type relintCompleteMsg struct {
	results     []*model.RuleFunctionResult
	specContent []byte
	selectedRow int // Preserve selected row position
}

type relintErrorMsg struct {
	err error
}

type continueWatchingMsg struct{} // Message to restart watching

type clearProcessingStateMsg struct{} // Message to clear processing state after delay

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
	uiState         UIState

	// filter management
	categories    []string // Unique categories from results
	categoryIndex int      // Current category filter index (-1 = all)
	rules         []string // Unique rule IDs from results
	ruleIndex     int      // Current rule filter index (-1 = all)

	// current selection
	modalContent *model.RuleFunctionResult // The current result being shown in the splitview
	docsState    DocsState                 // State of documentation loading
	docsContent  string                    // Loaded documentation content
	docsError    string                    // Error message if docs failed to load
	docsCache    map[string]string         // Cache of loaded documentation by rule ID
	docsSpinner  spinner.Model             // Spinner for loading state
	docsViewport viewport.Model            // Viewport for scrollable docs content
	codeViewport viewport.Model            // Viewport for expanded code view
	err          error                     // Track any errors that occur during operation

	// file watching
	watchConfig    *WatchConfig      // Configuration for file watching
	watchState     WatchState        // Current watch state
	watchError     string            // Error message for watch operations
	watcher        *fsnotify.Watcher // File system watcher
	watchedFiles   []string          // List of files being watched
	debounceTimer  *time.Timer       // Timer for debouncing file changes
	lastChangeTime time.Time         // Last time a file change was detected
	watchMsgChan   chan tea.Msg      // Channel for file watcher messages
}

// WatchConfig holds configuration for file watching
type WatchConfig struct {
	Enabled         bool
	BaseFlag        string
	SkipCheckFlag   bool
	TimeoutFlag     int
	HardModeFlag    bool
	RemoteFlag      bool
	IgnoreFile      string
	FunctionsFlag   string
	RulesetFlag     string
	CertFile        string
	KeyFile         string
	CAFile          string
	Insecure        bool
	Silent          bool
	CustomFunctions map[string]model.RuleFunction // Pre-loaded custom functions
}

// ShowDashboard displays the dashboard with channel-based communication
func ShowDashboard(cuiCommands chan interface{}, userInputs chan interface{}, fileName string) error {
	// For now, just display a placeholder until we fully implement the channel architecture
	fmt.Printf("Dashboard with channel architecture - file: %s\n", fileName)
	fmt.Println("Press 'q' to quit")
	
	// Simple input loop
	for {
		var input string
		fmt.Scanln(&input)
		if input == "q" {
			break
		}
	}
	
	return nil
}

// ShowViolationTableView displays results in an interactive console table (legacy)
func ShowViolationTableView(results []*model.RuleFunctionResult, fileName string, specContent []byte, watchConfig *WatchConfig) error {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\n\033[31mDashboard panic recovered: %v\033[0m\n", r)
			_, _ = fmt.Fprintf(os.Stderr, "Stack trace:\n")
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", buf[:n])
		}
	}()

	if len(results) == 0 {
		return nil
	}

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not get terminal size: %v\n", err)
		width = DefaultTerminalWidth
		height = DefaultTerminalHeight
	}
	if width == 0 {
		width = DefaultTerminalWidth
	}
	if height == 0 {
		height = DefaultTerminalHeight
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

		uiState: UIState{
			ViewMode:       ViewModeTable,
			ActiveModal:    ModalNone,
			ShowPath:       true,
			FilterState:    FilterAll,
			CategoryFilter: "",
			RuleFilter:     "",
		},

		categories:    categories,
		categoryIndex: -1, // -1 means "All"
		rules:         rules,
		ruleIndex:     -1, // -1 means "All"
		docsCache:     make(map[string]string),
		docsSpinner:   s,
		docsViewport:  vp,

		// watch initialization
		watchConfig:  watchConfig,
		watchState:   WatchStateIdle,
		watchedFiles: []string{},
		watchMsgChan: make(chan tea.Msg, 10), // Buffered channel for messages
	}

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		// Log the error details
		fmt.Fprintf(os.Stderr, "\n\033[31mDashboard error: %v\033[0m\n", err)
		return fmt.Errorf("dashboard exited with error: %w", err)
	}

	if finalM, ok := finalModel.(*ViolationResultTableModel); ok {
		// Cleanup watcher before exit
		if finalM.watcher != nil {
			_ = finalM.watcher.Close()
		}
		if finalM.watchMsgChan != nil {
			close(finalM.watchMsgChan)
		}

		if finalM.err != nil {
			fmt.Fprintf(os.Stderr, "\n\033[31mDashboard internal error: %v\033[0m\n", finalM.err)
			return finalM.err
		}
	}

	return nil
}

func (m *ViolationResultTableModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.docsSpinner.Tick)

	// Initialize file watcher if enabled
	if m.watchConfig != nil && m.watchConfig.Enabled {
		cmd := m.setupFileWatcher()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

func (m *ViolationResultTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	defer func() {
		if r := recover(); r != nil {
			m.err = fmt.Errorf("update table panic: %v", r)
			m.quitting = true

			_, _ = fmt.Fprintf(os.Stderr, "\n\033[31mUpdate panic: %v\033[0m\n", r)
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			_, _ = fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", buf[:n])
		}
	}()

	var cmd tea.Cmd
	var cmds []tea.Cmd

	// viewport updates when modal is open and loaded
	if m.uiState.ActiveModal == ModalDocs && m.docsState == DocsStateLoaded {
		m.docsViewport, cmd = m.docsViewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// spinner updates when loading
	if m.uiState.ActiveModal == ModalDocs && m.docsState == DocsStateLoading {
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
	// file watcher messages
	case fileChangeMsg:
		if m.watchConfig != nil && m.watchConfig.Enabled {
			cmd := m.handleFileChange(msg.fileName)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Continue listening for channel messages
			cmds = append(cmds, m.listenForChannelMessages())
		}

	case relintCompleteMsg:
		// Update results immediately but keep processing state for 700ms
		m.allResults = msg.results
		m.specContent = msg.specContent

		// Rebuild filtered results with current filters
		m.filterResults()

		// Rebuild table data
		columns, rows := BuildResultTableData(m.filteredResults, m.fileName, m.width, m.uiState.ShowPath)
		m.table.SetColumns(columns)
		m.table.SetRows(rows)
		m.rows = rows

		// Preserve selection by line number and column
		m.preserveSelection(msg.selectedRow)

		// Schedule clearing the processing state after 700ms
		cmds = append(cmds, m.clearProcessingStateAfterDelay())

		// Continue listening for channel messages
		cmds = append(cmds, m.listenForChannelMessages())

	case relintErrorMsg:
		// Re-linting failed
		m.watchState = WatchStateError
		m.watchError = msg.err.Error()

		// Continue listening for channel messages even after error
		cmds = append(cmds, m.listenForChannelMessages())

	case continueWatchingMsg:
		// Restart listening for channel messages
		if m.watchConfig != nil && m.watchConfig.Enabled {
			cmds = append(cmds, m.listenForChannelMessages())
		}

	case clearProcessingStateMsg:
		// Clear the processing state to hide the green circle
		if m.watchState == WatchStateProcessing {
			m.watchState = WatchStateIdle
		}

	case tea.MouseWheelMsg:
		// mouse wheel scrolling
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelUp:
			// up - same as pressing up arrow
			if m.uiState.ActiveModal == ModalCode {
				// code view is open, scroll in code view
				m.codeViewport.LineUp(3)
			} else {
				// scroll table up
				m.table.MoveUp(3)
			}
		case tea.MouseWheelDown:
			// down - same as pressing down arrow
			if m.uiState.ActiveModal == ModalCode {
				// code view is open, scroll in code view
				m.codeViewport.LineDown(3)
			} else {
				// Scroll table down
				m.table.MoveDown(3)
			}
		}
		// update selected item after scroll
		if m.table.Cursor() < len(m.filteredResults) {
			m.modalContent = m.filteredResults[m.table.Cursor()]
		}
		return m, nil

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
	if m.uiState.FilterState != FilterAll {
		builder.WriteString(" | ")

		// "Severity:" in gray, then colored icon and label
		grayStyle := lipgloss.NewStyle().Foreground(RGBGrey)
		builder.WriteString(grayStyle.Render("severity: "))

		// Build severity filter with colored icon
		var severityText string
		var filterStyle lipgloss.Style
		switch m.uiState.FilterState {
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

	if m.uiState.CategoryFilter != "" {
		builder.WriteString(" | ")
		categoryStyle := lipgloss.NewStyle().
			Foreground(RGBGrey)
		builder.WriteString(categoryStyle.Render("category: " + m.uiState.CategoryFilter))
	}

	if m.uiState.RuleFilter != "" {
		builder.WriteString(" | ")
		ruleStyle := lipgloss.NewStyle().
			Foreground(RGBGrey)
		builder.WriteString(ruleStyle.Render("rule: " + m.uiState.RuleFilter))
	}

	builder.WriteString("\n")

	contentHeight := m.height - ContentHeightMargin
	if contentHeight < MinTableHeight {
		contentHeight = MinTableHeight
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
	modalHeight := m.height - ModalHeightMargin

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
	modalHeight := m.height - ModalHeightMargin

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
	navBar := m.buildNavBar()

	// build base view based on view mode
	var baseView string
	if m.uiState.ViewMode == ViewModeTableWithSplit {
		detailsView := m.BuildDetailsView()
		baseView = lipgloss.JoinVertical(lipgloss.Left, tableView, detailsView, navBar)
	} else {
		baseView = lipgloss.JoinVertical(lipgloss.Left, tableView, navBar)
	}

	// create layers
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(baseView),
	}

	// add modal layer if active
	if m.uiState.ActiveModal != ModalNone {
		modal := m.renderActiveModal()
		if modal != "" {
			x, y := m.calculateModalPosition()
			layers = append(layers, lipgloss.NewLayer(modal).X(x).Y(y).Z(1))
		}
	}

	canvas := lipgloss.NewCanvas(layers...)
	return canvas.Render()
}

// buildNavBar builds the navigation bar at the bottom
func (m *ViolationResultTableModel) buildNavBar() string {
	navStyle := lipgloss.NewStyle().
		Foreground(RGBGrey).
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

// renderActiveModal renders the currently active modal
func (m *ViolationResultTableModel) renderActiveModal() string {
	switch m.uiState.ActiveModal {
	case ModalDocs:
		return m.buildModalView()
	case ModalCode:
		return m.BuildCodeView()
	default:
		return ""
	}
}

// state transition functions - update both old and new state during migration

// ToggleSplitView toggles between table and table with split view
func (m *ViolationResultTableModel) ToggleSplitView() {
	if m.uiState.ViewMode == ViewModeTable {
		m.uiState.ViewMode = ViewModeTableWithSplit
	} else {
		m.uiState.ViewMode = ViewModeTable
	}
}

// OpenModal opens a modal and closes any existing modal
func (m *ViolationResultTableModel) OpenModal(modal ModalType) {
	m.uiState.ActiveModal = modal
}

// CloseActiveModal closes the currently open modal
func (m *ViolationResultTableModel) CloseActiveModal() {
	m.uiState.ActiveModal = ModalNone
}

// TogglePathColumn toggles the path column visibility with viewport preservation
func (m *ViolationResultTableModel) TogglePathColumn() {
	m.uiState.ShowPath = !m.uiState.ShowPath

	currentCursor := m.table.Cursor()
	viewportHeight := m.table.Height()

	viewportStart := 0
	if currentCursor > viewportHeight/2 {
		viewportStart = currentCursor - viewportHeight/2
	}
	cursorOffsetInViewport := currentCursor - viewportStart

	columns, rows := BuildResultTableData(m.filteredResults, m.fileName, m.width, m.uiState.ShowPath)
	m.rows = rows

	// clear and update table
	m.table.SetRows([]table.Row{})
	m.table.SetColumns(columns)
	m.table.SetRows(rows)

	// reapply styles
	ApplyLintDetailsTableStyles(&m.table)

	// restore cursor position
	if currentCursor < len(rows) {
		m.table.SetCursor(currentCursor)
	} else if len(rows) > 0 {
		m.table.SetCursor(len(rows) - 1)
	}

	// scroll to maintain visible cursor position
	if viewportStart > 0 && currentCursor >= viewportStart+cursorOffsetInViewport {
		for i := 0; i < viewportStart; i++ {
			m.table.MoveDown(1)
		}
	}
}

// UpdateFilterState updates filter state
func (m *ViolationResultTableModel) UpdateFilterState(filter FilterState) {
	m.uiState.FilterState = filter
}

// UpdateCategoryFilter updates category filter
func (m *ViolationResultTableModel) UpdateCategoryFilter(category string) {
	m.uiState.CategoryFilter = category
}

// UpdateRuleFilter updates rule filter
func (m *ViolationResultTableModel) UpdateRuleFilter(rule string) {
	m.uiState.RuleFilter = rule
}

// setupFileWatcher initializes file watching if enabled
func (m *ViolationResultTableModel) setupFileWatcher() tea.Cmd {
	if m.watchConfig == nil || !m.watchConfig.Enabled {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		m.watchError = fmt.Sprintf("Failed to create file watcher: %v", err)
		m.watchState = WatchStateError
		return nil
	}

	m.watcher = watcher

	absPath, err := filepath.Abs(m.fileName)
	if err != nil {
		m.watchError = fmt.Sprintf("Failed to get absolute path for %s: %v", m.fileName, err)
		m.watchState = WatchStateError
		return nil
	}

	err = m.watcher.Add(absPath)
	if err != nil {
		m.watchError = fmt.Sprintf("Failed to watch file %s: %v", absPath, err)
		m.watchState = WatchStateError
		return nil
	}

	m.watchedFiles = []string{absPath}

	go m.watchFileChanges()

	return m.listenForChannelMessages()
}

// watchFileChanges runs in a goroutine to monitor file system events
func (m *ViolationResultTableModel) watchFileChanges() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
				select {
				case m.watchMsgChan <- fileChangeMsg{fileName: event.Name}:
				default:
				}
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			select {
			case m.watchMsgChan <- relintErrorMsg{err: fmt.Errorf("file watcher error: %w", err)}:
			default:
			}
		}
	}
}

// listenForChannelMessages returns a command that listens for messages from the watcher channel
func (m *ViolationResultTableModel) listenForChannelMessages() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		select {
		case msg := <-m.watchMsgChan:
			return msg
		case <-time.After(100 * time.Millisecond):
			return continueWatchingMsg{}
		}
	})
}

// handleFileChange processes file change events with debouncing
func (m *ViolationResultTableModel) handleFileChange(fileName string) tea.Cmd {
	m.lastChangeTime = time.Now()

	if m.debounceTimer != nil {
		m.debounceTimer.Stop()
	}

	// new debounce timer
	m.debounceTimer = time.NewTimer(WatchDebounceDelay)

	return tea.Cmd(func() tea.Msg {
		<-m.debounceTimer.C

		if time.Since(m.lastChangeTime) >= WatchDebounceDelay {
			return m.performRelint()
		}

		return nil
	})
}

// performRelint re-lints the specification with current configuration
func (m *ViolationResultTableModel) performRelint() tea.Msg {
	m.watchState = WatchStateProcessing

	currentRow := m.table.Cursor()

	specBytes, err := os.ReadFile(m.fileName)
	if err != nil {
		return relintErrorMsg{err: fmt.Errorf("failed to read spec file: %w", err)}
	}

	// Restored working linting logic
	var bufferedLogger *slog.Logger
	bufferedLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(bufferedLogger)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	// Use pre-loaded custom functions from dashboard command
	customFuncs := m.watchConfig.CustomFunctions

	// hard mode
	if m.watchConfig.HardModeFlag {
		selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
		owaspRules := rulesets.GetAllOWASPRules()
		for k, v := range owaspRules {
			selectedRS.Rules[k] = v
		}
	}

	// custom ruleset if specified
	if m.watchConfig.RulesetFlag != "" {
		if strings.HasPrefix(m.watchConfig.RulesetFlag, "http") {
			if !m.watchConfig.RemoteFlag {
				return relintErrorMsg{err: fmt.Errorf("remote ruleset specified but remote flag is disabled")}
			}
			downloadedRS, rsErr := rulesets.DownloadRemoteRuleSet(context.Background(), m.watchConfig.RulesetFlag, nil)
			if rsErr != nil {
				return relintErrorMsg{err: fmt.Errorf("unable to load remote ruleset '%s': %w", m.watchConfig.RulesetFlag, rsErr)}
			}
			selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(downloadedRS)
		} else {
			rsBytes, rsErr := os.ReadFile(m.watchConfig.RulesetFlag)
			if rsErr != nil {
				return relintErrorMsg{err: fmt.Errorf("unable to read ruleset file '%s': %w", m.watchConfig.RulesetFlag, rsErr)}
			}
			userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
			if userErr != nil {
				return relintErrorMsg{err: fmt.Errorf("unable to parse ruleset file '%s': %w", m.watchConfig.RulesetFlag, userErr)}
			}
			selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)
		}

		// Merge OWASP rules if hard mode is enabled
		if m.watchConfig.HardModeFlag {
			owaspRules := rulesets.GetAllOWASPRules()
			if selectedRS.Rules == nil {
				selectedRS.Rules = make(map[string]*model.Rule)
			}
			for k, v := range owaspRules {
				if selectedRS.Rules[k] == nil {
					selectedRS.Rules[k] = v
				}
			}
		}
	}

	// ignore file if specified
	var ignoredItems model.IgnoredItems
	if m.watchConfig.IgnoreFile != "" {
		raw, ferr := os.ReadFile(m.watchConfig.IgnoreFile)
		if ferr == nil {
			_ = yaml.Unmarshal(raw, &ignoredItems)
		}
	}

	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:           selectedRS,
		Spec:              specBytes,
		SpecFileName:      m.fileName,
		CustomFunctions:   customFuncs,
		Base:              m.watchConfig.BaseFlag,
		AllowLookup:       m.watchConfig.RemoteFlag,
		SkipDocumentCheck: m.watchConfig.SkipCheckFlag,
		Logger:            bufferedLogger,
		Timeout:           time.Duration(m.watchConfig.TimeoutFlag) * time.Second,
		HTTPClientConfig: utils.HTTPClientConfig{
			CertFile: m.watchConfig.CertFile,
			KeyFile:  m.watchConfig.KeyFile,
			CAFile:   m.watchConfig.CAFile,
			Insecure: m.watchConfig.Insecure,
		},
	})

	m.updateWatchedFilesFromRolodex(result.Index)

	if len(result.Errors) > 0 {
		return relintErrorMsg{err: fmt.Errorf("linting failed: %v", result.Errors[0])}
	}

	filteredResults := utils.FilterIgnoredResults(result.Results, ignoredItems)
	
	// Create result set and sort by line number
	tempResultSet := model.NewRuleResultSet(filteredResults)
	tempResultSet.SortResultsByLineNumber()
	sortedResults := tempResultSet.Results
	
	resultPointers := make([]*model.RuleFunctionResult, len(sortedResults))
	for i := range sortedResults {
		resultPointers[i] = sortedResults[i]
	}

	return relintCompleteMsg{
		results:     resultPointers,
		specContent: specBytes,
		selectedRow: currentRow,
	}
}

// preserveSelection tries to maintain selection at the same line/column or moves to next available
func (m *ViolationResultTableModel) preserveSelection(previousRow int) {
	if len(m.filteredResults) == 0 {
		return
	}

	// if we have previous results and current position is valid, try to find same line/column
	if previousRow < len(m.allResults) && len(m.allResults) > 0 {
		previousResult := m.allResults[previousRow]

		// try to find result with same line and column
		for i, result := range m.filteredResults {
			if result.StartNode != nil && previousResult.StartNode != nil &&
				result.StartNode.Line == previousResult.StartNode.Line &&
				result.StartNode.Column == previousResult.StartNode.Column {
				m.table.SetCursor(i)
				m.modalContent = result
				return
			}
		}
	}

	// if we can't find exact match, move to next available or first
	newCursor := 0
	if previousRow < len(m.filteredResults) {
		newCursor = previousRow
	} else if previousRow > 0 && len(m.filteredResults) > 0 {
		newCursor = len(m.filteredResults) - 1 // Last available
	}

	m.table.SetCursor(newCursor)
	if newCursor < len(m.filteredResults) {
		m.modalContent = m.filteredResults[newCursor]
	}
}

// filterResults applies current filters to allResults and updates filteredResults
func (m *ViolationResultTableModel) filterResults() {
	filtered := m.allResults

	// severity filter
	if m.uiState.FilterState != FilterAll {
		var severityFiltered []*model.RuleFunctionResult
		for _, result := range filtered {
			switch m.uiState.FilterState {
			case FilterErrors:
				if result.Rule.Severity == "error" {
					severityFiltered = append(severityFiltered, result)
				}
			case FilterWarnings:
				if result.Rule.Severity == "warn" {
					severityFiltered = append(severityFiltered, result)
				}
			case FilterInfo:
				if result.Rule.Severity == "info" {
					severityFiltered = append(severityFiltered, result)
				}
			}
		}
		filtered = severityFiltered
	}

	// category filter
	if m.uiState.CategoryFilter != "" {
		var categoryFiltered []*model.RuleFunctionResult
		for _, result := range filtered {
			if result.Rule.Formats != nil {
				for _, format := range result.Rule.Formats {
					if format == m.uiState.CategoryFilter {
						categoryFiltered = append(categoryFiltered, result)
						break
					}
				}
			}
		}
		filtered = categoryFiltered
	}

	// rule filter
	if m.uiState.RuleFilter != "" {
		var ruleFiltered []*model.RuleFunctionResult
		for _, result := range filtered {
			if result.Rule.Id == m.uiState.RuleFilter {
				ruleFiltered = append(ruleFiltered, result)
			}
		}
		filtered = ruleFiltered
	}

	m.filteredResults = filtered
}

// clearProcessingStateAfterDelay returns a command that clears the processing state after 700ms
func (m *ViolationResultTableModel) clearProcessingStateAfterDelay() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		time.Sleep(700 * time.Millisecond)
		return clearProcessingStateMsg{}
	})
}

// updateWatchedFilesFromRolodex adds all files from the document rolodex to the watcher
func (m *ViolationResultTableModel) updateWatchedFilesFromRolodex(specIndex *index.SpecIndex) {
	if m.watcher == nil || specIndex == nil {
		return
	}

	rolodex := specIndex.GetRolodex()
	if rolodex == nil {
		return
	}

	// track new files to avoid duplicates
	newFiles := make(map[string]bool)
	for _, existingFile := range m.watchedFiles {
		newFiles[existingFile] = true
	}

	allIndexes := rolodex.GetIndexes()
	for _, idx := range allIndexes {
		if idx == nil {
			continue
		}

		config := idx.GetConfig()
		if config != nil && config.SpecFilePath != "" {
			filePath := config.SpecFilePath

			absPath, err := filepath.Abs(filePath)
			if err != nil {
				continue
			}

			// skip if already watching
			if newFiles[absPath] {
				continue
			}

			// add to watcher
			err = m.watcher.Add(absPath)
			if err != nil {
				continue
			}

			m.watchedFiles = append(m.watchedFiles, absPath)
			newFiles[absPath] = true
		}
	}
}
