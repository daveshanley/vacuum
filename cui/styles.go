// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/charmbracelet/lipgloss/v2"
)

var (
	StyleCodeHighlight = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true).Italic(true)
	StyleQuotedText    = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true).Italic(true)

	StylePathGrey          = lipgloss.NewStyle().Foreground(RGBGrey)
	StyleFileItalic        = lipgloss.NewStyle().Foreground(RGBLightGrey).Italic(true)
	StyleDirectoryGrey     = lipgloss.NewStyle().Foreground(RGBGrey)
	StyleLineNumber        = lipgloss.NewStyle().Bold(true)
	StyleColumnNumber      = lipgloss.NewStyle().Foreground(RGBLightGrey)
	StyleLocationSeparator = lipgloss.NewStyle().Foreground(RGBLightGrey)

	StyleLogError  = lipgloss.NewStyle().Foreground(RGBRed).Bold(true)
	StyleLogWarn   = lipgloss.NewStyle().Foreground(RBGYellow).Italic(true)
	StyleLogInfo   = lipgloss.NewStyle().Foreground(RGBBlue).Italic(true)
	StyleLogDebug  = lipgloss.NewStyle().Foreground(RGBGrey).Italic(true)
	StyleLogPrefix = lipgloss.NewStyle().Italic(true)

	StyleSeverityError   = lipgloss.NewStyle().Foreground(RGBRed)
	StyleSeverityWarning = lipgloss.NewStyle().Foreground(RBGYellow)
	StyleSeverityInfo    = lipgloss.NewStyle().Foreground(RGBBlue)

	StyleSelectedRow = lipgloss.NewStyle().Foreground(RGBMutedPink)

	StylePathQuoted        = lipgloss.NewStyle().Foreground(RGBLightGrey).Italic(true)
	StylePathArrow         = lipgloss.NewStyle().Foreground(RGBRed)
	StylePathRef           = lipgloss.NewStyle().Foreground(RGBLightGrey)
	StyleSyntaxKey         = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true)
	StyleSyntaxString      = lipgloss.NewStyle().Foreground(RGBGreen)
	StyleSyntaxNumber      = lipgloss.NewStyle().Foreground(RBGYellow).Bold(true)
	StyleSyntaxBool        = lipgloss.NewStyle().Foreground(RGBGrey).Italic(true).Bold(true)
	StyleSyntaxComment     = lipgloss.NewStyle().Foreground(RGBPink).Italic(true)
	StyleSyntaxDash        = lipgloss.NewStyle().Foreground(RGBPink)
	StyleSyntaxDefault     = lipgloss.NewStyle().Foreground(RGBPink)
	StyleSyntaxSingleQuote = lipgloss.NewStyle().Foreground(RGBPink).Italic(true)
	StyleSyntaxRef         = lipgloss.NewStyle().Foreground(RGBGreen).Background(RGBDarkGrey).Bold(true)
)
