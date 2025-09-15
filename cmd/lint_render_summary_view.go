// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/v2/progress"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/list"
	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/dustin/go-humanize"
	"golang.org/x/term"
)

// column widths configuration
type columnWidths struct {
	category    int
	number      int
	rule        int
	violation   int
	impact      int
	fullHeaders bool
}

// table header configuration
type tableHeader struct {
	label string
	color string
	width int
}

// rule violation data
type ruleViolation struct {
	ruleId string
	count  int
}

// summary totals
type summaryTotals struct {
	errors   int
	warnings int
	info     int
}

// get terminal width with fallback
func getTerminalWidth() int {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 120
	}

	// In no-style mode, reduce width by 3 to avoid off-by-one truncation issues
	if cui.AreColorsDisabled() && width > 3 {
		width = width - 3
	}

	return width
}

// calculate responsive column widths based on terminal width
func calculateColumnWidths(width int) columnWidths {
	if width < 60 {
		return columnWidths{
			category:    10,
			number:      5,
			rule:        15,
			violation:   5,
			impact:      15,
			fullHeaders: false,
		}
	} else if width < 80 {
		return columnWidths{
			category:    12,
			number:      7,
			rule:        20,
			violation:   8,
			impact:      20,
			fullHeaders: false,
		}
	} else if width < 100 {
		return columnWidths{
			category:    15,
			number:      9,
			rule:        25,
			violation:   10,
			impact:      30,
			fullHeaders: true,
		}
	}
	// full width - make category table match rules table width
	// rules table: 40 + 12 + 50 + 4 (spaces) + 1 (leading) = 107
	// category table: 20 + 12 + 12 + X + 6 (spaces) + 1 (leading) = 107
	// [X = 107 - 20 - 12 - 12 - 6 - 1 = 56]
	return columnWidths{
		category:    20,
		number:      12,
		rule:        40,
		violation:   12,
		impact:      50,
		fullHeaders: true,
	}
}

// render table separator line
func renderTableSeparator(widths []int) {
	fmt.Printf(" %s", cui.ASCIIPink)
	for i, w := range widths {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Print(strings.Repeat("─", w))
	}
	fmt.Printf("%s\n", cui.ASCIIReset)
}

// render category table headers
func renderCategoryHeaders(widths columnWidths) {
	// calculate extended info width to match rules table
	infoWidth := 56

	headers := []tableHeader{
		{label: "category", color: cui.ASCIIPink, width: widths.category},
	}

	if widths.fullHeaders {
		headers = append(headers,
			tableHeader{label: "✗ errors", color: cui.ASCIIRed, width: widths.number},
			tableHeader{label: "▲ warnings", color: cui.ASCIIYellow, width: widths.number},
			tableHeader{label: "● info", color: cui.ASCIIBlue, width: infoWidth},
		)
	} else {
		headers = append(headers,
			tableHeader{label: "✗ error", color: cui.ASCIIRed, width: widths.number},
			tableHeader{label: "▲ warn", color: cui.ASCIIYellow, width: widths.number},
			tableHeader{label: "● info", color: cui.ASCIIBlue, width: infoWidth},
		)
	}

	fmt.Print(" ") // Add leading space to align with separator and data rows
	for i, h := range headers {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%s%-*s%s", h.color, h.width, h.label, cui.ASCIIReset)
	}
	fmt.Println()

	renderTableSeparator([]int{widths.category, widths.number, widths.number, 56})
}

// render category table row
func renderCategoryRow(name string, errors, warnings, info int, widths columnWidths) {
	// truncate category name if needed
	if len(name) > widths.category {
		name = name[:widths.category-3] + "..."
	}

	// calculate info column width to match rules table
	// rules table total: rule(40) + violation(12) + impact(50) + spaces(4) + leading(1) = 107
	// category table: category(20) + number(12) + number(12) + infoWidth + spaces(6) + leading(1) = 107
	// [infoWidth = 107 - 20 - 12 - 12 - 6 - 1 = 56]
	infoWidth := 56

	fmt.Printf(" %-*s  %-*s  %-*s  %-*s\n",
		widths.category, name,
		widths.number, humanize.Comma(int64(errors)),
		widths.number, humanize.Comma(int64(warnings)),
		infoWidth, humanize.Comma(int64(info)))
}

