// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/motor/autofix"
)

// repairSourceRows aligns the bounded draft excerpts by unchanged lines. Only
// changed lines receive diff colors; context uses the normal source highlighter.
func repairSourceRows(before, after string, width int, isYAML bool, offset int) ([]string, int) {
	lines := func(source string) []string {
		if source == "" {
			return nil
		}
		return strings.Split(source, "\n")
	}
	left, right := lines(before), lines(after)
	// Draft excerpts contain at most 40 source lines plus an omission notice.
	lcs := make([][]int, len(left)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(right)+1)
	}
	for i := len(left) - 1; i >= 0; i-- {
		for j := len(right) - 1; j >= 0; j-- {
			if left[i] == right[j] {
				lcs[i][j] = 1 + lcs[i+1][j+1]
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	lw, rw := repairColumns(width)
	var rows []string
	firstChange := -1
	appendLine := func(a, b string, changed bool) {
		if changed {
			if firstChange < 0 {
				firstChange = len(rows)
			}
			if a != "" {
				a = lipgloss.NewStyle().Foreground(color.RGBBlue).Render(a)
			}
			if b != "" {
				b = lipgloss.NewStyle().Foreground(color.RGBYellow).Render(b)
			}
		} else {
			a = ApplySyntaxHighlightingToLine(a, isYAML)
			b = ApplySyntaxHighlightingToLine(b, isYAML)
		}
		// One source line stays on one screen row. Cropping is explicit and can
		// be panned horizontally; presentation must not invent YAML/JSON newlines.
		rows = append(rows, repairColumnRow(repairSourceWindow(a, max(1, lw-2), offset), repairSourceWindow(b, max(1, rw-2), offset), width))
	}
	for i, j := 0, 0; i < len(left) || j < len(right); {
		if i < len(left) && j < len(right) && left[i] == right[j] {
			appendLine(left[i], right[j], false)
			i++
			j++
			continue
		}
		startI, startJ := i, j
		for i < len(left) || j < len(right) {
			if i < len(left) && j < len(right) && left[i] == right[j] {
				break
			}
			if j == len(right) || (i < len(left) && lcs[i+1][j] >= lcs[i][j+1]) {
				i++
			} else {
				j++
			}
		}
		for n := 0; n < max(i-startI, j-startJ); n++ {
			a, b := "", ""
			if startI+n < i {
				a = left[startI+n]
			}
			if startJ+n < j {
				b = right[startJ+n]
			}
			appendLine(a, b, true)
		}
	}
	return rows, max(0, firstChange)
}

// repairSourceWindow shows a terminal-cell slice with explicit crop markers.
func repairSourceWindow(line string, width, offset int) string {
	length := ansi.StringWidth(line)
	if offset >= length {
		return ""
	}
	tail := repairSourceSlice(line, offset, width)
	marker := lipgloss.NewStyle().Foreground(color.RGBGrey)
	prefix, suffix := "", ""
	available := width
	if offset > 0 {
		prefix = marker.Render("‹")
		available--
	}
	if length-offset > available {
		suffix = marker.Render("›")
		available--
	}
	return prefix + ansi.Truncate(tail, max(0, available), "") + suffix
}

func repairSourceMaxOffset(before, after string, width int) int {
	longest := 0
	for _, source := range []string{before, after} {
		for _, line := range strings.Split(source, "\n") {
			longest = max(longest, ansi.StringWidth(line))
		}
	}
	left, right := repairColumns(width)
	visible := max(1, min(left, right)-2)
	if longest <= visible {
		return 0
	}
	return max(0, longest-visible+1)
}

// repairSourceSlice aligns the left edge to a whole grapheme before cropping.
// It bounds allocations to the visible prefix, including for minified files.
func repairSourceSlice(line string, offset, width int) string {
	// Keep a second window for left-edge grapheme alignment, then work only
	// on that bounded prefix instead of rescanning the complete source line.
	line = ansi.Truncate(line, offset+2*width, "")
	if offset > 0 {
		prefix := ansi.Truncate(line, offset, "")
		if ansi.StringWidth(prefix) < offset {
			if ansi.StringWidth(line) <= offset {
				return ""
			}
			for ansi.StringWidth(prefix) < offset {
				offset++
				prefix = ansi.Truncate(line, offset, "")
			}
		}
	}
	return ansi.Cut(line, offset, offset+width)
}

// Each document keeps its own scroll position. Only visible rows are highlighted.
type overlayReview struct {
	panes  [2]viewport.Model
	yaml   [2]bool
	focus  int
	lines  []int // One-based candidate lines from the verified impact.
	change int
}

func (m *ViolationResultTableModel) newOverlayReview(p *autofix.FixPreview) *overlayReview {
	v := &overlayReview{}
	for i, data := range [][]byte{p.Overlay, p.Candidate} {
		v.panes[i] = viewport.New()
		v.panes[i].SetContent(overlayDisplayText(data))
		v.yaml[i] = !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{"))
	}
	for _, ch := range p.Impact.Changes {
		if ch == nil || ch.Context == nil || ch.Context.NewLine == nil || !m.rootRepairLocation(ch.Context.DocumentLocation) {
			continue
		}
		line := *ch.Context.NewLine
		if line > 0 && line <= v.panes[1].TotalLineCount() {
			v.lines = append(v.lines, line)
		}
	}
	slices.Sort(v.lines)
	v.lines = slices.Compact(v.lines)
	v.jump(0)
	return v
}

// Expand tabs before measuring terminal cells. Escape control characters in
// the display copy so source text cannot issue commands to the terminal.
func overlayDisplayText(data []byte) string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	special := func(r rune) bool { return r != '\n' && unicode.IsControl(r) }
	if strings.IndexFunc(text, special) < 0 {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		switch {
		case r == '\t':
			b.WriteString("    ")
		case special(r):
			b.WriteString(strings.Trim(strconv.QuoteRuneToASCII(r), "'"))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Unstyled ASCII maps one byte to one cell. Inspect only the visible prefix
// of minified JSON; defer to the grapheme-aware crop for Unicode.
func overlaySourceSlice(line string, offset, width int) string {
	end := min(len(line), offset+width)
	if strings.IndexFunc(line[:min(len(line), end+1)], func(r rune) bool { return r >= utf8.RuneSelf }) < 0 {
		return line[min(offset, end):end]
	}
	return repairSourceSlice(line, offset, width)
}

func (v *overlayReview) jump(step int) {
	if n := len(v.lines); n > 0 {
		v.change = (v.change + step + n) % n
		pane := &v.panes[1]
		pane.SetXOffset(0)
		pane.SetYOffset(max(0, v.lines[v.change]-4))
	}
}

func (v *overlayReview) render(width, height int) string {
	labels := [2]string{"Overlay file", "Resulting specification"}
	labels[v.focus] = lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true).Render("› " + labels[v.focus])
	left, right := repairColumns(width)
	widths := [2]int{left, right}
	var columns [2]string
	for i := range v.panes {
		pane := &v.panes[i]
		pane.SetWidth(widths[i])
		pane.SetHeight(max(1, height-2))
		pane.SetXOffset(pane.XOffset())
		pane.SetYOffset(pane.YOffset())
		view := *pane
		offset := view.XOffset()
		view.SetXOffset(0)
		view.StyleLineFunc = func(index int) lipgloss.Style {
			return lipgloss.NewStyle().Transform(func(line string) string {
				line = overlaySourceSlice(line, offset, widths[i])
				if i == 1 {
					if _, changed := slices.BinarySearch(v.lines, index+1); changed {
						return lipgloss.NewStyle().Foreground(color.RGBYellow).Render(line)
					}
				}
				return ApplySyntaxHighlightingToLine(line, v.yaml[i])
			})
		}
		columns[i] = lipgloss.NewStyle().Width(widths[i]).Render(ansi.Truncate(labels[i], widths[i], "…")) + "\n" +
			lipgloss.NewStyle().Foreground(color.RGBPink).Render(strings.Repeat("─", widths[i])) + "\n" + view.View()
	}
	pink := lipgloss.NewStyle().Foreground(color.RGBPink)
	divider := " │ \n─┼─" + strings.Repeat("\n │ ", max(1, height-2))
	return lipgloss.JoinHorizontal(lipgloss.Top, columns[0], pink.Render(divider), columns[1])
}
