// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
	"golang.org/x/term"
)

type BoxType string

const (
	BoxTypeError      BoxType = "error"
	BoxTypeWarning    BoxType = "warning"
	BoxTypeInfo       BoxType = "info"
	BoxTypeSuccess    BoxType = "success"
	BoxTypeHard       BoxType = "hard"
	BoxTypeComparison BoxType = "comparison"
)

func getTerminalWidth() int {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 120
	}

	if color.AreColorsDisabled() && width > 3 {
		width = width - 3
	}

	return width
}

func calculateBoxWidth(termWidth int) int {
	// simplified box width calculation based on terminal size
	if termWidth < 100 {
		boxWidth := termWidth - 13
		if boxWidth < 40 {
			return 40
		}
		return boxWidth
	}
	// for larger terminals, use a reasonable max width
	return 107
}

func RenderStyledBox(message string, boxType BoxType, noStyle bool) {
	if noStyle {
		fmt.Printf(" | %s\n\n", message)
		return
	}

	termWidth := getTerminalWidth()
	boxWidth := calculateBoxWidth(termWidth)

	messageStyle := lipgloss.NewStyle().
		Width(boxWidth-4).
		Padding(1, 2)

	var boxStyle lipgloss.Style
	switch boxType {
	case BoxTypeError, BoxTypeHard:
		boxStyle = lipgloss.NewStyle().
			Width(boxWidth).
			Foreground(color.RGBRed).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(color.RGBRed).
			Bold(true)
	case BoxTypeWarning:
		boxStyle = lipgloss.NewStyle().
			Width(boxWidth).
			Foreground(color.RBGYellow).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(color.RBGYellow).
			Bold(true)
	case BoxTypeInfo:
		boxStyle = lipgloss.NewStyle().
			Width(boxWidth).
			Foreground(color.RGBBlue).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(color.RGBBlue).
			Bold(true)
	case BoxTypeSuccess:
		boxStyle = lipgloss.NewStyle().
			Width(boxWidth).
			Foreground(color.RGBGreen).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(color.RGBGreen).
			Bold(true)
	case BoxTypeComparison:
		boxStyle = lipgloss.NewStyle().
			Width(boxWidth).
			Foreground(color.RGBCyan).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(color.RGBCyan).
			Bold(true)
	default:
		boxStyle = lipgloss.NewStyle().
			Width(boxWidth).
			Foreground(color.RGBWhite).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(color.RGBWhite).
			Bold(true)
	}

	fmt.Println(boxStyle.Render(messageStyle.Render(message)))
	fmt.Println()
}
