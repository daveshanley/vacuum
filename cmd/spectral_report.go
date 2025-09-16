// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"

	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/daveshanley/vacuum/cui"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
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
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				color.DisableColors()
			}

			if !stdIn && !stdOut {
				PrintBanner()
			}

			// check for file args
			if !stdIn && len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a spectral report, or use " +
					"the -i flag to use stdin"
				tui.RenderErrorString("%s", errText)
				return errors.New(errText)
			}

			timeFlag, _ := cmd.Flags().GetBool("time")
			noPretty, _ := cmd.Flags().GetBool("no-pretty")

			// Certificate/TLS configuration
			certFile, _ := cmd.Flags().GetString("cert-file")
			keyFile, _ := cmd.Flags().GetString("key-file")
			caFile, _ := cmd.Flags().GetString("ca-file")
			insecure, _ := cmd.Flags().GetBool("insecure")

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
				tui.RenderErrorString("Unable to read file '%s': %s", args[0], fileError.Error())
				return fileError
			}

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
					tui.RenderStyledBox(HardModeEnabled, tui.BoxTypeHard, noStyleFlag)
				}

			}

			functionsFlag, _ := cmd.Flags().GetString("functions")
			customFunctions, _ := LoadCustomFunctions(functionsFlag, true)

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {
				// Create HTTP client for remote ruleset downloads if needed
				var httpClient *http.Client
				httpClientConfig := utils.HTTPClientConfig{
					CertFile: certFile,
					KeyFile:  keyFile,
					CAFile:   caFile,
					Insecure: insecure,
				}
				if utils.ShouldUseCustomHTTPClient(httpClientConfig) {
					var clientErr error
					httpClient, clientErr = utils.CreateCustomHTTPClient(httpClientConfig)
					if clientErr != nil {
						tui.RenderErrorString("Failed to create custom HTTP client: %s", clientErr.Error())
						return clientErr
					}
				}

				var rsErr error
				selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, httpClient)
				if rsErr != nil {
					tui.RenderErrorString("Unable to load ruleset '%s': %s", rulesetFlag, rsErr.Error())
					return rsErr
				}

				// Merge OWASP rules if hard mode is enabled
				if MergeOWASPRulesToRuleSet(selectedRS, hardModeFlag) {
					if !stdIn && !stdOut {
						tui.RenderStyledBox(HardModeWithCustomRuleset, tui.BoxTypeHard, noStyleFlag)
					}
				}
			}

			if !stdIn && !stdOut {
				tui.RenderInfo("Linting against %d rules: %s", len(selectedRS.Rules), selectedRS.DocumentationURI)
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
				HTTPClientConfig: utils.HTTPClientConfig{
					CertFile: certFile,
					KeyFile:  keyFile,
					CAFile:   caFile,
					Insecure: insecure,
				},
			})

			resultSet := model.NewRuleResultSet(ruleset.Results)
			resultSet.SortResultsByLineNumber()

			resultSet.Results = utils.FilterIgnoredResultsPtr(resultSet.Results, ignoredItems)

			duration := time.Since(start)

			var source string
			if stdIn {
				source = "stdin"
			} else {
				source = args[0]
				// Make the path relative to current working directory for consistency
				if absPath, err := filepath.Abs(source); err == nil {
					if cwd, err := os.Getwd(); err == nil {
						if relPath, err := filepath.Rel(cwd, absPath); err == nil {
							source = relPath
						}
					}
				}
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
				tui.RenderErrorString("Unable to write report file: '%s': %s", reportOutput, err.Error())
				return err
			}

			tui.RenderSuccess("Report generated for '%s', written to '%s'", args[0], reportOutput)

			fi, _ := os.Stat(args[0])
			RenderTime(timeFlag, duration, fi.Size())

			return nil
		},
	}
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("no-pretty", "n", false, "Render JSON with no formatting")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	return cmd

}
