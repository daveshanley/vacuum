// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
)

// HighlightYAMLRefLine handles special highlighting for $ref lines
func HighlightYAMLRefLine(line string) (string, bool) {
	if strings.Contains(line, "$ref:") {
		if idx := strings.Index(line, "$ref:"); idx >= 0 {
			beforeRef := line[:idx]
			return beforeRef + syntaxRefStyle.Render(line[idx:]), true
		}
	}

	trimmed := strings.TrimSpace(line)
	if strings.Contains(trimmed, "'#/") || strings.Contains(trimmed, "\"#/") ||
		strings.Contains(trimmed, "#/components/") ||
		strings.Contains(trimmed, "#/definitions/") ||
		strings.Contains(trimmed, "#/schemas/") ||
		strings.Contains(trimmed, "#/parameters/") ||
		strings.Contains(trimmed, "#/responses/") ||
		strings.Contains(trimmed, "#/paths/") {
		// This is a $ref path - style the entire line
		return syntaxRefStyle.Render(line), true
	}

	return "", false
}

// HighlightYAMLComment handles comment highlighting for YAML
func HighlightYAMLComment(line string, isYAML bool) (string, bool) {
	if commentIndex := strings.IndexByte(line, '#'); commentIndex >= 0 {
		// make sure it's actually a comment, not part of a $ref
		if !strings.Contains(line[:commentIndex], "$ref") {
			beforeComment := line[:commentIndex]
			comment := line[commentIndex:]
			return ApplySyntaxHighlightingToLine(beforeComment, isYAML) + syntaxCommentStyle.Render(comment), true
		}
	}
	return "", false
}

// HighlightYAMLKeyValue handles key-value pair highlighting for YAML
func HighlightYAMLKeyValue(line string) (string, bool) {
	if matches := yamlKeyValueRegex.FindStringSubmatch(line); matches != nil {
		indent := matches[1]
		key := matches[2]
		separator := matches[3]
		value := matches[4]

		// special handling for $ref key
		coloredKey := key
		coloredValue := value

		if key == "$ref" {
			coloredKey = syntaxRefStyle.Render(key)
			coloredValue = syntaxRefStyle.Render(value)
		} else {
			coloredKey = syntaxKeyStyle.Render(key)
			coloredValue = HighlightYAMLValue(value)
		}

		return indent + coloredKey + separator + coloredValue, true
	}
	return "", false
}

// HighlightYAMLValue applies appropriate styling to a YAML value
func HighlightYAMLValue(value string) string {
	trimmedValue := strings.TrimSpace(value)

	// Check boolean values first
	switch trimmedValue {
	case "true", "false", "null":
		return syntaxBoolStyle.Render(value)
	default:
		if numberValueRegex.MatchString(trimmedValue) {
			return syntaxNumberStyle.Render(value)
		} else if len(value) > 0 && value[0] == '"' {
			// double-quoted strings are green
			return syntaxStringStyle.Render(value)
		} else if len(value) > 0 && value[0] == '\'' {
			// single-quoted strings are pink italic
			return syntaxSingleQuoteStyle.Render(value)
		} else if value != "" {
			// Default to pink for any unmatched value
			return syntaxDefaultStyle.Render(value)
		}
	}
	return value
}

// HighlightYAMLListItem handles list item highlighting for YAML
func HighlightYAMLListItem(line string) (string, bool) {
	if matches := yamlListItemRegex.FindStringSubmatch(line); matches != nil {
		// apply highlighting to the list item value
		itemValue := matches[3]
		coloredItem := HighlightYAMLValue(itemValue)
		return matches[1] + syntaxDashStyle.Render(matches[2]) + coloredItem, true
	}
	return "", false
}

// HighlightJSONLine handles JSON syntax highlighting
func HighlightJSONLine(line string) string {
	processed := false
	originalLine := line

	line = jsonKeyRegex.ReplaceAllStringFunc(line, func(match string) string {
		processed = true
		// check if it's $ref
		if strings.Contains(match, "$ref") {
			return syntaxRefStyle.Render(match)
		}
		return syntaxKeyStyle.Render(match)
	})

	line = jsonStringRegex.ReplaceAllStringFunc(line, func(match string) string {
		processed = true
		parts := strings.SplitN(match, "\"", 2)
		if len(parts) > 1 {
			return parts[0] + syntaxStringStyle.Render("\""+parts[1])
		}
		return match
	})

	// default pink for any remaining strings
	if !processed && line != "" {
		return syntaxDefaultStyle.Render(originalLine)
	}

	return line
}

// ApplySyntaxHighlightingToLine applies syntax highlighting to a single line
func ApplySyntaxHighlightingToLine(line string, isYAML bool) string {
	InitSyntaxStyles()

	// empty line, skip.
	if line == "" {
		return line
	}

	if isYAML {
		if result, handled := HighlightYAMLRefLine(line); handled {
			return result
		}
		if result, handled := HighlightYAMLComment(line, isYAML); handled {
			return result
		}
		if result, handled := HighlightYAMLKeyValue(line); handled {
			return result
		}
		if result, handled := HighlightYAMLListItem(line); handled {
			return result
		}
		return syntaxDefaultStyle.Render(line)
	} else {
		return HighlightJSONLine(line)
	}
}
