// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"
)

func RenderMarkdownTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	var output strings.Builder

	// calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// header row
	output.WriteString("|")
	for i, h := range headers {
		output.WriteString(fmt.Sprintf(" %-*s |", widths[i], h))
	}
	output.WriteString("\n")

	// separator row
	output.WriteString("|")
	for _, w := range widths {
		output.WriteString(fmt.Sprintf(" %s |", strings.Repeat("-", w)))
	}
	output.WriteString("\n")

	// data rows
	for _, row := range rows {
		output.WriteString("|")
		for i, cell := range row {
			if i < len(widths) {
				output.WriteString(fmt.Sprintf(" %-*s |", widths[i], cell))
			}
		}
		output.WriteString("\n")
	}

	return output.String()
}
