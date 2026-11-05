// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"

	"github.com/pb33f/testify/require"
)

// Both summary tables must fit the terminal and print at the same width, or
// their rows wrap and the separators break across lines.
func TestCalculateColumnWidths_TablesFitTerminal(t *testing.T) {
	for width := 60; width <= 200; width++ {
		widths := calculateColumnWidths(width)
		rules := 1 + widths.rule + 2 + widths.violation + 2 + widths.impact
		categories := 1 + widths.category + 2 + widths.number + 2 + widths.number + 2 + widths.info()

		require.LessOrEqual(t, rules, width, "rules table overflows a %d column terminal", width)
		require.Equal(t, rules, categories, "tables differ in width at %d columns", width)
	}
}

func TestCalculateColumnWidths_FullWidthUnchanged(t *testing.T) {
	widths := calculateColumnWidths(fullTableWidth)
	require.Equal(t, 56, widths.info())
	require.Equal(t, 40, widths.rule)
}

// A terminal of any width, even one or two columns, lays out at no less than the
// narrowest table, so the file header's rule never has a negative length.
func TestLayoutWidth_NeverNarrowerThanTheTables(t *testing.T) {
	require.Equal(t, 120, layoutWidth(0))
	for width := 1; width <= 200; width++ {
		layout := layoutWidth(width)
		require.GreaterOrEqual(t, layout, minTerminalWidth)
		// no-style output takes 3 more columns off (getTerminalWidth)
		require.GreaterOrEqual(t, fileHeaderWidth(layout-3)-1, 0, "header rule at %d columns", width)
	}
}

func TestFileHeaderWidth_MatchesSummaryTable(t *testing.T) {
	require.Equal(t, 67, fileHeaderWidth(80))
	require.Equal(t, fullTableWidth, fileHeaderWidth(120))
}
