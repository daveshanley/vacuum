// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	cmd.Flags().String("cert-file", "", "Path to client certificate file for HTTPS requests")
	cmd.Flags().String("key-file", "", "Path to client private key file for HTTPS requests")
	cmd.Flags().String("ca-file", "", "Path to CA certificate file for HTTPS requests")
	cmd.Flags().Bool("insecure", false, "Skip TLS certificate verification (insecure)")
	cmd.Flags().BoolP("debug", "w", false, "Enable debug logging")
	cmd.Flags().Int("min-score", 10, "Throw an error return code if the score is below this value")
	cmd.Flags().Bool("show-rules", false, "Show which rules are being used when linting")
	cmd.Flags().Bool("pipeline-output", false, "Renders CI/CD summary output, suitable for pipelines")

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
	certFile, _ := cmd.Flags().GetString("cert-file")
	keyFile, _ := cmd.Flags().GetString("key-file")
	caFile, _ := cmd.Flags().GetString("ca-file")
	insecure, _ := cmd.Flags().GetBool("insecure")
	debugFlag, _ := cmd.Flags().GetBool("debug")
	minScore, _ := cmd.Flags().GetInt("min-score")
	showRules, _ := cmd.Flags().GetBool("show-rules")
	pipelineOutput, _ := cmd.Flags().GetBool("pipeline-output")

	if !noStyleFlag && !pipelineOutput {
		fileInfo, _ := os.Stdout.Stat()
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			noStyleFlag = true
		}
	}

	if noStyleFlag && !pipelineOutput {
		cui.DisableColors()
	}

	if !silentFlag && !noBannerFlag && !pipelineOutput {
		PrintBanner()
	}

	// ignore file
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

	logLevel := slog.LevelError
	if debugFlag {
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)

	var resultSet *model.RuleResultSet
	var specBytes []byte
	var displayFileName string
	var stats *reports.ReportStatistics
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
			// Create HTTP client for remote ruleset downloads if needed
			var httpClient *http.Client
			httpClientConfig := utils.HTTPClientConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   caFile,
				Insecure: insecure,
			}
			if utils.ShouldUseCustomHTTPClient(httpClientConfig) {
				var clientErr error
				httpClient, clientErr = utils.CreateCustomHTTPClient(httpClientConfig)
				if clientErr != nil {
					fmt.Printf("\033[31mFailed to create custom HTTP client: %s\033[0m\n", clientErr.Error())
					return clientErr
				}
			}

			var rsErr error
			selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, httpClient)
			if rsErr != nil {
				fmt.Printf("\033[31mUnable to load ruleset '%s': %s\033[0m\n", rulesetFlag, rsErr.Error())
				return rsErr
			}
			if hardModeFlag {
				MergeOWASPRulesToRuleSet(selectedRS, true)
			}
		}

		// Show which rules are being used (after ruleset is fully loaded)
		if showRules && !pipelineOutput && !silentFlag {
			renderRulesList(selectedRS.Rules)
		}

		// Display linting info
		if !silentFlag && !pipelineOutput {
			fmt.Printf("%sLinting file '%s' against %d rules: %s%s\n\n",
				cui.ASCIIBlue, displayFileName, len(selectedRS.Rules), selectedRS.DocumentationURI, cui.ASCIIReset)
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
			HTTPClientConfig: utils.HTTPClientConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   caFile,
				Insecure: insecure,
			},
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

		// Create statistics for score checking and pipeline output
		if (minScore > 10 || pipelineOutput) && result.Index != nil && result.SpecInfo != nil {
			stats = statistics.CreateReportStatistics(result.Index, result.SpecInfo, resultSet)
		}
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

	// Update statistics if we have them from the report
	if reportOrSpec.IsReport && reportOrSpec.Report.Statistics != nil {
		stats = reportOrSpec.Report.Statistics
	}

	// Show detailed results if requested (but not in pipeline output mode)
	if detailsFlag && len(resultSet.Results) > 0 && !pipelineOutput {
		// Always use regular detailed view (no interactive UI)
		// Note: noMessageFlag is ignored when pipelineOutput is true (handled above)
		renderFixedDetails(resultSet.Results, specStringData, snippetsFlag, errorsFlag,
			silentFlag, noMessageFlag, allResultsFlag, noClipFlag, displayFileName, noStyleFlag)
	}

	// Render summary
	renderFixedSummary(resultSet, cats, stats, displayFileName, silentFlag, noStyleFlag, pipelineOutput, showRules)

	// Show timing (but not in pipeline output mode)
	duration := time.Since(start)
	if timeFlag && !pipelineOutput {
		renderFixedTiming(duration, fileInfo.Size())
	}

	// Check severity failure
	errs := resultSet.GetErrorCount()
	warnings := resultSet.GetWarnCount()
	informs := resultSet.GetInfoCount()

	// Check min score threshold
	if minScore > 10 && stats != nil {
		if stats.OverallScore < minScore {
			if !pipelineOutput && !silentFlag {
				fmt.Printf("\n%s🚨 SCORE THRESHOLD FAILED 🚨%s\n", cui.ASCIIRed, cui.ASCIIReset)
				fmt.Printf("%sOverall score is %d, but the threshold is %d%s\n\n",
					cui.ASCIIRed, stats.OverallScore, minScore, cui.ASCIIReset)
			} else if pipelineOutput {
				fmt.Printf("\n> 🚨 SCORE THRESHOLD FAILED, PIPELINE WILL FAIL 🚨\n\n")
			}
			return fmt.Errorf("score threshold failed, overall score is %d, and the threshold is %d",
				stats.OverallScore, minScore)
		}
	}

	// Check for failure but handle it gracefully without showing help
	failErr := CheckFailureSeverity(failSeverityFlag, errs, warnings, informs)
	if failErr != nil {
		os.Exit(1)
	}

	return nil
}

