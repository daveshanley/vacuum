// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/motor/autofix"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/pb33f/doctor/terminal/diffview"
	change "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

func TestRepairChangeReviewNavigationAndExactSave(t *testing.T) {
	m, disk := repairTestModel(t)
	for _, id := range []string{"info-description", "operation-description"} {
		selectRepairRule(t, m, id)
		repairKey(m, ' ', " ")
	}
	prepareRepairSave(t, m)
	r := m.repair
	require.Len(t, r.changes.changes, 2)
	original := string(*disk)
	candidate := append([]byte(nil), r.preview.Candidate...)
	selected := r.selectedChoices()
	first := ansi.Strip(m.repairReviewContent(100))
	require.Contains(t, first, "change 1/2")
	require.NotContains(t, first, "--- original")
	require.Contains(t, first, "description")
	repairKey(m, ']', "]")
	require.Contains(t, ansi.Strip(r.save.view.GetContent()), "change 2/2")
	require.NotEqual(t, first, ansi.Strip(m.repairReviewContent(100)))
	repairKey(m, 't', "t")
	require.Equal(t, r.diff, r.save.view.GetContent())
	require.Contains(t, r.save.view.GetContent(), "--- original")
	require.Contains(t, r.save.view.GetContent(), "Review API.")
	require.Contains(t, r.save.view.GetContent(), "GET")
	repairKey(m, '[', "[")
	require.False(t, r.changes.full)
	require.Contains(t, ansi.Strip(r.save.view.GetContent()), "change 1/2")
	require.Equal(t, selected, r.selectedChoices())
	require.Equal(t, original, string(*disk))
	require.Equal(t, candidate, r.preview.Candidate)
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, candidate, *disk)
	require.Empty(t, r.changes.changes)
}

func TestRepairChangeSourceUsesCorrectSnapshot(t *testing.T) {
	oldLine, newLine := 2, 4
	m := &ViolationResultTableModel{fileName: "/spec/api.yaml", repair: &dashboardRepair{
		changes: repairChangeView{original: []string{"old", "description: old"}, candidate: []string{"inserted", "wrong line", "owner:", "  description: new"}},
	}}
	ch := &change.Change{ChangeType: change.Modified, Property: "description", Original: "old", New: "new", Context: &change.ChangeContext{OriginalLine: &oldLine, NewLine: &newLine}}
	src := m.repairChangeSource(ch)
	require.Equal(t, diffview.Candidate, src.Side)
	require.Equal(t, 4, src.Line)
	require.Equal(t, "  description: new", src.Lines[src.Line-1])
	ch.Context.NewLine = nil
	require.Empty(t, m.repairChangeSource(ch).Lines, "original coordinates must not highlight candidate bytes")
	ch.ChangeType = change.PropertyRemoved
	src = m.repairChangeSource(ch)
	require.Equal(t, diffview.Original, src.Side)
	require.Equal(t, "description: old", src.Lines[src.Line-1])
	ch.Context.DocumentLocation = "/spec/external.yaml"
	require.Empty(t, m.repairChangeSource(ch).Lines, "external lines must not highlight the root")
	ch.Context.DocumentLocation = "/spec/api.yaml"
	require.NotEmpty(t, m.repairChangeSource(ch).Lines)
}

func TestRepairChangeFallbackAndInvalidation(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	prepareRepairSave(t, m)
	m.repair.preview.Impact = autofix.RepairImpact{Status: autofix.RepairImpactUnknown}
	m.repair.initChangeView()
	require.Equal(t, m.repair.diff, m.repairReviewContent(80))
	require.True(t, m.changeRepairView("]"))
	require.Equal(t, m.repair.diff, m.repairReviewContent(80))
	repairKey(m, 'e', "e")
	require.Empty(t, m.repair.changes)
	require.Empty(t, m.repairReviewContent(80))
}

func TestRepairChangeKeysInExpandedView(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	m.Update(cmd())
	require.Equal(t, ModalCode, m.uiState.ActiveModal)
	require.False(t, m.codeViewport.SoftWrap)
	repairKey(m, 't', "t")
	require.True(t, m.repair.changes.full)
	require.Contains(t, m.codeViewport.GetContent(), "--- original")
	repairKey(m, tea.KeyRight, "")
	require.Positive(t, m.codeViewport.XOffset())
	repairKey(m, 't', "t")
	require.False(t, m.repair.changes.full)
	require.Zero(t, m.codeViewport.XOffset())
	require.Contains(t, ansi.Strip(m.BuildCodeView()), "[]: changes | t: file diff")
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	require.Equal(t, 56, m.codeViewport.Width())
	for _, line := range strings.Split(ansi.Strip(m.BuildCodeView()), "\n") {
		require.LessOrEqual(t, ansi.StringWidth(line), 100)
	}
}

