// Copyright 2020-2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/logging"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

func GetDashboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dashboard",
		Short:   "Show interactive console dashboard for linting report",
		Long:    "Interactive console dashboard to explore linting report in detail using modern TUI",
		Example: "vacuum dashboard my-awesome-spec.yaml",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			PrintBanner()

			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
				messageStyle := lipgloss.NewStyle().Padding(1, 1)
				fmt.Println(style.Render(messageStyle.Render(errText)))
				fmt.Println()
				return errors.New(errText)
			}

			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			silent, _ := cmd.Flags().GetBool("silent")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")
			functionsFlag, _ := cmd.Flags().GetString("functions")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")
			certFile, _ := cmd.Flags().GetString("cert-file")
			keyFile, _ := cmd.Flags().GetString("key-file")
			caFile, _ := cmd.Flags().GetString("ca-file")
			insecure, _ := cmd.Flags().GetBool("insecure")
			watchFlag, _ := cmd.Flags().GetBool("watch")

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

			reportOrSpec, err := LoadFileAsReportOrSpec(args[0])
			if err != nil {
				message := fmt.Sprintf("Failed to load file: %v", err)
				style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
				messageStyle := lipgloss.NewStyle().Padding(1, 1)
				fmt.Println(style.Render(messageStyle.Render(message)))
				fmt.Println()
				return err
			}

			var resultSet *model.RuleResultSet
			var specBytes []byte
			displayFileName := reportOrSpec.FileName

			if reportOrSpec.IsReport {
				if !silent {
					message := fmt.Sprintf("loading pre-compiled vacuum report from '%s'", args[0])
					style := createResultBoxStyle(color.RGBBlue, color.RGBDarkBlue)
					messageStyle := lipgloss.NewStyle().Padding(1, 1)
					fmt.Println(style.Render(messageStyle.Render(message)))
					fmt.Println()
				}

				if reportOrSpec.ResultSet != nil && reportOrSpec.ResultSet.Results != nil {
					filteredResults := utils.FilterIgnoredResultsPtr(reportOrSpec.ResultSet.Results, ignoredItems)
					resultSet = model.NewRuleResultSetPointer(filteredResults)
				} else {
					resultSet = model.NewRuleResultSetPointer([]*model.RuleFunctionResult{})
				}

				specBytes = reportOrSpec.SpecBytes
			} else {
				// regular spec file - run linting (same as lint)
				specBytes = reportOrSpec.SpecBytes

				var logger *slog.Logger
				var bufferedLogger *logging.BufferedLogger
				bufferedLogger = logging.NewBufferedLoggerWithLevel(logging.LogLevelError)
				handler := logging.NewBufferedLogHandler(bufferedLogger)
				logger = slog.New(handler)

				defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
				selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
				customFuncs, _ := LoadCustomFunctions(functionsFlag, silent)

				if hardModeFlag {
					selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
					owaspRules := rulesets.GetAllOWASPRules()
					for k, v := range owaspRules {
						selectedRS.Rules[k] = v
					}
					if !silent {
						renderHardModeBox(HardModeEnabled, false)
					}
				}

				if rulesetFlag != "" {
					var rsErr error
					selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, nil)
					if rsErr != nil {
						if !silent {
							message := fmt.Sprintf("Unable to load ruleset '%s': %s", rulesetFlag, rsErr.Error())
							style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
							messageStyle := lipgloss.NewStyle().Padding(1, 1)
							fmt.Println(style.Render(messageStyle.Render(message)))
						}
						return rsErr
					}
					if hardModeFlag {
						if MergeOWASPRulesToRuleSet(selectedRS, true) {
							if !silent {
								renderHardModeBox(HardModeWithCustomRuleset, false)
							}
						}
					}
				}

				if !silent {
					fmt.Printf(" %svacuuming file '%s' against %d rules: %s%s\n\n",
						color.ASCIIBlue, displayFileName, len(selectedRS.Rules), selectedRS.DocumentationURI, color.ASCIIReset)
				}

				// Resolve base path for this specific file
				resolvedBase, baseErr := ResolveBasePathForFile(args[0], baseFlag)
				if baseErr != nil {
					return fmt.Errorf("failed to resolve base path: %w", baseErr)
				}

				result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
					RuleSet:           selectedRS,
					Spec:              specBytes,
					SpecFileName:      displayFileName, // THIS IS THE KEY FIX
					CustomFunctions:   customFuncs,
					Base:              resolvedBase,
					AllowLookup:       remoteFlag,
					SkipDocumentCheck: skipCheckFlag,
					Logger:            logger,
					Timeout:           time.Duration(timeoutFlag) * time.Second,
					HTTPClientConfig: utils.HTTPClientConfig{
						CertFile: certFile,
						KeyFile:  keyFile,
						CAFile:   caFile,
						Insecure: insecure,
					},
				})

				result.Results = utils.FilterIgnoredResults(result.Results, ignoredItems)

				RenderBufferedLogs(bufferedLogger, false)

				if len(result.Errors) > 0 {
					if !silent {
						// Create error box for each error
						for _, err := range result.Errors {
							message := fmt.Sprintf("Unable to process spec '%s': %s", displayFileName, err.Error())
							style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
							messageStyle := lipgloss.NewStyle().Padding(1, 1)
							fmt.Println(style.Render(messageStyle.Render(message)))
						}
					}
					return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
				}

				resultSet = model.NewRuleResultSet(result.Results)
				resultSet.SortResultsByLineNumber()
			}

			if resultSet == nil || len(resultSet.Results) == 0 {
				if !silent {
					renderResultBox(0, 0, 0) // Perfect score
				}
				return nil
			}

			if !silent {
				message := "launching interactive vacuum dashboard..."
				style := createResultBoxStyle(color.RGBBlue, color.RGBDarkBlue)
				messageStyle := lipgloss.NewStyle().Padding(1, 1)
				fmt.Println(style.Render(messageStyle.Render(message)))

				if watchFlag {
					watchMessage := fmt.Sprintf("watching for changes on file '%s'", displayFileName)
					fmt.Println(style.Render(messageStyle.Render(watchMessage)))
				}
			}

			// Load custom functions
			var customFuncs map[string]model.RuleFunction
			if functionsFlag != "" {
				customFuncs, err = LoadCustomFunctions(functionsFlag, silent)
				if err != nil && !silent {
					message := fmt.Sprintf("Failed to load custom functions: %v", err)
					style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
					messageStyle := lipgloss.NewStyle().Padding(1, 1)
					fmt.Println(style.Render(messageStyle.Render(message)))
				}
			}

			watchConfig := &tui.WatchConfig{
				Enabled:         watchFlag,
				BaseFlag:        baseFlag,
				SkipCheckFlag:   skipCheckFlag,
				TimeoutFlag:     timeoutFlag,
				HardModeFlag:    hardModeFlag,
				RemoteFlag:      remoteFlag,
				IgnoreFile:      ignoreFile,
				FunctionsFlag:   functionsFlag,
				RulesetFlag:     rulesetFlag,
				CertFile:        certFile,
				KeyFile:         keyFile,
				CAFile:          caFile,
				Insecure:        insecure,
				Silent:          silent,
				CustomFunctions: customFuncs,
			}

			err = tui.ShowViolationTableView(resultSet.Results, displayFileName, specBytes, watchConfig)
			if err != nil {
				if !silent {
					message := fmt.Sprintf("Failed to show dashboard: %v", err)
					style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
					messageStyle := lipgloss.NewStyle().Padding(1, 1)
					fmt.Println(style.Render(messageStyle.Render(message)))
				}
				return err
			}

			return nil
		},
	}

	// dashboard flags
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().BoolP("watch", "W", false, "Watch for file changes and automatically re-lint")

	return cmd
}
