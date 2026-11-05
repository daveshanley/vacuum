package tui

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/functions/core"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/testify/require"
)

func TestDashboardRepairModalShowsGeneratedBeforeAfter(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	repairKey(m, 'F', "F")
	require.Contains(t, m.repairStatus(), "Review fixes")
	require.NotContains(t, m.repairStatus(), "F to review")
	selectRepairRule(t, m, "info-description")
	option := m.currentRepairOption()
	require.True(t, option.Generated)
	require.False(t, m.repair.selected[option.Index])
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, ModalRepair, m.uiState.ActiveModal)
	view := m.View().Content
	require.Contains(t, view, "Currently")
	require.Contains(t, view, "Autofixed")
	require.Contains(t, view, "Review API.")
	require.Equal(t, original, string(*disk))
	repairKey(m, 'E', "E")
	require.Equal(t, ModalRepairEdit, m.uiState.ActiveModal)
	require.Equal(t, option.Value, m.repair.input.Value())
	m.repair.input.SetValue("")
	repairType(m, "Authored API description")
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, ModalRepair, m.uiState.ActiveModal)
	require.Contains(t, m.repair.draft.After, "Authored API description")
	require.False(t, m.repair.draft.Generated)
	require.True(t, m.repair.selected[option.Index])
	require.Nil(t, m.repair.preview)
	preview := repairKey(m, 'V', "V")
	require.NotNil(t, preview)
	require.Equal(t, original, string(*disk))
	m.Update(preview())
	require.NotNil(t, m.repair.preview)
	repairKey(m, 'A', "A")
	require.Contains(t, string(*disk), "Authored API description")
}

func TestDashboardRepairEditBlinkKeepsBackgroundAndBounds(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	m.uiState.ViewMode = ViewModeTableWithSplit
	m.UpdateDetailsViewContent()
	background := m.table.View() + m.BuildDetailsView()
	focus := repairKey(m, 'e', "e")
	require.NotNil(t, focus)
	before := m.buildRepairModal()
	require.Equal(t, background, m.table.View()+m.BuildDetailsView())
	_, blink := m.Update(focus())
	require.NotNil(t, blink)
	m.Update(blink())
	after := m.buildRepairModal()
	require.Equal(t, lipgloss.Width(before), lipgloss.Width(after))
	require.Equal(t, lipgloss.Height(before), lipgloss.Height(after))
	require.Equal(t, background, m.table.View()+m.BuildDetailsView())
	require.Contains(t, m.View().Content, "Edit proposed value")
}

func TestDashboardRepairModalResizeAndCancel(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	original := m.repair.values[m.currentRepairOption().Index]
	repairKey(m, 'e', "e")
	m.Update(tea.PasteMsg{Content: strings.Repeat("長い description ", 1000)})
	for _, size := range []tea.WindowSizeMsg{{Width: 140, Height: 40}, {Width: 80, Height: 24}, {Width: 40, Height: 12}} {
		m.Update(size)
		modal := m.buildRepairModal()
		require.LessOrEqual(t, lipgloss.Width(modal), size.Width)
		require.LessOrEqual(t, lipgloss.Height(modal), size.Height)
		require.Equal(t, ModalRepairEdit, m.uiState.ActiveModal)
	}
	repairKey(m, tea.KeyEscape, "")
	require.Equal(t, ModalNone, m.uiState.ActiveModal)
	require.Equal(t, original, m.repair.values[m.currentRepairOption().Index])
}

func TestDashboardRepairDiscoveryRejectsStaleCompletionAndDoesNotReopen(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.close()
	m.repair.options = nil
	calls := 0
	open := m.repair.config.Open
	m.repair.config.Open = func(source []byte) (*motor.FixReview, error) { calls++; return open(source) }
	discovery := m.discoverRepairs()
	require.NotNil(t, discovery)
	repairKey(m, 'f', "f")
	repairKey(m, 'v', "v")
	require.Zero(t, calls)
	m.Update(fileChangeMsg{fileName: "review.json"})
	msg := discovery().(repairDiscoveryMsg)
	m.Update(msg)
	require.Equal(t, 1, calls)
	require.Nil(t, m.repair.review)
	require.Nil(t, msg.job.review)
	require.True(t, msg.job.canceled)
}

func TestDashboardRepairDiscoveryPulseStopsAndEnterKeepsViolationView(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.close()
	m.repair.options = nil
	discovery := m.discoverRepairs()
	_, pulse := m.Update(discovery())
	require.NotNil(t, pulse)
	require.Contains(t, m.repairStatus(), "FIX AVAILABLE")
	require.Contains(t, m.repairStatus(), "suggested")
	require.False(t, m.repair.onlyFixes)
	selectRepairRule(t, m, "info-description")
	repairKey(m, tea.KeyEnter, "")
	require.NotEqual(t, ModalRepair, m.uiState.ActiveModal)
	generation := m.repair.pulseID
	for i := 0; i < 4; i++ {
		_, pulse = m.Update(repairPulseMsg{generation})
	}
	require.Nil(t, pulse)
	require.Zero(t, m.repair.pulse)
	require.Contains(t, m.repairStatus(), "FIX AVAILABLE")
	m.Update(fileChangeMsg{fileName: "review.json"})
	_, pulse = m.Update(repairPulseMsg{generation})
	require.Nil(t, pulse)
}

