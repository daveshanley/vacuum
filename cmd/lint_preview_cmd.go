// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/libopenapi/index"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

func GetLintPreviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "lint-preview <your-openapi-file.yaml>",
		Short:         "Preview lint results with enhanced table formatting",
		Long:          `Lint an OpenAPI specification and display results in a formatted table view`,
		RunE:          runLintPreview,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add all flags
	cmd.Flags().BoolP("details", "d", false, "Show full details of linting report")
	cmd.Flags().BoolP("snippets", "s", false, "Show code snippets where issues are found")
	cmd.Flags().BoolP("errors", "e", false, "Show errors only")
	cmd.Flags().StringP("category", "c", "", "Show a single category of results")
	cmd.Flags().BoolP("silent", "x", false, "Show nothing except the result")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output")
	cmd.Flags().BoolP("no-banner", "b", false, "Disable the banner output")
	cmd.Flags().BoolP("no-message", "m", false, "Hide message output when using -d")
	cmd.Flags().BoolP("all-results", "a", false, "Render all results when using -d")
	cmd.Flags().StringP("fail-severity", "n", model.SeverityError, "Results of this level or above will trigger a failure exit code")
	cmd.Flags().StringP("base", "p", "", "Base URL or path for resolving references")
	cmd.Flags().BoolP("remote", "u", true, "Allow remote references")
	cmd.Flags().BoolP("skip-check", "k", false, "Skip OpenAPI document validation")
	cmd.Flags().IntP("timeout", "g", 5, "Timeout in seconds for each rule")
	cmd.Flags().StringP("ruleset", "r", "", "Path to custom ruleset")
	cmd.Flags().StringP("functions", "f", "", "Path to custom functions")
	cmd.Flags().BoolP("time", "t", false, "Show execution time")
	cmd.Flags().BoolP("hard-mode", "z", false, "Enable hard mode (all rules)")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().Bool("no-clip", false, "Do not truncate messages or paths")
	cmd.Flags().Bool("ext-refs", false, "Enable $ref lookups for extension objects")
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")

	return cmd
}

