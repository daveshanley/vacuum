// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss/v2"
)

// Log level constants
const (
	LogLevelError = "ERROR"
	LogLevelWarn  = "WARN"
	LogLevelInfo  = "INFO"
	LogLevelDebug = "DEBUG"
)

type ColorizeMode int

const (
	ColorizeDefault ColorizeMode = iota
	ColorizePrimarySolid
	ColorizeSecondary
	ColorizeSecondarySolid
	ColorizeSubtlePrimary
	ColorizeSubtleSecondary
)

var (
	ASCIIRed             = "\033[38;5;196m"
	ASCIIGrey            = "\033[38;5;246m"
	ASCIIPink            = "\033[38;5;201m" // Bright neon pink to match RGBPink
	ASCIIMutedPink       = "\033[38;5;164m" // Muted pink for selected table rows
	ASCIILightGrey       = "\033[38;5;253m"
	ASCIIBlue            = "\033[38;5;45m"
	ASCIIYellow          = "\033[38;5;220m"
	ASCIIGreen           = "\033[38;5;46m"
	ASCIIGreenBold       = "\033[1;38;5;46m"
	ASCIILightGreyItalic = "\033[3;38;5;251m"
	ASCIIBold            = "\033[1m"
	ASCIIItalic          = "\033[3m"
	ASCIIReset           = "\033[0m"
	ASCIIBlueBoldItalic  = "\033[1;3;38;5;45m"        // Blue text with bold and italic
	ASCIIWarn            = "\033[48;5;220;1;38;5;0m"  // Yellow background with bold black text
	ASCIIError           = "\033[48;5;196;1;38;5;15m" // Red background with bold white text
	ASCIIInfo            = "\033[48;5;45;1;38;5;0m"   // Blue background with bold black text

	// Regex pattern to match text between single quotes with proper boundaries
	// Matches 'text' when preceded by space, start of string, or [ and followed by space, end of string, or ]
	QuotedTextPattern = regexp.MustCompile(`(?:^|\s|\[)'([^']+)'(?:\s|\]|$)`)

	// Store original values for restoration
	origRed             = "\033[38;5;196m"
	origGrey            = "\033[38;5;246m"
	origPink            = "\033[38;5;201m"
	origMutedPink       = "\033[38;5;164m"
	origLightGrey       = "\033[38;5;253m"
	origBlue            = "\033[38;5;45m"
	origYellow          = "\033[38;5;220m"
	origGreen           = "\033[38;5;46m"
	origLightGreyItalic = "\033[3;38;5;251m"
	origBold            = "\033[1m"
	origItalic          = "\033[3m"
	origReset           = "\033[0m"
	origBlueBoldItalic  = "\033[1;3;38;5;45m"
	origWarn            = "\033[48;5;220;1;38;5;0m"
	origError           = "\033[48;5;196;1;38;5;15m"
	origInfo            = "\033[48;5;45;1;38;5;0m"

	// Store original lipgloss colors
	origRGBBlue       = lipgloss.Color("45")
	origRGBPink       = lipgloss.Color("201")
	origRGBRed        = lipgloss.Color("196")
	origRGBDarkRed    = lipgloss.Color("124")
	origRGBYellow     = lipgloss.Color("220")
	origRGBDarkYellow = lipgloss.Color("172")
	origRGBGreen      = lipgloss.Color("46")
	origRGBDarkGreen  = lipgloss.Color("22")
	origRGBDarkBlue   = lipgloss.Color("24")
	origRGBOrange     = lipgloss.Color("208")
	origRGBPurple     = lipgloss.Color("135")
	origRGBGrey       = lipgloss.Color("246")
	origRGBDarkGrey   = lipgloss.Color("236")
	origRGBWhite      = lipgloss.Color("255")
	origRGBBlack      = lipgloss.Color("16")
	origRGBSubtleBlue = lipgloss.Color("#1a3a5a")
	origRGBSubtlePink = lipgloss.Color("#2a1a2a")
	origRGBLightGrey  = lipgloss.Color("253")
	origRGBMutedPink  = lipgloss.Color("164")

	// Current lipgloss colors (may be modified)
	RGBBlue       = origRGBBlue
	RGBPink       = origRGBPink
	RGBRed        = origRGBRed
	RGBDarkRed    = origRGBDarkRed
	RBGYellow     = origRGBYellow
	RGBDarkYellow = origRGBDarkYellow
	RGBGreen      = origRGBGreen
	RGBDarkGreen  = origRGBDarkGreen
	RGBDarkBlue   = origRGBDarkBlue
	RGBOrange     = origRGBOrange
	RGBPurple     = origRGBPurple
	RGBGrey       = origRGBGrey
	RGBDarkGrey   = origRGBDarkGrey
	RGBWhite      = origRGBWhite
	RGBBlack      = origRGBBlack
	RGBSubtleBlue = origRGBSubtleBlue
	RGBSubtlePink = origRGBSubtlePink
	RGBLightGrey  = origRGBLightGrey
	RGBMutedPink  = origRGBMutedPink
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func uintPtr(u uint) *uint    { return &u }

// Color constants for Glamour styles (as string pointers)
var (
	// ANSI 256 colors for general text
	ColorBlue       = strPtr("45")
	ColorSoftBlue   = strPtr("117")
	ColorBlueBg     = strPtr("#002329")
	ColorPink       = strPtr("201")
	ColorPinkBg     = strPtr("#2a1a2a")
	ColorRed        = strPtr("196")
	ColorYellow     = strPtr("220")
	ColorSoftYellow = strPtr("226")
	ColorGreen      = strPtr("46")
	ColorGrey       = strPtr("246")
	ColorDarkGrey   = strPtr("236")
	ColorLightGrey  = strPtr("253")
	ColorLightPink  = strPtr("164")

	// chroma specifc colors (non ANSI256)
	ChromaBlue      = strPtr("#00d7ff")
	ChromaPink      = strPtr("#ff5fff")
	ChromaRed       = strPtr("#ff0000")
	ChromaYellow    = strPtr("#ffd700")
	ChromaGreen     = strPtr("#00ff00")
	ChromaGrey      = strPtr("#8a8a8a")
	ChromaLightPink = strPtr("#d75fd7")
)

// ColorizeString highlights backtick-enclosed text with the specified style
func ColorizeString(text string, mode ColorizeMode) string {
	var style lipgloss.Style
	switch mode {
	case ColorizeDefault:
		style = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true)
	case ColorizePrimarySolid:
		style = lipgloss.NewStyle().Background(RGBBlue).Foreground(RGBBlack).Bold(true)
	case ColorizeSecondary:
		style = lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
	case ColorizeSecondarySolid:
		style = lipgloss.NewStyle().Background(RGBPink).Foreground(RGBBlack).Bold(true)
	case ColorizeSubtlePrimary:
		style = lipgloss.NewStyle().Background(RGBSubtleBlue).Foreground(RGBBlue).Bold(true)
	case ColorizeSubtleSecondary:
		style = lipgloss.NewStyle().Background(RGBSubtlePink).Foreground(RGBPink).Bold(true)
	}

	// find and replace backtick-enclosed text
	var result strings.Builder
	inBackticks := false
	backtickStart := 0

	for i, char := range text {
		if char == '`' {
			if !inBackticks {
				inBackticks = true
				backtickStart = i + 1
			} else {

				if i > backtickStart {
					content := text[backtickStart:i]
					result.WriteString(style.Render(content))
				}
				inBackticks = false
			}
		} else if !inBackticks {
			result.WriteRune(char)
		}
	}

	// handle unclosed backtick (treat rest as normal text)
	if inBackticks && backtickStart < len(text) {
		result.WriteString("`")
		result.WriteString(text[backtickStart:])
	}

	return result.String()
}