func TestDashboardRepairEmptyValueShowsSourceAndOpensEditor(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.onlyFixes = true
	selectRepairRule(t, m, "info-description")
	option := m.currentRepairOption()
	m.repair.values[option.Index] = ""
	m.repair.selected[option.Index] = false
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, ModalRepair, m.uiState.ActiveModal)
	require.True(t, m.repair.draft.NeedsValue)
	require.Contains(t, m.buildRepairModal(), "Currently")
	require.Contains(t, m.buildRepairModal(), "Value cleared")
	require.Contains(t, m.buildRepairModal(), "Review")
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, ModalRepairEdit, m.uiState.ActiveModal)
	repairKey(m, tea.KeyEnter, "")
	require.False(t, m.repair.selected[option.Index])
	require.Equal(t, ModalRepair, m.uiState.ActiveModal)
}

func TestDashboardRepairAvailabilityDoesNotHideErrorsOrStaleSource(t *testing.T) {
	m, _ := repairTestModel(t)
	require.Contains(t, m.repairStatus(), "FIX AVAILABLE")
	m.repair.status = "Preview failed: new diagnostic"
	require.Contains(t, m.repairStatus(), "Preview failed")
	require.NotContains(t, m.repairStatus(), "FIX AVAILABLE")
	m.Update(fileChangeMsg{fileName: "review.json"})
	require.NotContains(t, m.repairStatus(), "FIX AVAILABLE")
	m.repair.config.Open = func([]byte) (*motor.FixReview, error) { return nil, fmt.Errorf("source cannot be repaired") }
	repairKey(m, 'f', "f")
	require.Contains(t, m.repairStatus(), "source cannot be repaired")
	require.NotContains(t, m.repairStatus(), "FIX AVAILABLE")
}

type blockingRepairRule struct {
	core.Truthy
	enabled atomic.Bool
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *blockingRepairRule) RunRule(_ []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	if r.enabled.Load() {
		r.once.Do(func() { close(r.entered) })
		<-r.release
	}
	return nil
}

func TestDashboardRepairPendingVerificationDoesNotBlockModalKeys(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.close()
	blocker := &blockingRepairRule{entered: make(chan struct{}), release: make(chan struct{})}
	var release sync.Once
	defer release.Do(func() { close(blocker.release) })
	builtins := rulesets.GetAllBuiltInRules()
	review, err := motor.NewFixReview(&motor.RuleSetExecution{
		Spec: m.specContent, SpecFileName: "review.json", SilenceLogs: true,
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{
			"info-description":         builtins["info-description"],
			"duplicated-entry-in-enum": builtins["duplicated-entry-in-enum"],
			"blocking":                 {Id: "blocking", Given: "$.info", Severity: "warn", Then: &model.RuleAction{Function: "blocking"}},
		}}, CustomFunctions: map[string]model.RuleFunction{"blocking": blocker},
	}, nil)
	require.NoError(t, err)
	m.adoptRepairReview(review)
	m.repair.onlyFixes = true
	selectRepairRule(t, m, "info-description")
	blocker.enabled.Store(true)
	preview := repairKey(m, 'v', "v")
	require.NotNil(t, preview)
	done := make(chan tea.Msg, 1)
	go func() { done <- preview() }()
	select {
	case <-blocker.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("verification did not enter blocking rule")
	}
	// The verification now owns FixReview's mutex. These actions must not call Draft.
	responsive := func(action func()) {
		finished := make(chan struct{})
		go func() { action(); close(finished) }()
		select {
		case <-finished:
		case <-time.After(time.Second):
			release.Do(func() { close(blocker.release) })
			<-finished
			t.Fatal("modal action blocked behind verification")
		}
	}
	responsive(func() { repairKey(m, tea.KeyEnter, "") })
	require.Equal(t, ModalNone, m.uiState.ActiveModal)
	responsive(func() { repairKey(m, 'e', "e") })
	require.False(t, m.repair.editing)
	require.Contains(t, m.repair.status, "Verification running")
	// Cancellation retains the job until completion, so the same guards still apply.
	m.repair.invalidate()
	responsive(func() { repairKey(m, tea.KeyEnter, "") })
	m.OpenModal(ModalRepairEdit)
	m.repair.editReturn = true
	responsive(func() { m.finishRepairEdit() })
	require.Equal(t, ModalNone, m.uiState.ActiveModal)
	release.Do(func() { close(blocker.release) })
	m.Update(<-done)
	require.Nil(t, m.repair.pending)
}