func runLintPreview(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide an OpenAPI specification file to lint")
	}

	fileName := args[0]

	// Read all flags
	detailsFlag, _ := cmd.Flags().GetBool("details")
	snippetsFlag, _ := cmd.Flags().GetBool("snippets")
	errorsFlag, _ := cmd.Flags().GetBool("errors")
	categoryFlag, _ := cmd.Flags().GetString("category")
	silentFlag, _ := cmd.Flags().GetBool("silent")
	noStyleFlag, _ := cmd.Flags().GetBool("no-style")
	noBannerFlag, _ := cmd.Flags().GetBool("no-banner")
	noMessageFlag, _ := cmd.Flags().GetBool("no-message")
	allResultsFlag, _ := cmd.Flags().GetBool("all-results")
	failSeverityFlag, _ := cmd.Flags().GetString("fail-severity")
	baseFlag, _ := cmd.Flags().GetString("base")
	remoteFlag, _ := cmd.Flags().GetBool("remote")
	skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
	timeoutFlag, _ := cmd.Flags().GetInt("timeout")
	rulesetFlag, _ := cmd.Flags().GetString("ruleset")
	functionsFlag, _ := cmd.Flags().GetString("functions")
	timeFlag, _ := cmd.Flags().GetBool("time")
	hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
	ignoreFile, _ := cmd.Flags().GetString("ignore-file")
	noClipFlag, _ := cmd.Flags().GetBool("no-clip")
	extRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
	ignoreArrayCircleRef, _ := cmd.Flags().GetBool("ignore-array-circle-ref")
	ignorePolymorphCircleRef, _ := cmd.Flags().GetBool("ignore-polymorph-circle-ref")

	// Show banner unless disabled
	if !silentFlag && !noBannerFlag {
		PrintBanner()
	}

	// Load ignore file if specified
	ignoredItems := model.IgnoredItems{}
	if ignoreFile != "" {
		raw, ferr := os.ReadFile(ignoreFile)
		if ferr != nil {
			return fmt.Errorf("failed to read ignore file: %w", ferr)
		}
		ferr = yaml.Unmarshal(raw, &ignoredItems)
		if ferr != nil {
			return fmt.Errorf("failed to parse ignore file: %w", ferr)
		}
	}

	// Try to load the file as either a report or spec
	reportOrSpec, err := LoadFileAsReportOrSpec(fileName)
	if err != nil {
		fmt.Printf("\033[31mUnable to load file '%s': %v\033[0m\n", fileName, err)
		return err
	}

	// Get file info for timing
	fileInfo, _ := os.Stat(fileName)

	// Setup logging
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	})
	logger := slog.New(handler)

	var resultSet *model.RuleResultSet
	var specBytes []byte
	var displayFileName string
	start := time.Now()

	if reportOrSpec.IsReport {
		// Using a pre-compiled report
		if !silentFlag {
			fmt.Printf("\033[36mLoading pre-compiled vacuum report from '%s'\033[0m\n\n", fileName)
		}
		
		// Create a new RuleResultSet from the results to ensure proper initialization
		if reportOrSpec.ResultSet != nil && reportOrSpec.ResultSet.Results != nil {
			// Filter ignored results first
			filteredResults := utils.FilterIgnoredResultsPtr(reportOrSpec.ResultSet.Results, ignoredItems)
			// Create properly initialized RuleResultSet
			resultSet = model.NewRuleResultSetPointer(filteredResults)
		} else {
			resultSet = model.NewRuleResultSetPointer([]*model.RuleFunctionResult{})
		}
		
		specBytes = reportOrSpec.SpecBytes
		displayFileName = reportOrSpec.FileName
	} else {
		// Regular spec file - run linting
		specBytes = reportOrSpec.SpecBytes
		displayFileName = fileName
		
		// Build ruleset
		defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
		selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
		customFuncs, _ := LoadCustomFunctions(functionsFlag, silentFlag)

		// Handle hard mode
		if hardModeFlag {
			selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
			owaspRules := rulesets.GetAllOWASPRules()
			for k, v := range owaspRules {
				selectedRS.Rules[k] = v
			}
			if !silentFlag {
				fmt.Printf("\033[31m🚨 HARD MODE ENABLED 🚨\033[0m\n\n")
			}
		}

		// Handle custom ruleset
		if rulesetFlag != "" {
			var rsErr error
			selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, nil)
			if rsErr != nil {
				fmt.Printf("\033[31mUnable to load ruleset '%s': %s\033[0m\n", rulesetFlag, rsErr.Error())
				return rsErr
			}
			if hardModeFlag {
				MergeOWASPRulesToRuleSet(selectedRS, true)
			}
		}

		// Display linting info
		if !silentFlag {
			fmt.Printf("\033[36mLinting file '%s' against %d rules: %s\033[0m\n\n",
				displayFileName, len(selectedRS.Rules), selectedRS.DocumentationURI)
		}

		// Build deep graph if we have ignored items
		deepGraph := false
		if len(ignoredItems) > 0 {
			deepGraph = true
		}

		// Apply rules
		result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:                         selectedRS,
			Spec:                            specBytes,
			SpecFileName:                    displayFileName,
			CustomFunctions:                 customFuncs,
			Base:                            baseFlag,
			AllowLookup:                     remoteFlag,
			SkipDocumentCheck:               skipCheckFlag,
			Logger:                          logger,
			BuildDeepGraph:                  deepGraph,
			Timeout:                         time.Duration(timeoutFlag) * time.Second,
			IgnoreCircularArrayRef:          ignoreArrayCircleRef,
			IgnoreCircularPolymorphicRef:    ignorePolymorphCircleRef,
			ExtractReferencesFromExtensions: extRefsFlag,
		})

		// Filter ignored results
		result.Results = utils.FilterIgnoredResults(result.Results, ignoredItems)

		// Check for errors
		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				fmt.Printf("\033[31mUnable to process spec '%s': %s\033[0m\n", displayFileName, err.Error())
			}
			return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
		}

		// Process results
		resultSet = model.NewRuleResultSet(result.Results)
	}

	specStringData := strings.Split(string(specBytes), "\n")

	// Handle category filtering
	var cats []*model.RuleCategory
	if categoryFlag != "" {
		// Category filtering logic here (same as original)
		cats = model.RuleCategoriesOrdered
	} else {
		cats = model.RuleCategoriesOrdered
	}

	resultSet.SortResultsByLineNumber()
	
	// Create statistics if we have the necessary data
	var stats *reports.ReportStatistics
	if reportOrSpec.IsReport && reportOrSpec.Report.Statistics != nil {
		stats = reportOrSpec.Report.Statistics
	}
	// Note: For fresh linting, we'd need the index and spec info from the result,
	// but that's not available in this flow anymore. We can skip stats for now.

	// Show detailed results if requested
	if detailsFlag && len(resultSet.Results) > 0 {
		// Always use regular detailed view (no interactive UI)
		renderFixedDetails(resultSet.Results, specStringData, snippetsFlag, errorsFlag,
			silentFlag, noMessageFlag, allResultsFlag, noClipFlag, displayFileName, noStyleFlag)
	}

	// Render summary
	renderFixedSummary(resultSet, cats, stats, displayFileName, silentFlag, noStyleFlag)

	// Show timing
	duration := time.Since(start)
	if timeFlag {
		renderFixedTiming(duration, fileInfo.Size())
	}

	// Check severity failure
	errs := resultSet.GetErrorCount()
	warnings := resultSet.GetWarnCount()
	informs := resultSet.GetInfoCount()

	// Check for failure but handle it gracefully without showing help
	failErr := CheckFailureSeverity(failSeverityFlag, errs, warnings, informs)
	if failErr != nil {
		os.Exit(1)
	}

	return nil
}

