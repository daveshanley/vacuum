package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/motor"
)

type repairSaveStep int

const (
	repairSaveDestination repairSaveStep = iota
	repairSaveFilename
	repairSaveBundle
	repairSaveVerifying
	repairSavePreview
	repairSaveDone
	repairSaveError
)

// A second Enter from the bundle step must not accept an unseen preview.
const repairSaveAcceptDelay = 250 * time.Millisecond

type repairSaveState struct {
	acceptAfter time.Time
	retry       repairSaveStep
	step        repairSaveStep
	destination int
	bundle      int
	input       textinput.Model
	options     RepairSaveOptions
	view        viewport.Model
	err         string
	summary     string
}

func (s *repairSaveState) fail(message string, retry repairSaveStep) {
	s.err, s.retry, s.step = message, retry, repairSaveError
	s.view.SoftWrap = true
	s.view.SetContent(message)
	s.view.GotoTop()
}

func (m *ViolationResultTableModel) openRepairSave() {
	r := m.repair
	path := r.config.SourcePath
	if path == "" {
		path = m.fileName
	}
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	ext := filepath.Ext(path)
	input := textinput.New()
	value := strings.TrimSuffix(path, ext) + ".fixed" + ext
	width, _ := m.repairModalSize()
	width = min(width, max(72, min(100, ansi.StringWidth(value)+8)))
	input.SetWidth(max(1, width-8))
	input.SetValue(value)
	input.CharLimit = 4096
	styles := input.Styles()
	styles.Focused.Text = styles.Focused.Text.Foreground(color.RGBYellow)
	styles.Blurred.Text = styles.Blurred.Text.Foreground(color.RGBYellow)
	styles.Cursor.Color = color.RGBYellow
	input.SetStyles(styles)
	r.save = &repairSaveState{input: input, view: viewport.New()}
	r.save.view.SoftWrap = false
	m.OpenModal(ModalRepairSave)
	m.resizeRepairModal()
}

func (m *ViolationResultTableModel) closeRepairSave() {
	r := m.repair
	if r.save != nil && r.save.step != repairSaveDone {
		r.invalidate()
	}
	r.save = nil
	m.CloseActiveModal()
}

func (m *ViolationResultTableModel) handleRepairSave(msg tea.Msg) (bool, tea.Cmd) {
	r := m.repair
	s := r.save
	if s == nil {
		m.CloseActiveModal()
		return true, nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		switch msg.(type) {
		case tea.WindowSizeMsg, fileChangeMsg, relintCompleteMsg, relintErrorMsg, continueWatchingMsg, clearProcessingStateMsg, docsLoadedMsg, docsErrorMsg:
			return false, nil
		}
		var cmd tea.Cmd
		if s.step == repairSaveFilename {
			s.input, cmd = s.input.Update(msg)
		}
		if s.step == repairSavePreview || s.step == repairSaveError {
			s.view, cmd = s.view.Update(msg)
		}
		return true, cmd
	}
	if key.String() == "ctrl+c" {
		r.close()
		m.quitting = true
		return true, tea.Quit
	}
	if key.String() == "esc" {
		switch s.step {
		case repairSaveFilename:
			s.input.Blur()
			s.step = repairSaveDestination
			s.err = ""
		case repairSaveBundle:
			s.step = repairSaveDestination
			s.err = ""
		default:
			m.closeRepairSave()
		}
		return true, nil
	}
	switch s.step {
	case repairSaveDestination:
		switch key.String() {
		case "up", "k", "down", "j":
			s.destination = 1 - s.destination
		case "enter":
			if r.discovery != nil || r.pending != nil {
				s.err = "Verification is running. Please wait."
				return true, nil
			}
			if r.review == nil || len(r.selectedChoices()) == 0 {
				s.err = "Select at least one fix first. Press esc to return."
				return true, nil
			}
			s.err = ""
			if s.destination == 1 {
				s.step = repairSaveFilename
				return true, s.input.Focus()
			}
			s.step = repairSaveBundle
		}
	case repairSaveFilename:
		if key.String() == "enter" {
			if strings.TrimSpace(s.input.Value()) == "" {
				s.err = "Enter a filename."
				return true, nil
			}
			s.input.Blur()
			s.step = repairSaveBundle
			s.err = ""
			return true, nil
		}
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return true, cmd
	case repairSaveBundle:
		switch key.String() {
		case "up", "k":
			s.bundle = (s.bundle + 2) % 3
		case "down", "j":
			s.bundle = (s.bundle + 1) % 3
		case "enter":
			if r.pending != nil {
				s.err = "Verification is running. Please wait."
				return true, nil
			}
			options := RepairSaveOptions{BundleMode: []string{"keep", "inline", "composed"}[s.bundle]}
			if s.destination == 1 {
				options.OutputPath = strings.TrimSpace(s.input.Value())
			}
			choices := r.selectedChoices()
			if r.review == nil || len(choices) == 0 {
				s.err = "Select at least one fix first."
				return true, nil
			}
			r.invalidate()
			s.options, s.step, s.err = options, repairSaveVerifying, ""
			job := &repairPreviewJob{}
			r.pending = job
			review, previewOutput := r.review, r.config.PreviewOutput
			return true, func() tea.Msg {
				var p *motor.FixPreview
				var err error
				if previewOutput != nil {
					p, err = previewOutput(review, choices, options)
				} else {
					p, err = review.PreviewOutput(choices, motor.FixOutputOptions{BundleMode: options.BundleMode, OutputPath: options.OutputPath})
				}
				job.finish(p, err)
				return repairPreviewMsg{job}
			}
		}
	case repairSavePreview:
		if m.changeRepairView(key.String()) {
			return true, nil
		}
		switch strings.ToLower(key.String()) {
		case "e":
			r.invalidate()
			s.view.SetContent("")
			s.step = repairSaveDestination
			s.err = ""
		case "enter":
			if time.Now().Before(s.acceptAfter) {
				return true, nil
			}
			p := r.preview
			if p == nil {
				s.err = "Verify the output before saving."
				return true, nil
			}
			if p.Impact.Status != motor.RepairImpactNoBreaking && !r.confirmImpact {
				r.confirmImpact = true
				s.err = repairImpactLabel(p.Impact) + ". Press enter to accept this impact and save."
				return true, nil
			}
			var err error
			if r.config.Save != nil {
				err = r.config.Save(r.source, p.Candidate, s.options)
			} else if s.options.OutputPath == "" && r.config.Write != nil {
				err = r.config.Write(r.source, p.Candidate)
			} else {
				err = fmt.Errorf("file creation is not configured")
			}
			if err != nil {
				s.fail("Save failed: "+err.Error(), repairSavePreview)
				r.status = s.err
				return true, nil
			}
			s.summary = fmt.Sprintf("Saved %d fixes · %d findings remain · score %d → %d", p.Fixed, len(p.Results), p.Before.Score, p.After.Score)
			if s.options.OutputPath == "" {
				m.adoptSavedRepair()
			} else {
				r.status = s.summary + " · original unchanged"
				s.summary += "\nOriginal specification and findings unchanged."
				r.invalidate()
			}
			s.step, s.err = repairSaveDone, ""
			s.view.SetContent("")
		default:
			var cmd tea.Cmd
			s.view, cmd = s.view.Update(msg)
			return true, cmd
		}
	case repairSaveError:
		if key.String() == "enter" {
			s.step, s.err = s.retry, ""
			if s.step == repairSavePreview {
				s.acceptAfter = time.Now().Add(repairSaveAcceptDelay)
			}
			s.view.SoftWrap = s.step != repairSavePreview
			s.view.SetContent(m.repairReviewContent(s.view.Width()))
			s.view.GotoTop()
			return true, nil
		}
		var cmd tea.Cmd
		s.view, cmd = s.view.Update(msg)
		return true, cmd
	case repairSaveDone:
		if key.String() == "enter" {
			m.closeRepairSave()
		}
	}
	return true, nil
}

