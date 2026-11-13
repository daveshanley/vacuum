// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
	"github.com/pb33f/doctor/terminal/diffview"
	change "github.com/pb33f/libopenapi/what-changed/model"
)

// repairChangeView owns only text snapshots from the verified preview. It never
// parses documents or retains the comparison's document graph.
type repairChangeView struct {
	original, candidate []string
	changes             []*change.Change
	index               int
	full                bool
	width               int
	content             string
}

func (r *dashboardRepair) initChangeView() {
	r.changes = repairChangeView{}
	for _, ch := range r.preview.Impact.Changes {
		if ch != nil {
			r.changes.changes = append(r.changes.changes, ch)
		}
	}
	if len(r.changes.changes) != 0 {
		r.changes.original = strings.Split(string(r.source), "\n")
		r.changes.candidate = strings.Split(string(r.preview.Candidate), "\n")
	}
}

func repairChangeStyles() diffview.Styles {
	blue := lipgloss.NewStyle().Foreground(color.RGBBlue)
	yellow := lipgloss.NewStyle().Foreground(color.RGBYellow)
	return diffview.Styles{
		SelectedRow: yellow, RangeHighlight: yellow,
		AddedRow: yellow, AddedRange: yellow, RemovedRow: blue, RemovedRange: blue,
		Added: yellow, Modified: yellow, Removed: blue,
		Breaking: lipgloss.NewStyle().Foreground(color.RGBRed),
		Grey:     lipgloss.NewStyle().Foreground(color.RGBGrey),
		Detail:   lipgloss.NewStyle(), YAMLKey: color.StyleSyntaxKey,
	}
}

// repairChangeSource rejects external or ambiguous locations. Coordinates from
// the original document are never used against the candidate document.
func (m *ViolationResultTableModel) repairChangeSource(ch *change.Change) diffview.Source {
	if ch.Context == nil {
		return diffview.Source{}
	}
	ctx := ch.Context
	if !m.rootRepairLocation(ctx.DocumentLocation) {
		return diffview.Source{}
	}
	v := &m.repair.changes
	removed := ch.ChangeType == change.PropertyRemoved || ch.ChangeType == change.ObjectRemoved
	if !removed && ctx.NewLine != nil && *ctx.NewLine > 0 && *ctx.NewLine <= len(v.candidate) {
		return diffview.Source{Lines: v.candidate, Line: *ctx.NewLine, Side: diffview.Candidate}
	}
	if removed && ctx.OriginalLine != nil && *ctx.OriginalLine > 0 && *ctx.OriginalLine <= len(v.original) {
		return diffview.Source{Lines: v.original, Line: *ctx.OriginalLine, Side: diffview.Original}
	}
	return diffview.Source{}
}

func (m *ViolationResultTableModel) rootRepairLocation(location string) bool {
	if location == "" {
		return true
	}
	path := m.fileName
	if m.repair.config != nil && m.repair.config.SourcePath != "" {
		path = m.repair.config.SourcePath
	}
	root, err := filepath.Abs(path)
	return err == nil && filepath.Clean(location) == root
}

func (m *ViolationResultTableModel) hasRepairPreview() bool {
	return m.repair != nil && m.repair.preview != nil && m.repair.source != nil
}

func (m *ViolationResultTableModel) repairReviewContent(width int) string {
	r := m.repair
	if r == nil || r.preview == nil || r.source == nil {
		return ""
	}
	v := &r.changes
	if v.full || len(v.changes) == 0 {
		return r.diff
	}
	width = max(1, width)
	if v.content != "" && v.width == width {
		return v.content
	}
	ch := v.changes[v.index]
	source := m.repairChangeSource(ch)
	context := "Source context unavailable; t shows the complete file diff."
	if len(source.Lines) > 0 {
		context = "Autofixed source"
		if source.Side == diffview.Original {
			context = "Current source"
		}
	}
	v.content = fmt.Sprintf("%s · change %d/%d\n%s\n\n", repairImpactLabel(r.preview.Impact), v.index+1, len(v.changes), context) + diffview.RenderChange(ch, source, width, repairChangeStyles())
	v.width = width
	return v.content
}

// changeRepairView changes presentation only. Selection, acceptance, and the
// exact candidate passed to Save remain untouched.
func (m *ViolationResultTableModel) changeRepairView(key string) bool {
	r := m.repair
	if r == nil || r.preview == nil || r.source == nil {
		return false
	}
	v := &r.changes
	switch strings.ToLower(key) {
	case "[", "]":
		if len(v.changes) == 0 {
			return true
		}
		step := 1
		if key == "[" {
			step = -1
		}
		v.index = (v.index + step + len(v.changes)) % len(v.changes)
		v.full = false
	case "t":
		v.full = !v.full
	default:
		return false
	}
	v.content = ""
	if r.save != nil && r.save.step == repairSavePreview {
		r.save.view.SetContent(m.repairReviewContent(r.save.view.Width()))
		r.save.view.SetXOffset(0)
		r.save.view.GotoTop()
	} else if m.uiState.ActiveModal == ModalCode {
		m.codeViewport.SetContent(m.repairReviewContent(m.codeViewport.Width()))
		m.codeViewport.SetXOffset(0)
		m.codeViewport.GotoTop()
	}
	return true
}
