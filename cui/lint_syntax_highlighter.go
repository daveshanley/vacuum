// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
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

	// SUPER SIMPLE: Find colon, color everything before it (key) + colon blue
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return "", false
	}

	// Find leading whitespace
	leadingWhitespace := ""
	contentStart := 0
	for i, r := range line {
		if r != ' ' && r != '\t' {
			leadingWhitespace = line[:i]
			contentStart = i
			break
		}
	}

	// Find trailing whitespace
	trimmedLine := strings.TrimRight(line, " \t\r\n")
	trailingWhitespace := line[len(trimmedLine):]

	// Extract just the content (no leading or trailing whitespace)
	content := trimmedLine[contentStart:]

	// Check if this is a $ref line - style it green instead of blue
	var styledContent string
	if strings.HasPrefix(content, "$ref:") {
		// Style $ref lines green
		styledContent = syntaxRefStyle.Render(content)
	} else {
		// Style regular key-value lines blue
		styledContent = syntaxKeyStyle.Render(content)
	}

	return leadingWhitespace + styledContent + trailingWhitespace, true
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
