// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/motor/autofix"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	change "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

// prepareRepairSave opens the default update flow without a prior V preview.
func prepareRepairSave(t *testing.T, m *ViolationResultTableModel) {
	t.Helper()
	repairKey(m, 'a', "a")
	require.Equal(t, ModalRepairSave, m.uiState.ActiveModal)
	verifyRepairSave(t, m)
}
func verifyRepairSave(t *testing.T, m *ViolationResultTableModel) {
	t.Helper()
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveBundle, m.repair.save.step)
	cmd := repairKey(m, tea.KeyEnter, "")
	require.NotNil(t, cmd)
	require.Equal(t, repairSaveVerifying, m.repair.save.step)
	m.Update(cmd())
	require.Equal(t, repairSavePreview, m.repair.save.step, m.repair.save.err)
	require.NotNil(t, m.repair.preview)
	m.repair.save.acceptAfter = time.Time{} // The user has read the preview.
}

func TestRepairSaveDirectSelectionFlow(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	require.Nil(t, m.repair.preview)
	prepareRepairSave(t, m)
	require.Equal(t, original, string(*disk))
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "Review and save")
	require.Contains(t, m.repair.save.view.View(), "Review API.")
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveDone, m.repair.save.step)
	require.Contains(t, string(*disk), "Review API.")
	require.Equal(t, *disk, m.specContent)
	require.Contains(t, m.repair.save.summary, "Saved 1 fixes")
	saved := string(*disk)
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, ModalNone, m.uiState.ActiveModal)
	require.Equal(t, saved, string(*disk))
}

func TestRepairSaveCopyKeepsSourceAndFindings(t *testing.T) {
	m, disk := repairTestModel(t)
	original, findings := string(*disk), m.allResults
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	var verified, saved RepairSaveOptions
	var copied []byte
	m.repair.config.SourcePath = filepath.Join(t.TempDir(), "input.json")
	m.repair.config.PreviewOutput = func(r *autofix.FixReview, c []autofix.FixChoice, options RepairSaveOptions) (*autofix.FixPreview, error) {
		verified = options
		return r.PreviewOutput(c, autofix.FixOutputOptions{Format: "yaml", BundleMode: options.BundleMode})
	}
	m.repair.config.Save = func(before, after []byte, options RepairSaveOptions) error {
		require.Equal(t, original, string(before))
		copied = append([]byte(nil), after...)
		saved = options
		return nil
	}
	repairKey(m, 'a', "a")
	require.True(t, filepath.IsAbs(m.repair.save.input.Value()))
	repairKey(m, tea.KeyDown, "")
	focus := repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveFilename, m.repair.save.step)
	require.NotNil(t, focus)
	m.repair.save.input.SetValue("copy.yaml")
	repairKey(m, tea.KeyEnter, "")
	repairKey(m, tea.KeyDown, "")
	cmd := repairKey(m, tea.KeyEnter, "")
	require.NotNil(t, cmd)
	m.Update(cmd())
	require.Equal(t, repairSavePreview, m.repair.save.step, m.repair.save.err)
	m.repair.save.acceptAfter = time.Time{}
	require.Equal(t, RepairSaveOptions{OutputPath: "copy.yaml", BundleMode: "inline"}, verified)
	expected := append([]byte(nil), m.repair.preview.Candidate...)
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, verified, saved)
	require.Equal(t, expected, copied)
	require.Equal(t, original, string(*disk))
	require.Equal(t, original, string(m.specContent))
	require.Equal(t, findings, m.allResults)
	require.NotNil(t, m.repair.review)
	require.Nil(t, m.repair.preview)
	require.Contains(t, m.repair.save.summary, "Original specification and findings unchanged")
	require.Equal(t, filepath.Join(filepath.Dir(m.repair.config.SourcePath), "copy.yaml"), m.repairSavePath())
	// A long path may wrap in the compact completion modal, but no characters vanish.
	plain := ansi.Strip(m.renderActiveModal())
	compact := strings.Join(strings.Fields(strings.ReplaceAll(plain, "│", "")), "")
	require.Contains(t, compact, m.repairSavePath())
}

