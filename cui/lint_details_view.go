// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// splitViewDimensions holds calculated dimensions for the split view panels
type splitViewDimensions struct {
	splitWidth    int
	splitHeight   int
	contentHeight int
	detailsWidth  int
	howToFixWidth int
	codeWidth     int
}

// BuildDetailsView builds the details view for a violation when a user presses enter or return on a row.
func (m *ViolationResultTableModel) BuildDetailsView() string {
	if m.modalContent == nil {
		return ""
	}

	if m.height < 20 {
		return "" // terminal too small
	}

	dims := m.calculateSplitViewDimensions()
	
	pathBar := m.buildPathBar(dims.splitWidth)
	detailsPanel := m.buildDetailsPanel(dims.detailsWidth, dims.contentHeight)
	howToFixPanel := m.buildHowToFixPanel(dims.howToFixWidth, dims.contentHeight)
	codePanel := m.buildCodePanel(dims.codeWidth, dims.contentHeight)
	
	return m.assembleSplitView(dims, pathBar, detailsPanel, howToFixPanel, codePanel)
}

// calculateSplitViewDimensions calculates panel dimensions based on terminal size
func (m *ViolationResultTableModel) calculateSplitViewDimensions() splitViewDimensions {
	splitWidth := m.width
	innerWidth := splitWidth - 4 // account for container borders and padding
	
	return splitViewDimensions{
		splitWidth:    splitWidth,
		splitHeight:   SplitViewHeight,
		contentHeight: SplitContentHeight,
		detailsWidth:  int(float64(innerWidth) * float64(DetailsColumnPercent) / 100),
		howToFixWidth: int(float64(innerWidth) * float64(HowToFixColumnPercent) / 100),
		codeWidth:     innerWidth - int(float64(innerWidth)*float64(DetailsColumnPercent)/100) - int(float64(innerWidth)*float64(HowToFixColumnPercent)/100),
	}
}

// buildPathBar builds the path bar showing the JSONPath or error path
func (m *ViolationResultTableModel) buildPathBar(splitWidth int) string {
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
		path = path[:maxPathWidth-3] + "..."
	}

	return JSONPathBarStyle.Render(path)
}

// buildDetailsPanel builds the details panel with rule info and message
func (m *ViolationResultTableModel) buildDetailsPanel(detailsWidth, contentHeight int) string {

	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)
	
	var detailsContent strings.Builder
	
	// build title with severity icon
	severity := getRuleSeverity(m.modalContent)
	severityInfo := GetSeverityInfoFromText(severity)
	
	ruleName := "Issue"
	if m.modalContent.Rule != nil && m.modalContent.Rule.Id != "" {
		ruleName = m.modalContent.Rule.Id
	}
	
	titleStyle := severityInfo.TextStyle.Bold(true)
	detailsContent.WriteString(fmt.Sprintf("%s %s",
		severityInfo.IconStyle.Render(severityInfo.Icon), 
		titleStyle.Render(ruleName)))
	detailsContent.WriteString("\n")
	
	// add location
	location := formatFileLocation(m.modalContent, m.fileName)
	detailsContent.WriteString(lipgloss.NewStyle().Foreground(RGBBlue).Render(location))
	detailsContent.WriteString("\n\n")
	
	// add message
	colorizedMessage := ColorizeMessage(m.modalContent.Message)
	msgStyle := lipgloss.NewStyle().Width(detailsWidth - 2)
	detailsContent.WriteString(msgStyle.Render(colorizedMessage))
	
	return detailsStyle.Render(detailsContent.String())
}

// buildHowToFixPanel builds the how-to-fix panel with remediation suggestions
func (m *ViolationResultTableModel) buildHowToFixPanel(howToFixWidth, contentHeight int) string {

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
	
	return howToFixStyle.Render(howToFixContent.String())
}

