// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/charmbracelet/bubbles/v2/table"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
)

// HandleDocsMessages processes documentation-related messages
func (m *ViolationResultTableModel) HandleDocsMessages(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case docsLoadedMsg:
		// cache the content
		m.docsCache[msg.ruleID] = msg.content
		m.docsContent = msg.content
		m.docsState = DocsStateLoaded

		modalWidth := int(float64(m.width) - ModalWidthReduction)

		customStyle := CreatePb33fDocsStyle(modalWidth - 4)
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
				m.docsContent = msg.content
			}
		} else {
			m.docsContent = msg.content
		}

		// update viewport with rendered content
		m.docsViewport.SetContent(m.docsContent)
		m.docsViewport.GotoTop()
		return true, nil

	case docsErrorMsg:
		m.docsState = DocsStateError
		m.docsError = msg.err
		return true, nil
	}
	return false, nil
}

// HandleWindowResize handles terminal resize events
func (m *ViolationResultTableModel) HandleWindowResize(msg tea.WindowSizeMsg) tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height

	// Rebuild table with new dimensions
	columns, rows := BuildResultTableData(m.filteredResults, m.fileName, msg.Width, m.showPath)
	m.table.SetColumns(columns)
	m.table.SetRows(rows)
	m.table.SetWidth(msg.Width - 2) // border wrapper

	if m.showSplitView {
		// when details / split view is open, the table gets remaining space after fixed split view
		tableHeight := m.height - SplitViewHeight - SplitViewMargin
		if tableHeight < MinTableHeight {
			tableHeight = MinTableHeight
		}
		m.table.SetHeight(tableHeight)
	} else {
		m.table.SetHeight(msg.Height - 4)
	}

	ApplyLintDetailsTableStyles(&m.table)

	return nil
}

// HandleCodeViewKeys handles keyboard input when code view is open
func (m *ViolationResultTableModel) HandleCodeViewKeys(key string) (bool, tea.Cmd) {
	if !m.showCodeView {
		return false, nil
	}

	switch key {
	case "up", "k":
		m.codeViewport.LineUp(1)
		return true, nil
	case "down", "j":
		m.codeViewport.LineDown(1)
		return true, nil
	case "pgup", "pageup", "page up":
		m.codeViewport.ViewUp()
		return true, nil
	case "pgdn", "pagedown", "page down", "pgdown":
		m.codeViewport.ViewDown()
		return true, nil
	case "home", "g":
		m.codeViewport.GotoTop()
		return true, nil
	case "end", "G":
		m.codeViewport.GotoBottom()
		return true, nil
	case " ", "space":
		m.ReCenterCodeView()
		return true, nil
	case "esc", "q", "x":
		m.showCodeView = false
		return true, nil
	}

	// stop processing other keys when code view is open
	return true, nil
}

// HandleDocsModalKeys handles keyboard input when modal is open
func (m *ViolationResultTableModel) HandleDocsModalKeys(key string) (bool, tea.Cmd) {
	if !m.showModal {
		return false, nil
	}

	if m.docsState == DocsStateLoaded {
		switch key {
		case "up", "k":
			m.docsViewport.LineUp(1)
			return true, nil
		case "down", "j":
			m.docsViewport.LineDown(1)
			return true, nil
		case "pgup":
			m.docsViewport.ViewUp()
			return true, nil
		case "pgdn":
			m.docsViewport.ViewDown()
			return true, nil
		case "home", "g":
			m.docsViewport.GotoTop()
			return true, nil
		case "end", "G":
			m.docsViewport.GotoBottom()
			return true, nil
		}
	}

	switch key {
	case "esc", "q", "enter", "d":
		m.showModal = false
		// don't clear modalContent if the details split-view is still open
		if !m.showSplitView {
			m.modalContent = nil
		}
		// reset docs state for next open
		m.docsState = DocsStateLoading
		return true, nil
	}

	// Don't process other keys when modal is open
	return true, nil
}

// HandleFilterKeys handles filter-related keyboard shortcuts
func (m *ViolationResultTableModel) HandleFilterKeys(key string) (bool, tea.Cmd) {
	switch key {
	case "tab":
		// severity filter states
		m.filterState = (m.filterState + 1) % 4
		m.ApplyFilter()
		return true, nil
	case "c":
		// category filters
		m.categoryIndex = (m.categoryIndex + 1) % (len(m.categories) + 1)
		if m.categoryIndex == -1 || m.categoryIndex == len(m.categories) {
			m.categoryIndex = -1
			m.categoryFilter = ""
		} else {
			m.categoryFilter = m.categories[m.categoryIndex]
		}
		m.ApplyFilter()
		return true, nil
	case "r":
		// rule filters
		m.ruleIndex = (m.ruleIndex + 1) % (len(m.rules) + 1)
		if m.ruleIndex == -1 || m.ruleIndex == len(m.rules) {
			m.ruleIndex = -1
			m.ruleFilter = ""
		} else {
			m.ruleFilter = m.rules[m.ruleIndex]
		}
		m.ApplyFilter()
		return true, nil
	}
	return false, nil
}

