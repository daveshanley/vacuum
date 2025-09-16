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

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/logging"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/libopenapi/index"
	"github.com/spf13/cobra"
)

func GetLintCommand() *cobra.Command {
	validFileExtensions := []string{"yaml", "yml", "json"}
	cmd := &cobra.Command{
		Use:           "lint <your-openapi-file.yaml>",
		Short:         "Lint an OpenAPI specification",
		Long:          `Lint an OpenAPI specification, the output of the response will be in the terminal`,
		RunE:          runLint,
		SilenceUsage:  true,
		SilenceErrors: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return validFileExtensions, cobra.ShellCompDirectiveFilterFileExt
		},
	}

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
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().Bool("no-clip", false, "Do not truncate messages or paths")
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")
	cmd.Flags().Int("min-score", 10, "Throw an error return code if the score is below this value")
	cmd.Flags().Bool("show-rules", false, "Show which rules are being used when linting")
	cmd.Flags().Bool("pipeline-output", false, "Renders CI/CD summary output, suitable for pipelines")
	cmd.Flags().String("globbed-files", "", "Glob pattern of files to lint")
	// base, remote, skip-check, timeout, ruleset, functions, time, hard-mode are inherited from root as persistent flags
	// cert-file, key-file, ca-file, insecure, debug are inherited from root as persistent flags
	// ext-refs is inherited from root as a persistent flag

	return cmd
}

