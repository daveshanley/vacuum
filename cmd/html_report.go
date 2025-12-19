// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/daveshanley/vacuum/color"
	html_report "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/spf13/cobra"
)

// GetHTMLReportCommand returns a cobra command for generating an HTML Report.
func GetHTMLReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "html-report",
		Short:        "Generate an HTML report of a linting run",
		Long: `Generate an interactive and useful HTML report. Default output filename is 'report.html' located in the working directory.

For multiple files, use --globbed-files to specify a glob pattern:
  vacuum html-report --globbed-files "specs/*.yaml" --output-dir reports/

This generates one HTML report per input file, named after the source spec.`,
		Example: `vacuum html-report my-awesome-spec.yaml report.html
vacuum html-report --globbed-files "specs/*.yaml" --output-dir reports/
vacuum html-report --globbed-files "api/**/*.json"`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0:
				return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
			case 1:
				return []string{"html", "htm"}, cobra.ShellCompDirectiveFilterFileExt
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			noStyleFlag, _ := cmd.Flags().GetBool("no-style")
			noBannerFlag, _ := cmd.Flags().GetBool("no-banner")
			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			lookupTimeoutFlag, _ := cmd.Flags().GetInt("lookup-timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			silent, _ := cmd.Flags().GetBool("silent")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")
			changesFlag, _ := cmd.Flags().GetString("changes")
			originalFlag, _ := cmd.Flags().GetString("original")
			globPattern, _ := cmd.Flags().GetString("globbed-files")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			breakingConfigPath, _ := cmd.Flags().GetString("breaking-config")
			warnOnChanges, _ := cmd.Flags().GetBool("warn-on-changes")
			errorOnBreaking, _ := cmd.Flags().GetBool("error-on-breaking")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				color.DisableColors()
			}

			if !noBannerFlag {
				PrintBanner()
			}

			// Load and apply breaking rules config early, before any change comparison
			breakingConfig, breakingConfigErr := utils.LoadBreakingRulesConfig(breakingConfigPath)
			if breakingConfigErr != nil {
				var validationErr *utils.ConfigValidationError
				if errors.As(breakingConfigErr, &validationErr) {
					tui.RenderErrorString("Breaking config validation error in %s:", validationErr.FilePath)
					fmt.Print(validationErr.FormatValidationErrors())
					return breakingConfigErr
				}
				tui.RenderErrorString("Error loading breaking config: %v", breakingConfigErr)
				return breakingConfigErr
			}
			if breakingConfig != nil {
				utils.ApplyBreakingRulesConfig(breakingConfig)
				defer utils.ResetBreakingRulesConfig()
			}

			// Get files to process (handles glob patterns and direct args)
			filesToProcess, globErr := GetFilesToProcess(globPattern, args)
			if globErr != nil {
				tui.RenderErrorString("Error resolving files: %s", globErr.Error())
				return globErr
			}

			// check for file args
			if len(filesToProcess) == 0 {
				errText := "please supply an OpenAPI specification to generate an HTML Report"
				tui.RenderErrorString("%s", errText)
				return errors.New(errText)
			}

			// Ensure output directory exists for multi-file mode
			if outputDir != "" {
				if err := EnsureOutputDir(outputDir); err != nil {
					tui.RenderErrorString("Failed to create output directory '%s': %s", outputDir, err.Error())
					return err
				}
			}

			timeFlag, _ := cmd.Flags().GetBool("time")
			disableTimestamp, _ := cmd.Flags().GetBool("disableTimestamp")

			reportOutput := "report.html"

			if len(args) > 1 {
				reportOutput = args[1]
			}

			ignoredItems, err := LoadIgnoreFile(ignoreFile, silent, false, noStyleFlag)
			if err != nil {
				return err
			}

			// Multi-file mode detection
			isMultiFile := len(filesToProcess) > 1 || globPattern != ""

			if isMultiFile {
				tui.RenderInfo("Processing %d files...", len(filesToProcess))
				// Warn if change filtering flags are used with multi-file mode
				if changesFlag != "" || originalFlag != "" {
					tui.RenderInfo("Note: --changes and --original flags are ignored in multi-file mode")
				}
			}

			var processedFiles int

			for _, specFile := range filesToProcess {
				start := time.Now()

				vacuumReport, specBytes, _ := vacuum_report.BuildVacuumReportFromFile(specFile)
				if len(specBytes) <= 0 {
					tui.RenderErrorString("Failed to read specification: %v", specFile)
					if isMultiFile {
						continue
					}
					return errors.New("failed to read specification")
				}

				var resultSet *model.RuleResultSet
				var ruleset *motor.RuleSetExecutionResult
				var specIndex *index.SpecIndex
				var specInfo *datamodel.SpecInfo
				var stats *reports.ReportStatistics

				// if we have a pre-compiled report, jump straight to the end and collect $500
				if vacuumReport == nil {

					functionsFlag, _ := cmd.Flags().GetString("functions")
					customFunctions, _ := LoadCustomFunctions(functionsFlag, silent)

					rulesetFlag, _ := cmd.Flags().GetString("ruleset")

					// Certificate/TLS configuration
					certFile, _ := cmd.Flags().GetString("cert-file")
					keyFile, _ := cmd.Flags().GetString("key-file")
					caFile, _ := cmd.Flags().GetString("ca-file")
					insecure, _ := cmd.Flags().GetBool("insecure")

					// Resolve base path for this specific file
					resolvedBase, baseErr := ResolveBasePathForFile(specFile, baseFlag)
					if baseErr != nil {
						tui.RenderErrorString("Failed to resolve base path for '%s': %s", specFile, baseErr.Error())
						if isMultiFile {
							continue
						}
						return fmt.Errorf("failed to resolve base path: %w", baseErr)
					}

					httpFlags := &LintFlags{
						CertFile: certFile,
						KeyFile:  keyFile,
						CAFile:   caFile,
						Insecure: insecure,
					}
					httpClientConfig, cfgErr := GetHTTPClientConfig(httpFlags)
					if cfgErr != nil {
						return fmt.Errorf("failed to resolve TLS configuration: %w", cfgErr)
					}

					resultSet, ruleset, err = BuildResultsWithDocCheckSkip(false, hardModeFlag, rulesetFlag, specBytes, customFunctions,
						resolvedBase, remoteFlag, skipCheckFlag, time.Duration(timeoutFlag)*time.Second, time.Duration(lookupTimeoutFlag)*time.Millisecond, httpClientConfig, ignoredItems)
					if err != nil {
						tui.RenderError(err)
						if isMultiFile {
							continue
						}
						return err
					}
					specIndex = ruleset.Index
					specInfo = ruleset.SpecInfo

					if specInfo == nil {
						tui.RenderErrorString("Failed to parse specification: %v", specFile)
						if isMultiFile {
							continue
						}
						return errors.New("failed to parse specification")
					}
					specInfo.Generated = time.Now()
					stats = statistics.CreateReportStatistics(specIndex, specInfo, resultSet)

				} else {

					resultSet = model.NewRuleResultSetPointer(vacuumReport.ResultSet.Results)
					// Apply ignore filter to pre-compiled report results
					resultSet.Results = utils.FilterIgnoredResultsPtr(resultSet.Results, ignoredItems)
					specInfo = vacuumReport.SpecInfo
					stats = vacuumReport.Statistics

					// Recalculate error/warning/info counts and score after filtering
					if stats != nil && len(ignoredItems) > 0 {
						stats.TotalErrors = resultSet.GetErrorCount()
						stats.TotalWarnings = resultSet.GetWarnCount()
						stats.TotalInfo = resultSet.GetInfoCount()

						// Recalculate category statistics
						var catStats []*reports.CategoryStatistic
						for _, cat := range model.RuleCategoriesOrdered {
							var numIssues, numWarnings, numErrors, numInfo, numHints int
							numIssues = len(resultSet.GetResultsByRuleCategory(cat.Id))
							numWarnings = len(resultSet.GetWarningsByRuleCategory(cat.Id))
							numErrors = len(resultSet.GetErrorsByRuleCategory(cat.Id))
							numInfo = len(resultSet.GetInfoByRuleCategory(cat.Id))
							numHints = len(resultSet.GetHintByRuleCategory(cat.Id))
							numResults := len(resultSet.Results)
							var score int
							if numResults == 0 && numIssues == 0 {
								score = 100 // perfect
							} else if numResults > 0 {
								score = 100 - (numIssues * 100 / numResults)
							}
							catStats = append(catStats, &reports.CategoryStatistic{
								CategoryName: cat.Name,
								CategoryId:   cat.Id,
								NumIssues:    numIssues,
								Warnings:     numWarnings,
								Errors:       numErrors,
								Info:         numInfo,
								Hints:        numHints,
								Score:        score,
							})
						}
						stats.CategoryStatistics = catStats

						// Use the shared score calculation function
						stats.OverallScore = statistics.CalculateQualityScore(resultSet)
					}

					specInfo.Generated = vacuumReport.Generated
				}

				// Apply change-based filtering if --changes or --original is specified
				// Note: change filtering only makes sense for single-file mode
				if !isMultiFile {
					// Get DrDocument if available (only available for fresh linting, not pre-compiled reports)
					var drDoc *drModel.DrDocument
					if ruleset != nil && ruleset.RuleSetExecution != nil {
						drDoc = ruleset.RuleSetExecution.DrDocument
					}

					// Load changes first so we can use them for both filtering and violations
					var documentChanges *wcModel.DocumentChanges
					if originalFlag != "" {
						changeResult, changeErr := utils.GenerateChangeReportWithTree(originalFlag, specBytes, specFile)
						if changeErr != nil {
							if !silent {
								tui.RenderErrorString("Warning: Failed to generate change report: %v. Proceeding without change filtering.", changeErr)
							}
						} else if changeResult != nil {
							documentChanges = changeResult.DocumentChanges
						}
					} else if changesFlag != "" {
						var loadErr error
						documentChanges, loadErr = utils.LoadChangeReportFromFile(changesFlag)
						if loadErr != nil {
							if !silent {
								tui.RenderErrorString("Warning: Failed to load change report: %v. Proceeding without change filtering.", loadErr)
							}
						}
					}

					// Apply change filtering
					if documentChanges != nil {
						changeFilter := utils.NewChangeFilter(documentChanges, drDoc)
						resultSet.Results = changeFilter.FilterResults(resultSet.Results)
					}

					// Inject change violations if requested
					if documentChanges != nil && (warnOnChanges || errorOnBreaking) {
						changeViolations := utils.GenerateChangeViolations(documentChanges, utils.ChangeViolationOptions{
							WarnOnChanges:   warnOnChanges,
							ErrorOnBreaking: errorOnBreaking,
						})
						for _, v := range changeViolations {
							if v != nil {
								resultSet.Results = append(resultSet.Results, v)
							}
						}
					}

					// Recompute stats after change filtering/violations
					if stats != nil && documentChanges != nil {
						stats.TotalErrors = resultSet.GetErrorCount()
						stats.TotalWarnings = resultSet.GetWarnCount()
						stats.TotalInfo = resultSet.GetInfoCount()

						// Recalculate category statistics
						var catStats []*reports.CategoryStatistic
						for _, cat := range model.RuleCategoriesOrdered {
							numIssues := len(resultSet.GetResultsByRuleCategory(cat.Id))
							numWarnings := len(resultSet.GetWarningsByRuleCategory(cat.Id))
							numErrors := len(resultSet.GetErrorsByRuleCategory(cat.Id))
							numInfo := len(resultSet.GetInfoByRuleCategory(cat.Id))
							numHints := len(resultSet.GetHintByRuleCategory(cat.Id))
							numResults := len(resultSet.Results)
							var score int
							if numResults == 0 && numIssues == 0 {
								score = 100 // perfect
							} else if numResults > 0 {
								score = 100 - (numIssues * 100 / numResults)
							}
							catStats = append(catStats, &reports.CategoryStatistic{
								CategoryName: cat.Name,
								CategoryId:   cat.Id,
								NumIssues:    numIssues,
								Warnings:     numWarnings,
								Errors:       numErrors,
								Info:         numInfo,
								Hints:        numHints,
								Score:        score,
							})
						}
						stats.CategoryStatistics = catStats

						// Recalculate overall score
						stats.OverallScore = statistics.CalculateQualityScore(resultSet)
					}
				}

				duration := time.Since(start)

				// generate html report
				report := html_report.NewHTMLReport(specIndex, specInfo, resultSet, stats, disableTimestamp)

				generatedBytes := report.GenerateReport(false, GetVersion())

				// Determine output filename
				var outputFile string
				if isMultiFile {
					timestamp := time.Now().Format("01-02-06-15_04_05")
					outputFile = GenerateReportFileName(specFile, outputDir, "report", timestamp, ".html")
				} else {
					outputFile = reportOutput
				}

				err = os.WriteFile(outputFile, generatedBytes, 0664)

				if err != nil {
					tui.RenderErrorString("Unable to write HTML report file: '%s': %s", outputFile, err.Error())
					if isMultiFile {
						continue
					}
					return err
				}

				tui.RenderSuccess("HTML Report generated for '%s', written to '%s'", specFile, outputFile)

				fi, _ := os.Stat(specFile)
				if fi != nil {
					RenderTime(timeFlag, duration, fi.Size())
				}
				processedFiles++
			}

			// Summary for multi-file mode
			if isMultiFile {
				tui.RenderInfo("Processed %d files successfully", processedFiles)
			}

			return nil
		},
	}
	cmd.Flags().BoolP("disableTimestamp", "d", false, "Disable timestamp in report")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().BoolP("no-banner", "b", false, "Disable the banner output")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().String("globbed-files", "", "Glob pattern of files to process (e.g., 'specs/*.yaml')")
	cmd.Flags().String("output-dir", "", "Directory to write report files to (default: current directory)")

	return cmd
}
