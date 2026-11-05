package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	"github.com/pb33f/testify/require"
)

func TestRepairSplitHeaderAndContinuousContainer(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	o := m.currentRepairOption()
	o.Result.Message = "The API description is missing."
	o.Result.Rule.Description = "A description tells readers what this API provides."
	m.repair.onlyFixes = true
	for _, size := range []tea.WindowSizeMsg{{Width: 150, Height: 42}, {Width: 80, Height: 24}, {Width: 180, Height: 100}} {
		m.Update(size)
		repairKey(m, tea.KeyEnter, "")
		view := ansi.Strip(m.buildRepairModal())
		width, height := m.repairModalSize()
		require.Equal(t, width, lipgloss.Width(view))
		require.Equal(t, height, lipgloss.Height(view))
		require.LessOrEqual(t, height, 36)
		require.Contains(t, view, "The API description is missing.")
		require.NotContains(t, view, "Why:")
		require.NotContains(t, view, "Found:")
		require.NotContains(t, view, "Target:")
		lines := strings.Split(view, "\n")
		start, end := -1, -1
		for i, line := range lines {
			require.Equal(t, width, lipgloss.Width(line), "every frame row must meet the outer border")
			if strings.Contains(line, "┼") {
				start = i
			}
			if strings.Contains(line, "┴") {
				end = i
			}
		}
		require.Greater(t, start, 0)
		require.Greater(t, end, start+1)
		lw, _ := repairColumns(width)
		for _, line := range lines[start+1 : end] {
			runes := []rune(line)
			require.Equal(t, width, len(runes))
			require.Equal(t, '│', runes[0])
			require.Equal(t, '│', runes[lw+1], "divider must continue through blank body rows")
			require.Equal(t, '│', runes[width-1])
		}
		require.Equal(t, "┌", string([]rune(lines[0])[0]))
		require.True(t, strings.HasPrefix(lines[len(lines)-1], "└"))
		m.CloseActiveModal()
	}
}

func TestRepairSplitUsesBlueYellowAndPinkAndHonorsNoColor(t *testing.T) {
	blue, yellow, pink := color.RGBBlue, color.RGBYellow, color.RGBPink
	t.Cleanup(func() { color.RGBBlue, color.RGBYellow, color.RGBPink = blue, yellow, pink })
	color.RGBBlue = lipgloss.Color("#00aaff")
	color.RGBYellow = lipgloss.Color("#ffff00")
	color.RGBPink = lipgloss.Color("#ff00ff")
	row := repairColumnRow("original", "fixed", 80)
	require.NotContains(t, row, "\x1b[38;2;0;170;255m")
	require.NotContains(t, row, "\x1b[38;2;255;255;0m")
	require.Contains(t, row, "\x1b[38;2;255;0;255m│")
	color.RGBBlue, color.RGBYellow, color.RGBPink = lipgloss.NoColor{}, lipgloss.NoColor{}, lipgloss.NoColor{}
	row = repairColumnRow("original", "fixed", 80)
	require.Equal(t, row, ansi.Strip(row))
}

func TestRepairSplitLongHeaderAndSmallTerminal(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	o := m.currentRepairOption()
	o.Result.Message = strings.Repeat("A long issue description. ", 100)
	o.Result.Rule.Description = strings.Repeat("A long explanation. ", 100)
	m.repair.onlyFixes = true
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	repairKey(m, tea.KeyEnter, "")
	view := ansi.Strip(m.buildRepairModal())
	require.Contains(t, view, "…")
	require.Contains(t, view, "Currently")
	require.Contains(t, view, "Autofixed")
	require.Contains(t, view, "esc: back")
	require.Equal(t, 18, lipgloss.Height(view))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	view = ansi.Strip(m.buildRepairModal())
	require.Contains(t, view, "Currently")
	require.True(t, strings.HasSuffix(view, "┘"))
	m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	view = ansi.Strip(m.buildRepairModal())
	require.Contains(t, view, "Enlarge terminal")
	require.LessOrEqual(t, lipgloss.Width(view), 30)
	require.LessOrEqual(t, lipgloss.Height(view), 10)
}