func runLint(cmd *cobra.Command, args []string) error {
	flags := ReadLintFlags(cmd)

	SetupVacuumEnvironment(flags)

	validFileExtensions := []string{"yaml", "yml", "json"}
	filesToLint, err := getFilesToLint(flags.GlobPattern, args, validFileExtensions)
	if cmd.Flags().Changed("globbed-files") && err != nil {
		fmt.Printf("🚨 %s%sError getting files to lint: %v%s\n\n", color.ASCIIBold, color.ASCIIRed, err, color.ASCIIReset)
		return err
	}

	if len(filesToLint) < 1 {
		fmt.Printf("🚨 %s%sPlease supply an OpenAPI specification to lint%s\n\n",
			color.ASCIIBold, color.ASCIIRed, color.ASCIIReset)
		return fmt.Errorf("no file supplied")
	}

	// for multiple files, run each one and combine results
	if len(filesToLint) > 1 {
		return runMultipleFiles(cmd, filesToLint)
	}

	// single file processing continues below
	fileName := filesToLint[0]

	// ignore file
	ignoredItems, err := LoadIgnoreFile(flags.IgnoreFile, flags.SilentFlag, flags.PipelineOutput, flags.NoStyleFlag)
	if err != nil {
		return err
	}

	// try to load the file as either a report or spec
	reportOrSpec, err := LoadFileAsReportOrSpec(fileName)
	if err != nil {
		if !flags.SilentFlag {
			fmt.Printf("\033[31mUnable to load file '%s': %v\033[0m\n", fileName, err)
		}
		return err
	}

	fileInfo, _ := os.Stat(fileName)
	logger, bufferedLogger := createLogger(flags.DebugFlag)

	var resultSet *model.RuleResultSet
	var specBytes []byte
	var displayFileName string
	var stats *reports.ReportStatistics
	start := time.Now()

	if reportOrSpec.IsReport {
		// pre-compiled report
		if !flags.SilentFlag {
			fmt.Printf("\033[36mLoading pre-compiled vacuum report from '%s'\033[0m\n\n", fileName)
		}

		// create a new RuleResultSet from the results to ensure proper initialization
		if reportOrSpec.ResultSet != nil && reportOrSpec.ResultSet.Results != nil {
			// filter ignored results first
			filteredResults := utils.FilterIgnoredResultsPtr(reportOrSpec.ResultSet.Results, ignoredItems)
			// create properly initialized RuleResultSet
			resultSet = model.NewRuleResultSetPointer(filteredResults)
		} else {
			resultSet = model.NewRuleResultSetPointer([]*model.RuleFunctionResult{})
		}

		specBytes = reportOrSpec.SpecBytes
		displayFileName = reportOrSpec.FileName
	} else {
		// regular spec file - run linting
		specBytes = reportOrSpec.SpecBytes
		displayFileName = fileName

		customFuncs, _ := LoadCustomFunctions(flags.FunctionsFlag, flags.SilentFlag)

		// load and configure ruleset (handles hard mode, custom rulesets, etc.)
		selectedRS, err := LoadRulesetWithConfig(flags, logger)
		if err != nil {
			return err
		}

		if !flags.SilentFlag && !flags.PipelineOutput {
			fmt.Printf(" %svacuuming file '%s' against %d rules: %s%s\n\n",
				color.ASCIIBlue, displayFileName, len(selectedRS.Rules), selectedRS.DocumentationURI, color.ASCIIReset)
		}

		// deep graph is required if we have ignored items
		deepGraph := false
		if len(ignoredItems) > 0 {
			deepGraph = true
		}

		result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:                         selectedRS,
			Spec:                            specBytes,
			SpecFileName:                    displayFileName,
			CustomFunctions:                 customFuncs,
			Base:                            flags.BaseFlag,
			AllowLookup:                     flags.RemoteFlag,
			SkipDocumentCheck:               flags.SkipCheckFlag,
			Logger:                          logger,
			BuildDeepGraph:                  deepGraph,
			Timeout:                         time.Duration(flags.TimeoutFlag) * time.Second,
			IgnoreCircularArrayRef:          flags.IgnoreArrayCircleRef,
			IgnoreCircularPolymorphicRef:    flags.IgnorePolymorphCircleRef,
			ExtractReferencesFromExtensions: flags.ExtRefsFlag,
			HTTPClientConfig:                GetHTTPClientConfig(flags),
		})

		result.Results = utils.FilterIgnoredResults(result.Results, ignoredItems)

		// render out buffered logs
		RenderBufferedLogs(bufferedLogger, flags.NoStyleFlag)

		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				fmt.Printf("\033[31mUnable to process spec '%s': %s\033[0m\n", displayFileName, err.Error())
			}
			return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
		}

		resultSet = model.NewRuleResultSet(result.Results)

		if result.Index != nil && result.SpecInfo != nil {
			stats = statistics.CreateReportStatistics(result.Index, result.SpecInfo, resultSet)
		}
	}

	specStringData := strings.Split(string(specBytes), "\n")

	// handle category filtering
	var cats []*model.RuleCategory
	if flags.CategoryFlag != "" {
		resultSet.ResetCounts()
		var filteredResults []*model.RuleFunctionResult
		switch flags.CategoryFlag {
		case model.CategoryDescriptions:
			cats = append(cats, model.RuleCategories[model.CategoryDescriptions])
		case model.CategoryExamples:
			cats = append(cats, model.RuleCategories[model.CategoryExamples])
		case model.CategoryInfo:
			cats = append(cats, model.RuleCategories[model.CategoryInfo])
		case model.CategorySchemas:
			cats = append(cats, model.RuleCategories[model.CategorySchemas])
		case model.CategorySecurity:
			cats = append(cats, model.RuleCategories[model.CategorySecurity])
		case model.CategoryValidation:
			cats = append(cats, model.RuleCategories[model.CategoryValidation])
		case model.CategoryOperations:
			cats = append(cats, model.RuleCategories[model.CategoryOperations])
		case model.CategoryTags:
			cats = append(cats, model.RuleCategories[model.CategoryTags])
		case model.CategoryOWASP:
			cats = append(cats, model.RuleCategories[model.CategoryOWASP])
		default:
			if !flags.SilentFlag {
				fmt.Printf("%sWarning: Category '%s' is unknown, all categories are being considered.%s\n\n",
					color.ASCIIYellow, flags.CategoryFlag, color.ASCIIReset)
			}
			cats = model.RuleCategoriesOrdered
		}
		// filter results by category
		for _, val := range cats {
			categoryResults := resultSet.GetResultsByRuleCategory(val.Id)
			if len(categoryResults) > 0 {
				if len(cats) > 1 {
					filteredResults = append(filteredResults, categoryResults...)
				} else {
					filteredResults = categoryResults
				}
			}
		}
		resultSet.Results = filteredResults
	} else {
		cats = model.RuleCategoriesOrdered
	}

	resultSet.SortResultsByLineNumber()

	if reportOrSpec.IsReport && reportOrSpec.Report.Statistics != nil {
		stats = reportOrSpec.Report.Statistics
	}

	if flags.DetailsFlag && len(resultSet.Results) > 0 && !flags.PipelineOutput {
		renderFixedDetails(RenderDetailsOptions{
			Results:    resultSet.Results,
			SpecData:   specStringData,
			Snippets:   flags.SnippetsFlag,
			Errors:     flags.ErrorsFlag,
			Silent:     flags.SilentFlag,
			NoMessage:  flags.NoMessageFlag,
			AllResults: flags.AllResultsFlag,
			NoClip:     flags.NoClipFlag,
			FileName:   displayFileName,
			NoStyle:    flags.NoStyleFlag,
		})
	}

	renderFixedSummary(RenderSummaryOptions{
		RuleResultSet:  resultSet,
		RuleCategories: cats,
		Statistics:     stats,
		Filename:       displayFileName,
		Silent:         flags.SilentFlag,
		NoStyle:        flags.NoStyleFlag,
		PipelineOutput: flags.PipelineOutput,
		ShowRules:      flags.ShowRules,
	})

	// timing
	duration := time.Since(start)
	if flags.TimeFlag && !flags.PipelineOutput {
		renderFixedTiming(duration, fileInfo.Size())
	}

	// severity failure
	errs := resultSet.GetErrorCount()
	warnings := resultSet.GetWarnCount()
	informs := resultSet.GetInfoCount()

	// min score threshold
	if flags.MinScore > 10 && stats != nil {
		if stats.OverallScore < flags.MinScore {
			if !flags.PipelineOutput && !flags.SilentFlag {
				fmt.Printf("\n%s🚨 SCORE THRESHOLD FAILED 🚨%s\n", color.ASCIIRed, color.ASCIIReset)
				fmt.Printf("%sOverall score is %d, but the threshold is %d%s\n\n",
					color.ASCIIRed, stats.OverallScore, flags.MinScore, color.ASCIIReset)
			} else if flags.PipelineOutput {
				fmt.Printf("\n> 🚨 SCORE THRESHOLD FAILED, PIPELINE WILL FAIL 🚨\n\n")
			}
			return fmt.Errorf("score threshold failed, overall score is %d, and the threshold is %d",
				stats.OverallScore, flags.MinScore)
		}
	}

	failErr := CheckFailureSeverity(flags.FailSeverityFlag, errs, warnings, informs)
	if failErr != nil {
		if flags.SilentFlag {
			os.Exit(1)
		}
		return failErr
	}

	return nil
}

