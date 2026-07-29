// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"strings"
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestPositionIndexInUsesUTF16CodeUnits(t *testing.T) {
	content := "a😀b\nnext"
	tests := []struct {
		position Position
		want     int
	}{
		{Position{Line: 0, Character: 0}, 0},
		{Position{Line: 0, Character: 1}, 1},
		{Position{Line: 0, Character: 2}, 1},
		{Position{Line: 0, Character: 3}, 5},
		{Position{Line: 0, Character: 4}, 6},
		{Position{Line: 1, Character: 2}, 9},
		{Position{Line: 8, Character: 0}, 0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.position.IndexIn(content))
	}
}

func TestPositionIndexInClampsAtLineEndAndHandlesCRLF(t *testing.T) {
	content := "ab\r\ncd"
	assert.Equal(t, 2, (Position{Line: 0, Character: 99}).IndexIn(content))
	assert.Equal(t, 5, (Position{Line: 1, Character: 1}).IndexIn(content))
	assert.Equal(t, Position{Line: 0, Character: 2}, (Position{Line: 0}).EndOfLineIn(content))
}

func TestRangeIndexesIn(t *testing.T) {
	content := "hello\nw😀rld"
	start, end := (Range{
		Start: Position{Line: 1, Character: 1},
		End:   Position{Line: 1, Character: 4},
	}).IndexesIn(content)
	assert.Equal(t, 7, start)
	assert.Equal(t, 12, end)
}

func TestRangeValidIndexesInRejectsInvalidAndReversedRanges(t *testing.T) {
	content := "hello\nworld"
	start, end, ok := (Range{
		Start: Position{Line: 0, Character: 1},
		End:   Position{Line: 1, Character: 2},
	}).ValidIndexesIn(content)
	assert.True(t, ok)
	assert.Equal(t, 1, start)
	assert.Equal(t, 8, end)

	_, _, ok = (Range{
		Start: Position{Line: 1, Character: 2},
		End:   Position{Line: 0, Character: 1},
	}).ValidIndexesIn(content)
	assert.False(t, ok)

	_, _, ok = (Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: 8, Character: 0},
	}).ValidIndexesIn(content)
	assert.False(t, ok)
}

func TestRangeIndexesInPreservesIndependentInvalidPositionBehavior(t *testing.T) {
	start, end := (Range{
		Start: Position{Line: 8, Character: 0},
		End:   Position{Line: 1, Character: 2},
	}).IndexesIn("hello\nworld")

	assert.Equal(t, 0, start)
	assert.Equal(t, 8, end)
}

func TestRangeIndexesInMatchesIndependentPositionConversion(t *testing.T) {
	contents := []string{
		"",
		"hello\nworld",
		"a😀b\r\n次の行\n",
	}
	for _, content := range contents {
		for startLine := UInteger(0); startLine < 4; startLine++ {
			for startCharacter := UInteger(0); startCharacter < 8; startCharacter++ {
				for endLine := UInteger(0); endLine < 4; endLine++ {
					for endCharacter := UInteger(0); endCharacter < 8; endCharacter++ {
						target := Range{
							Start: Position{Line: startLine, Character: startCharacter},
							End:   Position{Line: endLine, Character: endCharacter},
						}
						wantStart, startValid := target.Start.indexIn(content)
						wantEnd, endValid := target.End.indexIn(content)

						start, end := target.IndexesIn(content)
						assert.Equal(t, wantStart, start)
						assert.Equal(t, wantEnd, end)

						validStart, validEnd, valid := target.ValidIndexesIn(content)
						assert.Equal(t, wantStart, validStart)
						assert.Equal(t, wantEnd, validEnd)
						assert.Equal(t, startValid && endValid && wantStart <= wantEnd, valid)
					}
				}
			}
		}
	}
}

func BenchmarkRangeValidIndexesInLargeDocument(b *testing.B) {
	content := strings.Repeat("0123456789abcdef\n", 10_000)
	target := Range{
		Start: Position{Line: 9_999, Character: 2},
		End:   Position{Line: 9_999, Character: 14},
	}

	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = target.ValidIndexesIn(content)
	}
}
