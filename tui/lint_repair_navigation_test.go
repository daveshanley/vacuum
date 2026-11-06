package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/pb33f/testify/require"
)

func TestRepairComparisonNavigatesWithoutLosingEditsOrSelection(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	repairKey(m, 'f', "f")
	selectRepairRule(t, m, "info-description")
	start := m.table.Cursor()
	repairKey(m, tea.KeyEnter, "")
	first := m.currentRepairOption().Index
	repairKey(m, 'e', "e")
	repairType(m, "Authored description")
	repairKey(m, tea.KeyEnter, "")
	require.True(t, m.repair.selected[first])
	firstDraft := m.repair.draft.After
	repairKey(m, tea.KeyDown, "")
	require.Equal(t, start+1, m.table.Cursor())
	require.Equal(t, ModalRepair, m.uiState.ActiveModal)
	require.NotEqual(t, first, m.currentRepairOption().Index)
	require.NotEqual(t, firstDraft, m.repair.draft.After)
	require.True(t, m.repair.selected[first])
	repairKey(m, tea.KeyUp, "")
	require.Equal(t, first, m.currentRepairOption().Index)
	require.Equal(t, firstDraft, m.repair.draft.After)
	require.Equal(t, "Authored description", m.repair.values[first])
	require.True(t, m.repair.selected[first])
	for range len(m.filteredResults) + 1 {
		repairKey(m, tea.KeyDown, "")
	}
	require.Equal(t, len(m.filteredResults)-1, m.table.Cursor(), "last fix does not wrap")
	for range len(m.filteredResults) + 1 {
		repairKey(m, tea.KeyUp, "")
	}
	require.Equal(t, 0, m.table.Cursor(), "first fix does not wrap")
	require.Equal(t, original, string(*disk))
}

func TestRepairComparisonPageKeysScrollWithoutChangingFix(t *testing.T) {
	m, _ := repairTestModel(t)
	repairKey(m, 'f', "f")
	repairKey(m, tea.KeyEnter, "")
	cursor := m.table.Cursor()
	m.repair.draftView.SetContent(strings.Repeat("long source line\n", 100))
	m.repair.draftView.SetYOffset(0)
	repairKey(m, tea.KeyPgDown, "")
	require.Positive(t, m.repair.draftView.YOffset())
	require.Equal(t, cursor, m.table.Cursor())
	repairKey(m, tea.KeyPgUp, "")
	require.Zero(t, m.repair.draftView.YOffset())
	require.Equal(t, cursor, m.table.Cursor())
}
