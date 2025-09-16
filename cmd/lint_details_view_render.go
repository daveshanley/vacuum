// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/tui"
	"github.com/dustin/go-humanize"
	"golang.org/x/term"
)

// Constants for summary table widths at different terminal sizes
const (
	// SummaryTableWidthFull is the width of summary tables for terminals >= 100 width
	// Calculated as: 40 (rule) + 12 (violations) + 50 (impact) + 4 (spacing) + 1 (leading) = 107
	SummaryTableWidthFull = 107

	// SummaryTableWidthMedium is the width for terminals 80-99 width
	// Calculated as: 25 (rule) + 10 (violations) + 30 (impact) + 4 (spacing) + 1 (leading) = 70
	SummaryTableWidthMedium = 70

	// SummaryTableWidthSmall is the width for terminals 60-79 width
	// Calculated as: 20 (rule) + 8 (violations) + 20 (impact) + 4 (spacing) + 1 (leading) = 53
	SummaryTableWidthSmall = 53
)

// TableConfig holds all table configuration and column widths
type TableConfig struct {
	Width         int
	ShowCategory  bool
	ShowPath      bool
	ShowRule      bool
	UseTreeFormat bool
	LocationWidth int
	SeverityWidth int
	MessageWidth  int
	RuleWidth     int
	CategoryWidth int
	PathWidth     int
	NoMessage     bool
	NoClip        bool
	NoStyle       bool
}

// SeverityInfo holds severity display information
type SeverityInfo struct {
	Icon      string
	Text      string
	Color     string
	Formatted string
}

// calculateMaxColumnWidths analyzes all results to determine natural column widths
func calculateMaxColumnWidths(results []*model.RuleFunctionResult, fileName string, errors bool) (locWidth, ruleWidth, catWidth, msgWidth int) {
	locWidth = len("Location")
	ruleWidth = len("Rule")
	catWidth = len("Category")
	msgWidth = len("Message")

	for _, r := range results {
		location := formatLocation(r, fileName)
		if len(location) > locWidth {
			locWidth = len(location)
		}

		if r.Rule != nil {
			if len(r.Rule.Id) > ruleWidth {
				ruleWidth = len(r.Rule.Id)
			}
			if r.Rule.RuleCategory != nil && len(r.Rule.RuleCategory.Name) > catWidth {
				catWidth = len(r.Rule.RuleCategory.Name)
			}
		}

		// only count message width if we're showing it
		if !errors || (r.Rule != nil && r.Rule.Severity == model.SeverityError) {
			if len(r.Message) > msgWidth {
				msgWidth = len(r.Message)
			}
		}
	}

	return
}

// calculateTableConfig determines the table layout based on terminal width
func calculateTableConfig(results []*model.RuleFunctionResult, fileName string, errors, noMessage, noClip, noStyle bool) *TableConfig {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 120
	}

	// In no-style mode, reduce width by 3 to avoid off-by-one truncation issues
	if noStyle && width > 3 {
		width = width - 3
	}

	config := &TableConfig{
		Width:         width,
		ShowCategory:  true,
		ShowPath:      true,
		ShowRule:      true,
		UseTreeFormat: false,
		NoMessage:     noMessage,
		NoClip:        noClip,
		NoStyle:       noStyle,
	}

	// get natural column widths
	locWidth, ruleWidth, catWidth, msgWidth := calculateMaxColumnWidths(results, fileName, errors)

	config.LocationWidth = locWidth
	config.SeverityWidth = 9
	config.RuleWidth = ruleWidth
	config.CategoryWidth = catWidth

	// responsive layout based on terminal width
	if width < 100 {
		config.UseTreeFormat = true
		return config
	} else if width >= 100 && width < 120 {
		config.ShowCategory = false
		config.ShowPath = false
		config.ShowRule = false
		config.CategoryWidth = 0
		config.RuleWidth = 0
		config.SeverityWidth = 2 // just the symbol
	} else if width >= 120 && width < 130 {
		config.ShowCategory = false
		config.ShowPath = false
		config.CategoryWidth = 0
	} else if width >= 130 && width < 160 {
		config.ShowCategory = false
		config.CategoryWidth = 0
	}
	// 160+ shows everything

	// calculate message and path widths
	separators := calculateSeparatorCount(config)
	fixedWidth := config.LocationWidth + config.SeverityWidth + config.RuleWidth + config.CategoryWidth + separators
	remainingWidth := width - fixedWidth

	if remainingWidth > 0 {
		if config.ShowPath {
			// message gets only what it needs, rest goes to path
			config.MessageWidth = msgWidth
			if config.MessageWidth > remainingWidth-20 {
				config.MessageWidth = remainingWidth - 20
			}
			config.PathWidth = remainingWidth - config.MessageWidth

			// enforce minimums
			if config.MessageWidth < 20 {
				config.MessageWidth = 20
				config.PathWidth = remainingWidth - config.MessageWidth
			}
			if config.PathWidth < 10 {
				config.PathWidth = 10
			}
		} else {
			config.MessageWidth = remainingWidth
			if config.MessageWidth < 20 {
				config.MessageWidth = 20
			}
		}
	} else {
		config.MessageWidth = 20
		if config.ShowPath {
			config.PathWidth = 10
		}
	}

	// adjust widths when no message column
	if config.NoMessage {
		if config.ShowPath {
			// Give all message width to path column
			config.PathWidth = config.PathWidth + config.MessageWidth + 2
		}
		// If path is not shown, the width just becomes right padding (as requested)
		config.MessageWidth = 0
	}

	return config
}

