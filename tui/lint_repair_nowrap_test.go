package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestRepairSourcePanningPreservesSourceRows(t *testing.T) {
	value := strings.Repeat("Long description ", 12) + "ENDING"
	for _, isYAML := range []bool{true, false} {
		source := "  description: \"" + value + "\""
		if !isYAML {
			source = "  \"description\": \"" + value + "\""
		}
		rows, _ := repairSourceRows(source, source, 72, isYAML, 0)
		require.Len(t, rows, 1)
		require.Contains(t, rows[0], "›")
		require.NotContains(t, rows[0], "ENDING")
		offset := repairSourceMaxOffset(source, source, 72)
		rows, _ = repairSourceRows(source, source, 72, isYAML, offset)
		require.Len(t, rows, 1)
		require.Contains(t, rows[0], "‹")
		require.Equal(t, 2, strings.Count(ansi.Strip(rows[0]), "ENDING"))
		// ANSI-aware clipping must not damage wide characters or exceed pane bounds.
		unicode := strings.Repeat("日本語 ", 20)
		for i := 0; i < 20; i++ {
			window := repairSourceWindow(unicode, 20, i)
			require.LessOrEqual(t, ansi.StringWidth(window), 20)
			require.NotContains(t, window, "\n")
			require.NotContains(t, window, "�")
		}
	}
}

func TestRepairLongValuePreviewAndSavedDocumentAgree(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"api.yaml", "openapi: 3.1.0\ninfo:\n  title: Long value\n  version: '1'\npaths: {}\n"},
		{"api.json", `{"openapi":"3.1.0","info":{"title":"Long value","version":"1"},"paths":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, disk := repairSourceTestModel(t, []byte(tc.source), tc.name, []string{"info-description"})
			m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
			selectRepairRule(t, m, "info-description")
			value := strings.Repeat("PreserveTheSourceLine", 12) + "ENDING"
			option := m.currentRepairOption()
			m.repair.values[option.Index] = value
			m.repair.selected[option.Index] = true
			m.openRepairDetail()
			draft := m.repair.draft.After
			require.Contains(t, draft, value)
			require.NotContains(t, ansi.Strip(m.renderActiveModal()), "ENDING")
			beforeRows := strings.Count(m.repair.draftView.GetContent(), "\n")
			cursor := m.table.Cursor()
			for i := 0; i < 80; i++ {
				repairKey(m, tea.KeyRight, "")
			}
			require.Equal(t, cursor, m.table.Cursor())
			require.Equal(t, beforeRows, strings.Count(m.repair.draftView.GetContent(), "\n"))
			require.Contains(t, ansi.Strip(m.renderActiveModal()), "ENDING")
			require.Equal(t, draft, m.repair.draft.After)
			require.Equal(t, tc.source, string(*disk))
			for i := 0; i < 80; i++ {
				repairKey(m, tea.KeyLeft, "")
			}
			require.Zero(t, m.repair.draftXOffset)
			prepareRepairSave(t, m)
			require.False(t, m.repair.save.view.SoftWrap)
			m.renderActiveModal() // Assign the final viewport dimensions.
			repairKey(m, tea.KeyRight, "")
			require.Positive(t, m.repair.save.view.XOffset())
			// A fresh output can have shorter lines. It must start at column zero.
			repairKey(m, 'e', "e")
			repairKey(m, tea.KeyEnter, "")
			repairKey(m, tea.KeyDown, "")
			cmd := repairKey(m, tea.KeyEnter, "")
			require.NotNil(t, cmd)
			m.Update(cmd())
			require.Equal(t, repairSavePreview, m.repair.save.step, m.repair.save.err)
			require.Zero(t, m.repair.save.view.XOffset())
			repairKey(m, tea.KeyEnter, "")
			var saved struct {
				Info struct {
					Description string `yaml:"description" json:"description"`
				} `yaml:"info" json:"info"`
			}
			require.NoError(t, yaml.Unmarshal(*disk, &saved))
			require.Equal(t, value, saved.Info.Description)
			if strings.HasSuffix(tc.name, "json") {
				require.True(t, json.Valid(*disk))
			}
		})
	}
}

func TestRepairSourceLastWideGlyphRemainsReachable(t *testing.T) {
	source := strings.Repeat("日本語", 20) + "終"
	for width := 40; width < 100; width++ {
		rows, _ := repairSourceRows(source, source, width, true, repairSourceMaxOffset(source, source, width))
		require.Len(t, rows, 1)
		require.Equal(t, 2, strings.Count(ansi.Strip(rows[0]), "終"), "width %d", width)
		require.Equal(t, width-2, ansi.StringWidth(rows[0]))
	}
}

func TestRepairVerifiedDiffNeverWrapsWideEdgeGlyphs(t *testing.T) {
	line := "+" + strings.Repeat("日本語", 50) + "終"
	for width := 40; width < 100; width++ {
		view := viewport.New(viewport.WithWidth(width), viewport.WithHeight(3))
		view.SetContent("header\n" + line + "\nfooter")
		view.SetXOffset(9999)
		offset := view.XOffset()
		rendered := repairDiffViewport(view)
		rows := strings.Split(ansi.Strip(rendered), "\n")
		require.Len(t, rows, 3, "width %d", width)
		require.Contains(t, rows[1], "終", "width %d", width)
		require.Empty(t, strings.TrimSpace(rows[2]), "short line must stay on its source row")
		require.Equal(t, offset, view.XOffset(), "render must not change navigation state")
		for _, row := range rows {
			require.Equal(t, width, ansi.StringWidth(row))
		}
	}
}