func PrintBanner() {
	banner := `
██╗   ██╗ █████╗  ██████╗██╗   ██╗██╗   ██╗███╗   ███╗
██║   ██║██╔══██╗██╔════╝██║   ██║██║   ██║████╗ ████║
██║   ██║███████║██║     ██║   ██║██║   ██║██╔████╔██║
╚██╗ ██╔╝██╔══██║██║     ██║   ██║██║   ██║██║╚██╔╝██║
 ╚████╔╝ ██║  ██║╚██████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝`

	fmt.Printf("%s%s%s\n\n", cui.ASCIIPink, banner, cui.ASCIIReset)
	fmt.Printf("%sversion: %s%s%s%s | compiled: %s%s%s\n", cui.ASCIIGreen,
		cui.ASCIIGreenBold, Version, cui.ASCIIReset, cui.ASCIIGreen, cui.ASCIIGreenBold, Date, cui.ASCIIReset)
	fmt.Printf("%s🔗 https://quobix.com/vacuum | https://github.com/daveshanley/vacuum%s\n\n", cui.ASCIIBlue, cui.ASCIIReset)
}

func renderFixedDetails(results []*model.RuleFunctionResult, specData []string,
	snippets, errors, silent, noMessage, allResults, noClip bool,
	fileName string, noStyle bool) {

	// print file header
	printFileHeader(fileName, silent)

	// calculate table configuration
	config := calculateTableConfig(results, fileName, errors, noMessage, noClip, noStyle)

	// render based on format
	if config.UseTreeFormat {
		renderTreeFormat(results, config, fileName, errors, allResults)
		return
	}

	// render table format (handles both normal and snippets mode)
	renderTableFormat(results, config, fileName, errors, allResults, snippets, specData)
}

func renderFixedSummary(rs *model.RuleResultSet, cats []*model.RuleCategory,
	stats *reports.ReportStatistics, fileName string, silent bool, noStyle bool,
	pipelineOutput bool, showRules bool) {

	if silent {
		return
	}

	// If pipeline output is requested, use the existing RenderSummary function
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

		RenderSummary(rso)
		return
	}

	// check if there are any results to display
	hasResults := rs != nil && rs.Results != nil && len(rs.Results) > 0

	if hasResults {
		width := getTerminalWidth()
		widths := calculateColumnWidths(width)

		// render category summary table
		renderCategoryTable(rs, cats, widths)

		// build and render rule violations table
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

	renderResultBox(errs, warnings, informs)

	// render quality score if available
	if stats != nil {
		fmt.Println()
		renderQualityScore(stats.OverallScore)
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