// render category totals row
func renderCategoryTotals(totals summaryTotals, widths columnWidths) {
	renderTableSeparator([]int{widths.category, widths.number, widths.number, 56})

	fmt.Printf(" %s%-*s%s  %s%s%-*s%s  %s%s%-*s%s  %s%s%-*s%s\n",
		cui.ASCIIBold, widths.category, "total", cui.ASCIIReset,
		cui.ASCIIRed, cui.ASCIIBold, widths.number, humanize.Comma(int64(totals.errors)), cui.ASCIIReset,
		cui.ASCIIYellow, cui.ASCIIBold, widths.number, humanize.Comma(int64(totals.warnings)), cui.ASCIIReset,
		cui.ASCIIBlue, cui.ASCIIBold, 56, humanize.Comma(int64(totals.info)), cui.ASCIIReset)
}

// render category summary table
func renderCategoryTable(rs *model.RuleResultSet, cats []*model.RuleCategory, widths columnWidths) summaryTotals {
	renderCategoryHeaders(widths)

	totals := summaryTotals{}

	for _, cat := range cats {
		errors := rs.GetErrorsByRuleCategory(cat.Id)
		warn := rs.GetWarningsByRuleCategory(cat.Id)
		info := rs.GetInfoByRuleCategory(cat.Id)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {
			renderCategoryRow(cat.Name, len(errors), len(warn), len(info), widths)

			totals.errors += len(errors)
			totals.warnings += len(warn)
			totals.info += len(info)
		}
	}

	renderCategoryTotals(totals, widths)
	fmt.Println()

	return totals
}

// build rule violations data
func buildRuleViolations(rs *model.RuleResultSet) []ruleViolation {
	ruleMap := make(map[string]*ruleViolation)

	for _, result := range rs.Results {
		if result.Rule != nil {
			if _, exists := ruleMap[result.Rule.Id]; !exists {
				ruleMap[result.Rule.Id] = &ruleViolation{
					ruleId: result.Rule.Id,
				}
			}
			ruleMap[result.Rule.Id].count++
		}
	}

	// convert to slice
	violations := make([]ruleViolation, 0, len(ruleMap))
	for _, rv := range ruleMap {
		violations = append(violations, *rv)
	}

	// sort by count (highest first)
	for i := 0; i < len(violations); i++ {
		for j := i + 1; j < len(violations); j++ {
			if violations[j].count > violations[i].count {
				violations[i], violations[j] = violations[j], violations[i]
			}
		}
	}

	return violations
}

// calculate rule violation stats
func calculateViolationStats(violations []ruleViolation) (total, max int) {
	for _, rv := range violations {
		total += rv.count
		if rv.count > max {
			max = rv.count
		}
	}
	return
}

// render rule violations table headers
func renderRuleHeaders(widths columnWidths) {
	fmt.Printf(" %s%-*s%s  %s%-*s%s  %s%-*s%s\n",
		cui.ASCIIPink, widths.rule, "rule", cui.ASCIIReset,
		cui.ASCIIPink, widths.violation, "violations", cui.ASCIIReset,
		cui.ASCIIPink, widths.impact, "quality impact", cui.ASCIIReset)

	renderTableSeparator([]int{widths.rule, widths.violation, widths.impact})
}

// render rule violation row
func renderRuleRow(rv ruleViolation, widths columnWidths, percentage float64, prog progress.Model) {
	// truncate rule name if needed
	ruleName := rv.ruleId
	if len(ruleName) > widths.rule {
		ruleName = ruleName[:widths.rule-3] + "..."
	}

	// render plain progress bar if colors are disabled
	if cui.AreColorsDisabled() {
		barLength := int(float64(widths.impact) * percentage)
		if barLength < 0 {
			barLength = 0
		}
		if barLength > widths.impact {
			barLength = widths.impact
		}
		bar := strings.Repeat("█", barLength) + strings.Repeat(" ", widths.impact-barLength)
		fmt.Printf(" %-*s  %-*s  %s\n",
			widths.rule, ruleName,
			widths.violation, humanize.Comma(int64(rv.count)),
			bar)
	} else {
		fmt.Printf(" %-*s  %-*s  %s\n",
			widths.rule, ruleName,
			widths.violation, humanize.Comma(int64(rv.count)),
			prog.ViewAs(percentage))
	}
}