func TestRepairSaveCancelAndLateCompletion(t *testing.T) {
	for _, phase := range []string{"destination", "filename", "bundle", "verifying", "preview"} {
		t.Run(phase, func(t *testing.T) {
			m, disk := repairTestModel(t)
			original := string(*disk)
			selectRepairRule(t, m, "info-description")
			repairKey(m, ' ', " ")
			repairKey(m, 'a', "a")
			var cmd tea.Cmd
			if phase == "filename" {
				repairKey(m, tea.KeyDown, "")
				repairKey(m, tea.KeyEnter, "")
			}
			if phase == "bundle" || phase == "verifying" || phase == "preview" {
				repairKey(m, tea.KeyEnter, "")
			}
			if phase == "verifying" || phase == "preview" {
				cmd = repairKey(m, tea.KeyEnter, "")
			}
			if phase == "preview" {
				m.Update(cmd())
				cmd = nil
			}
			repairKey(m, tea.KeyEscape, "")
			if m.uiState.ActiveModal == ModalRepairSave {
				repairKey(m, tea.KeyEscape, "")
			}
			require.Equal(t, ModalNone, m.uiState.ActiveModal)
			if cmd != nil {
				m.Update(cmd())
			}
			require.Nil(t, m.repair.preview)
			require.Nil(t, m.repair.pending)
			require.Nil(t, m.repair.save)
			require.Equal(t, original, string(*disk))
		})
	}
}

func TestRepairSaveFailuresStayVisibleAndDoNotWrite(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	repairKey(m, 'a', "a")
	repairKey(m, tea.KeyEnter, "")
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "Select at least one fix")
	repairKey(m, tea.KeyEscape, "")
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	m.repair.config.PreviewOutput = func(*autofix.FixReview, []autofix.FixChoice, RepairSaveOptions) (*autofix.FixPreview, error) {
		return nil, fmt.Errorf("reference is invalid")
	}
	repairKey(m, 'a', "a")
	repairKey(m, tea.KeyEnter, "")
	cmd := repairKey(m, tea.KeyEnter, "")
	m.Update(cmd())
	require.Equal(t, repairSaveError, m.repair.save.step)
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "reference is invalid")
	require.Equal(t, original, string(*disk))
	m.repair.config.PreviewOutput = nil
	repairKey(m, tea.KeyEnter, "")
	cmd = repairKey(m, tea.KeyEnter, "")
	m.Update(cmd())
	m.repair.save.acceptAfter = time.Time{}
	m.repair.config.Save = func([]byte, []byte, RepairSaveOptions) error { return fmt.Errorf("destination already exists") }
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveError, m.repair.save.step)
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "destination already exists")
	require.Equal(t, original, string(*disk))
}

func TestRepairSaveResizeBlinkAndWatch(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.config.SourcePath = "/" + strings.Repeat("long-source-directory/", 20) + "spec.yaml"
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	repairKey(m, 'a', "a")
	repairKey(m, tea.KeyDown, "")
	focus := repairKey(m, tea.KeyEnter, "")
	before := m.renderActiveModal()
	_, blink := m.Update(focus())
	require.NotNil(t, blink)
	m.Update(blink())
	after := m.renderActiveModal()
	require.Equal(t, lipgloss.Width(before), lipgloss.Width(after))
	require.Equal(t, lipgloss.Height(before), lipgloss.Height(after))
	for _, size := range []tea.WindowSizeMsg{{Width: 140, Height: 40}, {Width: 80, Height: 24}, {Width: 30, Height: 10}} {
		m.Update(size)
		view := m.renderActiveModal()
		require.LessOrEqual(t, lipgloss.Width(view), size.Width)
		require.LessOrEqual(t, lipgloss.Height(view), size.Height)
		if size.Width >= 80 {
			require.Equal(t, 1, strings.Count(ansi.Strip(view), "enter: continue | esc: back"))
		}
	}
	m.Update(fileChangeMsg{fileName: "review.json"})
	require.Nil(t, m.repair.save)
	require.Equal(t, ModalNone, m.uiState.ActiveModal)
}

func TestRepairSaveShortTerminalShowsFullScrollableError(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	message := "Preview failed: relative references remain.\n" + strings.Repeat("Detailed reference check\n", 20) + "Save in the source directory."
	m.repair.config.PreviewOutput = func(*autofix.FixReview, []autofix.FixChoice, RepairSaveOptions) (*autofix.FixPreview, error) {
		return nil, fmt.Errorf("%s", message)
	}
	repairKey(m, 'a', "a")
	repairKey(m, tea.KeyEnter, "")
	cmd := repairKey(m, tea.KeyEnter, "")
	m.Update(cmd())
	view := ansi.Strip(m.renderActiveModal())
	require.Contains(t, view, "relative references remain")
	require.Contains(t, view, "enter: back |")
	require.LessOrEqual(t, lipgloss.Height(view), 20)
	for i := 0; i < 6; i++ {
		repairKey(m, tea.KeyPgDown, "")
	}
	view = ansi.Strip(m.renderActiveModal())
	require.Contains(t, view, "Save in the source directory.")
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveBundle, m.repair.save.step)
	require.Equal(t, original, string(*disk))
}