// ColorizeMessage formats a message string with inline code highlighting.
// Text between backticks (`code`) or backtick with truncation (`code...)
// will be displayed in blue with bold and italic styling, keeping the backticks.
func ColorizeMessage(message string) string {
	if !strings.Contains(message, "`") {
		return message // No backticks, return as-is
	}

	// Use regex to find and colorize backtick-enclosed text
	return BacktickRegex.ReplaceAllStringFunc(message, func(match string) string {
		// Extract the content between backticks
		content := BacktickRegex.FindStringSubmatch(match)
		if len(content) > 1 {
			// Apply blue bold italic styling to backtick AND content using lipgloss
			return "`" + StyleCodeHighlight.Render(content[1]) + "`"
		}
		return match
	})
}

func ColorizeLogMessage(message, severity string) string {
	if severity == LogLevelError {
		return StyleLogError.Render(message)
	}
	if severity == LogLevelDebug {
		return StyleLogDebug.Render(message)
	}

	if !strings.HasPrefix(message, "[") {
		return message
	}

	// Use regex to find and colorize backtick-enclosed text
	res := LogPrefixRegex.ReplaceAllStringFunc(message, func(match string) string {
		// Extract the content between backticks
		content := LogPrefixRegex.FindStringSubmatch(match)
		if len(content) > 1 {
			// Apply styling using lipgloss
			switch severity {
			case LogLevelError:
				return StyleLogError.Copy().Italic(true).Render("[" + content[1] + "]")
			case LogLevelWarn:
				return StyleLogWarn.Copy().Render("[" + content[1] + "]")
			case LogLevelInfo:
				return StyleLogInfo.Copy().Render("[" + content[1] + "]")
			case LogLevelDebug:
				return StyleLogDebug.Copy().Render("[" + content[1] + "]")
			}

		}
		return match
	})
	return res
}