func TestVerifyOpensChangeViewFromReviewSurfaces(t *testing.T) {
	for _, surface := range []ModalType{ModalNone, ModalRepair, ModalCode} {
		for _, key := range []rune{'v', 'V'} {
			t.Run(fmt.Sprintf("%d/%c", surface, key), func(t *testing.T) {
				m, disk := repairTestModel(t)
				original := append([]byte(nil), (*disk)...)
				selectRepairRule(t, m, "info-description")
				repairKey(m, ' ', " ")
				switch surface {
				case ModalRepair:
					m.openRepairDetail()
				case ModalCode:
					repairKey(m, 'x', "x")
				}
				cmd := repairKey(m, key, string(key))
				require.NotNil(t, cmd, "V must start verification from this review surface")
				m.Update(cmd())
				require.Equal(t, ModalCode, m.uiState.ActiveModal, "verification must open a visible change view")
				require.Contains(t, ansi.Strip(m.renderActiveModal()), "change 1/1")
				require.Contains(t, ansi.Strip(m.renderActiveModal()), "Review API.")
				require.Equal(t, original, *disk)
				repairKey(m, 'a', "a")
				require.Equal(t, ModalRepairSave, m.uiState.ActiveModal, "A must open the existing save flow from the preview")
				require.Equal(t, original, *disk)
			})
		}
	}
}

func TestVerifyPreviewDoesNotDependOnSelectedRow(t *testing.T) {
	m, disk := repairTestModel(t)
	original := append([]byte(nil), (*disk)...)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	// A filter can remove the displayed row while verification runs.
	m.filteredResults = nil
	m.modalContent = nil
	m.Update(cmd())
	require.Equal(t, ModalCode, m.uiState.ActiveModal)
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "Review API.")
	require.Equal(t, original, *disk)
}

func TestVerifyFromCodeShowsEmptySelectionAndFailure(t *testing.T) {
	m, disk := repairTestModel(t)
	original := append([]byte(nil), (*disk)...)
	selectRepairRule(t, m, "info-description")
	repairKey(m, 'x', "x")
	require.Nil(t, repairKey(m, 'v', "v"))
	require.Equal(t, ModalNone, m.uiState.ActiveModal)
	require.Contains(t, m.repairStatus(), "Select a fix")
	repairKey(m, ' ', " ")
	repairKey(m, 'x', "x")
	require.NotNil(t, repairKey(m, 'v', "v"))
	job := m.repair.pending
	job.finish(nil, fmt.Errorf("invalid output"))
	m.Update(repairPreviewMsg{job: job})
	require.Equal(t, ModalNone, m.uiState.ActiveModal)
	require.Contains(t, m.repairStatus(), "Preview failed: invalid output")
	require.Nil(t, m.repair.preview)
	require.Equal(t, original, *disk)
}

func TestVerifyCompletionPreservesOpenSaveChoices(t *testing.T) {
	m, disk := repairTestModel(t)
	original := append([]byte(nil), (*disk)...)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	repairKey(m, 'a', "a")
	require.Equal(t, ModalRepairSave, m.uiState.ActiveModal)
	m.Update(cmd())
	require.Equal(t, ModalRepairSave, m.uiState.ActiveModal)
	require.Equal(t, repairSaveDestination, m.repair.save.step)
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "Where do you want to save")
	require.Equal(t, original, *disk)
	verifyRepairSave(t, m)
	require.Equal(t, repairSavePreview, m.repair.save.step)
}

func TestSourceReloadClosesVerifiedCodeView(t *testing.T) {
	for _, reload := range []bool{false, true} {
		t.Run(fmt.Sprintf("relint=%t", reload), func(t *testing.T) {
			m, disk := repairTestModel(t)
			original := append([]byte(nil), (*disk)...)
			selectRepairRule(t, m, "info-description")
			repairKey(m, ' ', " ")
			cmd := repairKey(m, 'v', "v")
			m.Update(cmd())
			require.Equal(t, ModalCode, m.uiState.ActiveModal)
			require.NotEmpty(t, m.codeViewport.GetContent())
			if reload {
				m.Update(relintCompleteMsg{specContent: original, results: m.allResults})
			} else {
				m.Update(fileChangeMsg{fileName: "review.json"})
			}
			require.Equal(t, ModalNone, m.uiState.ActiveModal)
			require.Empty(t, m.codeViewport.GetContent())
			require.Nil(t, m.repair.preview)
			require.Equal(t, original, *disk)
		})
	}
}