func TestRepairSaveFilenameUsesSourceDirectoryAndCompactYellowInput(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.config.SourcePath = "/apis/api.yaml"
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	repairKey(m, 'a', "a")
	require.Equal(t, "/apis/api.fixed.yaml", m.repair.save.input.Value())
	repairKey(m, tea.KeyDown, "")
	repairKey(m, tea.KeyEnter, "")
	rendered := m.renderActiveModal()
	require.Contains(t, rendered, lipgloss.NewStyle().Foreground(color.RGBYellow).Render("/apis/api.fixed.yaml"))
	require.Contains(t, ansi.Strip(rendered), "source spec directory")
	require.Contains(t, ansi.Strip(rendered), "enter: continue | esc: back")
	width, height := m.repairModalSize()
	require.Equal(t, width, lipgloss.Width(rendered))
	require.Equal(t, height, lipgloss.Height(rendered))
	require.LessOrEqual(t, height, 14)
	require.LessOrEqual(t, width, 100)
	x, y := m.calculateModalPosition()
	require.Equal(t, (m.width-width)/2, x)
	require.Equal(t, (m.height-height)/2, y)
	// Relative source paths use the same source-directory default, not a temp directory.
	repairKey(m, tea.KeyEscape, "")
	repairKey(m, tea.KeyEscape, "")
	m.repair.config.SourcePath = filepath.Join("specs", "api.json")
	repairKey(m, 'a', "a")
	expected, err := filepath.Abs(filepath.Join("specs", "api.fixed.json"))
	require.NoError(t, err)
	require.Equal(t, expected, m.repair.save.input.Value())
}

func TestRepairSaveCompactChoicesNeverHideAnOption(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	repairKey(m, 'a', "a")
	repairKey(m, tea.KeyEnter, "")
	for _, height := range []int{16, 20, 40} {
		m.Update(tea.WindowSizeMsg{Width: 80, Height: height})
		rendered := ansi.Strip(m.renderActiveModal())
		require.LessOrEqual(t, lipgloss.Height(rendered), height)
		if height == 16 {
			require.Contains(t, rendered, "Enlarge terminal")
		} else {
			require.Contains(t, rendered, "Keep references as they are")
			require.Contains(t, rendered, "Inline external references")
			require.Contains(t, rendered, "Compose reusable components")
			require.Contains(t, rendered, "enter: continue | esc: back")
		}
	}
}

func TestRepairSaveDividerAndControlsTouchBottomBorder(t *testing.T) {
	m, _ := repairTestModel(t)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	prepareRepairSave(t, m)
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 30}, {Width: 180, Height: 40}} {
		m.Update(size)
		for _, step := range []repairSaveStep{repairSaveDestination, repairSaveFilename, repairSaveBundle, repairSaveVerifying, repairSavePreview, repairSaveError, repairSaveDone} {
			m.repair.save.step = step
			rendered := m.renderActiveModal()
			rows := strings.Split(ansi.Strip(rendered), "\n")
			width, height := m.repairModalSize()
			require.Len(t, rows, height, "step %d", step)
			require.Equal(t, "├"+strings.Repeat("─", width-2)+"┤", rows[len(rows)-3], "step %d", step)
			require.Contains(t, rows[len(rows)-2], "esc", "controls must be the final content row")
			require.Equal(t, "└"+strings.Repeat("─", width-2)+"┘", rows[len(rows)-1])
			for _, row := range rows {
				require.Equal(t, width, ansi.StringWidth(row))
			}
		}
	}
}

func TestRepairSaveRequiresFreshEnterAfterPreview(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	repairKey(m, 'a', "a")
	repairKey(m, tea.KeyEnter, "")
	verify := repairKey(m, tea.KeyEnter, "")
	repairKey(m, tea.KeyEnter, "") // Already queued while verification runs.
	m.Update(verify())
	require.Equal(t, repairSavePreview, m.repair.save.step)
	m.View()
	repairKey(m, tea.KeyEnter, "") // Double-tap after a fast verification.
	require.Equal(t, repairSavePreview, m.repair.save.step)
	require.Equal(t, original, string(*disk))
	m.repair.save.acceptAfter = time.Now().Add(-time.Millisecond)
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, IsRepeat: true})
	require.Equal(t, repairSavePreview, m.repair.save.step)
	require.Equal(t, original, string(*disk))

	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveDone, m.repair.save.step)
	require.NotEqual(t, original, string(*disk))
}

