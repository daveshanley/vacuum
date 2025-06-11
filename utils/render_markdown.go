// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package utils

import (
	"strings"
	"unicode/utf8"
)

// RenderMarkdownTable builds a Markdown table from headers and rows.
func RenderMarkdownTable(headers []string, rows [][]string) string {
	colCount := len(headers)
	colWidths := make([]int, colCount)

	// determine max width per column (using rune count for proper Unicode support)
	for i, h := range headers {
		colWidths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			length := utf8.RuneCountInString(cell)
			if length > colWidths[i] {
				colWidths[i] = length
			}
		}
	}

	// helper to pad a string to `width` runes
	pad := func(s string, width int) string {
		padCount := width - utf8.RuneCountInString(s)
		return s + strings.Repeat(" ", padCount)
	}

	var sb strings.Builder

	// header row
	sb.WriteString("|")
	for i, h := range headers {
		sb.WriteString(" " + pad(h, colWidths[i]) + " |")
	}
	sb.WriteString("\n")

	// divider row
	sb.WriteString("|")
	for i := 0; i < colCount; i++ {
		sb.WriteString(" " + strings.Repeat("-", colWidths[i]) + " |")
	}
	sb.WriteString("\n")

	// data rows
	for _, row := range rows {
		sb.WriteString("|")
		for i, cell := range row {
			sb.WriteString(" " + pad(cell, colWidths[i]) + " |")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
