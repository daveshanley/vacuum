// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"strings"
	"unicode/utf8"
)

// Position identifies a zero-based line and UTF-16 character offset.
type Position struct {
	Line      UInteger `json:"line"`
	Character UInteger `json:"character"`
}

// IndexIn converts an LSP UTF-16 position into a byte offset. Invalid line
// positions retain the former Vacuum behavior and resolve to offset zero.
func (p Position) IndexIn(content string) int {
	offset, _ := p.indexIn(content)
	return offset
}

func (p Position) indexIn(content string) (int, bool) {
	lineStart := 0
	for line := UInteger(0); line < p.Line; line++ {
		next := strings.IndexByte(content[lineStart:], '\n')
		if next < 0 {
			return 0, false
		}
		lineStart += next + 1
	}

	lineEnd := len(content)
	if next := strings.IndexByte(content[lineStart:], '\n'); next >= 0 {
		lineEnd = lineStart + next
	}
	if lineEnd > lineStart && content[lineEnd-1] == '\r' {
		lineEnd--
	}

	target := int(p.Character)
	units := 0
	offset := lineStart
	for offset < lineEnd && units < target {
		r, width := utf8.DecodeRuneInString(content[offset:lineEnd])
		if r == utf8.RuneError && width == 0 {
			break
		}
		runeUnits := 1
		if r > 0xFFFF {
			runeUnits = 2
		}
		if units+runeUnits > target {
			break
		}
		units += runeUnits
		offset += width
	}
	return offset, true
}

// EndOfLineIn returns the UTF-16 position at the end of the selected line.
func (p Position) EndOfLineIn(content string) Position {
	lineStart := Position{Line: p.Line}.IndexIn(content)
	if p.Line > 0 && lineStart == 0 {
		return p
	}
	lineEnd := len(content)
	if next := strings.IndexByte(content[lineStart:], '\n'); next >= 0 {
		lineEnd = lineStart + next
	}
	if lineEnd > lineStart && content[lineEnd-1] == '\r' {
		lineEnd--
	}
	units := 0
	for _, r := range content[lineStart:lineEnd] {
		units++
		if r > 0xFFFF {
			units++
		}
	}
	return Position{Line: p.Line, Character: UInteger(units)}
}

// Range identifies a start and end position within a document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// IndexesIn converts an LSP range into byte offsets.
func (r Range) IndexesIn(content string) (int, int) {
	return r.Start.IndexIn(content), r.End.IndexIn(content)
}

// ValidIndexesIn converts a range into ordered byte offsets and reports
// whether both lines exist and the start does not follow the end.
func (r Range) ValidIndexesIn(content string) (int, int, bool) {
	start, startValid := r.Start.indexIn(content)
	end, endValid := r.End.indexIn(content)
	return start, end, startValid && endValid && start <= end
}