// VisibleLength calculates the visible length of a string, excluding ANSI escape codes.
func VisibleLength(s string) int {
	// Remove all ANSI escape sequences to get the actual visible length
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	clean := ansiRegex.ReplaceAllString(s, "")
	// Count runes instead of bytes for accurate character count
	return len([]rune(clean))
}

// ColorizeLocation formats a file location string with color codes for a terminal.
func ColorizeLocation(location string) string {
	// Expected format: path/to/file.ext:line:col
	// We need to parse and colorize each part
	if !strings.Contains(location, ":") {
		return location // Not a location format we recognize
	}

	// Split into file path and line:col
	lastColon := strings.LastIndex(location, ":")
	secondLastColon := strings.LastIndex(location[:lastColon], ":")

	if secondLastColon == -1 {
		return location // Not enough colons for file:line:col format
	}

	filePath := location[:secondLastColon]
	lineNum := location[secondLastColon+1 : lastColon]
	colNum := location[lastColon+1:]

	file := filepath.Base(filePath)
	dir := filepath.Dir(filePath)

	// Build colored path using lipgloss styles
	var result strings.Builder
	if dir != "." {
		result.WriteString(StyleDirectoryGrey.Render(dir))
		result.WriteString("/")
	}
	result.WriteString(StyleFileItalic.Render(file))
	result.WriteString(StyleLocationSeparator.Render(":"))
	result.WriteString(StyleLineNumber.Render(lineNum))
	result.WriteString(StyleLocationSeparator.Render(":"))
	result.WriteString(StyleColumnNumber.Render(colNum))

	return result.String()
}