func (m *ViolationResultTableModel) repairSavePath() string {
	s, config := m.repair.save, m.repair.config
	source := config.SourcePath
	if source == "" {
		source = m.fileName
	}
	if s.options.OutputPath == "" {
		return source
	}
	if filepath.IsAbs(s.options.OutputPath) {
		return filepath.Clean(s.options.OutputPath)
	}
	return filepath.Join(filepath.Dir(source), s.options.OutputPath)
}

// repairSaveModalSize keeps choices and filenames compact. Only the diff and
// error views need the full scrollable area. Blink state never affects sizing.
func (m *ViolationResultTableModel) repairSaveModalSize(width, height int) (int, int) {
	s := m.repair.save
	if s.step == repairSavePreview || s.step == repairSaveError {
		return width, height
	}
	preferred := 72
	if s.step == repairSaveFilename {
		preferred = max(preferred, min(100, ansi.StringWidth(s.input.Value())+8))
	} else if s.step == repairSaveVerifying || s.step == repairSaveDone {
		preferred = max(preferred, min(100, ansi.StringWidth(m.repairSavePath())+4))
	}
	width = min(width, preferred)
	_, body, _ := m.repairSaveContent(width, height)
	content := lipgloss.NewStyle().Width(max(1, width-4)).MaxWidth(max(1, width-4)).Render(body)
	return width, min(height, lipgloss.Height(content)+6)
}

