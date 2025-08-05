// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"time"
)

func GetSpectralReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "spectral-report",
		Short:        "Generate a Spectral compatible JSON report",
		Long: "Generate a JSON report using the same model as Spectral. Default output " +
			"filename is 'vacuum-spectral-report.json' located in the working directory. " +
			"Use the -i flag for using stdin instead of reading a file, and -o for stdout, instead of writing to a file.",
		Example: "vacuum spectral-report my-awesome-spec.yaml <vacuum-spectral-report.json>",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0:
				return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
			case 1:
				return []string{"json"}, cobra.ShellCompDirectiveFilterFileExt
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			stdIn, _ := cmd.Flags().GetBool("stdin")
			stdOut, _ := cmd.Flags().GetBool("stdout")
			noStyleFlag, _ := cmd.Flags().GetBool("no-style")
			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
			remoteFlag, _ := cmd.Flags().GetBool("remote")

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
				errText := "please supply an OpenAPI specification to generate a spectral report, or use " +
					"the -i flag to use stdin"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			timeFlag, _ := cmd.Flags().GetBool("time")
			noPretty, _ := cmd.Flags().GetBool("no-pretty")

			reportOutput := "vacuum-spectral-report.json"

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

			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			// read spec and parse to dashboard.
			defaultRuleSets := rulesets.BuildDefaultRuleSets()

			// default is recommended rules, based on spectral (for now anyway)
			selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

			// HARD MODE
			if hardModeFlag {
				selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()

				// extract all OWASP Rules
				owaspRules := rulesets.GetAllOWASPRules()
				allRules := selectedRS.Rules
				for k, v := range owaspRules {
					allRules[k] = v
				}
				if !stdIn && !stdOut {
					box := pterm.DefaultBox.WithLeftPadding(5).WithRightPadding(5)
					box.BoxStyle = pterm.NewStyle(pterm.FgLightRed)
					box.Println(pterm.LightRed("🚨 HARD MODE ENABLED 🚨"))
					pterm.Println()
				}

			}

			functionsFlag, _ := cmd.Flags().GetString("functions")
			customFunctions, _ := LoadCustomFunctions(functionsFlag, true)

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {
				var rsErr error
				selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag)
				if rsErr != nil {
					pterm.Error.Printf("Unable to load ruleset '%s': %s\n", rulesetFlag, rsErr.Error())
					pterm.Println()
					return rsErr
				}
			}

			if !stdIn && !stdOut {
				pterm.Info.Printf("Linting against %d rules: %s\n", len(selectedRS.Rules), selectedRS.DocumentationURI)
			}

			ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				RuleSet:                         selectedRS,
				Spec:                            specBytes,
				CustomFunctions:                 customFunctions,
				SilenceLogs:                     true,
				Base:                            baseFlag,
				AllowLookup:                     remoteFlag,
				SkipDocumentCheck:               skipCheckFlag,
				Timeout:                         time.Duration(timeoutFlag) * time.Second,
				ExtractReferencesFromExtensions: extensionRefsFlag,
			})

			resultSet := model.NewRuleResultSet(ruleset.Results)
			resultSet.SortResultsByLineNumber()

			duration := time.Since(start)

			var source string
			if stdIn {
				source = "stdin"
			} else {
				source = args[0] // todo: convert to full path.
			}
			// serialize
			spectralReport := resultSet.GenerateSpectralReport(source)

			var data []byte
			if noPretty {
				data, _ = json.Marshal(spectralReport)
			} else {
				data, _ = json.MarshalIndent(spectralReport, "", "    ")
			}

			if stdOut {
				fmt.Print(string(data))
				return nil
			}

			err := os.WriteFile(reportOutput, data, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write report file: '%s': %s\n", reportOutput, err.Error())
				pterm.Println()
				return err
			}

			pterm.Success.Printf("Report generated for '%s', written to '%s'\n", args[0], reportOutput)
			pterm.Println()

			fi, _ := os.Stat(args[0])
			RenderTime(timeFlag, duration, fi.Size())

			return nil
		},
	}
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("no-pretty", "n", false, "Render JSON with no formatting")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	return cmd

}