// ColorizeTableOutput adds ASCII color codes to a table output string based on the
// cursor position and content patterns.
func ColorizeTableOutput(tableView string, cursor int, rows []table.Row) string {
	lines := strings.Split(tableView, "\n")

	var selectedLocation string
	if cursor >= 0 && cursor < len(rows) {
		selectedLocation = rows[cursor][0]
	}

	var result strings.Builder
	for i, line := range lines {
		isSelectedLine := selectedLocation != "" && strings.Contains(line, selectedLocation)

		if isSelectedLine && i > 0 {
			line = StyleSelectedRow.Render(line)
		}

		if i >= 1 && !isSelectedLine {

			// location
			if LocationRegex.MatchString(line) {
				line = LocationRegex.ReplaceAllStringFunc(line, func(match string) string {
					parts := LocationRegex.FindStringSubmatch(match)
					if len(parts) == 4 {
						location := fmt.Sprintf("%s:%s:%s", parts[1], parts[2], parts[3])
						return ColorizeLocation(location)
					}
					return match
				})
			}

			// message
			for _, row := range rows {
				if len(row) > 2 && row[2] != "" && strings.Contains(line, row[2]) {
					// Use the actual ColorizeMessage function
					colorizedMsg := ColorizeMessage(row[2])
					line = strings.Replace(line, row[2], colorizedMsg, 1)
					break
				}
			}

			// path - handle both JSON paths and circular references
			// also check the actual path column from rows if available
			for _, row := range rows {
				if len(row) > 5 && row[5] != "" && strings.Contains(line, row[5]) {
					colorizedPath := ColorizePath(row[5])
					line = strings.Replace(line, row[5], colorizedPath, 1)
					break
				}
			}

			// handle inline paths that might appear in messages
			if JsonPathRegex.MatchString(line) {
				line = JsonPathRegex.ReplaceAllStringFunc(line, func(match string) string {
					return ColorizePath(match)
				})
			}

			// check for circular references (with arrows)
			if CircularRefRegex.MatchString(line) {
				line = CircularRefRegex.ReplaceAllStringFunc(line, func(match string) string {
					return ColorizePath(match)
				})
			}

			// severity - replace with lipgloss styles
			line = strings.Replace(line, "✗ error",
				StyleSeverityError.Render("✗ error"), -1)
			line = strings.Replace(line, "▲ warning",
				StyleSeverityWarning.Render("▲ warning"), -1)
			line = strings.Replace(line, "● info",
				StyleSeverityInfo.Render("● info"), -1)
		}

		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// ColorizeLogEntry applies formatting to log entries, highlighting quoted text
func ColorizeLogEntry(log, color string) string {
	if QuotedTextPattern != nil {
		return QuotedTextPattern.ReplaceAllStringFunc(log, func(match string) string {
			// extract the content between quotes
			content := QuotedTextPattern.FindStringSubmatch(match)
			if len(content) > 1 {
				// preserve the leading/trailing space or bracket
				prefix := ""
				suffix := ""
				if strings.HasPrefix(match, " ") {
					prefix = " "
				} else if strings.HasPrefix(match, "[") {
					prefix = "["
				}
				if strings.HasSuffix(match, " ") {
					suffix = " "
				} else if strings.HasSuffix(match, "]") {
					suffix = "]"
				}
				return prefix + StyleQuotedText.Render("'"+content[1]+"'") + color + suffix
			}
			return match
		})
	}
	return log
}

// ColorizePath formats a JSON/YAML path string with inline quote highlighting and circular reference detection.
func ColorizePath(path string) string {
	// Handle circular references first
	if CircularRefRegex.MatchString(path) {
		path = CircularRefRegex.ReplaceAllStringFunc(path, func(match string) string {
			var result strings.Builder
			parts := PartRegex.FindAllStringSubmatch(match, -1)
			for _, part := range parts {
				if part[1] != "" {
					// ref - use lipgloss style
					result.WriteString(StylePathRef.Render(part[1]))
				} else if part[2] != "" {
					// arrow - use lipgloss style
					result.WriteString(StylePathArrow.Render(part[2]))
				}
			}
			return result.String()
		})
	}

	// Handle quoted content
	if strings.Contains(path, "'") {
		var result strings.Builder
		lastIdx := 0

		// Find all single-quoted sections
		matches := SingleQuoteRegex.FindAllStringSubmatchIndex(path, -1)
		for _, match := range matches {
			// Add content before the quote
			result.WriteString(StylePathGrey.Render(path[lastIdx:match[0]]))
			// Add the quoted content
			if match[3] > match[2] {
				quotedText := "'" + path[match[2]:match[3]] + "'"
				result.WriteString(StylePathQuoted.Render(quotedText))
			}
			lastIdx = match[1]
		}
		// Add any remaining content
		if lastIdx < len(path) {
			result.WriteString(StylePathGrey.Render(path[lastIdx:]))
		}
		return result.String()
	}

	// handle unclosed quotes with truncation (e.g., 'text...)
	truncatedQuoteRegex := regexp.MustCompile(`'([^']+\.\.\.?)$`)
	if truncatedQuoteRegex.MatchString(path) {
		idx := strings.LastIndex(path, "'")
		if idx >= 0 {
			var result strings.Builder
			result.WriteString(StylePathGrey.Render(path[:idx]))
			result.WriteString(StylePathQuoted.Render(path[idx:]))
			return result.String()
		}
	}

	// The entire path should be wrapped in grey
	return StylePathGrey.Render(path)
}

// ApplyLintDetailsTableStyles applies custom styles to a table.Model for lint details display
func ApplyLintDetailsTableStyles(t *table.Model) {
	s := table.DefaultStyles()

	s.Header = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink).
		BorderBottom(true).
		BorderLeft(false).
		BorderRight(false).
		BorderTop(false).
		Foreground(RGBPink).
		Bold(true).
		Padding(0, 1)

	s.Selected = lipgloss.NewStyle().Bold(true).
		Foreground(RGBPink).
		Background(RGBSubtlePink).
		Padding(0, 0)

	s.Cell = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink).
		BorderRight(false).
		Padding(0, 1)

	t.SetStyles(s)
}