// HandleToggleKeys handles view toggle keyboard shortcuts
func (m *ViolationResultTableModel) HandleToggleKeys(key string) (bool, tea.Cmd) {
	switch key {
	case "enter":
		// toggle split view
		m.showSplitView = !m.showSplitView
		if m.showSplitView {
			// set content to the currently selected result
			if m.table.Cursor() < len(m.filteredResults) {
				m.modalContent = m.filteredResults[m.table.Cursor()]
			}
			// resize the table to leave room for the fixed-height split view
			tableHeight := m.height - SplitViewHeight - SplitViewMargin
			if tableHeight < MinTableHeight {
				tableHeight = MinTableHeight
			}
			m.table.SetHeight(tableHeight)
		} else {
			m.modalContent = nil
			// restore full height
			m.table.SetHeight(m.height - 4)
		}
		return true, nil

	case "x":
		// expanded code view modal
		if m.table.Cursor() < len(m.filteredResults) {
			if !m.showSplitView {
				m.modalContent = m.filteredResults[m.table.Cursor()]
			}
			m.showCodeView = !m.showCodeView

			// prepare code viewport if opening
			if m.showCodeView {
				m.PrepareCodeViewport()
			}
		}
		return true, nil

	case "d":
		// open documentation modal
		if m.table.Cursor() < len(m.filteredResults) {
			// If split view is open, preserve its modalContent
			if !m.showSplitView {
				m.modalContent = m.filteredResults[m.table.Cursor()]
			}
			m.showModal = !m.showModal

			// If opening modal, fetch documentation
			if m.showModal && m.modalContent != nil && m.modalContent.Rule != nil {
				return true, m.FetchOrLoadDocumentation()
			}
		}
		return true, nil

	case "p":
		// toggle path column visibility
		m.TogglePathColumn()
		return true, nil
	}
	return false, nil
}

// FetchOrLoadDocumentation loads documentation from cache or fetches it
func (m *ViolationResultTableModel) FetchOrLoadDocumentation() tea.Cmd {
	if m.modalContent == nil || m.modalContent.Rule == nil {
		return nil
	}

	ruleID := m.modalContent.Rule.Id

	// check cache first
	if cached, exists := m.docsCache[ruleID]; exists {
		m.docsContent = cached
		m.docsState = DocsStateLoaded

		// re-render markdown based on the current terminal size
		modalWidth := int(float64(m.width) - ModalWidthReduction)

		customStyle := CreatePb33fDocsStyle(modalWidth - 4)
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
				// raw content fallback
				m.docsContent = cached
			}
		} else {
			// raw content if we have an error.
			m.docsContent = cached
		}

		m.docsViewport.SetContent(m.docsContent)
		m.docsViewport.SetWidth(modalWidth - ViewportPadding)
		m.docsViewport.SetHeight(m.height - 14)
		m.docsViewport.GotoTop()
		return nil
	}

	m.docsState = DocsStateLoading
	m.docsContent = ""
	m.docsError = ""

	modalWidth := int(float64(m.width) - ModalWidthReduction)
	m.docsViewport.SetWidth(modalWidth - ViewportPadding)
	m.docsViewport.SetHeight(m.height - 14)

	return tea.Batch(fetchDocsFromDoctorAPI(ruleID), m.docsSpinner.Tick)
}

// TogglePathColumn handles toggling the path column visibility with viewport preservation
func (m *ViolationResultTableModel) TogglePathColumn() {
	m.showPath = !m.showPath

	currentCursor := m.table.Cursor()

	viewportHeight := m.table.Height()

	viewportStart := 0
	if currentCursor > viewportHeight/2 {
		viewportStart = currentCursor - viewportHeight/2
	}
	cursorOffsetInViewport := currentCursor - viewportStart

	columns, rows := BuildResultTableData(m.filteredResults, m.fileName, m.width, m.showPath)
	m.rows = rows

	// clear the rows to avoid index issues
	m.table.SetRows([]table.Row{})
	m.table.SetColumns(columns)
	m.table.SetRows(rows)

	// reapply styles
	ApplyLintDetailsTableStyles(&m.table)

	// restore cursor position and viewport
	if currentCursor < len(rows) {

		m.table.GotoTop()
		targetCursor := currentCursor

		// if we were scrolled down, overshoot and come back to position cursor correctly
		if viewportStart > 0 {
			// move past the target
			overshoot := cursorOffsetInViewport
			for i := 0; i < targetCursor+overshoot && i < len(rows)-1; i++ {
				m.table.MoveDown(1)
			}
			// move back up to get cursor in right viewport position
			for i := 0; i < overshoot; i++ {
				m.table.MoveUp(1)
			}
		} else {
			// just move to cursor position
			for i := 0; i < targetCursor; i++ {
				m.table.MoveDown(1)
			}
		}
	} else if len(rows) > 0 {
		m.table.SetCursor(0)
	}
}

// HandleEscapeKey handles the escape key with context-aware behavior
func (m *ViolationResultTableModel) HandleEscapeKey() (tea.Model, tea.Cmd) {
	// empty state (no results), clear all filters
	if len(m.filteredResults) == 0 && (m.filterState != FilterAll || m.categoryFilter != "" || m.ruleFilter != "") {

		m.filterState = FilterAll
		m.categoryFilter = ""
		m.ruleFilter = ""
		m.ApplyFilter()

		// rebuild the table with all results
		_, rows := BuildResultTableData(m.filteredResults, m.fileName, m.width, m.showPath)
		m.rows = rows
		m.table.SetRows(rows)

		// reset cursor position
		if len(rows) > 0 {
			m.table.SetCursor(0)
		}
		return m, nil
	}

	// if split view is open, close it on escape.
	if m.showSplitView {
		m.showSplitView = false
		m.modalContent = nil
		m.table.SetHeight(m.height - 4)
	} else {
		// just close it all down.
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// UpdateDetailsViewContent updates split view content when cursor changes
func (m *ViolationResultTableModel) UpdateDetailsViewContent() {
	if m.showSplitView {
		if m.table.Cursor() < len(m.filteredResults) {
			newContent := m.filteredResults[m.table.Cursor()]
			if m.modalContent != newContent {
				m.modalContent = newContent
			}
		}
	}
}
