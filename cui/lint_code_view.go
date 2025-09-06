// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/viewport"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/model"
)

// InitSyntaxStyles initializes the syntax highlighting styles once
func InitSyntaxStyles() {
	if !syntaxStylesInit {
		syntaxKeyStyle = lipgloss.NewStyle().Foreground(RGBBlue)
		syntaxStringStyle = lipgloss.NewStyle().Foreground(RGBGreen)
		syntaxNumberStyle = lipgloss.NewStyle().Foreground(RBGYellow).Italic(true).Bold(true)
		syntaxBoolStyle = lipgloss.NewStyle().Foreground(RGBGrey).Italic(true).Bold(true)
		syntaxCommentStyle = lipgloss.NewStyle().Foreground(RGBPink).Italic(true)
		syntaxDashStyle = lipgloss.NewStyle().Foreground(RGBPink)
		syntaxRefStyle = lipgloss.NewStyle().Foreground(RGBGreen).Background(RGBDarkGrey).Bold(true)
		syntaxDefaultStyle = lipgloss.NewStyle().Foreground(RGBPink)
		syntaxSingleQuoteStyle = lipgloss.NewStyle().Foreground(RGBPink).Italic(true)
		syntaxStylesInit = true
	}
}

// PrepareCodeViewport prepares the code viewport with the full spec and highlights the error line
func (m *ViolationResultTableModel) PrepareCodeViewport() {
	if m.modalContent == nil || m.specContent == nil {
		return
	}

	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - ModalHeightMargin

	m.codeViewport = viewport.New(viewport.WithWidth(modalWidth-4), viewport.WithHeight(modalHeight-4))

	// get the line number from the result
	targetLine := 0
	if m.modalContent.StartNode != nil {
		targetLine = m.modalContent.StartNode.Line
	} else if m.modalContent.Origin != nil {
		targetLine = m.modalContent.Origin.Line
	}

	content := m.FormatCodeWithHighlight(targetLine, modalWidth-8)
	m.codeViewport.SetContent(content)

	// scroll to the target line (try to center it in the viewport)
	if targetLine > 0 {
		// for windowed content, we need to calculate the position within the rendered content
		// the target line is always at position CodeWindowSize (or less if near the start of file)
		allLines := strings.Split(string(m.specContent), "\n")
		totalLines := len(allLines)
		const windowSize = CodeWindowSize

		// calculate where the target line appears in our rendered content
		var targetPositionInWindow int
		if totalLines <= (windowSize*2 + 1) {
			// no windowing, target is at its actual position
			targetPositionInWindow = targetLine
		} else {
			// windowing is active
			startLine := targetLine - windowSize
			if startLine < 1 {
				startLine = 1
			}
			// account for the "lines above not shown" notice if present
			if startLine > 1 {
				targetPositionInWindow = targetLine - startLine + 2 // +2 for the notice line
			} else {
				targetPositionInWindow = targetLine - startLine + 1
			}
		}

		// scroll to the center the target line in the viewport
		scrollTo := targetPositionInWindow - (m.codeViewport.Height() / 2)
		if scrollTo < 0 {
			scrollTo = 0
		}
		m.codeViewport.SetYOffset(scrollTo)
	}
}

// BuildCodeView builds the expanded code view modal
func (m *ViolationResultTableModel) BuildCodeView() string {
	modalWidth := int(float64(m.width) - 40)
	modalHeight := m.height - ModalHeightMargin

	if m.modalContent == nil {
		return ""
	}

	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(0, 1, 0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink)

	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(RGBBlue).
		Bold(true).
		Width(modalWidth - 4)

	targetLine := 0
	if m.modalContent.StartNode != nil {
		targetLine = m.modalContent.StartNode.Line
	} else if m.modalContent.Origin != nil {
		targetLine = m.modalContent.Origin.Line
	}

	title := fmt.Sprintf("📄 %s - line %d", m.fileName, targetLine)
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")

	sepStyle := lipgloss.NewStyle().
		Foreground(RGBPink).
		Width(modalWidth - 4)
	content.WriteString(sepStyle.Render(strings.Repeat("-", modalWidth-4)))
	content.WriteString("\n\n")

	// code viewport.
	content.WriteString(m.codeViewport.View())

	// calculate remaining lines for proper modal height
	currentLines := strings.Count(content.String(), "\n")
	neededLines := modalHeight - currentLines - 3
	if neededLines > 0 {
		content.WriteString(strings.Repeat("\n", neededLines))
	}

	// bottom bar with scroll percentage and controls
	var bottomBar string
	if m.codeViewport.TotalLineCount() > m.codeViewport.Height() {
		scrollPercent := fmt.Sprintf(" %.0f%%", m.codeViewport.ScrollPercent()*100)
		scrollStyle := lipgloss.NewStyle().Foreground(RGBBlue)

		controls := "↑↓/jk: scroll | pgup/pgdn: page | space: recenter | esc/x: close "
		controlsStyle := lipgloss.NewStyle().Foreground(RGBGrey)

		// calculate spacing
		scrollWidth := lipgloss.Width(scrollPercent)
		controlsWidth := lipgloss.Width(controls)
		spacerWidth := (modalWidth - 4) - scrollWidth - controlsWidth
		if spacerWidth < 0 {
			spacerWidth = 1
		}

		bottomBar = scrollStyle.Render(scrollPercent) +
			strings.Repeat(" ", spacerWidth) +
			controlsStyle.Render(controls)
	} else {
		// no scrolling needed
		navStyle := lipgloss.NewStyle().
			Foreground(RGBDarkGrey).
			Width(modalWidth - 4).
			Align(lipgloss.Center)
		bottomBar = navStyle.Render("esc/x: close")
	}

	content.WriteString(bottomBar)

	return modalStyle.Render(content.String())
}

