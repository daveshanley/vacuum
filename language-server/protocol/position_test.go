// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
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
	content := "hello\nworld"
	start, end := (Range{
		Start: Position{Line: 1, Character: 1},
		End:   Position{Line: 1, Character: 4},
	}).IndexesIn(content)
	assert.Equal(t, 7, start)
	assert.Equal(t, 10, end)
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
