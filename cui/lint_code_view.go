// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/v2/viewport"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/model"
)

// InitSyntaxStyles initializes the syntax highlighting styles once
// Now uses centralized styles from styles.go
func InitSyntaxStyles() {
	if !syntaxStylesInit {
		syntaxKeyStyle = StyleSyntaxKey
		syntaxStringStyle = StyleSyntaxString
		syntaxNumberStyle = StyleSyntaxNumber
		syntaxBoolStyle = StyleSyntaxBool
		syntaxCommentStyle = StyleSyntaxComment
		syntaxDashStyle = StyleSyntaxDash
		syntaxRefStyle = StyleSyntaxRef
		syntaxDefaultStyle = StyleSyntaxDefault
		syntaxSingleQuoteStyle = StyleSyntaxSingleQuote
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
	window := calculateCodeWindow(allLines, targetLine)
	
	var result strings.Builder
	
	// add top notice if needed
	if window.showAbove {
		result.WriteString(formatLinesNotShown(window.startLine-1, "above"))
		result.WriteString("\n")
	}
	
	// format the code lines
	isYAML := strings.HasSuffix(m.fileName, ".yaml") || strings.HasSuffix(m.fileName, ".yml")
	result.WriteString(m.formatCodeLines(window, targetLine, maxWidth, isYAML))
	
	// add bottom notice if needed
	if window.showBelow {
		result.WriteString("\n")
		result.WriteString(formatLinesNotShown(len(allLines)-window.endLine, "below"))
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

// codeWindow represents the window of lines to render
type codeWindow struct {
	startLine int      // 1-based start line number in file
	endLine   int      // 1-based end line number in file
	lines     []string // actual lines to render
	showAbove bool     // show "lines above not shown" notice
	showBelow bool     // show "lines below not shown" notice
}

// calculateCodeWindow determines which lines to show based on target line and file size
func calculateCodeWindow(allLines []string, targetLine int) codeWindow {
	const windowSize = CodeWindowSize
	totalLines := len(allLines)

	window := codeWindow{
		startLine: 1,
		endLine:   totalLines,
	}
	
	// check if we need windowing
	if totalLines > (windowSize*2 + 1) {
		if targetLine > 0 {
			// center window on target line
			window.startLine = targetLine - windowSize
			if window.startLine < 1 {
				window.startLine = 1
			}
			
			window.endLine = targetLine + windowSize
			if window.endLine > totalLines {
				window.endLine = totalLines
			}
		} else {
			// no target line, show first window
			window.endLine = windowSize*2 + 1
			if window.endLine > totalLines {
				window.endLine = totalLines
			}
		}
	}
	
	// extract lines (convert to 0-based indexing)
	window.lines = allLines[window.startLine-1 : window.endLine]
	window.showAbove = window.startLine > 1
	window.showBelow = window.endLine < totalLines
	
	return window
}

// formatLinesNotShown creates the "lines not shown" notice
func formatLinesNotShown(count int, position string) string {
	noticeStyle := lipgloss.NewStyle().Foreground(RGBGrey).Italic(true)
	return noticeStyle.Render(fmt.Sprintf("    ... (%d lines %s not shown) ...", count, position))
}

// formatCodeLines formats the actual code lines with line numbers and syntax highlighting
func (m *ViolationResultTableModel) formatCodeLines(window codeWindow, targetLine int, maxWidth int, isYAML bool) string {
	var result strings.Builder
	
	lineNumWidth := calculateLineNumberWidth(window.endLine)
	lineStyles := getLineFormattingStyles()
	
	for i, line := range window.lines {
		lineNum := window.startLine + i
		isHighlighted := lineNum == targetLine
		
		// format line number
		lineNumStr := formatLineNumber(lineNum, lineNumWidth, isHighlighted, lineStyles)
		result.WriteString(lineNumStr)
		
		// format line content
		lineContent := formatLineContent(line, maxWidth-lineNumWidth, isHighlighted, isYAML, lineStyles)
		result.WriteString(lineContent)
		
		if i < len(window.lines)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// calculateLineNumberWidth determines the width needed for line numbers
func calculateLineNumberWidth(maxLineNum int) int {
	width := len(fmt.Sprintf("%d", maxLineNum)) + 1
	if width < 5 {
		width = 5
	}
	return width
}

// lineFormattingStyles holds all the styles used for formatting code
type lineFormattingStyles struct {
	lineNum          lipgloss.Style
	lineNumHighlight lipgloss.Style
	pipe             lipgloss.Style
	triangle         lipgloss.Style
	highlight        lipgloss.Style
}

// getLineFormattingStyles returns all styles used for formatting
func getLineFormattingStyles() lineFormattingStyles {
	return lineFormattingStyles{
		lineNum:          lipgloss.NewStyle().Foreground(RGBGrey).Bold(true),
		lineNumHighlight: lipgloss.NewStyle().Foreground(RGBPink).Bold(true),
		pipe:             lipgloss.NewStyle().Foreground(RGBGrey),
		triangle:         lipgloss.NewStyle().Foreground(RGBPink).Bold(true),
		highlight:        lipgloss.NewStyle().Background(RGBSubtlePink).Foreground(RGBPink).Bold(true),
	}
}

// formatLineNumber formats the line number and marker
func formatLineNumber(lineNum int, width int, isHighlighted bool, styles lineFormattingStyles) string {
	lineNumStr := fmt.Sprintf("%*d ", width-1, lineNum)
	
	if isHighlighted {
		return styles.lineNumHighlight.Render(lineNumStr) + styles.triangle.Render("▶ ")
	}
	return styles.lineNum.Render(lineNumStr) + styles.pipe.Render("│ ")
}

// formatLineContent formats the actual line content with syntax highlighting
func formatLineContent(line string, maxWidth int, isHighlighted bool, isYAML bool, styles lineFormattingStyles) string {
	displayLine := ApplySyntaxHighlightingToLine(line, isYAML)

	if isHighlighted {
		// on windows, skip background highlighting as it breaks alignment
		if runtime.GOOS == "windows" {
			// just use syntax highlighting without background
			return displayLine
		}

		// on other platforms, apply background
		paddedLine := line
		if len(line) < maxWidth {
			paddedLine = line + strings.Repeat(" ", maxWidth-len(line))
		}
		return styles.highlight.Render(paddedLine)
	}

	return displayLine
}
