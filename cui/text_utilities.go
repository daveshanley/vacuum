// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/model"
)

// SeverityInfo contains all the display information for a severity level
type SeverityInfo struct {
	Icon      string
	Text      string
	IconStyle lipgloss.Style
	TextStyle lipgloss.Style
}

// GetSeverityInfo returns display information for a given severity
func GetSeverityInfo(severity string) SeverityInfo {
	switch severity {
	case model.SeverityError:
		return SeverityInfo{
			Icon:      "✗",
			Text:      "✗ error",
			IconStyle: lipgloss.NewStyle().Foreground(color.RGBRed).Bold(true),
			TextStyle: lipgloss.NewStyle().Foreground(color.RGBRed),
		}
	case model.SeverityWarn:
		return SeverityInfo{
			Icon:      "▲",
			Text:      "▲ warning",
			IconStyle: lipgloss.NewStyle().Foreground(color.RBGYellow).Bold(true),
			TextStyle: lipgloss.NewStyle().Foreground(color.RBGYellow),
		}
	default: // model.SeverityInfo and others
		return SeverityInfo{
			Icon:      "●",
			Text:      "● info",
			IconStyle: lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true),
			TextStyle: lipgloss.NewStyle().Foreground(color.RGBBlue),
		}
	}
}

// GetSeverityInfoFromText returns display information for severity text like "✗ error"
func GetSeverityInfoFromText(severityText string) SeverityInfo {
	switch severityText {
	case "✗ error":
		return SeverityInfo{
			Icon:      "✗",
			Text:      severityText,
			IconStyle: lipgloss.NewStyle().Foreground(color.RGBRed).Bold(true),
			TextStyle: lipgloss.NewStyle().Foreground(color.RGBRed),
		}
	case "▲ warning":
		return SeverityInfo{
			Icon:      "▲",
			Text:      severityText,
			IconStyle: lipgloss.NewStyle().Foreground(color.RBGYellow).Bold(true),
			TextStyle: lipgloss.NewStyle().Foreground(color.RBGYellow),
		}
	case "● info":
		return SeverityInfo{
			Icon:      "●",
			Text:      severityText,
			IconStyle: lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true),
			TextStyle: lipgloss.NewStyle().Foreground(color.RGBBlue),
		}
	default:
		return SeverityInfo{
			Icon:      "●",
			Text:      "● info",
			IconStyle: lipgloss.NewStyle().Foreground(color.RGBGrey).Bold(true),
			TextStyle: lipgloss.NewStyle().Foreground(color.RGBGrey),
		}
	}
}

func getRuleSeverity(r *model.RuleFunctionResult) string {
	if r == nil || r.Rule == nil {
		return "✗ error"
	}
	info := GetSeverityInfo(r.Rule.Severity)
	return info.Text
}

func getLintingFilterName(state FilterState) string {
	switch state {
	case FilterAll:
		return "All"
	case FilterErrors:
		return "Errors"
	case FilterWarnings:
		return "Warnings"
	case FilterInfo:
		return "Info"
	default:
		return "All"
	}
}

func extractCategories(results []*model.RuleFunctionResult) []string {
	// use struct{} instead of bool to save memory
	categoryMap := make(map[string]struct{})
	for _, r := range results {
		if r.Rule != nil && r.Rule.RuleCategory != nil {
			categoryMap[r.Rule.RuleCategory.Name] = struct{}{}
		}
	}

	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}

	// use sort.Strings for better performance than bubble sort
	sort.Strings(categories)

	return categories
}

func extractRules(results []*model.RuleFunctionResult) []string {
	// use struct{} instead of bool to save memory
	ruleMap := make(map[string]struct{})
	for _, r := range results {
		if r.Rule != nil && r.Rule.Id != "" {
			ruleMap[r.Rule.Id] = struct{}{}
		}
	}

	rules := make([]string, 0, len(ruleMap))
	for rule := range ruleMap {
		rules = append(rules, rule)
	}

	// use sort.Strings for better performance than bubble sort
	sort.Strings(rules)

	return rules
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) <= width {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

func renderEmptyState(width, height int) string {
	art := []string{
		"",
		" _|      _|     _|_|     _|_|_|_|_|   _|    _|   _|_|_|   _|      _|     _|_|_|  ",
		" _|_|    _|   _|    _|       _|       _|    _|     _|     _|_|    _|   _|        ",
		" _|  _|  _|   _|    _|       _|       _|_|_|_|     _|     _|  _|  _|   _|  _|_|  ",
		" _|    _|_|   _|    _|       _|       _|    _|     _|     _|    _|_|   _|    _|  ",
		" _|      _|     _|_|         _|       _|    _|   _|_|_|   _|      _|     _|_|_|  ",
		"",
		" _|    _|   _|_|_|_|   _|_|_|     _|_|_|_|  ",
		" _|    _|   _|         _|    _|   _|        ",
		" _|_|_|_|   _|_|_|     _|_|_|     _|_|_|    ",
		" _|    _|   _|         _|    _|   _|        ",
		" _|    _|   _|_|_|_|   _|    _|   _|_|_|_|  ",
		"",
		" Nothing to vacuum, the filters are too strict.",
		"",
		" To adjust them:",
		"",
		" > tab - cycle severity",
		" > c   - cycle categories",
		" > r   - cycle rules",
		" > esc - clear all filters",
	}

	artStr := strings.Join(art, "\n")

	maxLineWidth := 82 // width of the longest line in the art
	leftPadding := (width - maxLineWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	// add left padding to each line to center the entire block
	artLines := strings.Split(artStr, "\n")
	paddedLines := make([]string, len(artLines))
	padding := strings.Repeat(" ", leftPadding)
	for i, line := range artLines {
		if line != "" {
			paddedLines[i] = padding + line
		} else {
			paddedLines[i] = ""
		}
	}

	// calculate vertical centering
	totalLines := len(paddedLines)
	topPadding := (height - totalLines) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// build the result to exactly fill the height
	var resultLines []string
	for i := 0; i < topPadding; i++ {
		resultLines = append(resultLines, "")
	}

	// content
	resultLines = append(resultLines, paddedLines...)

	// bottom padding to exactly fill the height
	for len(resultLines) < height {
		resultLines = append(resultLines, "")
	}

	// ensure we don't exceed the height
	if len(resultLines) > height {
		resultLines = resultLines[:height]
	}

	textStyle := lipgloss.NewStyle().
		Foreground(color.RGBRed).
		Width(width)
	return textStyle.Render(strings.Join(resultLines, "\n"))
}

func addTableBorders(tableView string) string {
	tableStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(color.RGBPink).
		PaddingTop(0)

	return tableStyle.Render(tableView)
}

func formatFileLocation(r *model.RuleFunctionResult, fileName string) string {
	startLine := 0
	startCol := 0
	// Normalize the fileName path separators for the current OS
	f := filepath.FromSlash(fileName)

	if r != nil {
		if r.StartNode != nil {
			startLine = r.StartNode.Line
			startCol = r.StartNode.Column
		}

		if r.Origin != nil {
			// Normalize path separators for the current OS
			f = filepath.FromSlash(r.Origin.AbsoluteLocation)
			startLine = r.Origin.Line
			startCol = r.Origin.Column
		}
	}

	// Make path relative
	if absPath, err := filepath.Abs(f); err == nil {
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, absPath); err == nil {
				f = relPath
			}
		}
	}

	return fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
}
