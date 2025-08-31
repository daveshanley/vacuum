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

// TableLintModel holds the state for the interactive table view
type TableLintModel struct {
	table    table.Model
	results  []*model.RuleFunctionResult
	rows     []table.Row
	fileName string
	quitting bool
	width    int
	height   int
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
		table.WithHeight(height-4), // Leave room for title, blank, and status
		table.WithWidth(width),
	)

	// Configure styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#4B5263")).
		BorderBottom(true).
		Foreground(lipgloss.Color("#62c4ff")).
		Bold(true)
	
	s.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#3a3a3a"))
	
	s.Cell = s.Cell.Padding(0, 1)
	t.SetStyles(s)

	// Create and run model
	m := TableLintModel{
		table:    t,
		results:  results,
		rows:     rows,
		fileName: fileName,
		width:    width,
		height:   height,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func buildTableData(results []*model.RuleFunctionResult, fileName string, width int) ([]table.Column, []table.Row) {
	// Calculate column widths
	availableWidth := width - 12
	locWidth := availableWidth * 25 / 100
	sevWidth := 10
	msgWidth := availableWidth * 35 / 100
	ruleWidth := availableWidth * 15 / 100
	catWidth := 12
	pathWidth := availableWidth - locWidth - sevWidth - msgWidth - ruleWidth - catWidth

	// Minimum widths
	if locWidth < 25 {
		locWidth = 25
	}
	if msgWidth < 40 {
		msgWidth = 40
	}
	if ruleWidth < 15 {
		ruleWidth = 15
	}
	if pathWidth < 20 {
		pathWidth = 20
	}

	columns := []table.Column{
		{Title: "Location", Width: locWidth},
		{Title: "Severity", Width: sevWidth},
		{Title: "Message", Width: msgWidth},
		{Title: "Rule", Width: ruleWidth},
		{Title: "Category", Width: catWidth},
		{Title: "Path", Width: pathWidth},
	}

	rows := []table.Row{}
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

		rows = append(rows, table.Row{
			location,
			severity,
			r.Message,
			ruleID,
			category,
			r.Path,
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

func (m TableLintModel) Init() tea.Cmd {
	return nil
}

func (m TableLintModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 4)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
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

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f83aff")).
		Bold(true)
	builder.WriteString(titleStyle.Render("📋 Linting Results (Interactive View)"))
	builder.WriteString("\n\n")

	// Apply colors to table output
	tableView := colorizeTableOutput(m.table.View(), m.table.Cursor(), m.rows)
	builder.WriteString(tableView)
	builder.WriteString("\n")
	
	// Status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5263"))

	status := statusStyle.Render(fmt.Sprintf(
		" %d results • Row %d/%d • ↑↓/jk: navigate • pgup/pgdn: page • g/G: top/bottom • q: quit",
		len(m.results),
		m.table.Cursor()+1,
		len(m.results)))

	builder.WriteString(status)

	return builder.String()
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