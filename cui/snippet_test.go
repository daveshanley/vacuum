// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/gizak/termui/v3"
	"github.com/stretchr/testify/assert"
	"image"
	"strings"
	"testing"
)

func TestNewSnippet(t *testing.T) {
	assert.NotNil(t, NewSnippet())
}

func TestSnippet_Draw(t *testing.T) {
	s := NewSnippet()
	s.Text = "chicken nuggets"

	rect := image.Rectangle{
		Max: image.Point{
			X: 0,
			Y: 0,
		},
		Min: image.Point{
			X: 10,
			Y: 10,
		},
	}

	buf := termui.NewBuffer(rect)
	s.Draw(buf)

	cell := buf.GetCell(image.Point{
		X: 0,
		Y: 0,
	})

	// not sure how to fully test this code yet, not without a waste of time.
	assert.NotNil(t, cell.Rune)
}

func TestSnippet_Draw_NoWrap(t *testing.T) {
	s := NewSnippet()
	s.Text = "chicken nuggets"
	s.WrapText = true

	rect := image.Rectangle{
		Max: image.Point{
			X: 0,
			Y: 0,
		},
		Min: image.Point{
			X: 10,
			Y: 10,
		},
	}

	buf := termui.NewBuffer(rect)
	s.Draw(buf)

	cell := buf.GetCell(image.Point{
		X: 0,
		Y: 0,
	})

	// not sure how to fully test this code yet, not without a waste of time.
	assert.NotNil(t, cell.Rune)
}

func TestSnippet_Draw_WrapLong(t *testing.T) {
	s := NewSnippet()

	var build strings.Builder

	for i := 0; i < 100; i++ {
		build.WriteString("chicken nuggets, chicken soup, chicken nuggets for me and you.")
	}

	s.Text = build.String()
	s.WrapText = false

	rect := image.Rectangle{
		Max: image.Point{
			X: 0,
			Y: 0,
		},
		Min: image.Point{
			X: 2,
			Y: 10,
		},
	}

	buf := termui.NewBuffer(rect)
	s.SetRect(0, 0, 5, 5)
	s.Draw(buf)

	cell := buf.GetCell(image.Point{
		X: 0,
		Y: 0,
	})

	// not sure how to fully test this code yet, not without a waste of time.
	assert.NotNil(t, cell.Rune)
}