// calculateSeparatorCount returns the number of separator spaces needed
func calculateSeparatorCount(config *TableConfig) int {
	if !config.ShowRule && !config.ShowCategory && !config.ShowPath {
		return 4
	} else if !config.ShowCategory && !config.ShowPath {
		return 6
	} else if !config.ShowCategory {
		return 8
	}
	return 10
}

// formatLocation creates the location string for a result
func formatLocation(r *model.RuleFunctionResult, fileName string) string {
	startLine := 0
	startCol := 0
	if r.StartNode != nil {
		startLine = r.StartNode.Line
		startCol = r.StartNode.Column
	}

	f := fileName
	if r.Origin != nil {
		f = r.Origin.AbsoluteLocation
		startLine = r.Origin.Line
		startCol = r.Origin.Column
	}

	// make path relative
	if absPath, err := filepath.Abs(f); err == nil {
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, absPath); err == nil {
				f = relPath
			}
		}
	}

	return fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
}

// getSeverityInfo returns formatted severity information
func getSeverityInfo(r *model.RuleFunctionResult, showRule bool) *SeverityInfo {
	info := &SeverityInfo{}

	if r.Rule != nil {
		switch r.Rule.Severity {
		case model.SeverityError:
			info.Icon = "✗"
			info.Text = "error"
			info.Color = color.ASCIIRed
		case model.SeverityWarn:
			info.Icon = "▲"
			info.Text = "warning"
			info.Color = color.ASCIIYellow
		case model.SeverityInfo:
			info.Icon = "●"
			info.Text = "info"
			info.Color = color.ASCIIBlue
		default:
			info.Icon = "●"
			info.Text = string(r.Rule.Severity)
			info.Color = color.ASCIIBlue
		}
	} else {
		info.Icon = "●"
		info.Text = "info"
		info.Color = color.ASCIIBlue
	}

	// format based on display mode
	if !showRule {
		// narrow mode - just the colored symbol
		info.Formatted = fmt.Sprintf("%s%-2s%s", info.Color, info.Icon, color.ASCIIReset)
	} else {
		// normal mode - symbol and text
		paddedText := fmt.Sprintf("%s %-7s", info.Icon, info.Text)
		info.Formatted = fmt.Sprintf("%s%s%s", info.Color, paddedText, color.ASCIIReset)
	}

	return info
}

// printFileHeader prints the file header
func printFileHeader(fileName string, silent bool) {
	if silent {
		return
	}

	abs, _ := filepath.Abs(fileName)
	displayPath := abs
	if cwd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(cwd, abs); err == nil {
			displayPath = relPath
		}
	}

	// get terminal width and calculate table width
	termWidth := getTerminalWidth()
	widths := calculateColumnWidths(termWidth)

	// calculate actual table width (matching the summary table)
	// for full width: rule (40) + violation (12) + impact (50) + separators (4 spaces) + leading space (1) = 107
	tableWidth := widths.rule + widths.violation + widths.impact + 4 + 1
	if termWidth < 100 {
		// for smaller terminals, adjust table width accordingly
		tableWidth = termWidth - 13 // leave some margin
	}

	// use the same nice formatting as multi-file
	noStyle := color.AreColorsDisabled()
	if !noStyle {
		fmt.Printf("\n %s%s>%s %s%s%s\n", color.ASCIIBlue, color.ASCIIBold, color.ASCIIReset, color.ASCIIBlue, displayPath, color.ASCIIReset)
		fmt.Printf(" %s%s%s\n\n", color.ASCIIPink, strings.Repeat("-", tableWidth-1), color.ASCIIReset)
	} else {
		fmt.Printf("\n > %s\n", displayPath)
		fmt.Printf(" %s\n\n", strings.Repeat("-", tableWidth-1))
	}
}

