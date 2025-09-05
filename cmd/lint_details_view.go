// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

// BuildDetailsView builds the details view for a violation when a user presses enter or return on a row.
func (m *ViolationResultTableModel) BuildDetailsView() string {
	if m.modalContent == nil {
		return ""
	}

	// Calculate dimensions - FIXED height regardless of terminal size
	splitHeight := splitViewHeight // Fixed compact height with proper padding
	if m.height < 20 {
		return "" // Don't show split view if terminal too small
	}

	splitWidth := m.width // Match terminal width for consistency
	contentHeight := splitContentHeight   // Fixed content height for all columns

	// Column widths: details, how-to-fix, code
	// Adjust for container padding
	innerWidth := splitWidth - 4 // Account for container borders and padding
	detailsWidth := int(float64(innerWidth) * float64(detailsColumnPercent) / 100)
	howToFixWidth := int(float64(innerWidth) * float64(howToFixColumnPercent) / 100)
	codeWidth := innerWidth - detailsWidth - howToFixWidth

	codeSnippet, startLine := m.extractCodeSnippet(m.modalContent, 4)

	JSONPathBarStyle := lipgloss.NewStyle().
		Width(splitWidth-2).
		Padding(0, 1).
		Foreground(RGBGrey)

	path := m.modalContent.Path
	if path == "" && m.modalContent.Paths != nil && len(m.modalContent.Paths) > 0 {
		path = m.modalContent.Paths[0]
	}

	maxPathWidth := splitWidth - 6
	if len(path) > maxPathWidth && maxPathWidth > 3 {
		path = path[:maxPathWidth-3] + "..." // truncate if it's too long.
	}

	pathBar := JSONPathBarStyle.Render(path)

	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)
	var detailsContent strings.Builder

	severity := getRuleSeverity(m.modalContent)
	var asciiIcon string
	var asciiIconStyle lipgloss.Style
	switch severity {
	case "✗ error":
		asciiIcon = "✗"
		asciiIconStyle = lipgloss.NewStyle().Foreground(RGBRed).Bold(true)
	case "▲ warning":
		asciiIcon = "▲"
		asciiIconStyle = lipgloss.NewStyle().Foreground(RBGYellow).Bold(true)
	case "● info":
		asciiIcon = "●"
		asciiIconStyle = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true)
	default:
		asciiIcon = "●"
		asciiIconStyle = lipgloss.NewStyle().Foreground(RGBGrey).Bold(true)
	}

	ruleName := "Issue"
	if m.modalContent.Rule != nil && m.modalContent.Rule.Id != "" {
		ruleName = m.modalContent.Rule.Id
	}

	var titleStyle lipgloss.Style
	switch severity {
	case "✗ error":
		titleStyle = lipgloss.NewStyle().Foreground(RGBRed).Bold(true)
	case "▲ warning":
		titleStyle = lipgloss.NewStyle().Foreground(RBGYellow).Bold(true)
	case "● info":
		titleStyle = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true)
	default:
		titleStyle = lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
	}

	detailsContent.WriteString(fmt.Sprintf("%s %s", asciiIconStyle.Render(asciiIcon), titleStyle.Render(ruleName)))
	detailsContent.WriteString("\n")

	location := formatFileLocation(m.modalContent, m.fileName)
	detailsContent.WriteString(lipgloss.NewStyle().Foreground(RGBBlue).Render(location))
	detailsContent.WriteString("\n\n")

	colorizedMessage := ColorizeString(m.modalContent.Message, ColorizeSubtleSecondary)
	msgStyle := lipgloss.NewStyle().Width(detailsWidth - 2)
	detailsContent.WriteString(msgStyle.Render(colorizedMessage))

	detailsPanel := detailsStyle.Render(detailsContent.String())

	howToFixStyle := lipgloss.NewStyle().
		Width(howToFixWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)
	var howToFixContent strings.Builder

	if m.modalContent.Rule != nil && m.modalContent.Rule.HowToFix != "" {
		fixLines := strings.Split(m.modalContent.Rule.HowToFix, "\n")
		for i, line := range fixLines {
			if i > 0 {
				howToFixContent.WriteString("\n")
			}
			wrapped := wrapText(line, howToFixWidth-4)
			howToFixContent.WriteString(wrapped)
		}
	} else {
		howToFixContent.WriteString(lipgloss.NewStyle().Foreground(RGBGrey).Italic(true).
			Render("No fix suggestions available"))
	}

	howToFixPanel := howToFixStyle.Render(howToFixContent.String())

	codeStyle := lipgloss.NewStyle().
		Width(codeWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)

	var codeContent strings.Builder
	if codeSnippet != "" {
		codeLines := strings.Split(codeSnippet, "\n")
		lineNumStyle := lipgloss.NewStyle().Foreground(RGBGrey).Bold(true)
		codeTextStyle := lipgloss.NewStyle().Foreground(RGBWhite)
		highlightStyle := lipgloss.NewStyle().
			Background(RGBSubtlePink).
			Foreground(RGBPink).
			Bold(true)

		maxLineNum := startLine + len(codeLines) - 1
		lineNumWidth := len(fmt.Sprintf("%d", maxLineNum)) + 1 // +1 for space after number
		if lineNumWidth < 5 {
			lineNumWidth = 5
		}

		maxLineWidth := codeWidth - lineNumWidth - 2

		for i, codeLine := range codeLines {
			actualLineNum := startLine + i
			isHighlighted := false

			if m.modalContent.StartNode != nil && actualLineNum == m.modalContent.StartNode.Line {
				isHighlighted = true
			} else if m.modalContent.Origin != nil && actualLineNum == m.modalContent.Origin.Line {
				isHighlighted = true
			}

			lineNumStr := fmt.Sprintf("%*d ", lineNumWidth-1, actualLineNum)

			if isHighlighted {
				highlightedLineNumStyle := lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
				codeContent.WriteString(highlightedLineNumStyle.Render(lineNumStr))
			} else {
				codeContent.WriteString(lineNumStyle.Render(lineNumStr))
			}

			displayLine := codeLine
			if len(codeLine) > maxLineWidth {
				if maxLineWidth > 3 {
					displayLine = codeLine[:maxLineWidth-3] + "..." // truncate.
				} else {
					displayLine = codeLine[:maxLineWidth]
				}
			}

			if isHighlighted {
				paddedLine := displayLine
				if len(displayLine) < maxLineWidth {
					paddedLine = displayLine + strings.Repeat(" ", maxLineWidth-len(displayLine))
				}
				codeContent.WriteString(highlightStyle.Render(paddedLine))
			} else {
				codeContent.WriteString(codeTextStyle.Render(displayLine))
			}

			if i < len(codeLines)-1 {
				codeContent.WriteString("\n")
			}
		}
	} else {
		codeContent.WriteString(lipgloss.NewStyle().Foreground(RGBGrey).Italic(true).
			Render("No code context available"))
	}

	codePanel := codeStyle.Render(codeContent.String())

	// combine all three columns horizontally
	combinedPanels := lipgloss.JoinHorizontal(lipgloss.Top,
		detailsPanel,
		howToFixPanel,
		codePanel,
	)

	// Add a blank line between path and panels for spacing
	spacer := lipgloss.NewStyle().Height(1).Render(" ")

	combinedContent := lipgloss.JoinVertical(lipgloss.Left,
		pathBar,
		spacer,
		combinedPanels,
	)

	containerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBBlue).
		Width(splitWidth).
		Height(splitHeight)

	return containerStyle.Render(combinedContent)
}
