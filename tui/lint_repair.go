package tui

import (
	"charm.land/bubbles/v2/viewport"
	"fmt"
	"github.com/daveshanley/vacuum/color"
	"strings"
	"sync"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
)

// RepairConfig connects dashboard review to the command's lint settings and
// source-checked writer. Write must reject a changed original source.
type RepairConfig struct {
	Open  func(source []byte) (*motor.FixReview, error)
	Write func(original, candidate []byte) error
	// Reload accepts the effective rules and ignores after a successful watcher lint.
	Reload func(*rulesets.RuleSet, model.IgnoredItems)
}

type dashboardRepair struct {
	diff         string
	byIdentity   map[string]int
	config       *RepairConfig
	review       *motor.FixReview
	options      []motor.FixOption
	selected     map[int]bool
	values       map[int]string
	preview      *motor.FixPreview
	pending      *repairPreviewJob
	source       []byte
	status       string
	onlyFixes    bool
	editing      bool
	editIndex    int
	input        textinput.Model
	discovery    *repairDiscoveryJob
	pulse        int
	pulseID      int
	draft        *motor.FixDraft
	draftError   string
	draftHeading string
	draftView    viewport.Model
	editReturn   bool
	automatic    int
	suggested    int
	needsInput   int
}

// A command owns its result until the UI takes it. Cancellation also releases a
// result that finished just before quit or a file-watch event.
type repairPreviewJob struct {
	mu       sync.Mutex
	result   *motor.FixPreview
	err      error
	canceled bool
}
type repairPreviewMsg struct{ job *repairPreviewJob }

func (j *repairPreviewJob) cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.canceled = true
	if j.result != nil {
		j.result.Close()
		j.result = nil
	}
}
func (j *repairPreviewJob) finish(result *motor.FixPreview, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.canceled {
		if result != nil {
			result.Close()
		}
		return
	}
	j.result, j.err = result, err
}
func (j *repairPreviewJob) take() (*motor.FixPreview, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	result := j.result
	j.result = nil
	return result, j.err
}

func (r *dashboardRepair) invalidate() {
	r.diff = ""
	if r.preview != nil {
		r.preview.Close()
		r.preview = nil
	}
	if r.pending != nil {
		r.pending.cancel()
		// Keep the job until its completion arrives; a new preview must not
		// queue behind canceled work that is still using the review baseline.
	}
}
func (r *dashboardRepair) close() {
	r.invalidate()
	r.pulseID++
	r.pulse = 0
	r.automatic, r.suggested, r.needsInput = 0, 0, 0
	r.draft = nil
	r.editing = false
	r.input.Blur()
	if r.discovery != nil {
		r.discovery.cancel()
		r.discovery = nil
	}
	if r.review != nil {
		review := r.review
		r.review = nil
		// Review.Close waits for a running Preview before releasing its baseline.
		go review.Close()
	}
}
func (m *ViolationResultTableModel) openRepair() {
	r := m.repair
	if r == nil || r.config == nil || r.config.Open == nil {
		return
	}
	r.close()
	r.options = nil
	r.selected = make(map[int]bool)
	r.values = make(map[int]string)
	r.editing = false
	r.source = append([]byte(nil), m.specContent...)
	review, err := r.config.Open(r.source)
	if err != nil {
		r.status = "Fix review: " + err.Error()
		return
	}
	m.adoptRepairReview(review)
}