// buildCodePanel builds the code panel with syntax highlighted snippet
func (m *ViolationResultTableModel) buildCodePanel(codeWidth, contentHeight int) string {
	codeSnippet, startLine := m.ExtractCodeSnippet(m.modalContent, 4)

	codeStyle := lipgloss.NewStyle().
		Width(codeWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(0, 1)
	
	var codeContent strings.Builder
	
	if codeSnippet != "" {
		isYAML := strings.HasSuffix(m.fileName, ".yaml") || strings.HasSuffix(m.fileName, ".yml")

		codeLines := strings.Split(codeSnippet, "\n")
		lineNumStyle := lipgloss.NewStyle().Foreground(RGBGrey).Bold(true)
		highlightStyle := lipgloss.NewStyle().
			Background(RGBSubtlePink).
			Foreground(RGBPink).
			Bold(true)

		maxLineNum := startLine + len(codeLines) - 1
		lineNumWidth := len(fmt.Sprintf("%d", maxLineNum)) + 1 // +1 for space after number
		if lineNumWidth < 5 {
			lineNumWidth = 5
		}

		maxLineWidth := codeWidth - lineNumWidth - 4 // -4 to account for "▶ " or "│ " (2 chars) plus padding

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
				triangleStyle := lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
				codeContent.WriteString(triangleStyle.Render("▶ "))
			} else {
				codeContent.WriteString(lineNumStyle.Render(lineNumStr))
				pipeStyle := lipgloss.NewStyle().Foreground(RGBGrey)
				codeContent.WriteString(pipeStyle.Render("│ "))
			}

			syntaxHighlightedLine := ApplySyntaxHighlightingToLine(codeLine, isYAML)

			actualWidth := lipgloss.Width(syntaxHighlightedLine)
			if actualWidth > maxLineWidth {
				// we need to truncate the syntax-highlighted line
				// this is tricky because we need to preserve ANSI codes
				// for now, fall back to plain text truncation when line is too long
				if maxLineWidth > 3 {
					displayLine := codeLine[:min(len(codeLine), maxLineWidth-3)] + "..."
					syntaxHighlightedLine = ApplySyntaxHighlightingToLine(displayLine, isYAML)
				} else {
					displayLine := codeLine[:min(len(codeLine), maxLineWidth)]
					syntaxHighlightedLine = ApplySyntaxHighlightingToLine(displayLine, isYAML)
				}
			}

			if isHighlighted {
				// for highlighted lines, we need to apply the background color
				// to calculate padding needed
				currentWidth := lipgloss.Width(syntaxHighlightedLine)
				paddingNeeded := maxLineWidth - currentWidth
				if paddingNeeded > 0 {
					// add padding to the raw line, then apply highlighting
					paddedLine := codeLine + strings.Repeat(" ", paddingNeeded)
					if len(paddedLine) > maxLineWidth {
						paddedLine = codeLine[:min(len(codeLine), maxLineWidth)]
					}
					codeContent.WriteString(highlightStyle.Render(paddedLine))
				} else {
					// apply background to the truncated line
					codeContent.WriteString(highlightStyle.Render(codeLine[:min(len(codeLine), maxLineWidth)]))
				}
			} else {
				// use the syntax-highlighted line for normal lines
				codeContent.WriteString(syntaxHighlightedLine)
			}

			if i < len(codeLines)-1 {
				codeContent.WriteString("\n")
			}
		}
	} else {
		codeContent.WriteString(lipgloss.NewStyle().Foreground(RGBGrey).Italic(true).
			Render("No code context available"))
	}

	return codeStyle.Render(codeContent.String())
}

// assembleSplitView combines all panels into the final split view
func (m *ViolationResultTableModel) assembleSplitView(dims splitViewDimensions, pathBar, detailsPanel, howToFixPanel, codePanel string) string {
	// combine all three columns horizontally
	combinedPanels := lipgloss.JoinHorizontal(lipgloss.Top,
		detailsPanel,
		howToFixPanel,
		codePanel,
	)
	
	// blank line between path and panels for spacing
	spacer := lipgloss.NewStyle().Height(1).Render(" ")
	
	combinedContent := lipgloss.JoinVertical(lipgloss.Left,
		pathBar,
		spacer,
		combinedPanels,
	)
	
	containerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBBlue).
		Width(dims.splitWidth).
		Height(dims.splitHeight)
	
	return containerStyle.Render(combinedContent)
}