// fileResult holds the results and logs for a single file
type fileResult struct {
	fileName string
	results  []*model.RuleFunctionResult
	errors   int
	warnings int
	informs  int
	size     int64
	logs     []string
	err      error
}

func PrintBanner(noStyle ...bool) {
	banner := `   
 ██╗   ██╗ █████╗  ██████╗██╗   ██╗██╗   ██╗███╗   ███╗ 《《《─═─═── ·* · ˙*
 ██║   ██║██╔══██╗██╔════╝██║   ██║██║   ██║████╗ ████║《《《──═─═──· ··* ˙˙
 ██║   ██║███████║██║     ██║   ██║██║   ██║██╔████╔██║《《《───═─═─··· ˙˙ ˙
 ╚██╗ ██╔╝██╔══██║██║     ██║   ██║██║   ██║██║╚██╔╝██║《《──═─═──·* ·· ˙˙
  ╚████╔╝ ██║  ██║╚██████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║ 《《─═─═──* · · ˙
   ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝ 《───═─═─· ··* ˙˙ ˙
`
	skipColors := len(noStyle) > 0 && noStyle[0]

	if skipColors {
		fmt.Printf("%s\n", banner)
		fmt.Printf(" version: %s | compiled: %s\n", GetVersion(), GetDate())
		fmt.Printf(" https://quobix.com/vacuum/ | https://github.com/daveshanley/vacuum\n\n")
	} else {
		fmt.Printf(" %s%s%s\n", color.ASCIIPink, banner, color.ASCIIReset)
		fmt.Printf(" %sversion: %s%s%s%s | compiled: %s%s%s\n", color.ASCIIGreen,
			color.ASCIIGreenBold, GetVersion(), color.ASCIIReset, color.ASCIIGreen, color.ASCIIGreenBold, GetDate(), color.ASCIIReset)
		fmt.Printf("%s https://quobix.com/vacuum/ | https://github.com/daveshanley/vacuum%s\n\n", color.ASCIIBlue, color.ASCIIReset)
	}
}