func TestRepairSaveSelectedOverlay(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	m.repair.config.OverlayAvailable = true
	m.repair.config.SourcePath = filepath.Join(t.TempDir(), "input.json")
	var saved []byte
	m.repair.config.Save = func(before, data []byte, options RepairSaveOptions) error {
		require.True(t, options.Overlay)
		require.Equal(t, original, string(before))
		saved = append([]byte(nil), data...)
		return nil
	}
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	repairKey(m, 'a', "a")
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "Save as an overlay")
	repairKey(m, tea.KeyUp, "")
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, filepath.Join(filepath.Dir(m.repair.config.SourcePath), "input.overlay.yaml"), m.repair.save.input.Value())
	cmd := repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveVerifying, m.repair.save.step)
	require.NotNil(t, cmd)
	m.Update(cmd())
	require.Equal(t, repairSavePreview, m.repair.save.step, m.repair.save.err)
	preview := m.repair.preview
	split := m.repair.save.split
	require.NotNil(t, split)
	require.Equal(t, string(preview.Overlay), split.panes[0].GetContent())
	require.Equal(t, string(preview.Candidate), split.panes[1].GetContent())
	for _, size := range []tea.WindowSizeMsg{{Width: 200, Height: 55}, {Width: 86, Height: 24}} {
		m.Update(size)
		modal := ansi.Strip(m.renderActiveModal())
		require.Equal(t, size.Width-4, lipgloss.Width(modal))
		require.Equal(t, size.Height-2, lipgloss.Height(modal))
		require.Contains(t, modal, "› Overlay file")
		require.Contains(t, modal, "Resulting specification")
		require.Contains(t, modal, "overlay: 1.1.0")
		require.Contains(t, modal, "actions:")
		require.Contains(t, modal, `"openapi"`)
		require.Contains(t, modal, "esc: cancel")
	}
	repairKey(m, tea.KeyTab, "")
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "› Resulting specification")
	require.Empty(t, saved)
	m.repair.save.acceptAfter = time.Time{}
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveDone, m.repair.save.step)
	require.Equal(t, preview.Overlay, saved)
	require.Contains(t, string(saved), "Review API.")
	require.Equal(t, original, string(*disk))
	require.Equal(t, original, string(m.specContent))
}

func TestOverlayReviewIndependentPanesAndChangedLines(t *testing.T) {
	m, _ := repairTestModel(t)
	const originalLine = "\t\t\t\t\t\t\"description\": \"after界\","
	// Both documents have enough rows to scroll beyond the initial viewport.
	p := &autofix.FixPreview{
		Overlay:   []byte("overlay: 1.1.0\n" + strings.Repeat("  # action\n", 45)),
		Candidate: []byte("{\n" + strings.Repeat("\n", 18) + originalLine + "\n" + strings.Repeat("\n", 18) + "\"paths\": {}\n}"),
	}
	for _, l := range []int{39, 20} {
		p.Impact.Changes = append(p.Impact.Changes, &change.Change{Context: &change.ChangeContext{NewLine: &l}})
	}
	split := m.newOverlayReview(p)
	m.repair.preview = p
	m.openRepairSave()
	m.repair.save.step, m.repair.save.split = repairSavePreview, split
	read := func() string { return ansi.Strip(split.render(49, 9)) }
	before := read()
	require.Contains(t, before, "› Overlay file")
	require.NotContains(t, before, "description", "tab indentation extends beyond the pane")
	right := split.panes[1].YOffset()
	repairKey(m, tea.KeyDown, "")
	require.Positive(t, split.panes[0].YOffset())
	require.Equal(t, right, split.panes[1].YOffset())
	repairKey(m, tea.KeyTab, "")
	for range 12 {
		repairKey(m, tea.KeyRight, "")
	}
	after := read()
	require.Contains(t, after, "after界")
	require.NotContains(t, after, "\t")
	require.Len(t, strings.Split(after, "\n"), 9)
	require.Contains(t, string(p.Candidate), originalLine)
	repairKey(m, ']', "]")
	require.Contains(t, read(), "\"paths\": {}")
	require.Zero(t, split.panes[1].XOffset())
	repairKey(m, '[', "[")
	require.Equal(t, right, split.panes[1].YOffset())
}
