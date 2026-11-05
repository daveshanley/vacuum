package tui

import (
	"bytes"
	"fmt"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func repairTestModel(t *testing.T) (*ViolationResultTableModel, *[]byte) {
	t.Helper()
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a"]}}}}`)
	return repairSourceTestModel(t, source, "review.json", []string{"info-description", "operation-description", "duplicated-entry-in-enum"})
}

func repairSourceTestModel(t *testing.T, source []byte, filename string, ruleIDs []string) (*ViolationResultTableModel, *[]byte) {
	t.Helper()
	builtins := rulesets.GetAllBuiltInRules()
	rules := map[string]*model.Rule{}
	for _, id := range ruleIDs {
		rules[id] = builtins[id]
	}
	open := func(data []byte) (*motor.FixReview, error) {
		return motor.NewFixReview(&motor.RuleSetExecution{SpecFileName: filename, Spec: data, RuleSet: &rulesets.RuleSet{Rules: rules}, SilenceLogs: true}, nil)
	}
	baseline, err := open(source)
	require.NoError(t, err)
	var results []*model.RuleFunctionResult
	for _, o := range baseline.Options() {
		results = append(results, o.Result)
	}
	columns, rows := BuildResultTableData(results, filename, 180, true)
	m := &ViolationResultTableModel{width: 180, height: 40, fileName: filename, specContent: source, allResults: results, filteredResults: results, rows: rows, table: table.New(table.WithColumns(columns), table.WithRows(rows), table.WithFocused(true)), uiState: UIState{ShowPath: true}, categoryIndex: -1, ruleIndex: -1}
	disk := append([]byte(nil), source...)
	m.repair = &dashboardRepair{config: &RepairConfig{Open: open, Write: func(original, candidate []byte) error {
		if !bytes.Equal(original, disk) {
			return fmt.Errorf("source changed")
		}
		disk = append([]byte(nil), candidate...)
		return nil
	}}}
	m.openRepair()
	t.Cleanup(func() { m.repair.close(); baseline.Close() })
	return m, &disk
}
func repairKey(m *ViolationResultTableModel, code rune, text string) tea.Cmd {
	_, cmd := m.Update(tea.KeyPressMsg{Code: code, Text: text})
	return cmd
}

// repairType replaces the proposed value through the input before typing.
func repairType(m *ViolationResultTableModel, text string) {
	m.repair.input.SetValue("")
	for _, r := range text {
		repairKey(m, r, string(r))
	}
}
func selectRepairRule(t *testing.T, m *ViolationResultTableModel, id string) {
	t.Helper()
	for i, r := range m.filteredResults {
		if r.Rule.Id == id {
			m.table.SetCursor(i)
			m.UpdateDetailsViewContent()
			return
		}
	}
	t.Fatalf("missing rule %s", id)
}

func TestDashboardFixReviewEditPreviewApply(t *testing.T) {
	m, disk := repairTestModel(t)
	original := append([]byte(nil), (*disk)...)
	// A skipped mechanical fix remains a normal finding after metadata repairs.
	selectRepairRule(t, m, "duplicated-entry-in-enum")
	repairKey(m, ' ', " ")
	for _, id := range []string{"info-description", "operation-description"} {
		selectRepairRule(t, m, id)
		repairKey(m, 'e', "e")
		require.True(t, m.repair.editing)
		repairType(m, "Edited "+id)
		repairKey(m, tea.KeyEnter, "")
	}
	require.Equal(t, original, *disk)
	repairKey(m, 'a', "a")
	require.Contains(t, m.repair.status, "Preview")
	previewCmd := repairKey(m, 'v', "v")
	require.NotNil(t, previewCmd)
	m.Update(previewCmd())
	require.NotNil(t, m.repair.preview)
	require.Equal(t, 2, m.repair.preview.Fixed)
	require.Equal(t, original, *disk)
	require.Contains(t, m.repairDiff(), "Edited info-description")
	require.Contains(t, m.BuildDetailsView(), "Edited")
	repairKey(m, 'a', "a")
	require.NotEqual(t, original, *disk)
	require.Contains(t, string(*disk), "Edited operation-description")
	require.Len(t, m.allResults, 1)
	require.Equal(t, "duplicated-entry-in-enum", m.allResults[0].Rule.Id)
	require.Contains(t, m.repair.status, "APPLIED: 2 fixes")
	written := append([]byte(nil), (*disk)...)
	repairKey(m, 'a', "a")
	require.Equal(t, written, *disk)
}

func TestDashboardFixEditInvalidatesPreviewAndCancel(t *testing.T) {
	m, disk := repairTestModel(t)
	repairKey(m, 'f', "f")
	original := append([]byte(nil), (*disk)...)
	cmd := repairKey(m, 'v', "v")
	m.Update(cmd())
	require.NotNil(t, m.repair.preview)
	selectRepairRule(t, m, "info-description")
	repairKey(m, 'e', "e")
	repairType(m, "A value")
	repairKey(m, tea.KeyEnter, "")
	require.Nil(t, m.repair.preview)
	repairKey(m, 'a', "a")
	require.Equal(t, original, *disk)
	repairKey(m, 'e', "e")
	repairType(m, "discarded")
	repairKey(m, tea.KeyEscape, "")
	require.Equal(t, "A value", m.repair.values[m.currentRepairOption().Index])
	repairKey(m, tea.KeyEscape, "")
	require.Nil(t, m.repair.review)
	require.Nil(t, m.repair.preview)
	require.Equal(t, original, *disk)
}

func TestDashboardFixStaleCompletionAndWriteFailure(t *testing.T) {
	m, disk := repairTestModel(t)
	original := append([]byte(nil), (*disk)...)
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	selectRepairRule(t, m, "duplicated-entry-in-enum")
	repairKey(m, ' ', " ")
	m.Update(cmd())
	require.Nil(t, m.repair.preview)
	repairKey(m, ' ', " ")
	cmd = repairKey(m, 'v', "v")
	m.Update(cmd())
	require.NotNil(t, m.repair.preview)
	*disk = []byte("external edit")
	repairKey(m, 'a', "a")
	require.Contains(t, m.repair.status, "source changed")
	require.Equal(t, []byte("external edit"), *disk)
	require.Equal(t, original, m.specContent)
	m.Update(fileChangeMsg{fileName: "review.json"})
	require.Nil(t, m.repair.preview)
	require.Nil(t, m.repair.review)
}

func TestDashboardFixOpenFailureKeepsFindings(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.close()
	m.repair.config.Open = func([]byte) (*motor.FixReview, error) { return nil, fmt.Errorf("source changed") }
	repairKey(m, 'f', "f")
	require.False(t, m.repair.onlyFixes)
	require.Len(t, m.filteredResults, 3)
	require.Contains(t, m.repair.status, "source changed")
}

func TestDashboardFixEditingAllowsWatchAndResize(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	repairKey(m, 'e', "e")
	require.True(t, m.repair.editing)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	require.Equal(t, 120, m.width)
	require.Equal(t, 35, m.height)
	m.repair.input.SetValue("")
	m.Update(tea.PasteMsg{Content: "Pasted description"})
	require.Equal(t, "Pasted description", m.repair.input.Value())
	m.Update(fileChangeMsg{fileName: "review.json"})
	require.False(t, m.repair.editing)
	require.Nil(t, m.repair.review)
	require.Contains(t, m.repair.status, "Source changed")
}

func TestDashboardFixCanceledPreviewDoesNotQueueWork(t *testing.T) {
	m, _ := repairTestModel(t)
	first := repairKey(m, 'v', "v")
	require.NotNil(t, first)
	selectRepairRule(t, m, "duplicated-entry-in-enum")
	repairKey(m, ' ', " ")
	require.NotNil(t, m.repair.pending)
	require.Nil(t, repairKey(m, 'v', "v"))
	m.Update(first())
	require.Nil(t, m.repair.pending)
	require.Nil(t, m.repair.preview)
	repairKey(m, ' ', " ")
	require.NotNil(t, repairKey(m, 'v', "v"))
}

func TestDashboardFixCleanStateDoesNotBlameFilters(t *testing.T) {
	m, _ := repairTestModel(t)
	m.allResults = nil
	m.filteredResults = nil
	m.repair.status = "APPLIED: 3 fixes · 0 remain · score 100"
	view := m.buildTableView()
	require.Contains(t, view, "No findings remain")
	require.NotContains(t, view, "filters are too strict")
	require.Contains(t, view, "APPLIED: 3 fixes")
}

func TestDashboardFixReloadPassesCurrentSettings(t *testing.T) {
	m, _ := repairTestModel(t)
	called := false
	rules := &rulesets.RuleSet{Rules: map[string]*model.Rule{}}
	ignores := model.IgnoredItems{"rule": []string{"$.info"}}
	m.repair.config.Reload = func(got *rulesets.RuleSet, items model.IgnoredItems) {
		require.Same(t, rules, got)
		require.Equal(t, ignores, items)
		called = true
	}
	m.Update(relintCompleteMsg{results: m.allResults, specContent: m.specContent, repairRules: rules, repairIgnores: ignores})
	require.True(t, called)
	require.Nil(t, m.repair.review)
}
