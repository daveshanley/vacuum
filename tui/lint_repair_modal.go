package tui

import (
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/motor"
)

// Discovery owns its review until the dashboard accepts it. A reload or quit
// cancels ownership, including results which arrive after their source changed.
type repairDiscoveryJob struct {
	mu       sync.Mutex
	review   *motor.FixReview
	err      error
	canceled bool
}
type repairDiscoveryMsg struct{ job *repairDiscoveryJob }
type repairPulseMsg struct{ generation int }

func (j *repairDiscoveryJob) cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.canceled = true
	if j.review != nil {
		j.review.Close()
		j.review = nil
	}
}
func (j *repairDiscoveryJob) finish(review *motor.FixReview, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.canceled {
		if review != nil {
			review.Close()
		}
		return
	}
	j.review, j.err = review, err
}
func (j *repairDiscoveryJob) take() (*motor.FixReview, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	review := j.review
	j.review = nil
	return review, j.err
}
func (m *ViolationResultTableModel) discoverRepairs() tea.Cmd {
	r := m.repair
	if r == nil || r.config == nil || r.config.Open == nil || r.discovery != nil || r.review != nil {
		return nil
	}
	r.source = append([]byte(nil), m.specContent...)
	source := r.source
	open := r.config.Open
	job := &repairDiscoveryJob{}
	r.discovery = job
	r.status = "Checking available fixes"
	return func() tea.Msg { review, err := open(source); job.finish(review, err); return repairDiscoveryMsg{job} }
}
func repairPulse(generation int) tea.Cmd {
	return tea.Tick(650*time.Millisecond, func(time.Time) tea.Msg { return repairPulseMsg{generation} })
}
func (m *ViolationResultTableModel) repairModalSize() (int, int) {
	width := max(4, min(140, m.width-8))
	height := max(4, min(36, m.height-6))
	if m.uiState.ActiveModal == ModalRepairEdit {
		height = min(height, 12)
	}
	return min(max(1, m.width), width), min(max(1, m.height), height)
}
func (m *ViolationResultTableModel) resizeRepairModal() {
	if m.repair == nil {
		return
	}
	width, height := m.repairModalSize()
	m.repair.input.SetWidth(max(1, width-8))
	m.repair.draftView.SetWidth(max(1, width-2))
	m.repair.draftView.SetHeight(m.repairBodyHeight(width, height))
	m.setRepairDraftContent()
}
func (m *ViolationResultTableModel) openRepairDetail() {
	r := m.repair
	if r != nil && r.pending != nil {
		r.status = "Verification running · wait before reviewing another fix"
		return
	}
	o := m.currentRepairOption()
	if r == nil || r.review == nil || o == nil {
		return
	}
	choice := motor.FixChoice{Index: o.Index}
	if o.Editable {
		value := r.values[o.Index]
		choice.Value = &value
	}
	r.draft, r.draftError = nil, ""
	draft, err := r.review.Draft(choice)
	if err != nil {
		r.draftError = err.Error()
	} else {
		r.draft = draft
	}
	m.OpenModal(ModalRepair)
	width, height := m.repairModalSize()
	r.draftView = viewport.New(viewport.WithWidth(max(1, width-2)), viewport.WithHeight(m.repairBodyHeight(width, height)))
	m.setRepairDraftContent()
}

// repairColumns returns the widths inside the outer frame, including cell padding.
func repairColumns(width int) (int, int) {
	left := max(1, (width-3)/2)
	return left, max(1, width-3-left)
}

func repairColumnRow(left, right string, width int) string {
	lw, rw := repairColumns(width)
	cell := func(text string, width int, style lipgloss.Style) string {
		return style.Width(width).MaxWidth(width).Height(1).MaxHeight(1).Padding(0, 1).Render(text)
	}
	return cell(left, lw, lipgloss.NewStyle().Foreground(color.RGBBlue)) +
		lipgloss.NewStyle().Foreground(color.RGBPink).Render("│") +
		cell(right, rw, lipgloss.NewStyle().Foreground(color.RGBYellow))
}