// render rule violations totals
func renderRuleTotals(total int, widths columnWidths) {
	renderTableSeparator([]int{widths.rule, widths.violation, widths.impact})

	fmt.Printf(" %s%-*s%s  %s%s%-*s%s\n",
		cui.ASCIIBold, widths.rule, "total", cui.ASCIIReset,
		cui.ASCIIPink, cui.ASCIIBold,
		widths.violation, humanize.Comma(int64(total)),
		cui.ASCIIReset)
}

// render rule violations table
func renderRuleViolationsTable(violations []ruleViolation, widths columnWidths) {
	if len(violations) == 0 {
		return
	}

	total, max := calculateViolationStats(violations)

	// create progress bar
	var prog progress.Model
	if cui.AreColorsDisabled() {
		// Use grey to white gradient in no-style mode
		prog = progress.New(
			progress.WithScaledGradient("#606060", "#ffffff"),
			progress.WithWidth(widths.impact),
			progress.WithoutPercentage(),
			progress.WithFillCharacters('█', ' '),
		)
	} else {
		prog = progress.New(
			progress.WithScaledGradient("#62c4ff", "#f83aff"),
			progress.WithWidth(widths.impact),
			progress.WithoutPercentage(),
			progress.WithFillCharacters('█', ' '),
		)
	}

	renderRuleHeaders(widths)

	// show top 10 rules
	maxRules := 10
	if len(violations) < maxRules {
		maxRules = len(violations)
	}

	for i := 0; i < maxRules; i++ {
		percentage := float64(violations[i].count) / float64(max)
		renderRuleRow(violations[i], widths, percentage, prog)
	}

	renderRuleTotals(total, widths)

	if len(violations) > maxRules {
		fmt.Printf(" %s... and %s more rules%s\n", cui.ASCIIGrey, humanize.Comma(int64(len(violations)-maxRules)), cui.ASCIIReset)
	}
	fmt.Println()
}

// create result box style
func createResultBoxStyle(foreground, background color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(foreground).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeftForeground(foreground).
		BorderLeftBackground(background).
		BorderTop(false).
		Bold(true).
		BorderBottom(false).
		BorderLeft(true).
		Padding(0, 0, 0, 0).
		MarginLeft(1)
}

// render result box
func renderResultBox(errors, warnings, informs int) {
	if cui.AreColorsDisabled() {
		if errors > 0 {
			fmt.Printf(" | \u2717 Failed with %s errors, %s warnings and %s informs.\n",
				humanize.Comma(int64(errors)), humanize.Comma(int64(warnings)), humanize.Comma(int64(informs)))
		} else if warnings > 0 {
			fmt.Printf(" | \u25B2 Passed, but with %s warnings and %s informs.\n",
				humanize.Comma(int64(warnings)), humanize.Comma(int64(informs)))
		} else if informs > 0 {
			fmt.Printf(" | \u25CF Passed, with %s informs.\n", humanize.Comma(int64(informs)))
		} else {
			fmt.Println(" | \u2713 A perfect score! Like Mary Poppins, practically perfect in every way. Incredible, well done!")
		}
		fmt.Println()
		return
	}

	messageStyle := lipgloss.NewStyle().Padding(1, 1)

	if errors > 0 {
		message := fmt.Sprintf("\u2717 Failed with %s errors, %s warnings and %s informs.",
			humanize.Comma(int64(errors)), humanize.Comma(int64(warnings)), humanize.Comma(int64(informs)))
		style := createResultBoxStyle(cui.RGBRed, cui.RGBDarkRed)
		fmt.Println(style.Render(messageStyle.Render(message)))
	} else if warnings > 0 {
		message := fmt.Sprintf("\u25B2 Passed, but with %s warnings and %s informs.",
			humanize.Comma(int64(warnings)), humanize.Comma(int64(informs)))
		style := createResultBoxStyle(cui.RBGYellow, cui.RGBDarkYellow)
		fmt.Println(style.Render(messageStyle.Render(message)))
	} else if informs > 0 {
		message := fmt.Sprintf("\u25CF Passed, with %s informs.", humanize.Comma(int64(informs)))
		style := createResultBoxStyle(cui.RGBBlue, cui.RGBDarkBlue)
		fmt.Println(style.Render(messageStyle.Render(message)))
	} else {
		message := "\u2713 A perfect score! Like Mary Poppins, practically perfect in every way. Incredible, well done!"
		style := createResultBoxStyle(cui.RGBGreen, cui.RGBDarkGreen)
		fmt.Println(style.Render(messageStyle.Render(message)))
	}
	fmt.Println()
}

