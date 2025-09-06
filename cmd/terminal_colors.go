// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss/v2"
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
	ASCIIPink            = "\033[38;5;164m"
	ASCIILightGrey       = "\033[38;5;253m"
	ASCIIBlue            = "\033[38;5;45m"
	ASCIIYellow          = "\033[38;5;220m"
	ASCIIGreen           = "\033[38;5;46m"
	ASCIILightGreyItalic = "\033[3;38;5;251m"
	ASCIIBold            = "\033[1m"
	ASCIIReset           = "\033[0m"
	RGBBlue              = lipgloss.Color("45")
	RGBPink              = lipgloss.Color("201")
	RGBRed               = lipgloss.Color("196")
	RBGYellow            = lipgloss.Color("220")
	RGBGreen             = lipgloss.Color("46")
	RGBOrange            = lipgloss.Color("208")
	RGBPurple            = lipgloss.Color("135")
	RGBGrey              = lipgloss.Color("246")
	RGBDarkGrey          = lipgloss.Color("236")
	RGBWhite             = lipgloss.Color("255")
	RGBBlack             = lipgloss.Color("16")
	RGBSubtleBlue        = lipgloss.Color("#1a3a5a")
	RGBSubtlePink        = lipgloss.Color("#2a1a2a")
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
			line = ASCIIPink + line + ASCIIReset
		}

		if i >= 1 && !isSelectedLine {

			if locationRegex.MatchString(line) {

				line = locationRegex.ReplaceAllStringFunc(line, func(match string) string {
					parts := locationRegex.FindStringSubmatch(match)
					if len(parts) == 4 {
						filePath := parts[1]
						lineNum := parts[2]
						colNum := parts[3]

						file := filepath.Base(filePath)
						dir := filepath.Dir(filePath)
						filePath = fmt.Sprintf("%s/%s%s%s", dir, ASCIILightGreyItalic, file, ASCIIReset)

						// color parts with ASCII colors.
						coloredPath := fmt.Sprintf("%s%s%s", ASCIIGrey, filePath, ASCIIReset)
						coloredLine := fmt.Sprintf("%s%s%s", ASCIIBold, lineNum, ASCIIReset)
						coloredCol := fmt.Sprintf("%s%s%s", ASCIILightGrey, colNum, ASCIIReset)
						sep := fmt.Sprintf("%s:%s", ASCIILightGrey, ASCIIReset)
						return fmt.Sprintf("%s%s%s%s%s", coloredPath, sep, coloredLine, sep, coloredCol)
					}
					return match
				})
			}

			if jsonPathRegex.MatchString(line) {
				line = jsonPathRegex.ReplaceAllStringFunc(line, func(match string) string {
					return fmt.Sprintf("%s%s%s", ASCIIGrey, match, ASCIIReset)
				})
			}

			if circularRefRegex.MatchString(line) {
				line = circularRefRegex.ReplaceAllStringFunc(line, func(match string) string {
					circResult := ""

					parts := partRegex.FindAllStringSubmatch(match, -1)
					for _, part := range parts {
						if part[1] != "" {
							// ref
							circResult += fmt.Sprintf("%s%s%s", ASCIILightGrey, part[1], ASCIIReset)
						} else if part[2] != "" {
							// arrow
							circResult += fmt.Sprintf("%s%s%s", ASCIIRed, part[2], ASCIIReset)
						}
					}
					return circResult
				})
			}

			line = strings.Replace(line, "✗ error",
				fmt.Sprintf("%s%s%s", ASCIIRed, "✗ error", ASCIIReset), -1)
			line = strings.Replace(line, "▲ warning",
				fmt.Sprintf("%s%s%s", ASCIIYellow, "▲ warning", ASCIIReset), -1)
			line = strings.Replace(line, "● info",
				fmt.Sprintf("%s%s%s", ASCIIBlue, "● info", ASCIIReset), -1)
		}

		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func applyLintDetailsTableStyles(t *table.Model) {
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