// The header is bounded independently of document text and spans both columns.
func (m *ViolationResultTableModel) repairIssueHeader(width, height int) []string {
	r, o := m.repair, m.currentRepairOption()
	name, issue, why, location, pointer := "Fix", "", "", "", ""
	if o != nil && o.Result != nil {
		issue = o.Result.Message
		location = formatFileLocation(o.Result, m.fileName)
		pointer = o.Result.Path
		if o.Result.Rule != nil {
			name = o.Result.Rule.Id
			why = o.Result.Rule.Description
			if why == "" {
				why = o.Result.Rule.HowToFix
			}
		}
	}
	if r.draft != nil {
		pointer = r.draft.Pointer
	}
	state := "Not selected"
	if o != nil && r.selected[o.Index] {
		state = "Selected"
	}
	if o != nil && o.Generated && r.values[o.Index] == o.Value {
		state += " · Generated suggestion, review before use"
	} else {
		state += " · Proposal only; V verifies before A applies"
	}
	maxLines := 1
	if height >= 30 {
		maxLines = 2
	}
	var rows []string
	add := func(text string, limit int, style lipgloss.Style) {
		wrapped := strings.Split(style.Width(max(1, width-4)).MaxWidth(max(1, width-4)).Render(text), "\n")
		if len(wrapped) > limit {
			wrapped = wrapped[:limit]
			wrapped[limit-1] = ansi.Truncate(wrapped[limit-1], max(1, width-5), "") + "…"
		}
		rows = append(rows, wrapped...)
	}
	add("Review fix · "+name, 1, lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true))
	add("Issue: "+issue, maxLines, lipgloss.NewStyle())
	if why != "" {
		add("Why: "+why, maxLines, lipgloss.NewStyle())
	}
	add("Found: "+location, 1, lipgloss.NewStyle().Foreground(color.RGBGrey))
	add("Target: "+pointer, 1, lipgloss.NewStyle().Foreground(color.RGBGrey))
	add(state, 1, lipgloss.NewStyle().Foreground(color.RGBGrey))
	return rows
}

func (m *ViolationResultTableModel) repairBodyHeight(width, height int) int {
	// Outer edges, two column separators, labels, footer separator and two controls.
	return max(1, height-len(m.repairIssueHeader(width, height))-8)
}

func (m *ViolationResultTableModel) setRepairDraftContent() {
	r := m.repair
	if r == nil || r.draft == nil {
		return
	}
	width, height := m.repairModalSize()
	r.draftView.SetWidth(max(1, width-2))
	r.draftView.SetHeight(m.repairBodyHeight(width, height))
	afterLabel := "AFTER · fixed version"
	if r.draft.Generated {
		afterLabel = "AFTER · GENERATED SUGGESTION"
	}
	afterText := r.draft.After
	if r.draft.NeedsValue {
		afterLabel = "AFTER · VALUE REQUIRED"
		instruction := "Enter the required value."
		if o := m.currentRepairOption(); o != nil {
			instruction = o.Description + "."
		}
		afterText = "No value has been proposed.\n" + instruction + "\nPress E to edit.\nVerify the proposal before applying."
	}
	r.draftHeading = repairColumnRow("BEFORE · current source", afterLabel, width)
	lw, rw := repairColumns(width)
	leftStyle := lipgloss.NewStyle().Width(max(1, lw-2)).MaxWidth(max(1, lw-2))
	rightStyle := lipgloss.NewStyle().Width(max(1, rw-2)).MaxWidth(max(1, rw-2))
	left := strings.Split(leftStyle.Render(r.draft.Before), "\n")
	right := strings.Split(rightStyle.Render(afterText), "\n")
	rows := make([]string, max(len(left), len(right), r.draftView.Height()))
	for i := range rows {
		a, b := "", ""
		if i < len(left) {
			a = left[i]
		}
		if i < len(right) {
			b = right[i]
		}
		rows[i] = repairColumnRow(a, b, width)
	}
	beforeLines, afterLines := strings.Split(r.draft.Before, "\n"), strings.Split(afterText, "\n")
	changed := 0
	for changed < len(beforeLines) && changed < len(afterLines) && beforeLines[changed] == afterLines[changed] {
		changed++
	}
	changed = min(changed, len(afterLines)-1)
	target := 0
	if changed > 0 {
		target = lipgloss.Height(rightStyle.Render(strings.Join(afterLines[:changed], "\n")))
	}
	r.draftView.SetContent(strings.Join(rows, "\n"))
	r.draftView.SetYOffset(max(0, target-r.draftView.Height()/2))
}