func TestVerifyFollowsCurrentProposalWithoutChangingSelection(t *testing.T) {
	source := []byte("openapi: 3.1.0\ninfo:\n  title: Pets\n  version: '1'\npaths: {}\ncomponents:\n  schemas:\n    Cat:\n      type: string\n    Dog:\n      type: string\n")
	m, disk := repairSourceTestModel(t, source, "pets.yaml", []string{"component-description"})
	repairKey(m, 'f', "f")
	require.Len(t, m.repair.options, 2)
	for _, option := range m.repair.options {
		m.repair.selected[option.Index] = true
	}
	choices := m.repair.selectedChoices()
	m.table.SetCursor(0)
	repairKey(m, tea.KeyEnter, "")
	repairKey(m, tea.KeyDown, "")
	require.Equal(t, 1, m.table.Cursor())
	second := m.currentRepairOption().Index
	m.repair.values[second] = "Authored dog description"
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	m.Update(cmd())
	require.Equal(t, 1, m.repair.preview.Fixed)
	require.Contains(t, string(m.repair.preview.Candidate), "Authored dog description")
	require.NotContains(t, string(m.repair.preview.Candidate), "Cat schema.")
	require.Equal(t, 2, len(m.repair.selectedChoices()))
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "Authored dog description")
	repairKey(m, tea.KeyEscape, "")
	repairKey(m, tea.KeyEnter, "")
	repairKey(m, tea.KeyUp, "")
	require.Equal(t, 0, m.table.Cursor())
	cmd = repairKey(m, 'V', "V")
	require.NotNil(t, cmd)
	m.Update(cmd())
	require.Contains(t, string(m.repair.preview.Candidate), "Cat schema.")
	require.NotContains(t, string(m.repair.preview.Candidate), "Authored dog description")
	require.Equal(t, choices[0].Index, m.repair.selectedChoices()[0].Index)
	require.Equal(t, choices[1].Index, m.repair.selectedChoices()[1].Index)
	require.Equal(t, source, *disk)
	// Save always checks the entire selected batch, not only the displayed fix.
	prepareRepairSave(t, m)
	require.Equal(t, 2, m.repair.preview.Fixed)
	require.Contains(t, string(m.repair.preview.Candidate), "Cat schema.")
	require.Contains(t, string(m.repair.preview.Candidate), "Authored dog description")
	require.Equal(t, source, *disk)
}

func TestVerifyUnselectedProposalDoesNotSelectIt(t *testing.T) {
	m, disk := repairTestModel(t)
	original := append([]byte(nil), (*disk)...)
	selectRepairRule(t, m, "info-description")
	m.openRepairDetail()
	require.Empty(t, m.repair.selectedChoices())
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	m.Update(cmd())
	require.Contains(t, string(m.repair.preview.Candidate), "Review API.")
	require.Empty(t, m.repair.selectedChoices())
	require.Equal(t, original, *disk)
	repairKey(m, 'a', "a")
	repairKey(m, tea.KeyEnter, "")
	require.Contains(t, m.repair.save.err, "Select at least one fix")
	require.Equal(t, original, *disk)
}

func TestVerifyAgainKeepsVerifiedChangeAndScroll(t *testing.T) {
	m, _ := repairTestModel(t)
	for _, id := range []string{"info-description", "operation-description"} {
		selectRepairRule(t, m, id)
		repairKey(m, ' ', " ")
	}
	cmd := repairKey(m, 'v', "v")
	m.Update(cmd())
	repairKey(m, ']', "]")
	m.codeViewport.SetHeight(2)
	m.codeViewport.SetYOffset(1)
	before := m.codeViewport.GetContent()
	preview := m.repair.preview
	require.Nil(t, repairKey(m, 'v', "v"))
	require.Same(t, preview, m.repair.preview)
	require.Equal(t, 1, m.repair.changes.index)
	require.Equal(t, 1, m.codeViewport.YOffset())
	require.Equal(t, before, m.codeViewport.GetContent())
}