// printTableHeaders prints the table headers based on configuration
func printTableHeaders(config *TableConfig) {
	fmt.Printf("%s%s", color.ASCIIPink, color.ASCIIBold)

	printColumns := []struct {
		show  bool
		width int
		title string
	}{
		{true, config.LocationWidth, "Location"},
		{true, config.SeverityWidth, getSeverityHeaderText(config)},
		{!config.NoMessage, config.MessageWidth, "Message"},
		{config.ShowRule, config.RuleWidth, "Rule"},
		{config.ShowCategory, config.CategoryWidth, "Category"},
		{config.ShowPath, config.PathWidth, "Path"},
	}

	first := true
	for _, col := range printColumns {
		if !col.show {
			continue
		}
		if !first {
			fmt.Print("  ")
		}
		fmt.Printf("%-*s", col.width, col.title)
		first = false
	}

	fmt.Printf("%s\n", color.ASCIIReset)
}

// getSeverityHeaderText returns the header text for severity column
func getSeverityHeaderText(config *TableConfig) string {
	if !config.ShowRule {
		return "" // no header for symbol-only mode
	}
	return "Severity"
}

// printTableSeparator prints the separator line
func printTableSeparator(config *TableConfig) {
	fmt.Printf("%s%s", color.ASCIIPink, color.ASCIIBold)

	printColumns := []struct {
		show  bool
		width int
	}{
		{true, config.LocationWidth},
		{true, config.SeverityWidth},
		{!config.NoMessage, config.MessageWidth},
		{config.ShowRule, config.RuleWidth},
		{config.ShowCategory, config.CategoryWidth},
		{config.ShowPath, config.PathWidth},
	}

	first := true
	for _, col := range printColumns {
		if !col.show {
			continue
		}
		if !first {
			fmt.Print("  ")
		}
		fmt.Print(strings.Repeat("─", col.width))
		first = false
	}

	fmt.Printf("%s\n", color.ASCIIReset)
}

// renderTreeFormat renders results in tree format for narrow terminals
func renderTreeFormat(results []*model.RuleFunctionResult, config *TableConfig, fileName string, errors, allResults bool) {
	for i, r := range results {
		if i > 1000 && !allResults {
			fmt.Printf("%s...%s more violations not rendered%s\n", color.ASCIIRed, humanize.Comma(int64(len(results)-1000)), color.ASCIIReset)
			break
		}

		if errors && r.Rule != nil && r.Rule.Severity != model.SeverityError {
			continue
		}

		location := formatLocation(r, fileName)
		coloredLocation := color.ColorizeLocation(location)
		severity := getSeverityInfo(r, false)

		// location line with severity
		fmt.Printf("%s  %s%s %s%s\n", coloredLocation, severity.Color, severity.Icon, severity.Text, color.ASCIIReset)

		// message line with truncation
		maxMsgWidth := config.Width - 4
		message := r.Message
		if len(message) > maxMsgWidth && maxMsgWidth > 3 {
			message = message[:maxMsgWidth-3] + "..."
		}
		coloredMessage := color.ColorizeMessage(message)
		fmt.Printf(" %s├─%s %s\n", color.ASCIIGrey, color.ASCIIReset, coloredMessage)

		// rule and category line
		ruleId := ""
		category := ""
		if r.Rule != nil {
			ruleId = r.Rule.Id
			if r.Rule.RuleCategory != nil {
				category = r.Rule.RuleCategory.Name
			}
		}

		ruleCatLine := ""
		if ruleId != "" && category != "" {
			ruleCatLine = fmt.Sprintf("Rule: %s | Category: %s", ruleId, category)
		} else if ruleId != "" {
			ruleCatLine = fmt.Sprintf("Rule: %s", ruleId)
		} else if category != "" {
			ruleCatLine = fmt.Sprintf("Category: %s", category)
		}

		if ruleCatLine != "" {
			maxRuleCatWidth := config.Width - 4
			if len(ruleCatLine) > maxRuleCatWidth && maxRuleCatWidth > 3 {
				ruleCatLine = ruleCatLine[:maxRuleCatWidth-3] + "..."
			}
			fmt.Printf(" %s├─%s %s\n", color.ASCIIGrey, color.ASCIIReset, ruleCatLine)
		}

		// path line
		if r.Path != "" {
			maxPathWidth := config.Width - 10
			pathText := r.Path
			if len(pathText) > maxPathWidth && maxPathWidth > 3 {
				pathText = pathText[:maxPathWidth-3] + "..."
			}
			coloredPath := color.ColorizePath(pathText)
			fmt.Printf(" %s└─%s Path: %s%s%s\n", color.ASCIIGrey, color.ASCIIReset, color.ASCIIGrey, coloredPath, color.ASCIIReset)
		}

		fmt.Println()
	}
}