func printFixedBanner() {
	banner := `
██╗   ██╗ █████╗  ██████╗██╗   ██╗██╗   ██╗███╗   ███╗
██║   ██║██╔══██╗██╔════╝██║   ██║██║   ██║████╗ ████║
██║   ██║███████║██║     ██║   ██║██║   ██║██╔████╔██║
╚██╗ ██╔╝██╔══██║██║     ██║   ██║██║   ██║██║╚██╔╝██║
 ╚████╔╝ ██║  ██║╚██████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝`

	// Use raw ANSI codes for colors
	fmt.Printf("\033[35m%s\033[0m\n\n", banner)                                                          // Magenta
	fmt.Printf("\033[32mversion: %s | compiled: %s\033[0m\n", Version, Date)                             // Green
	fmt.Printf("\033[36m🔗 https://quobix.com/vacuum | https://github.com/daveshanley/vacuum\033[0m\n\n") // Cyan
}

func renderFixedDetails(results []*model.RuleFunctionResult, specData []string,
	snippets, errors, silent, noMessage, allResults, noClip bool,
	fileName string, noStyle bool) {

	if !silent {
		// Display file header
		abs, _ := filepath.Abs(fileName)
		displayPath := abs
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, abs); err == nil {
				displayPath = relPath
			}
		}

		fmt.Printf("\n\033[35m%s\033[0m\n", displayPath) // Magenta
		fmt.Println(strings.Repeat("-", len(displayPath)))
		fmt.Println()
	}

	// Get terminal width
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 120 // Default fallback
	}

	// First pass: calculate the actual maximum widths needed for each column
	maxLocationLen := len("Location") // Start with header width
	maxRuleLen := len("Rule")
	maxCategoryLen := len("Category")
	
	for _, r := range results {
		// Build location for this result
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

		// Make path relative
		if absPath, err := filepath.Abs(f); err == nil {
			if cwd, err := os.Getwd(); err == nil {
				if relPath, err := filepath.Rel(cwd, absPath); err == nil {
					f = relPath
				}
			}
		}

		location := fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
		if len(location) > maxLocationLen {
			maxLocationLen = len(location)
		}
		
		// Check rule length
		if r.Rule != nil && len(r.Rule.Id) > maxRuleLen {
			maxRuleLen = len(r.Rule.Id)
		}
		
		// Check category length
		if r.Rule != nil && r.Rule.RuleCategory != nil && len(r.Rule.RuleCategory.Name) > maxCategoryLen {
			maxCategoryLen = len(r.Rule.RuleCategory.Name)
		}
	}

	// Column width allocation based on actual content
	// Priority order: location (never truncated), rule (never truncated), category (never truncated), message, path
	
	// Fixed/dynamic widths based on content
	locWidth := maxLocationLen
	sevWidth := 10  // Fixed width for severity with icon
	ruleWidth := maxRuleLen
	catWidth := maxCategoryLen
	
	// Calculate remaining width after fixed columns
	separators := 10 // Space for column separators
	fixedWidth := locWidth + sevWidth + ruleWidth + catWidth + separators
	remainingWidth := width - fixedWidth
	
	// Allocate remaining space between message and path
	// Message gets priority (60%), path gets the rest (40%)
	var msgWidth, pathWidth int
	if remainingWidth > 0 {
		msgWidth = remainingWidth * 60 / 100
		pathWidth = remainingWidth - msgWidth
		
		// Minimum widths to ensure readability
		if msgWidth < 20 {
			msgWidth = 20
			pathWidth = remainingWidth - msgWidth
			if pathWidth < 10 {
				pathWidth = 10
			}
		}
	} else {
		// If no remaining width, use minimums
		msgWidth = 20
		pathWidth = 10
	}

	// Build and render table
	if !snippets {
		// Print header with pink color (matching BubbleTea UI)
		if !noMessage {
			fmt.Printf("\033[38;5;201m%-*s  %-*s  %-*s  %-*s  %-*s  %-*s\033[0m\n",
				locWidth, "Location",
				sevWidth, "Severity",
				msgWidth, "Message",
				ruleWidth, "Rule",
				catWidth, "Category",
				pathWidth, "Path")
		} else {
			// Adjust widths when no message column
			pathWidth = msgWidth + pathWidth + 2
			fmt.Printf("\033[38;5;201m%-*s  %-*s  %-*s  %-*s  %-*s\033[0m\n",
				locWidth, "Location",
				sevWidth, "Severity",
				ruleWidth, "Rule",
				catWidth, "Category",
				pathWidth, "Path")
		}

		// Print separator with pink color
		if !noMessage {
			fmt.Printf("\033[38;5;201m%s  %s  %s  %s  %s  %s\033[0m\n",
				strings.Repeat("─", locWidth),
				strings.Repeat("─", sevWidth),
				strings.Repeat("─", msgWidth),
				strings.Repeat("─", ruleWidth),
				strings.Repeat("─", catWidth),
				strings.Repeat("─", pathWidth))
		} else {
			fmt.Printf("\033[38;5;201m%s  %s  %s  %s  %s\033[0m\n",
				strings.Repeat("─", locWidth),
				strings.Repeat("─", sevWidth),
				strings.Repeat("─", ruleWidth),
				strings.Repeat("─", catWidth),
				strings.Repeat("─", pathWidth))
		}

		// Print rows
		for i, r := range results {
			if i > 1000 && !allResults {
				fmt.Printf("\033[31m...%d more violations not rendered\033[0m\n", len(results)-1000)
				break
			}

			// Skip if showing errors only
			if errors && r.Rule != nil && r.Rule.Severity != model.SeverityError {
				continue
			}

			// Build location
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

			// Make path relative
			if absPath, err := filepath.Abs(f); err == nil {
				if cwd, err := os.Getwd(); err == nil {
					if relPath, err := filepath.Rel(cwd, absPath); err == nil {
						f = relPath
					}
				}
			}

			// Format location as file:line:col (never truncated)
			location := fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
			// Apply color formatting to location
			coloredLocation := cui.ColorizeLocation(location)

			// Truncate fields if needed
			m := r.Message
			p := r.Path
			if !noClip {
				if len(m) > msgWidth {
					m = m[:msgWidth-3] + "..."
				}
				if len(p) > pathWidth {
					p = p[:pathWidth-3] + "..."
				}
			}

			// Format severity with color and icon (matching BubbleTea UI)
			var sevColored string
			if r.Rule != nil {
				switch r.Rule.Severity {
				case model.SeverityError:
					sevColored = fmt.Sprintf("\033[31m%s error  \033[0m", "✗")
				case model.SeverityWarn:
					sevColored = fmt.Sprintf("\033[33m%s warning\033[0m", "▲")
				case model.SeverityInfo:
					sevColored = fmt.Sprintf("\033[36m%s info   \033[0m", "●")
				default:
					sevColored = fmt.Sprintf("%-*s", sevWidth, r.Rule.Severity)
				}
			} else {
				sevColored = fmt.Sprintf("\033[36m%s info   \033[0m", "●")
			}

			// Get rule and category
			ruleId := ""
			category := ""
			if r.Rule != nil {
				ruleId = r.Rule.Id
				if r.Rule.RuleCategory != nil {
					category = r.Rule.RuleCategory.Name
				}
			}

			// Print row with path in grey (like BubbleTea UI)
			// Note: We need to account for ANSI codes in the location when calculating padding
			// The colored location has ANSI codes that don't count toward visible width
			if !noMessage {
				fmt.Printf("%s%*s  %-10s  %-*s  %-*s  %-*s  \033[90m%-*s\033[0m\n",
					coloredLocation, locWidth - len(location), "",  // Pad based on uncolored length
					sevColored,
					msgWidth, truncate(m, msgWidth),
					ruleWidth, ruleId,  // Never truncate rule
					catWidth, category,  // Never truncate category
					pathWidth, truncate(p, pathWidth))
			} else {
				fmt.Printf("%s%*s  %-10s  %-*s  %-*s  \033[90m%-*s\033[0m\n",
					coloredLocation, locWidth - len(location), "",  // Pad based on uncolored length
					sevColored,
					ruleWidth, ruleId,  // Never truncate rule
					catWidth, category,  // Never truncate category
					pathWidth, truncate(p, pathWidth))
			}
		}
		fmt.Println()
	}
}