// renderHardModeBox displays the hard mode enabled message using lipgloss
func renderHardModeBox(message string, noStyle bool) {
	if noStyle {
		fmt.Printf(" | %s\n\n", message)
		return
	}

	// get terminal width and calculate box width to match summary tables
	termWidth := getTerminalWidth()
	widths := calculateColumnWidths(termWidth)

	// calculate actual table width (matching the summary table)
	// for full width: rule (40) + violation (12) + impact (50) + separators (4 spaces) + leading space (1) = 107
	boxWidth := widths.rule + widths.violation + widths.impact + 4 + 1
	if termWidth < 100 {
		// for smaller terminals, adjust box width accordingly
		boxWidth = termWidth - 13 // leave some margin
		if boxWidth < 40 {
			boxWidth = 40
		}
	}

	// center the message in the box
	messageStyle := lipgloss.NewStyle().
		Width(boxWidth-2).
		Align(lipgloss.Center).
		Padding(1, 0)

	boxStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Foreground(lipgloss.Color("196")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("196")).
		Bold(true)

	fmt.Println(boxStyle.Render(messageStyle.Render(message)))
	fmt.Println()
}

// renderInfoMessage displays an info message using lipgloss
func renderInfoMessage(message string, noStyle bool) {
	if noStyle {
		fmt.Printf(" %s\n", message)
		return
	}

	fmt.Printf(" %s%s%s\n", color.ASCIIBlue, message, color.ASCIIReset)
}

// renderIgnoredItems displays the ignored paths and rules in tree format
func renderIgnoredItems(ignoredItems model.IgnoredItems, noStyle bool) {
	type ignoredItem struct {
		rule string
		path string
	}
	var items []ignoredItem

	// collect all ignored items from the map
	for category, paths := range ignoredItems {
		if len(paths) > 0 {
			for _, path := range paths {
				items = append(items, ignoredItem{
					rule: category,
					path: path,
				})
			}
		}
	}

	if len(items) == 0 {
		return
	}

	fmt.Printf(" %signored items:%s\n", color.ASCIIGrey, color.ASCIIReset)

	// render in tree format
	for i, item := range items {
		isLast := i == len(items)-1
		if !noStyle {
			// format: rule (pink bold) : path (colorized)
			formattedItem := fmt.Sprintf("%s%s%s%s: %s",
				color.ASCIIPink, color.ASCIIBold, item.rule, color.ASCIIReset,
				color.ColorizePath(item.path))

			if isLast {
				fmt.Printf(" %s└─%s %s\n", color.ASCIIPink, color.ASCIIReset, formattedItem)
			} else {
				fmt.Printf(" %s├─%s %s\n", color.ASCIIPink, color.ASCIIReset, formattedItem)
			}
		} else {
			if isLast {
				fmt.Printf(" └─ %s: %s\n", item.rule, item.path)
			} else {
				fmt.Printf(" ├─ %s: %s\n", item.rule, item.path)
			}
		}
	}
	fmt.Println()
}

// createLogger creates a debug logger using slog with lipgloss formatting
func createLogger(debugFlag bool) (*slog.Logger, *logging.BufferedLogger) {
	var bufferedLogger *logging.BufferedLogger
	if debugFlag {
		bufferedLogger = logging.NewBufferedLoggerWithLevel(logging.LogLevelDebug)
	} else {
		bufferedLogger = logging.NewBufferedLoggerWithLevel(logging.LogLevelError)
	}

	handler := logging.NewBufferedLogHandler(bufferedLogger)

	logger := slog.New(handler)

	return logger, bufferedLogger
}

