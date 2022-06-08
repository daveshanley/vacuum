// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"time"
)

func GetVacuumReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a vacuum report",
		Long: "Generate a full report of a linting run. This can be used as a result set, or can be used to replay a linting run. " +
			"the default filename is 'vacuum-report-MM-DD-YY-HH_MM_SS.json' located in the working directory.",
		Example: "vacuum report my-awesome-spec.yaml <vacuum-spectral-report.json>",
		RunE: func(cmd *cobra.Command, args []string) error {

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			timeFlag, _ := cmd.Flags().GetBool("time")
			useYaml, _ := cmd.Flags().GetBool("yaml")
			noPretty, _ := cmd.Flags().GetBool("no-pretty")
			compress, _ := cmd.Flags().GetBool("compress")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			extension := ".json"

			if useYaml {
				extension = ".yaml"
			}
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

			// pre-render
			resultSet.PrepareForSerialization(ruleset.SpecInfo)

			var data []byte
			var err error

			// create vacuum report
			vr := model.VacuumReport{
				Generated: time.Now(),
				SpecInfo:  ruleset.SpecInfo,
				ResultSet: resultSet,
			}

			if !useYaml {
				if noPretty || compress {
					data, err = json.Marshal(vr)
				} else {
					data, err = json.MarshalIndent(vr, "", "    ")
				}
			} else {
				data, err = yaml.Marshal(vr)
			}

			if err != nil {
				pterm.Error.Printf("Unable to marshal report '%s': %s\n", args[0], fileError.Error())
				pterm.Println()
				return err
			}

			reportData := data

			if compress {

				var b bytes.Buffer
				gz := gzip.NewWriter(&b)
				if _, err = gz.Write(data); err != nil {
					pterm.Error.Printf("Unable to compress report '%s': %s\n", args[0], err.Error())
					pterm.Println()
					return err
				}
				if err = gz.Close(); err != nil {
					pterm.Error.Printf("Unable to close compressed report '%s': %s\n", args[0], err.Error())
					pterm.Println()
					return err
				}
				reportData = b.Bytes()
				extension = ".gz"
			}

			reportOutputName := fmt.Sprintf("%s-%s%s",
				reportOutput, vr.Generated.Format("01-02-06-15_04_05"), extension)

			err = ioutil.WriteFile(reportOutputName, reportData, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write report file: '%s': %s\n", reportOutputName, fileError.Error())
				pterm.Println()
				return err
			}

			pterm.Info.Printf("Report generated for '%s', written to '%s'\n", args[0], reportOutputName)
			pterm.Println()

			fi, _ := os.Stat(args[0])
			cui.RenderTime(timeFlag, duration, fi)

			return nil
		},
	}
	cmd.Flags().BoolP("compress", "c", false, "Compress results using gzip")
	cmd.Flags().BoolP("yaml", "y", false, "Render using YAML instead of JSON")
	cmd.Flags().BoolP("no-pretty", "n", false, "Render JSON with no indenting (not available for YAML)")
	return cmd
}
