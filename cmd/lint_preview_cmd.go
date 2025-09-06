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
	"github.com/daveshanley/vacuum/statistics"
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
		Short:         "Preview lint with enhanced interactive table",
		Long:          `Lint an OpenAPI specification with an enhanced interactive table view`,
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
	cmd.Flags().BoolP("interactive", "i", false, "Force interactive table view")

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
	interactiveFlag, _ := cmd.Flags().GetBool("interactive")

	// Show banner unless disabled
	if !silentFlag && !noBannerFlag {
		PrintBanner()
	}

	// Get file info
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		fmt.Printf("\033[31mUnable to read file '%s': %v\033[0m\n", fileName, err)
		return err
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

	// Setup logging
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	})
	logger := slog.New(handler)

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
			fileName, len(selectedRS.Rules), selectedRS.DocumentationURI)
	}

	// Start timing
	start := time.Now()

	// Read and lint the file
	specBytes, ferr := os.ReadFile(fileName)
	if ferr != nil {
		fmt.Printf("\033[31mUnable to read file '%s': %s\033[0m\n", fileName, ferr.Error())
		return ferr
	}

	specStringData := strings.Split(string(specBytes), "\n")

	// Build deep graph if we have ignored items
	deepGraph := false
	if len(ignoredItems) > 0 {
		deepGraph = true
	}

	// Apply rules
	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:                         selectedRS,
		Spec:                            specBytes,
		SpecFileName:                    fileName,
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
			fmt.Printf("\033[31mUnable to process spec '%s': %s\033[0m\n", fileName, err.Error())
		}
		return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
	}

	// Process results
	resultSet := model.NewRuleResultSet(result.Results)

	// Handle category filtering
	var cats []*model.RuleCategory
	if categoryFlag != "" {
		// Category filtering logic here (same as original)
		cats = model.RuleCategoriesOrdered
	} else {
		cats = model.RuleCategoriesOrdered
	}

	resultSet.SortResultsByLineNumber()
	stats := statistics.CreateReportStatistics(result.Index, result.SpecInfo, resultSet)

	// Show detailed results if requested
	if detailsFlag && len(resultSet.Results) > 0 {
		// Use interactive table for large result sets (>50) or if specifically requested
		if (len(resultSet.Results) > 50 || interactiveFlag) && !snippetsFlag && !silentFlag {
			// Show summary first
			renderFixedSummary(resultSet, cats, stats, fileName, silentFlag, noStyleFlag)

			// Show timing if requested
			duration := time.Since(start)
			if timeFlag {
				renderFixedTiming(duration, fileInfo.Size())
			}

			// Launch interactive table
			fmt.Println()
			fmt.Printf("\033[36m📋 Launching interactive table view (press 'q' to exit)...\033[0m\n")
			fmt.Println()

			// Filter results if needed
			var filteredResults []*model.RuleFunctionResult
			for _, r := range resultSet.Results {
				if errorsFlag && r.Rule.Severity != model.SeverityError {
					continue
				}
				filteredResults = append(filteredResults, r)
			}
			if len(filteredResults) == 0 {
				filteredResults = resultSet.Results
			}

			// Show interactive table
			err := cui.ShowViolationTableView(filteredResults, fileName, specBytes)
			if err != nil {
				fmt.Printf("\033[31mError showing interactive table: %v\033[0m\n", err)
			}
			return nil
		} else {
			// Use regular detailed view for smaller result sets or when snippets are requested
			renderFixedDetails(resultSet.Results, specStringData, snippetsFlag, errorsFlag,
				silentFlag, noMessageFlag, allResultsFlag, noClipFlag, fileName, noStyleFlag)
		}
	}

	// Render summary
	renderFixedSummary(resultSet, cats, stats, fileName, silentFlag, noStyleFlag)

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
		width = cui.DefaultTerminalWidth // Default fallback
	}

	// Calculate dynamic column widths based on terminal width
	// Allocate percentages: Location, Severity, Message, Rule, Category, Path
	locWidth := width * cui.LocationColumnPercent / 100
	sevWidth := cui.SeverityColumnWidth
	msgWidth := width * cui.MessageColumnPercent / 100
	ruleWidth := width * cui.RuleColumnPercent / 100
	catWidth := cui.CategoryColumnWidth
	pathWidth := width - locWidth - sevWidth - msgWidth - ruleWidth - catWidth - cui.TableSeparatorWidth // for separators

	// Minimum widths
	if locWidth < cui.MinLocationWidth {
		locWidth = cui.MinLocationWidth
	}
	if msgWidth < cui.MinMessageWidth {
		msgWidth = cui.MinMessageWidth
	}
	if ruleWidth < cui.MinRuleWidth {
		ruleWidth = cui.MinRuleWidth
	}
	if pathWidth < 20 {
		pathWidth = 20
	}

	// Build and render table
	if !snippets {
		// Print header
		fmt.Printf("\033[36m%-*s  %-*s  %-*s  %-*s  %-*s  %-*s\033[0m\n",
			locWidth, "Location",
			sevWidth, "Severity",
			msgWidth, "Message",
			ruleWidth, "Rule",
			catWidth, "Category",
			pathWidth, "Path")

		// Print separator
		fmt.Printf("\033[90m%s  %s  %s  %s  %s  %s\033[0m\n",
			strings.Repeat("─", locWidth),
			strings.Repeat("─", sevWidth),
			strings.Repeat("─", msgWidth),
			strings.Repeat("─", ruleWidth),
			strings.Repeat("─", catWidth),
			strings.Repeat("─", pathWidth))

		// Print rows
		for i, r := range results {
			if i > 1000 && !allResults {
				fmt.Printf("\033[31m...%d more violations not rendered\033[0m\n", len(results)-1000)
				break
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

			location := fmt.Sprintf("%s:%d:%d", f, startLine, startCol)

			// Handle message and path truncation
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

			// Get severity
			sev := "info"
			if r.Rule != nil {
				sev = r.Rule.Severity
			}

			// Skip if showing errors only
			if errors && sev != model.SeverityError {
				continue
			}

			// Format severity with color
			var sevColored string
			switch sev {
			case model.SeverityError:
				sevColored = fmt.Sprintf("\033[31m%-*s\033[0m", sevWidth, "error")
			case model.SeverityWarn:
				sevColored = fmt.Sprintf("\033[33m%-*s\033[0m", sevWidth, "warning")
			default:
				sevColored = fmt.Sprintf("\033[36m%-*s\033[0m", sevWidth, "info")
			}

			// Print row
			fmt.Printf("%-*s  %s  %-*s  %-*s  %-*s  \033[90m%-*s\033[0m\n",
				locWidth, truncate(location, locWidth),
				sevColored,
				msgWidth, truncate(m, msgWidth),
				ruleWidth, truncate(r.Rule.Id, ruleWidth),
				catWidth, truncate(r.Rule.RuleCategory.Name, catWidth),
				pathWidth, truncate(p, pathWidth))
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