func renderFixedDetails(opts RenderDetailsOptions) {
	printFileHeader(opts.FileName, opts.Silent)

	// calculate table configuration
	config := calculateTableConfig(opts.Results, opts.FileName, opts.Errors, opts.NoMessage, opts.NoClip, opts.NoStyle)

	if config.UseTreeFormat {
		renderTreeFormat(opts.Results, config, opts.FileName, opts.Errors, opts.AllResults)
		return
	}

	renderTableFormat(opts.Results, config, opts.FileName, opts.Errors, opts.AllResults, opts.Snippets, opts.SpecData)
}

func renderFixedSummary(opts RenderSummaryOptions) {
	rs := opts.RuleResultSet
	cats := opts.RuleCategories
	stats := opts.Statistics
	fileName := opts.Filename
	silent := opts.Silent
	pipelineOutput := opts.PipelineOutput
	showRules := opts.ShowRules

	if silent {
		return
	}

	// If pipeline output is requested, use the existing RenderMarkdownSummary function
	if pipelineOutput {
		var ruleset *rulesets.RuleSet
		if rs != nil && len(rs.Results) > 0 && rs.Results[0].Rule != nil {
			ruleset = &rulesets.RuleSet{
				Rules: make(map[string]*model.Rule),
			}
			seenRules := make(map[string]bool)
			for _, result := range rs.Results {
				if result.Rule != nil && !seenRules[result.Rule.Id] {
					ruleset.Rules[result.Rule.Id] = result.Rule
					seenRules[result.Rule.Id] = true
				}
			}
		}

		rso := RenderSummaryOptions{
			RuleResultSet:  rs,
			RuleCategories: cats,
			RuleSet:        ruleset,
			PipelineOutput: true,
			RenderRules:    showRules,
			ReportStats:    stats,
			Filename:       fileName,
			TotalFiles:     1,
			Silent:         false,
		}

		RenderMarkdownSummary(rso)
		return
	}

	// check if there are any results to display
	hasResults := rs != nil && rs.Results != nil && len(rs.Results) > 0

	if hasResults {
		width := getTerminalWidth()
		widths := calculateColumnWidths(width)

		renderCategoryTable(rs, cats, widths)

		violations := buildRuleViolations(rs)
		renderRuleViolationsTable(violations, widths)
	}

	// render result box
	errs := 0
	warnings := 0
	informs := 0
	if rs != nil {
		errs = rs.GetErrorCount()
		warnings = rs.GetWarnCount()
		informs = rs.GetInfoCount()
	}

	fmt.Printf(" --> use the %s<dashboard>%s command to be able to navigate results interactively <--\n",
		color.ASCIIGreenBold, color.ASCIIReset)

	// render quality score if available
	if stats != nil {
		fmt.Println()
		renderQualityScore(stats.OverallScore)
	}

	renderResultBox(errs, warnings, informs)

	fmt.Printf(" %suse --debug if you want to enable developer logging%s\n\n", color.ASCIILightGreyItalic, color.ASCIIReset)

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

// getFilesToLint handles both individual files and glob patterns
func getFilesToLint(globPattern string, filepaths []string, validFileExtensions []string) ([]string, error) {
	// Note that if some of the paths are absolute and the others are relative,
	// then we turn all paths into relative ones.
	if globPattern == "" {
		return deduplicate(filepaths), nil
	}

	var filesToLint = filepaths

	// Get all files that match the glob pattern
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return []string{}, err
	}
	filesToLint = append(filesToLint, matches...)

	// Remove any duplicates
	filesToLint = deduplicate(filesToLint)

	// Ensure that all files have valid file extensions
	for _, file := range filesToLint {
		if !hasValidExtension(file, validFileExtensions) {
			return []string{}, fmt.Errorf("File %q has an invalid file extension. Only %v are supported.\n", file, validFileExtensions)
		}
	}

	return filesToLint, nil
}

func deduplicate(input []string) []string {
	seen := make(map[string]bool)
	deduplicated := []string{}
	for _, val := range input {
		if _, alreadySeen := seen[val]; !alreadySeen {
			seen[val] = true
			deduplicated = append(deduplicated, val)
		}
	}
	return deduplicated
}

func hasValidExtension(filename string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}
