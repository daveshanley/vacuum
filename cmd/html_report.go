// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	html_report "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/statistics"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"time"
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

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				pterm.DisableColor()
				pterm.DisableStyling()
			}

			PrintBanner()

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate an HTML Report"
				pterm.Error.Println(errText)
				pterm.Println()
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
				pterm.Error.Printf("Failed to read specification: %v\n\n", args[0])
				return err
			}

			var resultSet *model.RuleResultSet
			var ruleset *motor.RuleSetExecutionResult
			var specIndex *index.SpecIndex
			var specInfo *datamodel.SpecInfo
			var stats *reports.ReportStatistics

			// if we have a pre-compiled report, jump straight to the end and collect $500
			if vacuumReport == nil {

				functionsFlag, _ := cmd.Flags().GetString("functions")
				customFunctions, _ := LoadCustomFunctions(functionsFlag)

				rulesetFlag, _ := cmd.Flags().GetString("ruleset")
				resultSet, ruleset, err = BuildResultsWithDocCheckSkip(rulesetFlag, specBytes, customFunctions,
					baseFlag, skipCheckFlag, time.Duration(timeoutFlag)*time.Second)
				if err != nil {
					pterm.Error.Printf("Failed to generate report: %v\n\n", err)
					return err
				}
				specIndex = ruleset.Index
				specInfo = ruleset.SpecInfo

				specInfo.Generated = time.Now()
				stats = statistics.CreateReportStatistics(specIndex, specInfo, resultSet)

			} else {

				resultSet = model.NewRuleResultSetPointer(vacuumReport.ResultSet.Results)
				specInfo = vacuumReport.SpecInfo
				stats = vacuumReport.Statistics
				specInfo.Generated = vacuumReport.Generated
			}

			duration := time.Since(start)

			// generate html report
			report := html_report.NewHTMLReport(specIndex, specInfo, resultSet, stats, disableTimestamp)

			generatedBytes := report.GenerateReport(false, Version)
			//generatedBytes := report.GenerateReport(true) // test mode

			err = os.WriteFile(reportOutput, generatedBytes, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write HTML report file: '%s': %s\n", reportOutput, err.Error())
				pterm.Println()
				return err
			}

			pterm.Success.Printf("HTML Report generated for '%s', written to '%s'\n", args[0], reportOutput)
			pterm.Println()

			fi, _ := os.Stat(args[0])
			RenderTime(timeFlag, duration, fi.Size())

			return nil
		},
	}
	cmd.Flags().BoolP("disableTimestamp", "d", false, "Disable timestamp in report")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")

	return cmd
}
