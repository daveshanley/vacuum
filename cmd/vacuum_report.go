// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"time"
)

func GetVacuumReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "report",
		Short:         "Generate a vacuum sealed, replayable report",
		Long: "Generate a full report of a linting run. This can be used as a result set, or can be used to replay a linting run. " +
			"the default filename is 'vacuum-report-MM-DD-YY-HH_MM_SS.json' located in the working directory.",
		Example: "vacuum report <my-awesome-spec.yaml> <report-prefix>",
		RunE: func(cmd *cobra.Command, args []string) error {

			PrintBanner()

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			timeFlag, _ := cmd.Flags().GetBool("time")
			noPretty, _ := cmd.Flags().GetBool("no-pretty")
			compress, _ := cmd.Flags().GetBool("compress")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			extension := ".json"

			reportOutput := "vacuum-report"

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

			// read spec and parse to dashboard.
			defaultRuleSets := rulesets.BuildDefaultRuleSets()

			// default is recommended rules, based on spectral (for now anyway)
			selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

			functionsFlag, _ := cmd.Flags().GetString("functions")
			var customFunctions map[string]model.RuleFunction

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {

				customFunctions, _ = LoadCustomFunctions(functionsFlag)

				rsBytes, rsErr := ioutil.ReadFile(rulesetFlag)
				if rsErr != nil {
					pterm.Error.Printf("Unable to read ruleset file '%s': %s\n", rulesetFlag, rsErr.Error())
					pterm.Println()
					return rsErr
				}
				selectedRS, rsErr = BuildRuleSetFromUserSuppliedSet(rsBytes, defaultRuleSets)
				if rsErr != nil {
					return rsErr
				}
			}

			pterm.Info.Printf("Linting against %d rules: %s\n", len(selectedRS.Rules), selectedRS.DocumentationURI)

			ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				RuleSet:         selectedRS,
				Spec:            specBytes,
				CustomFunctions: customFunctions,
			})

			resultSet := model.NewRuleResultSet(ruleset.Results)
			resultSet.SortResultsByLineNumber()

			duration := time.Since(start)

			// pre-render
			resultSet.PrepareForSerialization(ruleset.SpecInfo)

			var data []byte
			var err error

			// generate statistics
			stats := statistics.CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

			// create vacuum report
			vr := vacuum_report.VacuumReport{
				Generated:  time.Now(),
				SpecInfo:   ruleset.SpecInfo,
				ResultSet:  resultSet,
				Statistics: stats,
			}

			if noPretty || compress {
				data, _ = json.Marshal(vr)
			} else {
				data, _ = json.MarshalIndent(vr, "", "    ")
			}

			reportData := data

			if compress {

				var b bytes.Buffer
				gz := gzip.NewWriter(&b)
				_, wErr := gz.Write(data)
				if wErr != nil {
					return wErr
				}
				wErr = gz.Close()
				if wErr != nil {
					return wErr
				}
				reportData = b.Bytes()
				extension = ".json.gz"
			}

			reportOutputName := fmt.Sprintf("%s-%s%s",
				reportOutput, vr.Generated.Format("01-02-06-15_04_05"), extension)

			err = ioutil.WriteFile(reportOutputName, reportData, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write report file: '%s': %s\n", reportOutputName, err.Error())
				pterm.Println()
				return err
			}

			pterm.Success.Printf("Report generated for '%s', written to '%s'\n", args[0], reportOutputName)
			pterm.Println()

			fi, _ := os.Stat(args[0])
			RenderTime(timeFlag, duration, fi)

			return nil
		},
	}
	cmd.Flags().BoolP("compress", "c", false, "Compress results using gzip")
	cmd.Flags().BoolP("no-pretty", "n", false, "Render JSON with no formatting")
	return cmd
}
