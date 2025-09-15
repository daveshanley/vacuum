// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
	"runtime"

	"github.com/charmbracelet/lipgloss/v2"
)

// Central lipgloss styles to replace all raw ANSI usage
// These styles use the existing RGBColor variables for 100% color compatibility
var (
	// Message and inline code styling
	StyleCodeHighlight     = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true).Italic(true)
	StyleQuotedText        = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true).Italic(true)

	// Path and location styling
	StylePathGrey          = lipgloss.NewStyle().Foreground(RGBGrey)
	StyleFileItalic        = lipgloss.NewStyle().Foreground(RGBLightGrey).Italic(true)
	StyleDirectoryGrey     = lipgloss.NewStyle().Foreground(RGBGrey)
	StyleLineNumber        = lipgloss.NewStyle().Bold(true)
	StyleColumnNumber      = lipgloss.NewStyle().Foreground(RGBLightGrey)
	StyleLocationSeparator = lipgloss.NewStyle().Foreground(RGBLightGrey)

	// Log message styling by severity
	StyleLogError          = lipgloss.NewStyle().Foreground(RGBRed).Bold(true)
	StyleLogWarn           = lipgloss.NewStyle().Foreground(RBGYellow).Italic(true)
	StyleLogInfo           = lipgloss.NewStyle().Foreground(RGBBlue).Italic(true)
	StyleLogDebug          = lipgloss.NewStyle().Foreground(RGBGrey).Italic(true)
	StyleLogPrefix         = lipgloss.NewStyle().Italic(true)

	// Severity markers for table output
	StyleSeverityError     = lipgloss.NewStyle().Foreground(RGBRed)
	StyleSeverityWarning   = lipgloss.NewStyle().Foreground(RBGYellow)
	StyleSeverityInfo      = lipgloss.NewStyle().Foreground(RGBBlue)

	// Table row styling
	StyleSelectedRow       = lipgloss.NewStyle().Foreground(RGBMutedPink)

	// Path component styling for ColorizePath (currently unused - using original ANSI approach)
	StylePathQuoted        = lipgloss.NewStyle().Foreground(RGBLightGrey).Italic(true)
	StylePathArrow         = lipgloss.NewStyle().Foreground(RGBRed)
	StylePathRef           = lipgloss.NewStyle().Foreground(RGBLightGrey)

	// Syntax highlighting styles (consolidate from existing variables)
	StyleSyntaxKey         = lipgloss.NewStyle().Foreground(RGBBlue).Bold(true)
	StyleSyntaxString      = lipgloss.NewStyle().Foreground(RGBGreen)
	StyleSyntaxNumber      = lipgloss.NewStyle().Foreground(RBGYellow).Italic(true).Bold(true)
	StyleSyntaxBool        = lipgloss.NewStyle().Foreground(RGBGrey).Italic(true).Bold(true)
	StyleSyntaxComment     = lipgloss.NewStyle().Foreground(RGBPink).Italic(true)
	StyleSyntaxDash        = lipgloss.NewStyle().Foreground(RGBPink)
	StyleSyntaxRef         lipgloss.Style // initialized in init()
	StyleSyntaxDefault     = lipgloss.NewStyle().Foreground(RGBPink)
	StyleSyntaxSingleQuote = lipgloss.NewStyle().Foreground(RGBPink).Italic(true)
)

// InitStyles initializes all styles once - replaces InitSyntaxStyles
var stylesInitialized bool

func init() {
	// Initialize StyleSyntaxRef based on platform
	if runtime.GOOS == "windows" {
		// no background on windows as it breaks alignment
		StyleSyntaxRef = lipgloss.NewStyle().Foreground(RGBGreen).Bold(true)
	} else {
		StyleSyntaxRef = lipgloss.NewStyle().Foreground(RGBGreen).Background(RGBDarkGrey).Bold(true)
	}
}

func InitStyles() {
	if !stylesInitialized {
		// Styles are already defined above with lipgloss.NewStyle()
		// This function exists for backwards compatibility
		stylesInitialized = true
	}
}

// GetStylesInitialized returns whether styles have been initialized
func GetStylesInitialized() bool {
	return stylesInitialized
}

// SetStylesInitialized sets the initialization state (for testing)
func SetStylesInitialized(state bool) {
	stylesInitialized = state
}