// CreatePb33fDocsStyle creates a custom Glamour style for documentation rendering
// using the existing princess beef heavy industries color scheme.
func CreatePb33fDocsStyle(termWidth int) ansi.StyleConfig {

	truePointer := boolPtr(true)
	falsePointer := boolPtr(false)

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       ColorPink,
				Bold:        truePointer,
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix:     "\n",
				BackgroundColor: ColorBlueBg,
				Prefix:          fmt.Sprintf("%s\n \u2605 ", strings.Repeat("", termWidth)),
				Suffix:          fmt.Sprintf("\n%s\n", strings.Repeat("", termWidth)),
				Color:           ColorBlue,
				Bold:            truePointer,
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix:     "\n",
				BackgroundColor: ColorBlueBg,
				Prefix:          fmt.Sprintf("%s\n \u2605 ", strings.Repeat("", termWidth)),
				Suffix:          fmt.Sprintf("\n%s\n", strings.Repeat("", termWidth)),
				Color:           ColorBlue,
				Bold:            truePointer,
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ColorBlue,
				Bold:  truePointer,
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ColorPink,
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: ColorPink,
			},
		},
		Emph: ansi.StylePrimitive{
			Color:  ColorPink,
			Italic: truePointer,
		},
		Strong: ansi.StylePrimitive{
			Color:           ColorPink,
			BackgroundColor: ColorPinkBg,
			Bold:            truePointer,
			Underline:       truePointer,
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "[",
				Suffix:          "]",
				Bold:            truePointer,
				Color:           ColorGreen,
				BackgroundColor: ColorDarkGrey,
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					BackgroundColor: ColorPinkBg,
					Color:           ColorLightGrey,
				},
				Margin: uintPtr(1),
			},
			Theme: "monokai",
			Chroma: &ansi.Chroma{
				Keyword: ansi.StylePrimitive{
					Color: ChromaBlue,
					Bold:  falsePointer,
				},
				Text: ansi.StylePrimitive{
					Color: ChromaPink,
					Bold:  truePointer,
				},
				LiteralString: ansi.StylePrimitive{
					Color: ChromaGreen,
				},
				LiteralNumber: ansi.StylePrimitive{
					Color: ChromaPink,
				},
				Comment: ansi.StylePrimitive{
					Color:  ChromaGrey,
					Italic: truePointer,
				},
				NameFunction: ansi.StylePrimitive{
					Color: ChromaGreen,
				},
				NameTag: ansi.StylePrimitive{
					Color: ChromaBlue,
					Bold:  falsePointer,
				},
				NameAttribute: ansi.StylePrimitive{
					Color: ChromaGreen,
				},
				Operator: ansi.StylePrimitive{
					Color: ChromaYellow,
				},
				Punctuation: ansi.StylePrimitive{
					Color: ChromaGrey,
				},
				NameBuiltin: ansi.StylePrimitive{
					Color: ChromaBlue,
				},
				NameClass: ansi.StylePrimitive{
					Color: ChromaGreen,
					Bold:  truePointer,
				},
				NameConstant: ansi.StylePrimitive{
					Color: ChromaLightPink,
				},
			},
		},
		Link: ansi.StylePrimitive{
			Color:     ColorSoftBlue,
			Underline: truePointer,
		},
		LinkText: ansi.StylePrimitive{
			Color:  ColorBlue,
			Prefix: "[",
			Suffix: "]",
			Bold:   truePointer,
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: ColorPink,
				},
				Indent: uintPtr(2),
			},
			LevelIndent: 2,
		},
		Item: ansi.StylePrimitive{
			Prefix: "> ",
			Color:  ColorBlue,
		},
		Enumeration: ansi.StylePrimitive{
			Color: ColorBlue,
		},

		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  ColorGrey,
				Italic: truePointer,
			},
			Indent:      uintPtr(1),
			IndentToken: strPtr("│ "),
		},

		HorizontalRule: ansi.StylePrimitive{
			Color:  ColorPink,
			Format: fmt.Sprintf("\n%s\n", strings.Repeat("-", termWidth)),
		},

		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{},
			},
			CenterSeparator: strPtr("┼"),
			ColumnSeparator: strPtr("│"),
			RowSeparator:    strPtr("─"),
		},

		Strikethrough: ansi.StylePrimitive{
			CrossedOut: truePointer,
			Color:      ColorGrey,
		},

		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{},
			Ticked:         "✓ ",
			Unticked:       "☐ ",
		},

		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
			Margin:         uintPtr(1),
		},

		DefinitionTerm: ansi.StylePrimitive{
			Color: ColorPink,
			Bold:  truePointer,
		},
		DefinitionDescription: ansi.StylePrimitive{
			Color: ColorLightGrey,
		},
	}
}