func TestDashboardRepairContactStartsWithEditableProposal(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.close()
	review, err := motor.NewFixReview(&motor.RuleSetExecution{
		Spec: m.specContent, SpecFileName: "review.json", SilenceLogs: true,
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-contact": rulesets.GetAllBuiltInRules()["info-contact"]}},
	}, nil)
	require.NoError(t, err)
	m.repair.selected = make(map[int]bool)
	m.repair.values = make(map[int]string)
	m.adoptRepairReview(review)
	options := review.Options()
	require.Len(t, options, 1)
	m.allResults = []*model.RuleFunctionResult{options[0].Result}
	m.filteredResults = m.allResults
	m.refreshRepairRows()
	m.table.SetCursor(0)
	m.repair.onlyFixes = true
	repairKey(m, tea.KeyEnter, "")
	require.False(t, m.repair.draft.NeedsValue)
	require.True(t, m.repair.draft.Generated)
	require.Contains(t, m.repair.draft.After, `"contact"`)
	require.NotContains(t, ansi.Strip(m.buildRepairModal()), "VALUE REQUIRED")
	repairKey(m, 'e', "e")
	require.Equal(t, ModalRepairEdit, m.uiState.ActiveModal)
	require.NotEmpty(t, m.repair.input.Value())
	require.Equal(t, options[0].Value, m.repair.input.Value())
	repairKey(m, tea.KeyEscape, "")
	require.Equal(t, ModalRepair, m.uiState.ActiveModal)
	require.Contains(t, m.repair.draft.After, `"contact"`)
}

func TestDashboardRepairOpensAtChangeNearOwnerEnd(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.close()
	var source strings.Builder
	source.WriteString("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    User:\n      type: object\n      properties:\n")
	for i := 0; i < 14; i++ {
		fmt.Fprintf(&source, "        field%d:\n          type: string\n", i)
	}
	review, err := motor.NewFixReview(&motor.RuleSetExecution{Spec: []byte(source.String()), SpecFileName: "review.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"component-description": rulesets.GetAllBuiltInRules()["component-description"]}}}, nil)
	require.NoError(t, err)
	m.specContent = []byte(source.String())
	m.repair.source = m.specContent
	m.repair.selected = make(map[int]bool)
	m.repair.values = make(map[int]string)
	m.adoptRepairReview(review)
	options := review.Options()
	require.Len(t, options, 1)
	m.allResults = []*model.RuleFunctionResult{options[0].Result}
	m.filteredResults = m.allResults
	m.refreshRepairRows()
	m.table.SetCursor(0)
	m.repair.onlyFixes = true
	m.Update(tea.WindowSizeMsg{Width: 150, Height: 42})
	repairKey(m, tea.KeyEnter, "")
	require.GreaterOrEqual(t, strings.Count(m.repair.draft.Before, "\n"), 30)
	require.Greater(t, m.repair.draftView.YOffset(), 0)
	require.Contains(t, ansi.Strip(m.repair.draftView.View()), "description:")
	require.Contains(t, ansi.Strip(m.repair.draftView.View()), "User schema.")
	require.Contains(t, ansi.Strip(m.buildRepairModal()), "Currently")
	require.Contains(t, ansi.Strip(m.buildRepairModal()), "Autofixed")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	require.Contains(t, ansi.Strip(m.repair.draftView.View()), "User schema.")
}

func TestDashboardRepairDiscoveryKeepsOrdinaryEscapeBehavior(t *testing.T) {
	for _, action := range []string{"split", "table", "empty-filter"} {
		t.Run(action, func(t *testing.T) {
			m, _ := repairTestModel(t)
			m.repair.close()
			m.repair.options = nil
			discovery := m.discoverRepairs()
			m.Update(discovery())
			require.NotNil(t, m.repair.review)
			require.False(t, m.repair.onlyFixes)
			switch action {
			case "split":
				repairKey(m, tea.KeyEnter, "")
				require.Equal(t, ViewModeTableWithSplit, m.uiState.ViewMode)
			case "empty-filter":
				m.uiState.RuleFilter = "no-such-rule"
				m.ApplyFilter()
				require.Empty(t, m.filteredResults)
			}
			repairKey(m, tea.KeyEscape, "")
			switch action {
			case "table":
				require.True(t, m.quitting)
			case "split":
				require.Equal(t, ViewModeTable, m.uiState.ViewMode)
				require.False(t, m.quitting)
			case "empty-filter":
				require.Empty(t, m.uiState.RuleFilter)
				require.NotEmpty(t, m.filteredResults)
				require.False(t, m.quitting)
			}
			if action != "table" {
				require.NotNil(t, m.repair.review)
				require.Contains(t, m.repairStatus(), "FIX AVAILABLE")
			}
		})
	}
}