// renderTableRow renders a single table row
func renderTableRow(r *model.RuleFunctionResult, config *TableConfig, fileName string) {
	location := formatLocation(r, fileName)
	coloredLocation := color.ColorizeLocation(location)

	// truncate message and path if needed
	message := r.Message
	path := r.Path
	if !config.NoClip {
		// Use rune-aware truncation to avoid cutting in the middle of multi-byte characters
		if len(message) > config.MessageWidth && config.MessageWidth > 3 {
			msgRunes := []rune(message)
			if len(msgRunes) > config.MessageWidth-3 {
				message = string(msgRunes[:config.MessageWidth-3]) + "..."
			}
		}
		if len(path) > config.PathWidth && config.PathWidth > 3 {
			pathRunes := []rune(path)
			if len(pathRunes) > config.PathWidth-3 {
				path = string(pathRunes[:config.PathWidth-3]) + "..."
			}
		}
	}

	coloredMessage := color.ColorizeMessage(message)
	coloredPath := ""
	if config.ShowPath {
		truncatedPath := path
		if len(truncatedPath) > config.PathWidth {
			truncatedPath = truncate(truncatedPath, config.PathWidth)
		}
		coloredPath = color.ColorizePath(truncatedPath)
	}

	severity := getSeverityInfo(r, config.ShowRule)

	ruleId := ""
	category := ""
	if r.Rule != nil {
		ruleId = r.Rule.Id
		if r.Rule.RuleCategory != nil {
			category = r.Rule.RuleCategory.Name
		}
	}

	// calculate padding for colored fields
	locPadding := config.LocationWidth - color.VisibleLength(coloredLocation)
	if locPadding < 0 {
		locPadding = 0
	}

	// After truncation, the message should already be at or under MessageWidth
	// So we calculate padding based on the actual visible length
	msgPadding := config.MessageWidth - color.VisibleLength(coloredMessage)
	if msgPadding < 0 {
		msgPadding = 0
	}

	pathPadding := 0
	if config.ShowPath {
		pathPadding = config.PathWidth - color.VisibleLength(coloredPath)
		if pathPadding < 0 {
			pathPadding = 0
		}
	}

	// build the row output
	fmt.Printf("%s%*s", coloredLocation, locPadding, "")
	fmt.Printf("  %-10s", severity.Formatted)

	if !config.NoMessage {
		fmt.Printf("  %s%*s", coloredMessage, msgPadding, "")
	}

	if config.ShowRule {
		fmt.Printf("  %-*s", config.RuleWidth, ruleId)
	}

	if config.ShowCategory {
		fmt.Printf("  %-*s", config.CategoryWidth, category)
	}

	if config.ShowPath {
		fmt.Printf("  %s%s%*s%s", color.ASCIIGrey, coloredPath, pathPadding, "", color.ASCIIReset)
	}

	fmt.Println()
}

