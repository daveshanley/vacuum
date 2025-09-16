// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"

	"github.com/daveshanley/vacuum/color"
	html_report "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/daveshanley/vacuum/utils"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"

	"os"
	"time"

	"github.com/daveshanley/vacuum/tui"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

// GetHTMLReportCommand returns a cobra command for generating an HTML Report.
func GetHTMLReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "html-report",
		Short:        "Generate an HTML report of a linting run",
		Long: "Generate an interactive and useful HTML report. Default output " +
			"filename is 'report.html' located in the working directory.",
		Example: "vacuum html-report <my-awesome-spec.yaml> <report.html>",
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
			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			silent, _ := cmd.Flags().GetBool("silent")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				color.DisableColors()
			}

			PrintBanner()

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate an HTML Report"
				tui.RenderErrorString("%s", errText)
				return errors.New(errText)
			}

			timeFlag, _ := cmd.Flags().GetBool("time")
			disableTimestamp, _ := cmd.Flags().GetBool("disableTimestamp")

			reportOutput := "report.html"

			if len(args) > 1 {
				reportOutput = args[1]
			}

			start := time.Now()
			var err error
			vacuumReport, specBytes, _ := vacuum_report.BuildVacuumReportFromFile(args[0])
			if len(specBytes) <= 0 {
				tui.RenderErrorString("Failed to read specification: %v", args[0])
				return err
			}

			var resultSet *model.RuleResultSet
			var ruleset *motor.RuleSetExecutionResult
			var specIndex *index.SpecIndex
			var specInfo *datamodel.SpecInfo
			var stats *reports.ReportStatistics

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

				resultSet, ruleset, err = BuildResultsWithDocCheckSkip(false, hardModeFlag, rulesetFlag, specBytes, customFunctions,
					baseFlag, remoteFlag, skipCheckFlag, time.Duration(timeoutFlag)*time.Second, utils.HTTPClientConfig{
						CertFile: certFile,
						KeyFile:  keyFile,
						CAFile:   caFile,
						Insecure: insecure,
					}, ignoredItems)
				if err != nil {
					tui.RenderError(err)
					return err
				}
				specIndex = ruleset.Index
				specInfo = ruleset.SpecInfo

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
						} else {
							score = numIssues / numResults * 100
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

			duration := time.Since(start)

			// generate html report
			report := html_report.NewHTMLReport(specIndex, specInfo, resultSet, stats, disableTimestamp)

			generatedBytes := report.GenerateReport(false, GetVersion())
			//generatedBytes := report.GenerateReport(true) // test mode

			err = os.WriteFile(reportOutput, generatedBytes, 0664)

			if err != nil {
				tui.RenderErrorString("Unable to write HTML report file: '%s': %s", reportOutput, err.Error())
				return err
			}

			tui.RenderSuccess("HTML Report generated for '%s', written to '%s'", args[0], reportOutput)

			fi, _ := os.Stat(args[0])
			RenderTime(timeFlag, duration, fi.Size())

			return nil
		},
	}
	cmd.Flags().BoolP("disableTimestamp", "d", false, "Disable timestamp in report")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")

	return cmd
}