// colorsDisabled tracks whether colors are currently disabled
var colorsDisabled = false

// DisableColors sets all ANSI color codes to empty strings for monochrome output
func DisableColors() {
	colorsDisabled = true
	ASCIIRed = ""
	ASCIIGrey = ""
	ASCIIPink = ""
	ASCIIMutedPink = ""
	ASCIILightGrey = ""
	ASCIIBlue = ""
	ASCIIYellow = ""
	ASCIIGreen = ""
	ASCIILightGreyItalic = ""
	ASCIIBold = ""
	ASCIIItalic = ""
	ASCIIReset = ""
	ASCIIBlueBoldItalic = ""
	ASCIIWarn = ""
	ASCIIError = ""
	ASCIIInfo = ""

	// Also disable lipgloss colors - use NoColor for most, dark grey for backgrounds
	RGBBlue = lipgloss.NoColor{}
	RGBPink = lipgloss.NoColor{}
	RGBRed = lipgloss.NoColor{}
	RGBDarkRed = lipgloss.Color("238")
	RBGYellow = lipgloss.NoColor{}
	RGBDarkYellow = lipgloss.Color("238")
	RGBGreen = lipgloss.NoColor{}
	RGBDarkGreen = lipgloss.Color("238")
	RGBDarkBlue = lipgloss.Color("238")
	RGBOrange = lipgloss.NoColor{}
	RGBPurple = lipgloss.NoColor{}
	RGBGrey = lipgloss.NoColor{}
	RGBDarkGrey = lipgloss.Color("238")
	RGBWhite = lipgloss.NoColor{}
	RGBBlack = lipgloss.NoColor{}
	RGBSubtleBlue = lipgloss.Color("238")
	RGBSubtlePink = lipgloss.Color("238")
	RGBLightGrey = lipgloss.NoColor{}
	RGBMutedPink = lipgloss.NoColor{}
}

// AreColorsDisabled returns true if colors are currently disabled
func AreColorsDisabled() bool {
	return colorsDisabled
}

// EnableColors restores all ANSI color codes to their original values
func EnableColors() {
	colorsDisabled = false
	ASCIIRed = origRed
	ASCIIGrey = origGrey
	ASCIIPink = origPink
	ASCIIMutedPink = origMutedPink
	ASCIILightGrey = origLightGrey
	ASCIIBlue = origBlue
	ASCIIYellow = origYellow
	ASCIIGreen = origGreen
	ASCIILightGreyItalic = origLightGreyItalic
	ASCIIBold = origBold
	ASCIIItalic = origItalic
	ASCIIReset = origReset
	ASCIIBlueBoldItalic = origBlueBoldItalic
	ASCIIWarn = origWarn
	ASCIIError = origError
	ASCIIInfo = origInfo

	// Restore lipgloss colors
	RGBBlue = origRGBBlue
	RGBPink = origRGBPink
	RGBRed = origRGBRed
	RGBDarkRed = origRGBDarkRed
	RBGYellow = origRGBYellow
	RGBDarkYellow = origRGBDarkYellow
	RGBGreen = origRGBGreen
	RGBDarkGreen = origRGBDarkGreen
	RGBDarkBlue = origRGBDarkBlue
	RGBOrange = origRGBOrange
	RGBPurple = origRGBPurple
	RGBGrey = origRGBGrey
	RGBDarkGrey = origRGBDarkGrey
	RGBWhite = origRGBWhite
	RGBBlack = origRGBBlack
	RGBSubtleBlue = origRGBSubtleBlue
	RGBSubtlePink = origRGBSubtlePink
	RGBLightGrey = origRGBLightGrey
	RGBMutedPink = origRGBMutedPink
}
