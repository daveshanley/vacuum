// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"github.com/daveshanley/vacuum/cui"
	html_report "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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

			// read file.
			specBytes, fileError := ioutil.ReadFile(args[0])

			if fileError != nil {
				pterm.Error.Printf("Unable to read file '%s': %s\n", args[0], fileError.Error())
				pterm.Println()
				return fileError
			}

			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			// read spec and parse
			defaultRuleSets := rulesets.BuildDefaultRuleSets()

			// default is recommended rules, based on spectral (for now anyway)
			selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {

				rsBytes, rsErr := ioutil.ReadFile(rulesetFlag)
				if rsErr != nil {
					pterm.Error.Printf("Unable to read ruleset file '%s': %s\n", rulesetFlag, rsErr.Error())
					pterm.Println()
					return rsErr
				}
				selectedRS, rsErr = cui.BuildRuleSetFromUserSuppliedSet(rsBytes, defaultRuleSets)
				if rsErr != nil {
					return rsErr
				}
			}

			pterm.Info.Printf("Running vacuum against spec '%s' against %d rules: %s\n\n%s\n", args[0],
				len(selectedRS.Rules), selectedRS.DocumentationURI, selectedRS.Description)
			pterm.Println()

			ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				RuleSet: selectedRS,
				Spec:    specBytes,
			})

			resultSet := model.NewRuleResultSet(ruleset.Results)
			resultSet.SortResultsByLineNumber()

			duration := time.Since(start)

			// generate statistics
			stats := statistics.CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

			// generate html report
			report := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats)
			//generatedBytes := report.GenerateReport(false)
			generatedBytes := report.GenerateReport(true)

			err := ioutil.WriteFile(reportOutput, generatedBytes, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write HTML report file: '%s': %s\n", reportOutput, fileError.Error())
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
