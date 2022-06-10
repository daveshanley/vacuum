// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"github.com/daveshanley/vacuum/cui"
	html_report "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"time"
)

// GetHTMLReportCommand returns a cobra command for generating an HTML Report.
func GetHTMLReportCommand() *cobra.Command {

	// TODO: there is a large duplicate of code in here, copied from the spectral report command.
	// this needs to be unified and refactored into shared code.

	return &cobra.Command{
		Use:   "html-report",
		Short: "Generate an HTML report (Work In Progress)",
		Long: "Generate an interactive and useful HTML report (this is not ready yet). Default output " +
			"filename is 'report.html' located in the working directory.",
		Example: "vacuum html-report <my-awesome-spec.yaml> <report.html>",
		RunE: func(cmd *cobra.Command, args []string) error {

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
				resultSet, ruleset, err = buildResults(rulesetFlag, specBytes)
				specIndex = ruleset.Index
				specInfo = ruleset.SpecInfo
				specInfo.Generated = time.Now()
				stats = statistics.CreateReportStatistics(specIndex, specInfo, resultSet)

			} else {

				resultSet = model.NewRuleResultSetPointer(vacuumReport.ResultSet.Results)

				// now we need to re-index everything, but we don't run any rules.
				var rootNode yaml.Node
				err = yaml.Unmarshal(*vacuumReport.SpecInfo.SpecBytes, &rootNode)
				if err != nil {
					pterm.Error.Printf("Unable to read spec bytes from report file '%s': %s\n", args[0], err.Error())
					pterm.Println()
					return err
				}

				specIndex = model.NewSpecIndex(&rootNode)
				specInfo = vacuumReport.SpecInfo
				stats = vacuumReport.Statistics
				specInfo.Generated = vacuumReport.Generated
			}

			duration := time.Since(start)

			// generate html report
			report := html_report.NewHTMLReport(specIndex, specInfo, resultSet, stats)

			generatedBytes := report.GenerateReport(false)
			//generatedBytes := report.GenerateReport(true)

			err = ioutil.WriteFile(reportOutput, generatedBytes, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write HTML report file: '%s': %s\n", reportOutput, err.Error())
				pterm.Println()
				return err
			}

			pterm.Info.Printf("HTML Report generated for '%s', written to '%s'\n", args[0], reportOutput)
			pterm.Println()

			fi, _ := os.Stat(args[0])
			cui.RenderTime(timeFlag, duration, fi)

			return nil
		},
	}
}

func buildResults(rulesetFlag string, specBytes []byte) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {

	// read spec and parse
	defaultRuleSets := rulesets.BuildDefaultRuleSets()

	// default is recommended rules, based on spectral (for now anyway)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	// if ruleset has been supplied, lets make sure it exists, then load it in
	// and see if it's valid. If so - let's go!
	if rulesetFlag != "" {

		rsBytes, rsErr := ioutil.ReadFile(rulesetFlag)
		if rsErr != nil {
			return nil, nil, rsErr
		}
		selectedRS, rsErr = cui.BuildRuleSetFromUserSuppliedSet(rsBytes, defaultRuleSets)
		if rsErr != nil {
			return nil, nil, rsErr
		}
	}

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	resultSet.SortResultsByLineNumber()
	return resultSet, ruleset, nil
}
