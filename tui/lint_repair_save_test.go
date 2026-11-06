package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/motor"
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
	m.repair.config.PreviewOutput = func(r *motor.FixReview, c []motor.FixChoice, options RepairSaveOptions) (*motor.FixPreview, error) {
		verified = options
		return r.PreviewOutput(c, motor.FixOutputOptions{Format: "yaml", BundleMode: options.BundleMode})
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
	require.Contains(t, ansi.Strip(m.renderActiveModal()), filepath.Join(filepath.Dir(m.repair.config.SourcePath), "copy.yaml"))
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
	m.repair.config.PreviewOutput = func(*motor.FixReview, []motor.FixChoice, RepairSaveOptions) (*motor.FixPreview, error) {
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
	m.repair.config.Save = func([]byte, []byte, RepairSaveOptions) error { return fmt.Errorf("destination already exists") }
	repairKey(m, tea.KeyEnter, "")
	require.Equal(t, repairSaveError, m.repair.save.step)
	require.Contains(t, ansi.Strip(m.renderActiveModal()), "destination already exists")
	require.Equal(t, original, string(*disk))
}

func TestRepairSaveResizeBlinkAndWatch(t *testing.T) {
	m, _ := repairTestModel(t)
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
	m.repair.config.PreviewOutput = func(*motor.FixReview, []motor.FixChoice, RepairSaveOptions) (*motor.FixPreview, error) {
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
