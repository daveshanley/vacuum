// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"golang.org/x/term"
)

// TableConfig holds all table configuration and column widths
type TableConfig struct {
	Width          int
	ShowCategory   bool
	ShowPath       bool
	ShowRule       bool
	UseTreeFormat  bool
	LocationWidth  int
	SeverityWidth  int
	MessageWidth   int
	RuleWidth      int
	CategoryWidth  int
	PathWidth      int
	NoMessage      bool
	NoClip         bool
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
func calculateTableConfig(results []*model.RuleFunctionResult, fileName string, errors, noMessage, noClip bool) *TableConfig {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 120
	}

	config := &TableConfig{
		Width:         width,
		ShowCategory:  true,
		ShowPath:      true,
		ShowRule:      true,
		UseTreeFormat: false,
		NoMessage:     noMessage,
		NoClip:        noClip,
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
	if config.NoMessage && config.ShowPath {
		config.PathWidth = config.MessageWidth + config.PathWidth + 2
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
			info.Color = cui.ASCIIRed
		case model.SeverityWarn:
			info.Icon = "▲"
			info.Text = "warning"
			info.Color = cui.ASCIIYellow
		case model.SeverityInfo:
			info.Icon = "●"
			info.Text = "info"
			info.Color = cui.ASCIIBlue
		default:
			info.Icon = "●"
			info.Text = string(r.Rule.Severity)
			info.Color = cui.ASCIIBlue
		}
	} else {
		info.Icon = "●"
		info.Text = "info"
		info.Color = cui.ASCIIBlue
	}

	// format based on display mode
	if !showRule {
		// narrow mode - just the colored symbol
		info.Formatted = fmt.Sprintf("%s%-2s%s", info.Color, info.Icon, cui.ASCIIReset)
	} else {
		// normal mode - symbol and text
		paddedText := fmt.Sprintf("%s %-7s", info.Icon, info.Text)
		info.Formatted = fmt.Sprintf("%s%s%s", info.Color, paddedText, cui.ASCIIReset)
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

	fmt.Printf("\n%s%s%s\n", cui.ASCIIPink, displayPath, cui.ASCIIReset)
	fmt.Println(strings.Repeat("-", len(displayPath)))
	fmt.Println()
}

// printTableHeaders prints the table headers based on configuration
func printTableHeaders(config *TableConfig) {
	fmt.Printf("%s%s", cui.ASCIIPink, cui.ASCIIBold)
	
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
	
	fmt.Printf("%s\n", cui.ASCIIReset)
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
	fmt.Printf("%s%s", cui.ASCIIPink, cui.ASCIIBold)
	
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
	
	fmt.Printf("%s\n", cui.ASCIIReset)
}

// renderTreeFormat renders results in tree format for narrow terminals
func renderTreeFormat(results []*model.RuleFunctionResult, config *TableConfig, fileName string, errors, allResults bool) {
	for i, r := range results {
		if i > 1000 && !allResults {
			fmt.Printf("%s...%d more violations not rendered%s\n", cui.ASCIIRed, len(results)-1000, cui.ASCIIReset)
			break
		}

		if errors && r.Rule != nil && r.Rule.Severity != model.SeverityError {
			continue
		}

		location := formatLocation(r, fileName)
		coloredLocation := cui.ColorizeLocation(location)
		severity := getSeverityInfo(r, false)

		// location line with severity
		fmt.Printf("%s  %s%s %s%s\n", coloredLocation, severity.Color, severity.Icon, severity.Text, cui.ASCIIReset)

		// message line with truncation
		maxMsgWidth := config.Width - 4
		message := r.Message
		if len(message) > maxMsgWidth && maxMsgWidth > 3 {
			message = message[:maxMsgWidth-3] + "..."
		}
		coloredMessage := cui.ColorizeMessage(message)
		fmt.Printf(" %s├─%s %s\n", cui.ASCIIGrey, cui.ASCIIReset, coloredMessage)

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
			fmt.Printf(" %s├─%s %s\n", cui.ASCIIGrey, cui.ASCIIReset, ruleCatLine)
		}

		// path line
		if r.Path != "" {
			maxPathWidth := config.Width - 10
			pathText := r.Path
			if len(pathText) > maxPathWidth && maxPathWidth > 3 {
				pathText = pathText[:maxPathWidth-3] + "..."
			}
			coloredPath := cui.ColorizePath(pathText)
			fmt.Printf(" %s└─%s Path: %s%s%s\n", cui.ASCIIGrey, cui.ASCIIReset, cui.ASCIIGrey, coloredPath, cui.ASCIIReset)
		}

		fmt.Println()
	}
}

// renderTableRow renders a single table row
func renderTableRow(r *model.RuleFunctionResult, config *TableConfig, fileName string) {
	location := formatLocation(r, fileName)
	coloredLocation := cui.ColorizeLocation(location)
	
	// truncate message and path if needed
	message := r.Message
	path := r.Path
	if !config.NoClip {
		if len(message) > config.MessageWidth && config.MessageWidth > 3 {
			message = message[:config.MessageWidth-3] + "..."
		}
		if len(path) > config.PathWidth && config.PathWidth > 3 {
			path = path[:config.PathWidth-3] + "..."
		}
	}
	
	coloredMessage := cui.ColorizeMessage(message)
	coloredPath := ""
	if config.ShowPath {
		truncatedPath := path
		if len(truncatedPath) > config.PathWidth {
			truncatedPath = truncate(truncatedPath, config.PathWidth)
		}
		coloredPath = cui.ColorizePath(truncatedPath)
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
	locPadding := config.LocationWidth - cui.VisibleLength(coloredLocation)
	if locPadding < 0 {
		locPadding = 0
	}
	
	msgPadding := config.MessageWidth - cui.VisibleLength(coloredMessage)
	if msgPadding < 0 {
		msgPadding = 0
	}
	
	pathPadding := 0
	if config.ShowPath {
		pathPadding = config.PathWidth - cui.VisibleLength(coloredPath)
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
		fmt.Printf("  %s%s%*s%s", cui.ASCIIGrey, coloredPath, pathPadding, "", cui.ASCIIReset)
	}
	
	fmt.Println()
}

// renderTableFormat renders the results in table format
func renderTableFormat(results []*model.RuleFunctionResult, config *TableConfig, 
	fileName string, errors, allResults, snippets bool) {
	
	if !snippets {
		printTableHeaders(config)
		printTableSeparator(config)
		
		for i, r := range results {
			if i > 1000 && !allResults {
				fmt.Printf("%s...%d more violations not rendered%s\n", cui.ASCIIRed, len(results)-1000, cui.ASCIIReset)
				break
			}
			
			if errors && r.Rule != nil && r.Rule.Severity != model.SeverityError {
				continue
			}
			
			renderTableRow(r, config, fileName)
		}
		
		fmt.Println()
	}
}