// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"

	color2 "github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/doctor/changerator/renderer"
	"github.com/pb33f/doctor/terminal"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
)

// WhatChangedModeMessage is the message displayed when what-changed mode is active
const WhatChangedModeMessage = "WHAT CHANGED MODE: Only linting changed areas"

// renderComparisonModeSummary displays the comparison mode box, change statistics, and optionally the tree
// The tree is only rendered when showTree is true (when --changes-summary flag is passed)
func renderComparisonModeSummary(changeResult *utils.ChangeResult, changes *wcModel.DocumentChanges, noStyle bool, showTree bool) {
	if changes == nil {
		return
	}

	// Render the what-changed mode box
	tui.RenderStyledBox(WhatChangedModeMessage, tui.BoxTypeComparison, noStyle)

	// Extract and render change statistics
	stats := utils.ExtractChangeStats(changes)
	if stats.HasChanges() {
		renderChangeStatistics(stats, noStyle)
	}

	// Render the change tree if available and requested via --changes-summary
	if showTree && changeResult != nil && changeResult.RootNode != nil {
		renderChangeTree(changeResult, noStyle)
	}
}

// renderChangeTree renders the change tree using the doctor's TreeRenderer
func renderChangeTree(changeResult *utils.ChangeResult, noStyle bool) {
	if changeResult == nil || changeResult.RootNode == nil {
		return
	}

	// determine color scheme based on noStyle flag
	// when styled (!noStyle): use GrayscaleColorScheme (dims tree chrome)
	// when not styled (noStyle): use nil (NoColorScheme, no ANSI codes)
	var colorScheme terminal.ColorScheme
	if !noStyle {
		colorScheme = terminal.GrayscaleColorScheme{}
	}

	config := &renderer.TreeConfig{
		UseEmojis:       !noStyle, // Use emojis when styled, ASCII when not
		ShowLineNumbers: true,
		ShowStatistics:  true,
		ColorScheme:     colorScheme,
	}

	treeRenderer := renderer.NewTreeRenderer(changeResult.RootNode, config)
	treeOutput := treeRenderer.Render()

	if treeOutput == "" {
		return
	}

	if noStyle {
		fmt.Printf(" change tree\n")
		fmt.Printf(" %s\n", strings.Repeat("-", 40))
	} else {
		fmt.Printf(" %s%schange tree%s\n", color2.ASCIICyan, color2.ASCIIBold, color2.ASCIIReset)
		renderCyanSeparator(40)
	}

	// Indent the tree output
	lines := strings.Split(treeOutput, "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Printf(" %s\n", line)
		}
	}

	if noStyle {
		fmt.Printf(" %s\n\n", strings.Repeat("-", 40))
	} else {
		renderCyanSeparator(40)
		fmt.Println()
	}
}

// renderChangeStatistics displays the change statistics summary
func renderChangeStatistics(stats *utils.ChangeStats, noStyle bool) {
	if noStyle {
		renderChangeStatisticsPlain(stats)
	} else {
		renderChangeStatisticsStyled(stats)
	}
}

// renderChangeStatisticsPlain renders change statistics without color
func renderChangeStatisticsPlain(stats *utils.ChangeStats) {
	fmt.Printf(" change summary\n")
	fmt.Printf(" %s\n", strings.Repeat("-", 40))

	fmt.Printf(" %-20s %s\n", "total changes:", humanize.Comma(int64(stats.TotalChanges)))

	if stats.Added > 0 {
		fmt.Printf(" %-20s %s\n", "added:", humanize.Comma(int64(stats.Added)))
	}
	if stats.Modified > 0 {
		fmt.Printf(" %-20s %s\n", "modified:", humanize.Comma(int64(stats.Modified)))
	}
	if stats.Removed > 0 {
		fmt.Printf(" %-20s %s\n", "removed:", humanize.Comma(int64(stats.Removed)))
	}
	if stats.BreakingChanges > 0 {
		fmt.Printf(" %-20s %s\n", "breaking:", humanize.Comma(int64(stats.BreakingChanges)))
	}

	fmt.Printf(" %s\n\n", strings.Repeat("-", 40))
}

// renderChangeStatisticsStyled renders change statistics with color
func renderChangeStatisticsStyled(stats *utils.ChangeStats) {
	fmt.Printf(" %s%schange summary%s\n", color2.ASCIICyan, color2.ASCIIBold, color2.ASCIIReset)
	renderCyanSeparator(40)

	// Total changes
	fmt.Printf(" %s%-20s%s %s%s%s\n",
		color2.ASCIIGrey, "total changes:", color2.ASCIIReset,
		color2.ASCIIBold, humanize.Comma(int64(stats.TotalChanges)), color2.ASCIIReset)

	// Added (green)
	if stats.Added > 0 {
		fmt.Printf(" %s%-20s%s %s%s%s\n",
			color2.ASCIIGrey, "added:", color2.ASCIIReset,
			color2.ASCIIGreen, humanize.Comma(int64(stats.Added)), color2.ASCIIReset)
	}

	// Modified (yellow)
	if stats.Modified > 0 {
		fmt.Printf(" %s%-20s%s %s%s%s\n",
			color2.ASCIIGrey, "modified:", color2.ASCIIReset,
			color2.ASCIIYellow, humanize.Comma(int64(stats.Modified)), color2.ASCIIReset)
	}

	// Removed (red)
	if stats.Removed > 0 {
		fmt.Printf(" %s%-20s%s %s%s%s\n",
			color2.ASCIIGrey, "removed:", color2.ASCIIReset,
			color2.ASCIIRed, humanize.Comma(int64(stats.Removed)), color2.ASCIIReset)
	}

	// Breaking changes (red bold)
	if stats.BreakingChanges > 0 {
		fmt.Printf(" %s%-20s%s %s%s%s%s\n",
			color2.ASCIIGrey, "breaking:", color2.ASCIIReset,
			color2.ASCIIRed, color2.ASCIIBold, humanize.Comma(int64(stats.BreakingChanges)), color2.ASCIIReset)
	}

	renderCyanSeparator(40)
	fmt.Println()
}

// renderCyanSeparator renders a cyan colored separator line
func renderCyanSeparator(width int) {
	fmt.Printf(" %s%s%s\n", color2.ASCIICyan, strings.Repeat("─", width), color2.ASCIIReset)
}
