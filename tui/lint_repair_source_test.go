package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	"github.com/pb33f/testify/require"
)

func TestRepairSourceColorsOnlyChangedLines(t *testing.T) {
	InitSyntaxStyles()
	blue, yellow, key := color.RGBBlue, color.RGBYellow, SyntaxKeyStyle
	t.Cleanup(func() { color.RGBBlue, color.RGBYellow, SyntaxKeyStyle = blue, yellow, key })
	color.RGBBlue, color.RGBYellow = lipgloss.Color("#00aaff"), lipgloss.Color("#ffff00")
	SyntaxKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#aaccee"))
	oldStyle := lipgloss.NewStyle().Foreground(color.RGBBlue)
	newStyle := lipgloss.NewStyle().Foreground(color.RGBYellow)
	for _, tc := range []struct {
		name, before, after, context, removed, added string
		yaml                                         bool
	}{
		{"insert", "type: string\nformat: binary", "type: string\ndescription: Image data.\nformat: binary", "format: binary", "", "description: Image data.", true},
		{"delete", "type: string\ndescription: obsolete\nformat: binary", "type: string\nformat: binary", "format: binary", "description: obsolete", "", true},
		{"replace", "type: string\ndescription: old\nformat: binary", "type: string\ndescription: new\nformat: binary", "format: binary", "description: old", "description: new", true},
		{"json comma", "{\n  \"type\": \"string\"\n}", "{\n  \"type\": \"string\",\n  \"description\": \"Text\"\n}", "}", "  \"type\": \"string\"", "  \"type\": \"string\",", false},
		{"separated edits", "a: old\nkeep: unchanged\nz: old", "a: new\nkeep: unchanged\nz: new", "keep: unchanged", "a: old", "a: new", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, first := repairSourceRows(tc.before, tc.after, 100, tc.yaml, 0)
			view := strings.Join(rows, "\n")
			require.GreaterOrEqual(t, first, 0)
			// Context stays on the same row in both panes, using the source view's syntax styles.
			styledContext := ApplySyntaxHighlightingToLine(tc.context, tc.yaml)
			require.Equal(t, 2, strings.Count(view, styledContext))
			for _, row := range rows {
				if strings.Contains(ansi.Strip(row), tc.context) {
					require.Equal(t, 2, strings.Count(row, styledContext))
				}
				require.Equal(t, 98, lipgloss.Width(row))
			}
			if tc.removed != "" {
				require.Contains(t, view, oldStyle.Render(tc.removed))
			} else {
				require.NotContains(t, view, "\x1b[38;2;0;170;255m")
			}
			if tc.added != "" {
				require.Contains(t, view, newStyle.Render(tc.added))
			} else {
				require.NotContains(t, view, "\x1b[38;2;255;255;0m")
			}
			if tc.name == "insert" || tc.name == "delete" {
				lw, _ := repairColumns(100)
				changed := ansi.Strip(rows[first])
				if tc.name == "insert" {
					require.Empty(t, strings.TrimSpace(changed[:lw]))
				} else {
					require.Empty(t, strings.TrimSpace(changed[lw+len("│"):]))
				}
			}
		})
	}
}

func TestRepairSourceLongLinesStayOnOneRowWithoutColor(t *testing.T) {
	blue, yellow, pink, grey := color.RGBBlue, color.RGBYellow, color.RGBPink, color.RGBGrey
	t.Cleanup(func() { color.RGBBlue, color.RGBYellow, color.RGBPink, color.RGBGrey = blue, yellow, pink, grey })
	color.RGBBlue, color.RGBYellow, color.RGBPink, color.RGBGrey = lipgloss.NoColor{}, lipgloss.NoColor{}, lipgloss.NoColor{}, lipgloss.NoColor{}
	rows, first := repairSourceRows("existing", "existing\n"+strings.Repeat("long new value ", 20), 72, true, 0)
	require.Equal(t, 1, first)
	require.Len(t, rows, 2)
	require.Contains(t, rows[1], "›")
	for _, row := range rows {
		require.Equal(t, row, ansi.Strip(row))
		require.Equal(t, 70, lipgloss.Width(row))
	}
}

func TestRepairSplitCompactMessageAndSingleGrayFooter(t *testing.T) {
	gray := color.RGBGrey
	t.Cleanup(func() { color.RGBGrey = gray })
	color.RGBGrey = lipgloss.Color("#888888")
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	message := "The `description` is missing."
	m.currentRepairOption().Result.Message = message
	m.openRepairDetail()
	header := m.repairIssueHeader(140, 36)
	require.Equal(t, []string{color.ColorizeMessage(message)}, header)
	view := m.buildRepairModal()
	plain := ansi.Strip(view)
	for _, removed := range []string{"Review fix", "Issue:", "Why:", "Found:", "Target:", "Generated suggestion", "No files written"} {
		require.NotContains(t, plain, removed)
	}
	require.Contains(t, plain, "Currently")
	require.Contains(t, plain, "Autofixed")
	require.Contains(t, view, lipgloss.NewStyle().Foreground(color.RGBGrey).Render("↑↓: fixes | enter: accept | e: edit | v: verify fix | a: save | esc: back | pgup/pgdn: scroll | ←→: pan"))
	rows := strings.Split(plain, "\n")
	require.Contains(t, rows[len(rows)-3], "┴")
	require.Contains(t, rows[len(rows)-2], "enter: accept")
}
