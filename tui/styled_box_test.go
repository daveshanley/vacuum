// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"testing"

	"github.com/pb33f/testify/require"
)

// A box wider than the terminal wraps every line of its border.
func TestCalculateBoxWidth_FitsTerminal(t *testing.T) {
	for width := 53; width <= 200; width++ {
		require.LessOrEqual(t, calculateBoxWidth(width), width, "box overflows a %d column terminal", width)
	}
	require.Equal(t, fullBoxWidth, calculateBoxWidth(200))
}
