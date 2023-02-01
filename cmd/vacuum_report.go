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
	"os"
	"time"
)

func GetVacuumReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "report",
		Short:         "Generate a vacuum sealed, re-playable report",
		Long: "Generate a full report of a linting run. This can be used as a result set, or can be used to replay a linting run. " +
			"the default filename is 'vacuum-report-MM-DD-YY-HH_MM_SS.json' located in the working directory. " +
			"Use the -i flag for using stdin instead of reading a file, and -o for stdout, instead of writing to a file.",
		Example: "vacuum report <my-awesome-spec.yaml> <report-prefix>",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			stdIn, _ := cmd.Flags().GetBool("stdin")
			stdOut, _ := cmd.Flags().GetBool("stdout")
			noStyleFlag, _ := cmd.Flags().GetBool("no-style")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				pterm.DisableColor()
				pterm.DisableStyling()
			}

			if !stdIn && !stdOut {
				PrintBanner()
			}

			// check for file args
			if !stdIn && len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report, or use the -i flag to use stdin"
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

			var specBytes []byte
			var fileError error

			if stdIn {
				// read file from stdin
				inputReader := cmd.InOrStdin()
				buf := &bytes.Buffer{}
				_, fileError = buf.ReadFrom(inputReader)
				specBytes = buf.Bytes()

			} else {
				// read file from filesystem
				specBytes, fileError = os.ReadFile(args[0])
			}

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

				rsBytes, rsErr := os.ReadFile(rulesetFlag)
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

			if !stdIn && !stdOut {
				pterm.Info.Printf("Linting against %d rules: %s\n", len(selectedRS.Rules), selectedRS.DocumentationURI)
			}

			ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				RuleSet:         selectedRS,
				Spec:            specBytes,
				CustomFunctions: customFunctions,
				SilenceLogs:     true,
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

			if stdOut {
				fmt.Print(string(reportData))
				return nil
			}

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

			err = os.WriteFile(reportOutputName, reportData, 0664)
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
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("compress", "c", false, "Compress results using gzip")
	cmd.Flags().BoolP("no-pretty", "n", false, "Render JSON with no formatting")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	return cmd
}
