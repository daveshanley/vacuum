// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
	"strings"
)

// HighlightYAMLRefLine handles special highlighting for $ref lines
func HighlightYAMLRefLine(line string) (string, bool) {
	if strings.Contains(line, "$ref:") {
		idx := strings.Index(line, "$ref:")
		if idx >= 0 {
			// extract parts
			before := line[:idx]
			refKey := "$ref"
			colon := ":"
			afterColon := line[idx+5:] // after "$ref:"

			// build result styling individual tokens
			var result strings.Builder
			result.WriteString(before)
			result.WriteString(syntaxRefStyle.Render(refKey))
			result.WriteString(colon)

			// find and style the value
			trimmed := strings.TrimSpace(afterColon)
			leadingSpaces := afterColon[:len(afterColon)-len(trimmed)]
			result.WriteString(leadingSpaces)

			// style just the value, not trailing content
			if trimmed != "" {
				// find where value ends (at comment or end of line)
				valueEnd := len(trimmed)
				if commentIdx := strings.Index(trimmed, "#"); commentIdx > 0 {
					valueEnd = commentIdx
				}
				value := strings.TrimSpace(trimmed[:valueEnd])
				afterValue := trimmed[valueEnd:]

				result.WriteString(syntaxRefStyle.Render(value))
				result.WriteString(afterValue)
			}

			return result.String(), true
		}
	}

	// check for ref paths - but only style the path, not the whole line
	trimmed := strings.TrimSpace(line)
	if strings.Contains(trimmed, "'#/") || strings.Contains(trimmed, "\"#/") {
		// find the ref path and style just that
		start := strings.Index(line, "'#/")
		if start == -1 {
			start = strings.Index(line, "\"#/")
		}
		if start >= 0 {
			// find the end quote
			quote := line[start]
			end := strings.IndexByte(line[start+1:], quote)
			if end > 0 {
				end += start + 2

				var result strings.Builder
				result.WriteString(line[:start])
				result.WriteString(syntaxRefStyle.Render(line[start:end]))
				result.WriteString(line[end:])
				return result.String(), true
			}
		}
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
	// windows-safe approach: parse and style individual tokens
	if strings.Contains(line, "\":") {
		// find the key
		keyStart := strings.Index(line, "\"")
		if keyStart >= 0 {
			keyEnd := strings.Index(line[keyStart+1:], "\"")
			if keyEnd > 0 {
				keyEnd += keyStart + 1

				// extract parts
				before := line[:keyStart]
				key := line[keyStart:keyEnd+1]
				afterKey := line[keyEnd+1:]

				// check if it's a $ref
				isRef := strings.Contains(key, "$ref")

				// find the colon and value
				colonIndex := strings.Index(afterKey, ":")
				if colonIndex >= 0 {
					beforeValue := afterKey[:colonIndex+1]
					afterColon := afterKey[colonIndex+1:]

					// trim to find value
					trimmed := strings.TrimSpace(afterColon)
					leadingSpaces := afterColon[:len(afterColon)-len(trimmed)]

					// build result
					var result strings.Builder
					result.WriteString(before)

					// style the key
					if isRef {
						result.WriteString(syntaxRefStyle.Render(key))
					} else {
						result.WriteString(syntaxKeyStyle.Render(key))
					}

					result.WriteString(beforeValue)
					result.WriteString(leadingSpaces)

					// style the value if it's a string
					if strings.HasPrefix(trimmed, "\"") {
						endQuote := strings.Index(trimmed[1:], "\"")
						if endQuote > 0 {
							endQuote += 2
							value := trimmed[:endQuote]
							afterValue := trimmed[endQuote:]

							if isRef {
								result.WriteString(syntaxRefStyle.Render(value))
							} else {
								result.WriteString(syntaxStringStyle.Render(value))
							}
							result.WriteString(afterValue)
							return result.String()
						}
					} else if trimmed != "" && !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
						// simple value (number, boolean, etc)
						valueEnd := len(trimmed)
						for i, char := range trimmed {
							if char == ',' || char == ' ' || char == '\t' {
								valueEnd = i
								break
							}
						}

						value := trimmed[:valueEnd]
						afterValue := trimmed[valueEnd:]

						// style based on type
						if value == "true" || value == "false" || value == "null" {
							result.WriteString(syntaxBoolStyle.Render(value))
						} else if NumberValueRegex.MatchString(value) {
							result.WriteString(syntaxNumberStyle.Render(value))
						} else {
							result.WriteString(syntaxDefaultStyle.Render(value))
						}
						result.WriteString(afterValue)
						return result.String()
					}

					// couldn't parse value, just add the rest
					result.WriteString(afterColon)
					return result.String()
				}

				// no colon, just style the key
				var result strings.Builder
				result.WriteString(before)
				if isRef {
					result.WriteString(syntaxRefStyle.Render(key))
				} else {
					result.WriteString(syntaxKeyStyle.Render(key))
				}
				result.WriteString(afterKey)
				return result.String()
			}
		}
	}

	// no key-value pair found
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