func (m *ViolationResultTableModel) adoptRepairReview(review *motor.FixReview) {
	r := m.repair
	r.review = review
	r.options = review.Options()
	r.byIdentity = make(map[string]int, len(r.options))
	r.automatic, r.suggested, r.needsInput = 0, 0, 0
	for i, o := range r.options {
		r.byIdentity[fixIdentity(o.Result)] = i
		r.selected[o.Index] = o.Selected
		r.values[o.Index] = o.Value
		if o.Generated {
			r.suggested++
		} else if o.Editable {
			r.needsInput++
		} else if o.Reason == "" {
			r.automatic++
		}
	}
	r.status = "Fixes available · F to review"
	m.refreshRepairRows()
}
func fixIdentity(r *model.RuleFunctionResult) string {
	if r == nil {
		return ""
	}
	id := ""
	if r.Rule != nil {
		id = r.Rule.Id
	}
	line, column := 0, 0
	if r.StartNode != nil {
		line = r.StartNode.Line
		column = r.StartNode.Column
	}
	return fmt.Sprintf("%s\x00%s\x00%s\x00%d:%d\x00%s", id, r.Path, r.DiagnosticCode, line, column, r.Message)
}
func (m *ViolationResultTableModel) repairOption(result *model.RuleFunctionResult) *motor.FixOption {
	if m.repair == nil {
		return nil
	}
	key := fixIdentity(result)
	if i, ok := m.repair.byIdentity[key]; ok && i < len(m.repair.options) {
		return &m.repair.options[i]
	}
	return nil
}
func (m *ViolationResultTableModel) currentRepairOption() *motor.FixOption {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.filteredResults) {
		return nil
	}
	return m.repairOption(m.filteredResults[i])
}
func (m *ViolationResultTableModel) handleRepairMessage(msg tea.Msg) (bool, tea.Cmd) {
	r := m.repair
	if r == nil {
		return false, nil
	}
	if handled, cmd := m.handleRepairDiscovery(msg); handled {
		return true, cmd
	}
	if done, ok := msg.(repairPreviewMsg); ok {
		if r.pending != done.job {
			done.job.cancel()
			return true, nil
		}
		r.pending = nil
		preview, err := done.job.take()
		if err != nil {
			r.status = "Preview failed: " + err.Error()
			return true, nil
		}
		if preview == nil {
			return true, nil
		}
		r.preview = preview
		r.diff = utils.RepairDiff(r.source, preview.Candidate)
		r.status = fmt.Sprintf("PREVIEW: %d fixes · E %d→%d W %d→%d I %d→%d · score %d→%d · no writes", preview.Fixed, preview.Before.Errors, preview.After.Errors, preview.Before.Warnings, preview.After.Warnings, preview.Before.Info, preview.After.Info, preview.Before.Score, preview.After.Score)
		if m.uiState.ViewMode != ViewModeTableWithSplit {
			m.ToggleSplitView()
		}
		m.UpdateDetailsViewContent()
		m.table.SetHeight(max(MinTableHeight, m.height-SplitViewHeight-SplitViewMargin))
		return true, nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if r.editing {
			switch msg.(type) {
			case tea.WindowSizeMsg, fileChangeMsg, relintCompleteMsg, relintErrorMsg, continueWatchingMsg, clearProcessingStateMsg, docsLoadedMsg, docsErrorMsg:
				return false, nil
			}
			var cmd tea.Cmd
			r.input, cmd = r.input.Update(msg)
			return true, cmd
		}
		if m.uiState.ActiveModal == ModalRepair {
			switch msg.(type) {
			case tea.WindowSizeMsg, fileChangeMsg, relintCompleteMsg, relintErrorMsg, continueWatchingMsg, clearProcessingStateMsg, docsLoadedMsg, docsErrorMsg:
				return false, nil
			}
			var cmd tea.Cmd
			r.draftView, cmd = r.draftView.Update(msg)
			return true, cmd
		}
		return false, nil
	}
	if r.editing {
		switch key.String() {
		case "esc":
			r.editing = false
			r.input.Blur()
			r.status = "Edit canceled"
			m.finishRepairEdit()
			return true, nil
		case "ctrl+c":
			r.close()
			m.quitting = true
			return true, tea.Quit
		case "enter":
			r.values[r.editIndex] = r.input.Value()
			r.selected[r.editIndex] = strings.TrimSpace(r.input.Value()) != ""
			r.editing = false
			r.input.Blur()
			r.invalidate()
			r.status = "Value saved for review · v preview before applying"
			m.refreshRepairRows()
			m.finishRepairEdit()
			return true, nil
		}
		var cmd tea.Cmd
		r.input, cmd = r.input.Update(msg)
		return true, cmd
	}
	if len(key.Text) == 1 && strings.Contains("FEVA", key.Text) {
		key.Text = strings.ToLower(key.Text)
		key.Code = rune(key.Text[0])
	}
	if key.String() == "ctrl+c" {
		r.close()
		m.quitting = true
		return true, tea.Quit
	}
	if m.uiState.ActiveModal == ModalRepair {
		switch key.String() {
		case "esc", "q":
			m.CloseActiveModal()
			return true, nil
		case "enter", " ", "space", "e", "v", "a":
			if key.String() == "enter" {
				key.Text = " "
				key.Code = ' '
				if o := m.currentRepairOption(); o != nil && o.Editable && strings.TrimSpace(r.values[o.Index]) == "" {
					key.Text = "e"
					key.Code = 'e'
				}
			}
			if key.String() == "v" || key.String() == "a" {
				m.CloseActiveModal()
			}
		default:
			var cmd tea.Cmd
			r.draftView, cmd = r.draftView.Update(msg)
			return true, cmd
		}
	} else if m.uiState.ActiveModal != ModalNone {
		return false, nil
	}
	switch key.String() {
	case "esc":
		if r.onlyFixes || r.pending != nil || (r.preview != nil && r.source != nil) {
			r.close()
			r.options = nil
			r.selected = nil
			r.values = nil
			r.onlyFixes = false
			r.editing = false
			r.status = "Fix review canceled · f to review again"
			m.ApplyFilter()
			m.UpdateDetailsViewContent()
			return true, nil
		}
		return false, nil
	case "f", "F":
		if r.discovery != nil {
			r.status = "Checking available fixes, please wait"
			return true, nil
		}
		if r.review == nil {
			m.openRepair()
		}
		if r.review == nil {
			return true, nil
		}
		r.onlyFixes = !r.onlyFixes
		if r.preview == nil && r.pending == nil && !strings.HasPrefix(r.status, "APPLIED:") {
			if r.onlyFixes {
				r.status = "Review fixes · Enter before/after · Space select · V verify"
			} else {
				r.status = "Fixes available · F to review"
			}
		}
		m.ApplyFilter()
		m.UpdateDetailsViewContent()
		return true, nil
	case "enter":
		if r.onlyFixes && r.review != nil {
			m.openRepairDetail()
			return true, nil
		}
		return false, nil
	case " ", "space":
		if o := m.currentRepairOption(); o != nil && (o.Reason == "" || o.Editable) {
			r.selected[o.Index] = !r.selected[o.Index]
			r.invalidate()
			r.status = "Selection changed · v preview before applying"
			m.refreshRepairRows()
		}
		return true, nil
	case "e":
		if r.pending != nil {
			r.status = "Verification running · wait before editing"
			return true, nil
		}
		o := m.currentRepairOption()
		if o == nil || !o.Editable {
			r.status = "This fix has no editable value"
			return true, nil
		}
		r.invalidate()
		r.editIndex = o.Index
		r.editing = true
		r.input = textinput.New()
		r.input.Placeholder = o.Description
		r.input.SetValue(r.values[o.Index])
		r.input.CharLimit = 65536
		r.editReturn = m.uiState.ActiveModal == ModalRepair
		m.OpenModal(ModalRepairEdit)
		m.resizeRepairModal()
		r.status = "Edit value · enter save · esc cancel"
		return true, r.input.Focus()
	case "v":
		if r.discovery != nil {
			r.status = "Checking available fixes, please wait"
			return true, nil
		}
		if r.pending != nil {
			r.status = "Preview is running"
			return true, nil
		}
		if r.review == nil {
			m.openRepair()
		}
		if r.review == nil {
			return true, nil
		}
		var choices []motor.FixChoice
		for _, o := range r.options {
			if r.selected[o.Index] {
				choice := motor.FixChoice{Index: o.Index}
				if o.Editable {
					v := r.values[o.Index]
					choice.Value = &v
				}
				choices = append(choices, choice)
			}
		}
		if len(choices) == 0 {
			r.status = "Select a fix or enter a value first"
			return true, nil
		}
		r.invalidate()
		job := &repairPreviewJob{}
		r.pending = job
		review := r.review
		r.status = "Verifying selected fixes · no files written"
		return true, func() tea.Msg { p, err := review.Preview(choices); job.finish(p, err); return repairPreviewMsg{job} }
	case "a":
		if r.preview == nil || r.source == nil {
			r.status = "Preview the current selection before applying"
			return true, nil
		}
		if r.config.Write == nil {
			r.status = "This source is read-only"
			return true, nil
		}
		p := r.preview
		if err := r.config.Write(r.source, p.Candidate); err != nil {
			r.status = "Apply failed: " + err.Error()
			return true, nil
		}
		// Retain final nodes until a subsequent review or quit releases the preview.
		m.allResults = p.Results
		m.specContent = append([]byte(nil), p.Candidate...)
		r.onlyFixes = false
		m.uiState.FilterState = FilterAll
		m.uiState.CategoryFilter = ""
		m.uiState.RuleFilter = ""
		m.categories = extractCategories(m.allResults)
		m.rules = extractRules(m.allResults)
		m.ApplyFilter()
		m.UpdateDetailsViewContent()
		r.status = fmt.Sprintf("APPLIED: %d fixes · %d remain · E %d→%d W %d→%d · score %d→%d", p.Fixed, len(p.Results), p.Before.Errors, p.After.Errors, p.Before.Warnings, p.After.Warnings, p.Before.Score, p.After.Score)
		// Keep the completed preview as the owner of displayed results, but prevent
		// applying it again. A new review will release both old owners.
		r.options = nil
		r.selected = nil
		r.source = nil
		r.diff = ""
		old := r.review
		r.review = nil
		if old != nil {
			go old.Close()
		}
		return true, nil
	}
	return false, nil
}
func (m *ViolationResultTableModel) repairDetails() string {
	r := m.repair
	if r == nil {
		return ""
	}

	o := m.repairOption(m.modalContent)
	if o == nil {
		return ""
	}
	state := "Skipped"
	if r.selected[o.Index] {
		state = "Selected"
	}
	if o.Reason != "" && !o.Editable {
		state = "Unavailable"
	}
	text := state + "\n" + o.Description
	if o.Editable {
		if o.Generated && r.values[o.Index] == o.Value {
			text += "\nGenerated suggestion, review before use"
		}
		value := r.values[o.Index]
		if value == "" {
			value = "(value required)"
		}
		text += "\n\n" + value + "\n\ne: edit value"
	}
	if o.Reason != "" && (!o.Editable || r.values[o.Index] == "") {
		text += "\n" + o.Reason
	}
	return text
}
func (m *ViolationResultTableModel) repairDiff() string {
	if m.repair == nil || m.repair.preview == nil || m.repair.source == nil {
		return ""
	}
	return m.repair.diff
}
func (m *ViolationResultTableModel) repairStatus() string {
	if m.repair == nil {
		return ""
	}
	r := m.repair
	selected := 0
	for _, v := range r.selected {
		if v {
			selected++
		}
	}
	prefix := ""
	if r.review != nil && !strings.HasPrefix(r.status, "PREVIEW:") {
		prefix = fmt.Sprintf("%d selected · ", selected)
	}
	available := r.automatic + r.suggested + r.needsInput
	if available > 0 && !r.onlyFixes && strings.HasPrefix(r.status, "Fixes available") {
		badge := lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true)
		if r.pulse > 0 && r.pulse%2 == 0 {
			badge = badge.Foreground(color.RGBYellow)
		}
		return badge.MaxWidth(max(1, m.width-2)).Render(fmt.Sprintf(" | FIX AVAILABLE · F review · %d auto / %d suggested / %d input", r.automatic, r.suggested, r.needsInput))
	}
	return lipgloss.NewStyle().MaxWidth(max(1, m.width-2)).Render(" | " + prefix + strings.ReplaceAll(r.status, "\n", " "))
}

func (m *ViolationResultTableModel) buildRepairTableData(results []*model.RuleFunctionResult, fileName string, width int, showPath bool) ([]table.Column, []table.Row) {
	columns, rows := BuildResultTableData(results, fileName, width, showPath)
	if m.repair != nil {
		for i, result := range results {
			if o := m.repairOption(result); o != nil {
				state := "[-] "
				if o.Editable && m.repair.values[o.Index] == "" {
					state = "[?] "
				} else if o.Reason == "" || o.Editable {
					state = "[ ] "
					if m.repair.selected[o.Index] {
						state = "[x] "
					}
				}
				label := ""
				if o.Generated && m.repair.values[o.Index] == o.Value {
					label = "generated "
				} else if o.Editable && m.repair.values[o.Index] == "" {
					label = "input needed "
				} else if o.Reason == "" {
					label = "auto "
				}
				rows[i][2] = state + label + rows[i][2]
			}
		}
	}
	return columns, rows
}
func (m *ViolationResultTableModel) refreshRepairRows() {
	columns, rows := m.buildRepairTableData(m.filteredResults, m.fileName, m.width, m.uiState.ShowPath)
	m.rows = rows
	m.table.SetColumns(columns)
	m.table.SetRows(rows)
}