// render quality score box
func renderQualityScore(score int) {
	var boxType cui.BoxType
	var grade string

	switch {
	case score > 95:
		boxType = cui.BoxTypeSuccess
		grade = "A+"
	case score > 90 && score <= 95:
		boxType = cui.BoxTypeSuccess
		grade = "A"
	case score > 85 && score <= 90:
		boxType = cui.BoxTypeSuccess
		grade = "B"
	case score > 75 && score <= 85:
		boxType = cui.BoxTypeInfo
		grade = "C"
	case score > 65 && score <= 75:
		boxType = cui.BoxTypeWarning
		grade = "D"
	case score > 55 && score <= 65:
		boxType = cui.BoxTypeWarning
		grade = "F"
	case score > 25 && score <= 55:
		boxType = cui.BoxTypeError
		grade = "🤒"
	case score >= 10 && score <= 25:
		boxType = cui.BoxTypeError
		grade = "🥵"
	case score >= 5 && score < 10:
		boxType = cui.BoxTypeError
		grade = "😵"
	case score >= 1 && score < 5:
		boxType = cui.BoxTypeError
		grade = "💀"
	default:
		boxType = cui.BoxTypeError
		grade = "💀"
	}

	message := fmt.Sprintf("Quality Score: %d/100 [%s]", score, grade)
	cui.RenderStyledBox(message, boxType, cui.AreColorsDisabled())
}

// renderRulesList renders the list of rules using lipgloss list
func renderRulesList(rules map[string]*model.Rule) {
	fmt.Println(" The following rules are going to be used:")
	fmt.Println()

	// Sort rules for consistent output
	var sortedRules []*model.Rule
	for _, rule := range rules {
		sortedRules = append(sortedRules, rule)
	}
	// Sort by rule ID for consistency
	for i := 0; i < len(sortedRules); i++ {
		for j := i + 1; j < len(sortedRules); j++ {
			if sortedRules[i].Id > sortedRules[j].Id {
				sortedRules[i], sortedRules[j] = sortedRules[j], sortedRules[i]
			}
		}
	}

	// Create list
	l := list.New()

	// Custom enumerator with blue numbered bullets
	l.Enumerator(func(items list.Items, i int) string {
		// Right-align numbers with padding and add space after
		numStr := fmt.Sprintf("[%d] ", i+1)
		if i+1 < 10 {
			numStr = fmt.Sprintf("  [%d] ", i+1)
		} else if i+1 < 100 {
			numStr = fmt.Sprintf(" [%d] ", i+1)
		}
		return numStr
	})

	// Style the enumerator in blue
	l.EnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("45"))) // Blue

	// Add items with styled rule names using ANSI codes directly
	for _, rule := range sortedRules {
		// Format: bold pink rule ID + normal description
		// Using ANSI codes directly: bold pink for rule ID, reset for description
		formattedItem := fmt.Sprintf("%s%s%s%s: %s",
			cui.ASCIIBold,
			cui.ASCIIPink,
			rule.Id,
			cui.ASCIIReset,
			rule.Description)
		l.Item(formattedItem)
	}

	// Set indentation
	l.Indenter(func(list.Items, int) string { return "  " })

	fmt.Println(l)
	fmt.Println()
}
