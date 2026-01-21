// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"sort"
	"strings"

	color2 "github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
)

// renderChangeFilterSummary displays what the change filter removed
func renderChangeFilterSummary(stats *utils.ChangeFilterStats, widths columnWidths, noStyle bool) {
	if stats == nil || stats.ResultsDropped == 0 {
		return
	}

	renderChangeFilterHeader(widths, noStyle)
	renderChangeFilterTotals(stats, widths, noStyle)

	if len(stats.RulesFullyFiltered) > 0 {
		renderFullyFilteredRules(stats.RulesFullyFiltered, noStyle)
	}

	fmt.Println()
}

// renderChangeFilterHeader renders the header for the change filter summary
func renderChangeFilterHeader(widths columnWidths, noStyle bool) {
	if noStyle {
		fmt.Println(" change filter summary")
		fmt.Printf(" %s\n", strings.Repeat("-", widths.rule+widths.violation+widths.impact+4))
	} else {
		fmt.Printf(" %s%schange filter summary%s\n", color2.ASCIIPink, color2.ASCIIBold, color2.ASCIIReset)
		renderTableSeparator([]int{widths.rule + widths.violation + widths.impact + 4})
	}
}

// renderChangeFilterTotals renders the totals of filtered results
func renderChangeFilterTotals(stats *utils.ChangeFilterStats, widths columnWidths, noStyle bool) {
	percentage := stats.GetDroppedPercentage()

	filteredLabel := "results filtered"
	rulesLabel := "rules fully removed"

	filteredValue := fmt.Sprintf("%s of %s (%d%%)",
		humanize.Comma(int64(stats.ResultsDropped)),
		humanize.Comma(int64(stats.TotalResultsBefore)),
		percentage)

	if noStyle {
		fmt.Printf(" %-20s %s\n", filteredLabel, filteredValue)
		if len(stats.RulesFullyFiltered) > 0 {
			fmt.Printf(" %-20s %s\n", rulesLabel, humanize.Comma(int64(len(stats.RulesFullyFiltered))))
		}
		fmt.Printf(" %s\n", strings.Repeat("-", widths.rule+widths.violation+widths.impact+4))
	} else {
		fmt.Printf(" %s%-20s%s %s%s%s\n",
			color2.ASCIIGrey, filteredLabel, color2.ASCIIReset,
			color2.ASCIIBold, filteredValue, color2.ASCIIReset)
		if len(stats.RulesFullyFiltered) > 0 {
			fmt.Printf(" %s%-20s%s %s%s%s\n",
				color2.ASCIIGrey, rulesLabel, color2.ASCIIReset,
				color2.ASCIIBold, humanize.Comma(int64(len(stats.RulesFullyFiltered))), color2.ASCIIReset)
		}
		renderTableSeparator([]int{widths.rule + widths.violation + widths.impact + 4})
	}
}

// renderFullyFilteredRules renders the list of rules where all results were filtered out
func renderFullyFilteredRules(rules []string, noStyle bool) {
	// Sort for consistent output
	sort.Strings(rules)

	if noStyle {
		fmt.Println()
		fmt.Println(" rules with all results filtered:")
		for i, rule := range rules {
			isLast := i == len(rules)-1
			if isLast {
				fmt.Printf("  └─ %s\n", rule)
			} else {
				fmt.Printf("  ├─ %s\n", rule)
			}
		}
	} else {
		fmt.Println()
		fmt.Printf(" %srules with all results filtered:%s\n", color2.ASCIIGrey, color2.ASCIIReset)
		for i, rule := range rules {
			isLast := i == len(rules)-1
			if isLast {
				fmt.Printf(" %s└─%s %s%s%s\n", color2.ASCIIPink, color2.ASCIIReset, color2.ASCIIBold, rule, color2.ASCIIReset)
			} else {
				fmt.Printf(" %s├─%s %s%s%s\n", color2.ASCIIPink, color2.ASCIIReset, color2.ASCIIBold, rule, color2.ASCIIReset)
			}
		}
	}
}