func (m *ViolationResultTableModel) repairSaveContent(width, height int) (string, string, string) {
	r := m.repair
	s := r.save
	inner := max(1, width-4)
	s.input.SetWidth(max(1, width-8))
	title := "Save fixes"
	footer := "↑↓: choose | enter: continue | esc: back"
	var body string
	choices := func(labels []string, selected int) string {
		lines := make([]string, len(labels))
		for i, label := range labels {
			lines[i] = "  " + label
			if i == selected {
				lines[i] = lipgloss.NewStyle().Foreground(color.RGBPink).Bold(true).Render("› " + label)
			}
		}
		return strings.Join(lines, "\n\n")
	}
	switch s.step {
	case repairSaveDestination:
		body = fmt.Sprintf("%d fixes selected. Where do you want to save them?\n\n", len(r.selectedChoices())) + choices([]string{"Update the current specification", "Create a new specification"}, s.destination)
	case repairSaveFilename:
		title = "Create a new specification"
		body = "Filename\n\n" + s.input.View() + "\n\nDefault: the source spec directory. Use .yaml or .json.\nExisting files will not be replaced."
		footer = "enter: continue | esc: back"
	case repairSaveBundle:
		title = "Bundle references"
		body = choices([]string{"Keep references as they are", "Inline external references", "Compose reusable components"}, s.bundle) + "\n\nThe final output will be checked before you save it."
	case repairSaveVerifying:
		title = "Checking final output"
		body = lipgloss.NewStyle().Foreground(color.RGBYellow).Render(m.repairSavePath()) + "\n\nChecking fixes, references, document structure, and contract impact.\nNo files have been written."
		footer = "esc: cancel"
	case repairSavePreview:
		title = "Review and save"
		p := r.preview
		if p != nil {
			body = lipgloss.NewStyle().Foreground(color.RGBYellow).Render(m.repairSavePath()) + "\n" + fmt.Sprintf("%d fixes · findings %d → %d · score %d → %d", p.Fixed, p.Before.Errors+p.Before.Warnings+p.Before.Info, p.After.Errors+p.After.Warnings+p.After.Info, p.Before.Score, p.After.Score) + "\n"
			prefix := lipgloss.NewStyle().Width(inner).Render(body)
			if s.view.Width() != inner {
				s.view.SetWidth(inner)
				s.view.SetContent(m.repairReviewContent(inner))
			}
			s.view.SetHeight(max(1, height-9-lipgloss.Height(prefix)))
			body = prefix + "\n" + repairDiffViewport(s.view)
		}
		footer = "enter: save | e: options | []: changes | t: file diff | ↑↓: scroll | ←→: pan | esc: cancel"
		if ansi.StringWidth(footer) > inner {
			footer = "enter: save | e: options | []: changes | t: diff | esc: cancel"
		}
		if r.confirmImpact {
			footer = "enter: accept impact and save | esc: cancel"
		}
	case repairSaveError:
		title = "Cannot save specification"
		s.view.SetWidth(inner)
		s.view.SetHeight(max(1, height-6))
		body = s.view.View()
		footer = "enter: back | ↑↓/pgup/pgdn: scroll | esc: cancel"
	case repairSaveDone:
		title = "Specification saved"
		body = lipgloss.NewStyle().Foreground(color.RGBYellow).Render(m.repairSavePath()) + "\n\n" + s.summary
		footer = "enter/esc: close"
	}
	if s.err != "" && s.step != repairSaveError {
		body += "\n\n" + lipgloss.NewStyle().Foreground(color.RGBYellow).Render(s.err)
	}
	return title, body, footer
}

func (m *ViolationResultTableModel) buildRepairSaveModal() string {
	if m.repair == nil || m.repair.save == nil {
		return ""
	}
	width, height := m.repairModalSize()
	inner := max(1, width-4)
	frame := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(color.RGBPink).Width(width).Height(height).MaxWidth(width).MaxHeight(height)
	title, body, footer := m.repairSaveContent(width, height)
	minimumHeight := lipgloss.Height(lipgloss.NewStyle().Width(inner).MaxWidth(inner).Render(body)) + 6
	if m.repair.save.step == repairSavePreview || m.repair.save.step == repairSaveError {
		minimumHeight = 14
	}
	if width < 50 || height < minimumHeight {
		return frame.Render("Enlarge terminal to save fixes")
	}
	content := lipgloss.NewStyle().Width(inner).MaxWidth(inner).Height(height - 6).MaxHeight(height - 6).Render(body)
	header := lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true).Render(title)
	controls := lipgloss.NewStyle().Foreground(color.RGBGrey).Render(ansi.Truncate(footer, inner, ""))
	// Draw the footer as part of the frame so no automatic padding can put
	// an empty row below the controls. Match the comparison modal's divider.
	pink := lipgloss.NewStyle().Foreground(color.RGBPink)
	row := func(text string) string {
		cell := lipgloss.NewStyle().Width(inner).MaxWidth(inner).Height(1).MaxHeight(1).Render(text)
		return pink.Render("│") + " " + cell + " " + pink.Render("│")
	}
	edge := strings.Repeat("─", width-2)
	lines := []string{pink.Render("┌" + edge + "┐"), row(header), row("")}
	for _, line := range strings.Split(content, "\n") {
		lines = append(lines, row(line))
	}
	lines = append(lines, pink.Render("├"+edge+"┤"), row(controls), pink.Render("└"+edge+"┘"))
	return strings.Join(lines, "\n")
}

// repairDiffViewport keeps native navigation but crops visible source lines at
// whole graphemes. Native cell slicing can otherwise reflow a wide edge glyph.
func repairDiffViewport(view viewport.Model) string {
	offset, width := view.XOffset(), view.Width()
	view.SetXOffset(0)
	crop := lipgloss.NewStyle().Transform(func(line string) string {
		return repairSourceSlice(line, offset, width)
	})
	view.StyleLineFunc = func(int) lipgloss.Style { return crop }
	return view.View()
}
