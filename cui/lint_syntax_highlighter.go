// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/charmbracelet/lipgloss/v2"
	"strings"
)

// HighlightYAMLRefLine handles special highlighting for $ref lines
func HighlightYAMLRefLine(line string) (string, bool) {
	// DISABLED - return false to let normal processing handle it
	return "", false
}

// HighlightYAMLComment handles comment highlighting for YAML
func HighlightYAMLComment(line string, isYAML bool) (string, bool) {
	// DISABLED - return false to let normal processing handle it
	return "", false
}

// HighlightYAMLKeyValue handles key-value pair highlighting for YAML
func HighlightYAMLKeyValue(line string) (string, bool) {

	if strings.Contains(line, "$ref") {
		colonIdx := strings.Index(line, ":")
		keyAndColon := line[:colonIdx+1]
		after := line[colonIdx+1:]

		// Find linebreak in 'after' and style only up to it
		lineBreakIdx := strings.IndexAny(after, "\n\r")

		// Style value up to linebreak, keep linebreak unstyled
		valueBeforeBreak := after[:lineBreakIdx]
		lineBreakAndRest := after[lineBreakIdx:]
		s := lipgloss.NewStyle().Foreground(RGBGreen).Bold(true)
		return s.Render(keyAndColon+valueBeforeBreak) + lineBreakAndRest, true

	}

	// SUPER SIMPLE: Find colon, color everything before it (key) + colon blue
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return "", false
	}

	// Color: blue(everything before colon + colon) + everything after
	keyAndColon := line[:colonIdx+1]
	after := line[colonIdx+1:]

	return syntaxKeyStyle.Render(keyAndColon) + after, true
}

// HighlightYAMLValue applies appropriate styling to a YAML value
func HighlightYAMLValue(value string) string {
	// DISABLED - just return the value as-is
	return value
}

// HighlightYAMLListItem handles list item highlighting for YAML
func HighlightYAMLListItem(line string) (string, bool) {
	// DISABLED - return false to let normal processing handle it
	return "", false
}

// HighlightJSONLine handles JSON syntax highlighting
func HighlightJSONLine(line string) string {
	// SUPER SIMPLE: Just find "key": pattern and color key+colon blue
	if idx := strings.Index(line, "\":"); idx > 0 {
		// Find where the key starts (opening quote)
		keyStart := strings.LastIndex(line[:idx], "\"")
		if keyStart >= 0 {
			// Color: everything before key + blue(key and colon) + everything after colon
			before := line[:keyStart]
			keyAndColon := line[keyStart : idx+2] // includes both quotes and colon
			after := line[idx+2:]

			return before + syntaxKeyStyle.Render(keyAndColon) + after
		}
	}
	// No key found, return as-is
	return line
}

// ApplySyntaxHighlightingToLine applies syntax highlighting to a single line
func ApplySyntaxHighlightingToLine(line string, isYAML bool) string {
	// Ensure styles are initialized
	if !syntaxStylesInit {
		InitSyntaxStyles()
	}

	// empty line, skip.
	if line == "" {
		return line
	}

	if isYAML {
		// Try simple key-value highlighting
		if result, handled := HighlightYAMLKeyValue(line); handled {
			return result
		}
		// No match, return as-is
		return line
	} else {
		return HighlightJSONLine(line)
	}
}