// FormatCodeWithHighlight formats the spec content with line numbers and highlights the error line
func (m *ViolationResultTableModel) FormatCodeWithHighlight(targetLine int, maxWidth int) string {
	allLines := strings.Split(string(m.specContent), "\n")
	totalLines := len(allLines)

	const windowSize = CodeWindowSize

	// calculate the window of lines to render
	startLine := 1
	endLine := totalLines
	actualTargetLine := targetLine // Track the actual line number for highlighting

	if totalLines > (windowSize*2 + 1) {
		// limit the window
		if targetLine > 0 {
			// calculate the window centered on the target line
			startLine = targetLine - windowSize
			if startLine < 1 {
				startLine = 1
			}
			endLine = targetLine + windowSize
			if endLine > totalLines {
				endLine = totalLines
			}
		} else {
			// no target line, show first 2001 lines
			endLine = windowSize*2 + 1
			endLine = totalLines
		}
	}

	// extract the lines to render (convert to 0-based indexing)
	lines := allLines[startLine-1 : endLine]

	var result strings.Builder
	lineNumStyle := lipgloss.NewStyle().Foreground(RGBGrey)
	highlightStyle := lipgloss.NewStyle().
		Background(RGBSubtlePink).
		Foreground(RGBPink).
		Bold(true)

	isYAML := strings.HasSuffix(m.fileName, ".yaml") || strings.HasSuffix(m.fileName, ".yml")

	// calculate line number width based on the actual max line numbers
	lineNumWidth := len(fmt.Sprintf("%d", endLine)) + 1
	if lineNumWidth < 5 {
		lineNumWidth = 5
	}

	// add a notice if we're showing a limited window
	if startLine > 1 {
		noticeStyle := lipgloss.NewStyle().Foreground(RGBGrey).Italic(true)
		result.WriteString(noticeStyle.Render(fmt.Sprintf("    ... (%d lines above not shown) ...", startLine-1)))
		result.WriteString("\n")
	}

	// track if we're in a multi-line markdown block
	inMarkdownBlock := false
	markdownIndent := ""
	var markdownContent strings.Builder
	markdownStartLine := 0

	for i, line := range lines {
		lineNum := startLine + i // actual line number in the file
		isHighlighted := lineNum == actualTargetLine

		// check if this is a description field with block scalar (| or >-)
		if isYAML && !inMarkdownBlock {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "description:") {
				// is it a block scalar?
				afterKey := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
				if afterKey == "|" || afterKey == "|-" || afterKey == ">-" || afterKey == ">" {
					inMarkdownBlock = true
					markdownContent.Reset()
					markdownStartLine = lineNum
					// indent level of the next content
					if i+1 < len(lines) {
						nextLine := lines[i+1]
						// calculate indent by finding first non-space character
						for j, ch := range nextLine {
							if ch != ' ' && ch != '\t' {
								markdownIndent = nextLine[:j]
								break
							}
						}
					}
				}
			}
		}

		// format line number
		lineNumStr := fmt.Sprintf("%*d ", lineNumWidth-1, lineNum)

		if isHighlighted {
			highlightedLineNumStyle := lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
			result.WriteString(highlightedLineNumStyle.Render(lineNumStr))
		} else {
			result.WriteString(lineNumStyle.Render(lineNumStr))
		}

		// handle markdown block content
		if inMarkdownBlock && lineNum > markdownStartLine {
			// check if we're still in the markdown block (lines must maintain same or greater indent)
			if len(markdownIndent) > 0 && !strings.HasPrefix(line, markdownIndent) && strings.TrimSpace(line) != "" {
				// end of markdown block - don't render with glamour for performance
				inMarkdownBlock = false
				// process current line normally
				coloredLine := ApplySyntaxHighlightingToLine(line, isYAML)
				if isHighlighted {
					displayLine := line
					if len(line) < maxWidth-lineNumWidth {
						displayLine = line + strings.Repeat(" ", maxWidth-lineNumWidth-len(line))
					}
					result.WriteString(highlightStyle.Render(displayLine))
				} else {
					result.WriteString(coloredLine)
				}
			} else {
				// still in markdown block, just apply syntax highlighting normally
				if isHighlighted {
					displayLine := line
					if len(line) < maxWidth-lineNumWidth {
						displayLine = line + strings.Repeat(" ", maxWidth-lineNumWidth-len(line))
					}
					result.WriteString(highlightStyle.Render(displayLine))
				} else {
					result.WriteString(ApplySyntaxHighlightingToLine(line, isYAML))
				}
			}
		} else {
			// normal line - apply syntax highlighting
			coloredLine := ApplySyntaxHighlightingToLine(line, isYAML)

			if isHighlighted {
				// pad the line to full width for background color
				displayLine := line
				if len(line) < maxWidth-lineNumWidth {
					displayLine = line + strings.Repeat(" ", maxWidth-lineNumWidth-len(line))
				}
				result.WriteString(highlightStyle.Render(displayLine))
			} else {
				result.WriteString(coloredLine)
			}
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	// add a notice if we're cutting off lines at the bottom
	if endLine < totalLines {
		result.WriteString("\n")
		noticeStyle := lipgloss.NewStyle().Foreground(RGBGrey).Italic(true)
		result.WriteString(noticeStyle.Render(fmt.Sprintf("    ... (%d lines below not shown) ...", totalLines-endLine)))
	}

	return result.String()
}

// ReCenterCodeView re-centers the viewport on the highlighted error line
func (m *ViolationResultTableModel) ReCenterCodeView() {
	if m.modalContent == nil {
		return
	}

	// get the target line number
	targetLine := 0
	if m.modalContent.StartNode != nil {
		targetLine = m.modalContent.StartNode.Line
	} else if m.modalContent.Origin != nil {
		targetLine = m.modalContent.Origin.Line
	}

	if targetLine > 0 {
		// calculate the position of the target line within the rendered content
		allLines := strings.Split(string(m.specContent), "\n")
		totalLines := len(allLines)
		const windowSize = CodeWindowSize

		var targetPositionInWindow int
		if totalLines <= (windowSize*2 + 1) {
			// no windowing, target is at its actual position
			targetPositionInWindow = targetLine
		} else {
			// windowing is active
			startLine := targetLine - windowSize
			if startLine < 1 {
				startLine = 1
			}
			// account for the "lines above not shown" notice if present
			if startLine > 1 {
				targetPositionInWindow = targetLine - startLine + 2 // +2 for the notice line
			} else {
				targetPositionInWindow = targetLine - startLine + 1
			}
		}

		// center the target line in the viewport
		scrollTo := targetPositionInWindow - (m.codeViewport.Height() / 2)
		if scrollTo < 0 {
			scrollTo = 0
		}
		m.codeViewport.SetYOffset(scrollTo)
	}
}

// ExtractCodeSnippet extracts lines around the issue with context
func (m *ViolationResultTableModel) ExtractCodeSnippet(result *model.RuleFunctionResult, contextLines int) (string, int) {
	if m.specContent == nil || result == nil {
		return "", 0
	}

	line := 0
	if result.StartNode != nil {
		line = result.StartNode.Line
	}
	if result.Origin != nil {
		line = result.Origin.Line
	}

	if line == 0 {
		return "", 0
	}

	lines := bytes.Split(m.specContent, []byte("\n"))

	startLine := line - contextLines - 1 // -1 because line numbers are 1-based
	if startLine < 0 {
		startLine = 0
	}

	endLine := line + contextLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var snippet strings.Builder
	for i := startLine; i < endLine; i++ {
		snippet.Write(lines[i])
		if i < endLine-1 {
			snippet.WriteString("\n")
		}
	}

	return snippet.String(), startLine + 1
}
