// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package tui

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/daveshanley/vacuum/color"
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

		customStyle := color.CreatePb33fDocsStyle(modalWidth - 4)
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
	columns, rows := BuildResultTableData(m.filteredResults, m.fileName, msg.Width, m.uiState.ShowPath)
	m.table.SetColumns(columns)
	m.table.SetRows(rows)
	m.table.SetWidth(msg.Width - 2) // border wrapper

	if m.uiState.ViewMode == ViewModeTableWithSplit {
		// when details / split view is open, the table gets remaining space after fixed split view
		tableHeight := m.height - SplitViewHeight - SplitViewMargin
		if tableHeight < MinTableHeight {
			tableHeight = MinTableHeight
		}
		m.table.SetHeight(tableHeight)
	} else {
		m.table.SetHeight(msg.Height - 4)
	}

	color.ApplyLintDetailsTableStyles(&m.table)

	return nil
}

// HandleCodeViewKeys handles keyboard input when code view is open
func (m *ViolationResultTableModel) HandleCodeViewKeys(key string) (bool, tea.Cmd) {
	if m.uiState.ActiveModal != ModalCode {
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
		m.CloseActiveModal()
		return true, nil
	}

	// stop processing other keys when code view is open
	return true, nil
}

// HandleDocsModalKeys handles keyboard input when modal is open
func (m *ViolationResultTableModel) HandleDocsModalKeys(key string) (bool, tea.Cmd) {
	if m.uiState.ActiveModal != ModalDocs {
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
		m.CloseActiveModal()
		// don't clear modalContent if the details split-view is still open
		if m.uiState.ViewMode != ViewModeTableWithSplit {
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
		newFilter := (m.uiState.FilterState + 1) % 4
		m.UpdateFilterState(FilterState(newFilter))
		m.ApplyFilter()
		return true, nil
	case "c":
		// category filters
		m.categoryIndex = (m.categoryIndex + 1) % (len(m.categories) + 1)
		if m.categoryIndex == -1 || m.categoryIndex == len(m.categories) {
			m.categoryIndex = -1
			m.UpdateCategoryFilter("")
		} else {
			m.UpdateCategoryFilter(m.categories[m.categoryIndex])
		}
		m.ApplyFilter()
		return true, nil
	case "r":
		// rule filters
		m.ruleIndex = (m.ruleIndex + 1) % (len(m.rules) + 1)
		if m.ruleIndex == -1 || m.ruleIndex == len(m.rules) {
			m.ruleIndex = -1
			m.UpdateRuleFilter("")
		} else {
			m.UpdateRuleFilter(m.rules[m.ruleIndex])
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
		m.ToggleSplitView()
		if m.uiState.ViewMode == ViewModeTableWithSplit {
			// set content to the currently selected result with safety checks
			cursor := m.table.Cursor()
			if cursor >= 0 && cursor < len(m.filteredResults) && m.filteredResults != nil {
				m.modalContent = m.filteredResults[cursor]
			} else {
				// cursor is invalid, reset split view
				m.uiState.ViewMode = ViewModeTable
				return true, nil
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
		// expanded code view modal with safety checks
		cursor := m.table.Cursor()
		if cursor >= 0 && cursor < len(m.filteredResults) && m.filteredResults != nil {
			if m.uiState.ViewMode != ViewModeTableWithSplit {
				m.modalContent = m.filteredResults[cursor]
			}
			m.OpenModal(ModalCode)
			// prepare code viewport if opening
			if m.uiState.ActiveModal == ModalCode {
				m.PrepareCodeViewport()
			}
		}
		return true, nil

	case "d":
		// open documentation modal with safety checks
		cursor := m.table.Cursor()
		if cursor >= 0 && cursor < len(m.filteredResults) && m.filteredResults != nil {
			// if split view is open, preserve its modalContent
			if m.uiState.ViewMode != ViewModeTableWithSplit {
				m.modalContent = m.filteredResults[cursor]
			}
			m.OpenModal(ModalDocs)
			// if opening modal, fetch documentation
			if m.uiState.ActiveModal == ModalDocs && m.modalContent != nil && m.modalContent.Rule != nil {
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

		customStyle := color.CreatePb33fDocsStyle(modalWidth - 4)
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

// HandleEscapeKey handles the escape key with context-aware behavior
func (m *ViolationResultTableModel) HandleEscapeKey() (tea.Model, tea.Cmd) {
	// empty state (no results), clear all filters
	if len(m.filteredResults) == 0 && (m.uiState.FilterState != FilterAll || m.uiState.CategoryFilter != "" || m.uiState.RuleFilter != "") {

		m.uiState.FilterState = FilterAll
		m.uiState.CategoryFilter = ""
		m.uiState.RuleFilter = ""
		m.ApplyFilter()

		// rebuild the table with all results
		_, rows := BuildResultTableData(m.filteredResults, m.fileName, m.width, m.uiState.ShowPath)
		m.rows = rows
		m.table.SetRows(rows)

		// reset cursor position
		if len(rows) > 0 {
			m.table.SetCursor(0)
		}
		return m, nil
	}

	// handle escape based on current state
	if m.uiState.ActiveModal != ModalNone {
		// close active modal
		m.CloseActiveModal()
	} else if m.uiState.ViewMode == ViewModeTableWithSplit {
		// close split view
		m.uiState.ViewMode = ViewModeTable
		m.modalContent = nil
		m.table.SetHeight(m.height - 4)
	} else {
		// quit the application
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// UpdateDetailsViewContent updates split view content when cursor changes
func (m *ViolationResultTableModel) UpdateDetailsViewContent() {
	if m.uiState.ViewMode == ViewModeTableWithSplit {
		cursor := m.table.Cursor()
		// nil check and bounds checking
		if m.filteredResults != nil && cursor >= 0 && cursor < len(m.filteredResults) {
			newContent := m.filteredResults[cursor]
			if m.modalContent != newContent {
				m.modalContent = newContent
			}
		} else {
			// invalid cursor position, clear the modal content
			m.modalContent = nil
		}
	}
}