func (m *ViolationResultTableModel) buildRepairSplit() string {
	r := m.repair
	width, height := m.repairModalSize()
	lw, rw := repairColumns(width)
	pink := lipgloss.NewStyle().Foreground(color.RGBPink)
	header := m.repairIssueHeader(width, height)
	bodyHeight := height - len(header) - 8
	if width < 40 || bodyHeight < 1 {
		return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(color.RGBPink).Width(max(1, width-2)).Height(max(1, height-2)).MaxWidth(width).MaxHeight(height).Render("Enlarge terminal to review fixes")
	}
	edge := func(left, middle, right string) string {
		return pink.Render(left + strings.Repeat("─", lw) + middle + strings.Repeat("─", rw) + right)
	}
	full := func(text string) string {
		content := lipgloss.NewStyle().Width(max(1, width-4)).MaxWidth(max(1, width-4)).Height(1).MaxHeight(1).Render(text)
		return pink.Render("│") + " " + content + " " + pink.Render("│")
	}
	lines := []string{pink.Render("┌" + strings.Repeat("─", max(1, width-2)) + "┐")}
	for _, row := range header {
		lines = append(lines, full(row))
	}
	lines = append(lines, edge("├", "┬", "┤"))
	heading := r.draftHeading
	body := strings.Split(r.draftView.View(), "\n")
	if r.draftError != "" {
		heading = repairColumnRow("BEFORE · current source", "AFTER · proposal unavailable", width)
		body = strings.Split(lipgloss.NewStyle().Width(max(1, rw-2)).Render("Cannot preview proposal\n"+r.draftError+repairEditHint(m.currentRepairOption())), "\n")
		for i := range body {
			body[i] = repairColumnRow("", body[i], width)
		}
	}
	lines = append(lines, pink.Render("│")+heading+pink.Render("│"), edge("├", "┼", "┤"))
	for i := 0; i < bodyHeight; i++ {
		row := repairColumnRow("", "", width)
		if i < len(body) && strings.TrimSpace(body[i]) != "" {
			row = body[i]
		}
		lines = append(lines, pink.Render("│")+row+pink.Render("│"))
	}
	accept := "Enter: accept fix"
	if o := m.currentRepairOption(); o != nil {
		if o.Editable && strings.TrimSpace(r.values[o.Index]) == "" {
			accept = "Enter: edit value"
		} else if r.selected[o.Index] {
			accept = "Enter: deselect fix"
		}
	}
	lines = append(lines, edge("├", "┴", "┤"), full(accept+"   E: edit   V: verify selection"), full("↑↓: scroll   Esc: back   No files written"), pink.Render("└"+strings.Repeat("─", max(1, width-2))+"┘"))
	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(strings.Join(lines, "\n"))
}
func (m *ViolationResultTableModel) finishRepairEdit() {
	if m.repair.pending != nil {
		m.CloseActiveModal()
		m.repair.status = "Verification running · wait before reviewing another fix"
		return
	}
	if m.repair.editReturn {
		m.openRepairDetail()
	} else {
		m.CloseActiveModal()
	}
}
func (m *ViolationResultTableModel) buildRepairModal() string {
	r := m.repair
	if r == nil {
		return ""
	}
	if !r.editing {
		return m.buildRepairSplit()
	}
	width, height := m.repairModalSize()
	inner := max(1, width-4)
	heading := lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true)
	subdued := lipgloss.NewStyle().Foreground(color.RGBGrey)
	generated := lipgloss.NewStyle().Foreground(color.RGBYellow)
	o := m.currentRepairOption()
	name := "Fix"
	if o != nil && o.Result.Rule != nil {
		name = o.Result.Rule.Id
	}
	var lines []string
	if r.editing {
		lines = append(lines, heading.Render("Edit proposed value · "+name))
		hint := "Your edit changes this proposal only. Preview before applying."
		if o != nil && o.Generated && r.values[o.Index] == o.Value {
			hint = generated.Render("Generated suggestion, review and edit before use.")
		}
		lines = append(lines, lipgloss.NewStyle().Width(inner).MaxHeight(2).Render(hint), "")
		lines = append(lines, lipgloss.NewStyle().Width(inner).Height(1).MaxHeight(1).Render(r.input.View()))
		lines = append(lines, "", subdued.Render("Enter: save value   Esc: cancel"))
	}
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(color.RGBPink).Padding(0, 1).Width(max(1, width-2)).Height(max(1, height-2)).MaxWidth(width).MaxHeight(height).Render(content)
}

func (m *ViolationResultTableModel) handleRepairDiscovery(msg tea.Msg) (bool, tea.Cmd) {
	r := m.repair
	switch msg := msg.(type) {
	case repairDiscoveryMsg:
		if r.discovery != msg.job {
			msg.job.cancel()
			return true, nil
		}
		r.discovery = nil
		review, err := msg.job.take()
		if err != nil {
			r.status = "Fix review: " + err.Error()
			return true, nil
		}
		if review == nil {
			return true, nil
		}
		r.selected = make(map[int]bool)
		r.values = make(map[int]string)
		m.adoptRepairReview(review)
		count := 0
		for _, o := range r.options {
			if o.Reason == "" || o.Editable {
				count++
			}
		}
		if count == 0 {
			r.status = "No supported fixes for these findings"
			return true, nil
		}
		r.status = "Fixes available · F to review"
		r.pulseID++
		r.pulse = 4
		return true, repairPulse(r.pulseID)
	case repairPulseMsg:
		if msg.generation != r.pulseID || r.pulse <= 0 {
			return true, nil
		}
		r.pulse--
		if r.pulse > 0 {
			return true, repairPulse(r.pulseID)
		}
		return true, nil
	}
	return false, nil
}

func repairEditHint(o *motor.FixOption) string {
	if o != nil && o.Editable {
		return "\n\nPress E to supply or correct the value."
	}
	return ""
}
