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
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"time"
)

// GetHTMLReportCommand returns a cobra command for generating an HTML Report.
func GetHTMLReportCommand() *cobra.Command {

	return &cobra.Command{
		Use:   "html-report",
		Short: "Generate an HTML report (Work In Progress)",
		Long: "Generate an interactive and useful HTML report (this is not ready yet). Default output " +
			"filename is 'report.html' located in the working directory.",
		Example: "vacuum html-report <my-awesome-spec.yaml> <report.html>",
		RunE: func(cmd *cobra.Command, args []string) error {

			PrintBanner()

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate an HTML Report"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			timeFlag, _ := cmd.Flags().GetBool("time")

			reportOutput := "report.html"

			if len(args) > 1 {
				reportOutput = args[1]
			}

			start := time.Now()
			var err error
			vacuumReport, specBytes, _ := vacuum_report.BuildVacuumReportFromFile(args[0])

			var resultSet *model.RuleResultSet
			var ruleset *motor.RuleSetExecutionResult
			var specIndex *model.SpecIndex
			var specInfo *model.SpecInfo
			var stats *reports.ReportStatistics

			// if we have a pre-compiled report, jump straight to the end and collect $500
			if vacuumReport == nil {

				rulesetFlag, _ := cmd.Flags().GetString("ruleset")
				resultSet, ruleset, err = BuildResults(rulesetFlag, specBytes)
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
			report := html_report.NewHTMLReport(specIndex, specInfo, resultSet, stats)

			generatedBytes := report.GenerateReport(false)
			//generatedBytes := report.GenerateReport(true) // test mode

			err = ioutil.WriteFile(reportOutput, generatedBytes, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write HTML report file: '%s': %s\n", reportOutput, err.Error())
				pterm.Println()
				return err
			}

			pterm.Info.Printf("HTML Report generated for '%s', written to '%s'\n", args[0], reportOutput)
			pterm.Println()

			fi, _ := os.Stat(args[0])
			RenderTime(timeFlag, duration, fi)

			return nil
		},
	}
}
