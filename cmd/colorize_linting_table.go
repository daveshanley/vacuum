// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/table"
)

func colorizeTableOutput(tableView string, cursor int, rows []table.Row) string {
	lines := strings.Split(tableView, "\n")

	// Get selected row's location to identify it
	var selectedLocation string
	if cursor >= 0 && cursor < len(rows) {
		selectedLocation = rows[cursor][0]
	}

	red := "\033[38;5;196m"  // Bright red for errors
	grey := "\033[38;5;246m" // Nice readable gray #909090
	lightGreyItalic := "\033[3;38;5;251m"
	secondaryPink := "\033[38;5;164m"
	lightGrey := "\033[38;5;253m"
	blue := "\033[38;5;45m" // Bright blue for paths
	bold := "\033[1m"

	reset := "\033[0m"

	var result strings.Builder
	for i, line := range lines {
		// Skip coloring for headers and selected row
		isSelectedLine := selectedLocation != "" && strings.Contains(line, selectedLocation)

		// Color the path content if not selected
		if isSelectedLine && i > 0 {
			// Selected lines (including siblings) get pink
			line = secondaryPink + line + reset
		}

		if i >= 1 && !isSelectedLine { // Start from line 1 (skip header row at 0)

			if locationRegex.MatchString(line) {

				line = locationRegex.ReplaceAllStringFunc(line, func(match string) string {
					parts := locationRegex.FindStringSubmatch(match)
					if len(parts) == 4 {
						filePath := parts[1]
						lineNum := parts[2]
						colNum := parts[3]

						file := filepath.Base(filePath)
						dir := filepath.Dir(filePath)
						filePath = fmt.Sprintf("%s/%s%s%s", dir, lightGreyItalic, file, reset)

						// Apply tertiary color to file path and line/col numbers
						coloredPath := fmt.Sprintf("%s%s%s", grey, filePath, reset)
						coloredLine := fmt.Sprintf("%s%s%s", bold, lineNum, reset)
						coloredCol := fmt.Sprintf("%s%s%s", lightGrey, colNum, reset)
						sep := fmt.Sprintf("%s:%s", lightGrey, reset)
						return fmt.Sprintf("%s%s%s%s%s", coloredPath, sep, coloredLine, sep, coloredCol)
					}
					return match // Fallback to original if something goes wrong
				})
			}

			if jsonPathRegex.MatchString(line) {
				line = jsonPathRegex.ReplaceAllStringFunc(line, func(match string) string {
					return fmt.Sprintf("%s%s%s", grey, match, reset)
				})
			}

			if circularRefRegex.MatchString(line) {
				line = circularRefRegex.ReplaceAllStringFunc(line, func(match string) string {
					circResult := ""

					parts := partRegex.FindAllStringSubmatch(match, -1)
					for _, part := range parts {
						if part[1] != "" {
							// ref
							circResult += fmt.Sprintf("%s%s%s", lightGrey, part[1], reset)
						} else if part[2] != "" {
							// arrow
							circResult += fmt.Sprintf("%s%s%s", red, part[2], reset)
						}
					}
					return circResult
				})
			}

			line = strings.Replace(line, "✗ error", fmt.Sprintf("%s%s", "\033[38;5;196m", "✗ error\033[0m"), -1)
			line = strings.Replace(line, "▲ warning", fmt.Sprintf("%s%s", "\033[38;5;220m", "▲ warning\033[0m"), -1)
			line = strings.Replace(line, "● info", fmt.Sprintf("%s%s", blue, "● info\033[0m"), -1)
		}

		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
