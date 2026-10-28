// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"io"
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

// WriteMarkdownTable writes an unpadded table for portable reports. Callers must
// escape pipes, newlines, and markup in cells before calling it. Unlike the
// terminal table, a long cell does not add padding to every other row.
func WriteMarkdownTable(w io.Writer, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}
	writeRow := func(row []string) error {
		if _, err := io.WriteString(w, "| "+strings.Join(row, " | ")+" |\n"); err != nil {
			return err
		}
		return nil
	}
	if err := writeRow(headers); err != nil {
		return err
	}
	separators := make([]string, len(headers))
	for i := range separators {
		separators[i] = "---"
	}
	if err := writeRow(separators); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeRow(row); err != nil {
			return err
		}
	}
	return nil
}