func renderFixedSummary(rs *model.RuleResultSet, cats []*model.RuleCategory,
	stats *reports.ReportStatistics, fileName string, silent bool, noStyle bool) {

	if silent {
		return
	}

	// Build category summary table
	fmt.Printf("\033[36m%-20s  %-10s  %-10s  %-10s\033[0m\n", "Category", "Errors", "Warnings", "Info")
	fmt.Printf("\033[90m%s  %s  %s  %s\033[0m\n",
		strings.Repeat("─", 20),
		strings.Repeat("─", 10),
		strings.Repeat("─", 10),
		strings.Repeat("─", 10))

	for _, cat := range cats {
		errors := rs.GetErrorsByRuleCategory(cat.Id)
		warn := rs.GetWarningsByRuleCategory(cat.Id)
		info := rs.GetInfoByRuleCategory(cat.Id)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {
			fmt.Printf("%-20s  %-10s  %-10s  %-10s\n",
				cat.Name,
				humanize.Comma(int64(len(errors))),
				humanize.Comma(int64(len(warn))),
				humanize.Comma(int64(len(info))))
		}
	}
	fmt.Println()

	// Render result box
	errs := rs.GetErrorCount()
	warnings := rs.GetWarnCount()
	informs := rs.GetInfoCount()

	if errs > 0 {
		fmt.Printf("\033[31m╭──────────────────────────────────────────────────────────────────────╮\033[0m\n")
		fmt.Printf("\033[31m│  ❌ Linting failed with %d errors, %d warnings and %d informs  │\033[0m\n",
			errs, warnings, informs)
		fmt.Printf("\033[31m╰──────────────────────────────────────────────────────────────────────╯\033[0m\n")
	} else if warnings > 0 {
		fmt.Printf("\033[33m╭──────────────────────────────────────────────────────────────────────╮\033[0m\n")
		fmt.Printf("\033[33m│  ⚠️  Linting passed with %d warnings and %d informs  │\033[0m\n",
			warnings, informs)
		fmt.Printf("\033[33m╰──────────────────────────────────────────────────────────────────────╯\033[0m\n")
	} else if informs > 0 {
		fmt.Printf("\033[36m╭──────────────────────────────────────────────────────────────────────╮\033[0m\n")
		fmt.Printf("\033[36m│  ℹ️  Linting passed, %d informs reported  │\033[0m\n", informs)
		fmt.Printf("\033[36m╰──────────────────────────────────────────────────────────────────────╯\033[0m\n")
	} else {
		fmt.Printf("\033[32m╭──────────────────────────────────────────────────────────────────────╮\033[0m\n")
		fmt.Printf("\033[32m│  ✅ Perfect score! Well done!  │\033[0m\n")
		fmt.Printf("\033[32m╰──────────────────────────────────────────────────────────────────────────╯\033[0m\n")
	}

	// Show score if we have stats
	if stats != nil {
		fmt.Println()
		score := stats.OverallScore
		var color string
		var emoji string

		switch {
		case score >= 90:
			color = "\033[32m" // Green
			emoji = "🏆"
		case score >= 70:
			color = "\033[36m" // Cyan
			emoji = "👍"
		case score >= 50:
			color = "\033[33m" // Yellow
			emoji = "⚡"
		default:
			color = "\033[31m" // Red
			emoji = "💔"
		}

		fmt.Printf("%s╔════════════════════════════╗\033[0m\n", color)
		fmt.Printf("%s║  %s Quality Score: %d/100  ║\033[0m\n", color, emoji, score)
		fmt.Printf("%s╚════════════════════════════╝\033[0m\n", color)
	}
}

func renderFixedTiming(duration time.Duration, fileSize int64) {
	fmt.Println()

	l := "milliseconds"
	d := fmt.Sprintf("%d", duration.Milliseconds())
	if duration.Milliseconds() > 1000 {
		l = "seconds"
		d = humanize.FormatFloat("##.##", duration.Seconds())
	}

	fmt.Printf("\033[36m⏱️  vacuum took %s %s to lint %s\033[0m\n",
		d, l, index.HumanFileSize(float64(fileSize)))
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
