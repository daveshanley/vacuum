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
	lineStart, valid, _, _ := lineStarts(content, p.Line, p.Line)
	if !valid {
		return 0, false
	}
	offset, _ := characterOffsets(contentLine(content, lineStart), p.Character, p.Character)
	return lineStart + offset, true
}

// EndOfLineIn returns the UTF-16 position at the end of the selected line.
func (p Position) EndOfLineIn(content string) Position {
	lineStart, valid, _, _ := lineStarts(content, p.Line, p.Line)
	if !valid {
		return p
	}
	units := 0
	for _, r := range contentLine(content, lineStart) {
		units += utf16RuneUnits(r)
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
	start, end, _, _ := r.indexesIn(content)
	return start, end
}

// ValidIndexesIn converts a range into ordered byte offsets and reports
// whether both lines exist and the start does not follow the end.
func (r Range) ValidIndexesIn(content string) (int, int, bool) {
	start, end, startValid, endValid := r.indexesIn(content)
	return start, end, startValid && endValid && start <= end
}

func (r Range) indexesIn(content string) (int, int, bool, bool) {
	startLine, startValid, endLine, endValid := lineStarts(
		content,
		r.Start.Line,
		r.End.Line,
	)
	if !startValid && !endValid {
		return 0, 0, false, false
	}
	if startLine == endLine && startValid && endValid {
		line := contentLine(content, startLine)
		start, end := characterOffsets(line, r.Start.Character, r.End.Character)
		return startLine + start, startLine + end, true, true
	}

	start := 0
	if startValid {
		line := contentLine(content, startLine)
		offset, _ := characterOffsets(line, r.Start.Character, r.Start.Character)
		start = startLine + offset
	}
	end := 0
	if endValid {
		line := contentLine(content, endLine)
		offset, _ := characterOffsets(line, r.End.Character, r.End.Character)
		end = endLine + offset
	}
	return start, end, startValid, endValid
}

func lineStarts(content string, first, second UInteger) (int, bool, int, bool) {
	firstStart, secondStart := 0, 0
	firstValid, secondValid := false, false
	lineStart := 0
	for line := UInteger(0); ; line++ {
		if line == first {
			firstStart, firstValid = lineStart, true
		}
		if line == second {
			secondStart, secondValid = lineStart, true
		}
		if firstValid && secondValid {
			return firstStart, true, secondStart, true
		}
		next := strings.IndexByte(content[lineStart:], '\n')
		if next < 0 {
			return firstStart, firstValid, secondStart, secondValid
		}
		lineStart += next + 1
	}
}

func contentLine(content string, lineStart int) string {
	lineEnd := len(content)
	if next := strings.IndexByte(content[lineStart:], '\n'); next >= 0 {
		lineEnd = lineStart + next
	}
	if lineEnd > lineStart && content[lineEnd-1] == '\r' {
		lineEnd--
	}
	return content[lineStart:lineEnd]
}

func characterOffsets(line string, first, second UInteger) (int, int) {
	targets := [2]int{int(first), int(second)}
	offsets := [2]int{-1, -1}
	for index, target := range targets {
		if target == 0 {
			offsets[index] = 0
		}
	}

	units := 0
	for offset := 0; offset < len(line) && (offsets[0] < 0 || offsets[1] < 0); {
		r, width := utf8.DecodeRuneInString(line[offset:])
		if r == utf8.RuneError && width == 0 {
			break
		}
		runeUnits := utf16RuneUnits(r)
		nextUnits := units + runeUnits
		for index, target := range targets {
			if offsets[index] >= 0 {
				continue
			}
			switch {
			case target < nextUnits:
				offsets[index] = offset
			case target == nextUnits:
				offsets[index] = offset + width
			}
		}
		units = nextUnits
		offset += width
	}
	for index := range offsets {
		if offsets[index] < 0 {
			offsets[index] = len(line)
		}
	}
	return offsets[0], offsets[1]
}

func utf16RuneUnits(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}
