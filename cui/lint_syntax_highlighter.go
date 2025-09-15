// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
	"strings"
)

// HighlightYAMLRefLine handles special highlighting for $ref lines
func HighlightYAMLRefLine(line string) (string, bool) {
	if strings.Contains(line, "$ref:") {
		if idx := strings.Index(line, "$ref:"); idx >= 0 {
			// Build the styled line using proper string building for Windows compatibility
			var result strings.Builder
			result.WriteString(line[:idx]) // before ref
			result.WriteString(syntaxRefStyle.Render(line[idx:])) // ref part
			return result.String(), true
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
			// Build the styled line using proper string building for Windows compatibility
			var result strings.Builder
			result.WriteString(ApplySyntaxHighlightingToLine(line[:commentIndex], isYAML)) // before comment
			result.WriteString(syntaxCommentStyle.Render(line[commentIndex:])) // comment
			return result.String(), true
		}
	}
	return "", false
}

// HighlightYAMLKeyValue handles key-value pair highlighting for YAML
func HighlightYAMLKeyValue(line string) (string, bool) {
	if matches := YamlKeyValueRegex.FindStringSubmatch(line); matches != nil {
		indent := matches[1]
		key := matches[2]
		separator := matches[3]
		value := matches[4]

		// Build the styled line using proper string building for Windows compatibility
		var result strings.Builder
		result.WriteString(indent)

		// special handling for $ref key
		if key == "$ref" {
			result.WriteString(syntaxRefStyle.Render(key))
			result.WriteString(separator)
			result.WriteString(syntaxRefStyle.Render(value))
		} else {
			result.WriteString(syntaxKeyStyle.Render(key))
			result.WriteString(separator)
			result.WriteString(HighlightYAMLValue(value))
		}

		return result.String(), true
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
		if NumberValueRegex.MatchString(trimmedValue) {
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
	if matches := YamlListItemRegex.FindStringSubmatch(line); matches != nil {
		// Build the styled line using proper string building for Windows compatibility
		var result strings.Builder
		result.WriteString(matches[1]) // indent
		result.WriteString(syntaxDashStyle.Render(matches[2])) // dash
		result.WriteString(HighlightYAMLValue(matches[3])) // item value
		return result.String(), true
	}
	return "", false
}

// HighlightJSONLine handles JSON syntax highlighting
func HighlightJSONLine(line string) string {
	processed := false
	originalLine := line

	// handle $ref lines specially - highlight only the $ref part
	if strings.Contains(line, "\"$ref\"") {
		refIndex := strings.Index(line, "\"$ref\"")
		if refIndex >= 0 {
			// find the end of the $ref value (next quote after the colon)
			afterRefStart := refIndex + 6 // length of "$ref"
			colonIndex := strings.Index(line[afterRefStart:], ":")
			if colonIndex >= 0 {
				valueStart := afterRefStart + colonIndex + 1
				// find the closing quote
				quoteStart := strings.Index(line[valueStart:], "\"")
				if quoteStart >= 0 {
					quoteEnd := strings.Index(line[valueStart+quoteStart+1:], "\"")
					if quoteEnd >= 0 {
						refEnd := valueStart + quoteStart + quoteEnd + 2
						// Build the styled line using proper string building for Windows compatibility
						var result strings.Builder
						result.WriteString(line[:refIndex]) // before ref
						result.WriteString(syntaxRefStyle.Render(line[refIndex:refEnd])) // ref part
						result.WriteString(line[refEnd:]) // after ref
						return result.String()
					}
				}
			}
		}
		// fallback: highlight the key only if we can't parse the value
		line = strings.ReplaceAll(line, "\"$ref\"", syntaxRefStyle.Render("\"$ref\""))
		processed = true
	}

	line = JsonKeyRegex.ReplaceAllStringFunc(line, func(match string) string {
		processed = true
		return syntaxKeyStyle.Render(match)
	})

	line = JsonStringRegex.ReplaceAllStringFunc(line, func(match string) string {
		processed = true
		parts := strings.SplitN(match, "\"", 2)
		if len(parts) > 1 {
			// Build styled string properly for Windows compatibility
			var result strings.Builder
			result.WriteString(parts[0])
			result.WriteString(syntaxStringStyle.Render("\"" + parts[1]))
			return result.String()
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
