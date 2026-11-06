package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
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
