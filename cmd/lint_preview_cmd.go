// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/libopenapi/index"
	"github.com/spf13/cobra"
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

	// Use color constants
	fmt.Printf("%s%s%s\n\n", cui.ASCIIPink, banner, cui.ASCIIReset)
	fmt.Printf("%sversion: %s | compiled: %s%s\n", cui.ASCIIGreen, Version, Date, cui.ASCIIReset)
	fmt.Printf("%s🔗 https://quobix.com/vacuum | https://github.com/daveshanley/vacuum%s\n\n", cui.ASCIIBlue, cui.ASCIIReset)
}

func renderFixedDetails(results []*model.RuleFunctionResult, specData []string,
	snippets, errors, silent, noMessage, allResults, noClip bool,
	fileName string, noStyle bool) {

	// print file header
	printFileHeader(fileName, silent)

	// calculate table configuration
	config := calculateTableConfig(results, fileName, errors, noMessage, noClip)

	// render based on format
	if config.UseTreeFormat {
		renderTreeFormat(results, config, fileName, errors, allResults)
		return
	}

	// render table format
	if !snippets {
		renderTableFormat(results, config, fileName, errors, allResults, snippets)
	}
}

func renderFixedSummary(rs *model.RuleResultSet, cats []*model.RuleCategory,
	stats *reports.ReportStatistics, fileName string, silent bool, noStyle bool) {

	if silent {
		return
	}

	// Check if there are any results to display
	hasResults := rs != nil && rs.Results != nil && len(rs.Results) > 0

	if hasResults {
		// Build category summary table with colored headers
		fmt.Printf(" %s%-20s%s  %s%-12s%s  %s%-12s%s  %s%-12s%s\n",
			cui.ASCIIPink, "Category", cui.ASCIIReset,
			cui.ASCIIRed, "✗ Errors", cui.ASCIIReset,
			cui.ASCIIYellow, "▲ Warnings", cui.ASCIIReset,
			cui.ASCIIBlue, "● Info", cui.ASCIIReset)
		fmt.Printf(" %s%s  %s  %s  %s%s\n",
			cui.ASCIIPink,
			strings.Repeat("─", 20),
			strings.Repeat("─", 12),
			strings.Repeat("─", 12),
			strings.Repeat("─", 12),
			cui.ASCIIReset)

		for _, cat := range cats {
			errors := rs.GetErrorsByRuleCategory(cat.Id)
			warn := rs.GetWarningsByRuleCategory(cat.Id)
			info := rs.GetInfoByRuleCategory(cat.Id)

			if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {
				fmt.Printf(" %-20s  %-12s  %-12s  %-12s\n",
					cat.Name,
					humanize.Comma(int64(len(errors))),
					humanize.Comma(int64(len(warn))),
					humanize.Comma(int64(len(info))))
			}
		}
		fmt.Println()

		// Build rule violation summary table
		type ruleViolation struct {
			ruleId string
			count  int
		}

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

		// convert map to slice and sort by count
		var ruleViolations []ruleViolation
		for _, rv := range ruleMap {
			ruleViolations = append(ruleViolations, *rv)
		}

		// sort by violation count (highest first)
		for i := 0; i < len(ruleViolations); i++ {
			for j := i + 1; j < len(ruleViolations); j++ {
				if ruleViolations[j].count > ruleViolations[i].count {
					ruleViolations[i], ruleViolations[j] = ruleViolations[j], ruleViolations[i]
				}
			}
		}

		// print rule violations table if there are any
		if len(ruleViolations) > 0 {
			// calculate total violations for percentage
			totalViolations := 0
			for _, rv := range ruleViolations {
				totalViolations += rv.count
			}

			// create progress bar with gradient from blue to pink
			// using RGB hex values that match our theme
			prog := progress.New(
				progress.WithScaledGradient("#62c4ff", "#f83aff"), // blue to pink
				progress.WithWidth(30),
				progress.WithoutPercentage(),
				progress.WithFillCharacters('█', ' '), // solid bar with no background
			)

			fmt.Printf(" %s%-40s%s  %s%-12s%s  %s%-30s%s\n",
				cui.ASCIIPink, "Rule", cui.ASCIIReset,
				cui.ASCIIPink, "Violations", cui.ASCIIReset,
				cui.ASCIIPink, "Quality Impact", cui.ASCIIReset)
			fmt.Printf(" %s%s  %s  %s%s\n",
				cui.ASCIIPink,
				strings.Repeat("─", 40),
				strings.Repeat("─", 12),
				strings.Repeat("─", 30),
				cui.ASCIIReset)

			// show top 10 most violated rules
			maxRules := 10
			if len(ruleViolations) < maxRules {
				maxRules = len(ruleViolations)
			}

			for i := 0; i < maxRules; i++ {
				rv := ruleViolations[i]
				// truncate rule name if too long
				ruleName := rv.ruleId
				if len(ruleName) > 40 {
					ruleName = ruleName[:37] + "..."
				}

				// calculate percentage of total violations
				percentage := float64(rv.count) / float64(totalViolations)

				// render the row with progress bar
				fmt.Printf(" %-40s  %-12s  %s\n",
					ruleName,
					humanize.Comma(int64(rv.count)),
					prog.ViewAs(percentage))
			}

			if len(ruleViolations) > maxRules {
				fmt.Printf(" %s... and %d more rules%s\n", cui.ASCIIGrey, len(ruleViolations)-maxRules, cui.ASCIIReset)
			}
			fmt.Println()
		}
	}

	// Render result box
	errs := 0
	warnings := 0
	informs := 0
	if rs != nil {
		errs = rs.GetErrorCount()
		warnings = rs.GetWarnCount()
		informs = rs.GetInfoCount()
	}

	if errs > 0 {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIRed, cui.ASCIIReset)
		fmt.Printf("%s│  ❌ Linting failed with %d errors, %d warnings and %d informs  │%s\n",
			cui.ASCIIRed, errs, warnings, informs, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIRed, cui.ASCIIReset)
	} else if warnings > 0 {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIYellow, cui.ASCIIReset)
		fmt.Printf("%s│  ⚠️  Linting passed with %d warnings and %d informs  │%s\n",
			cui.ASCIIYellow, warnings, informs, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIYellow, cui.ASCIIReset)
	} else if informs > 0 {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIBlue, cui.ASCIIReset)
		fmt.Printf("%s│  ℹ️  Linting passed, %d informs reported  │%s\n", cui.ASCIIBlue, informs, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIBlue, cui.ASCIIReset)
	} else {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIGreen, cui.ASCIIReset)
		fmt.Printf("%s│  ✅ Perfect score! Well done!  │%s\n", cui.ASCIIGreen, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIGreen, cui.ASCIIReset)
	}

	// Show score if we have stats
	if stats != nil {
		fmt.Println()
		score := stats.OverallScore
		var color string
		var emoji string

		switch {
		case score >= 90:
			color = cui.ASCIIGreen
			emoji = "🏆"
		case score >= 70:
			color = cui.ASCIIBlue
			emoji = "👍"
		case score >= 50:
			color = cui.ASCIIYellow
			emoji = "⚡"
		default:
			color = cui.ASCIIRed
			emoji = "💔"
		}

		fmt.Printf("%s╔════════════════════════════╗%s\n", color, cui.ASCIIReset)
		fmt.Printf("%s║  %s Quality Score: %d/100  ║%s\n", color, emoji, score, cui.ASCIIReset)
		fmt.Printf("%s╚════════════════════════════╝%s\n", color, cui.ASCIIReset)
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