// renderTableFormat renders the results in table format
func renderTableFormat(results []*model.RuleFunctionResult, config *TableConfig,
	fileName string, errors, allResults, snippets bool, specData []string) {

	if !snippets {
		printTableHeaders(config)
		printTableSeparator(config)

		for i, r := range results {
			if i > 1000 && !allResults {
				fmt.Printf("%s...%s more violations not rendered%s\n", color.ASCIIRed, humanize.Comma(int64(len(results)-1000)), color.ASCIIReset)
				break
			}

			if errors && r.Rule != nil && r.Rule.Severity != model.SeverityError {
				continue
			}

			renderTableRow(r, config, fileName)
		}

		if !config.NoStyle {
			fmt.Println()
		}
	} else {
		// snippets mode - render each result with its code snippet
		printTableHeaders(config)
		printTableSeparator(config)

		for i, r := range results {
			if i > 1000 && !allResults {
				fmt.Printf("%s...%s more violations not rendered%s\n", color.ASCIIRed, humanize.Comma(int64(len(results)-1000)), color.ASCIIReset)
				break
			}

			if errors && r.Rule != nil && r.Rule.Severity != model.SeverityError {
				continue
			}

			renderTableRow(r, config, fileName)

			renderCodeSnippetWithHighlight(r, specData, fileName)
			// Only add blank line after snippet if colors are enabled
			if !config.NoStyle {
				fmt.Println()
			}
		}
	}
}

// calculateCodeSnippetHighlightWidth calculates the width for the highlighted code line
// to match the summary table width
func calculateCodeSnippetHighlightWidth(lineNumWidth int) int {
	width := getTerminalWidth()

	var summaryTableWidth int
	if width < 60 {
		// very narrow terminal - use minimum table width
		summaryTableWidth = 40
	} else if width < 80 {
		summaryTableWidth = SummaryTableWidthSmall
	} else if width < 100 {
		summaryTableWidth = SummaryTableWidthMedium
	} else {
		summaryTableWidth = SummaryTableWidthFull
	}

	highlightWidth := summaryTableWidth - lineNumWidth - 3

	// minimum width
	if highlightWidth < 40 {
		highlightWidth = 40
	}

	return highlightWidth
}

// renderCodeSnippetWithHighlight renders a code snippet around the error line with syntax highlighting
func renderCodeSnippetWithHighlight(r *model.RuleFunctionResult, specData []string, fileName string) {
	tui.InitSyntaxStyles()

	if specData == nil {
		return
	}

	// get the target line from either StartNode or Origin
	targetLine := 0
	if r.StartNode != nil {
		targetLine = r.StartNode.Line
	}
	if r.Origin != nil {
		targetLine = r.Origin.Line
	}
	if targetLine <= 0 || targetLine > len(specData) {
		return
	}

	isYAML := strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")

	// show 2 lines before and after the error line
	startLine := targetLine - 2
	endLine := targetLine + 2

	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(specData) {
		endLine = len(specData)
	}

	// calculate line number width
	lineNumWidth := len(fmt.Sprintf("%d", endLine))
	if lineNumWidth < 4 {
		lineNumWidth = 4
	}

	// calculate highlight width to match summary table width
	highlightWidth := calculateCodeSnippetHighlightWidth(lineNumWidth)

	// Only add blank line before snippet if colors are enabled
	if color.ASCIIReset != "" {
		fmt.Println()
	}

	// render the code snippet
	for i := startLine; i <= endLine; i++ {
		lineNum := fmt.Sprintf("%*d", lineNumWidth, i)
		line := ""
		if i-1 >= 0 && i-1 < len(specData) {
			line = specData[i-1]
		}

		// apply syntax highlighting
		highlightedLine := tui.ApplySyntaxHighlightingToLine(line, isYAML)

		// highlight the error line
		if i == targetLine {
			// error line with pink arrow, bold line number, and full-line background
			// use dynamic width format string for padding
			fmt.Printf("%s%s%s %s%s▶%s \033[48;5;53m%s%s%-*s%s\n",
				color.ASCIIPink,
				color.ASCIIBold,
				lineNum,
				color.ASCIIReset,
				color.ASCIIPink,
				color.ASCIIReset,
				color.ASCIIPink,
				color.ASCIIBold,
				highlightWidth,
				line, // use raw line instead of highlighted to avoid color conflicts
				color.ASCIIReset)
		} else {
			// normal line
			fmt.Printf("%s%s %s│%s %s\n",
				color.ASCIIGrey,
				lineNum,
				color.ASCIIGrey,
				color.ASCIIReset,
				highlightedLine)
		}
	}
}
