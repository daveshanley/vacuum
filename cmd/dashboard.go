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
	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
			// Print banner
			PrintBanner()

			// Check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				style := createResultBoxStyle(cui.RGBRed, cui.RGBDarkRed)
				messageStyle := lipgloss.NewStyle().Padding(1, 1)
				fmt.Println(style.Render(messageStyle.Render(errText)))
				fmt.Println()
				return errors.New(errText)
			}

			// Read flags
			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			silent, _ := cmd.Flags().GetBool("silent")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")
			functionsFlag, _ := cmd.Flags().GetString("functions")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			// Certificate/TLS configuration
			certFile, _ := cmd.Flags().GetString("cert-file")
			keyFile, _ := cmd.Flags().GetString("key-file")
			caFile, _ := cmd.Flags().GetString("ca-file")
			insecure, _ := cmd.Flags().GetBool("insecure")

			// Load ignore file if specified
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

			// Try to load the file as either a report or spec
			reportOrSpec, err := LoadFileAsReportOrSpec(args[0])
			if err != nil {
				message := fmt.Sprintf("Failed to load file: %v", err)
				style := createResultBoxStyle(cui.RGBRed, cui.RGBDarkRed)
				messageStyle := lipgloss.NewStyle().Padding(1, 1)
				fmt.Println(style.Render(messageStyle.Render(message)))
				fmt.Println()
				return err
			}

			var resultSet *model.RuleResultSet
			var specBytes []byte
			displayFileName := reportOrSpec.FileName

			if reportOrSpec.IsReport {
				// Using a pre-compiled report
				if !silent {
					// Create info box for loading report
					message := fmt.Sprintf("Loading pre-compiled vacuum report from '%s'", args[0])
					style := createResultBoxStyle(cui.RGBBlue, cui.RGBDarkBlue)
					messageStyle := lipgloss.NewStyle().Padding(1, 1)
					fmt.Println(style.Render(messageStyle.Render(message)))
					fmt.Println()
				}

				// Create a new RuleResultSet from the results to ensure proper initialization
				if reportOrSpec.ResultSet != nil && reportOrSpec.ResultSet.Results != nil {
					// Filter ignored results
					filteredResults := utils.FilterIgnoredResultsPtr(reportOrSpec.ResultSet.Results, ignoredItems)
					// Create properly initialized RuleResultSet
					resultSet = model.NewRuleResultSetPointer(filteredResults)
				} else {
					resultSet = model.NewRuleResultSetPointer([]*model.RuleFunctionResult{})
				}

				specBytes = reportOrSpec.SpecBytes
			} else {
				// Regular spec file - run linting (same as lint-preview)
				specBytes = reportOrSpec.SpecBytes

				// Setup logging with BufferedLogger
				var logger *slog.Logger
				var bufferedLogger *BufferedLogger
				bufferedLogger = NewBufferedLoggerWithLevel(cui.LogLevelError)
				handler := NewBufferedLogHandler(bufferedLogger)
				logger = slog.New(handler)

				// Build ruleset
				defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
				selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
				customFuncs, _ := LoadCustomFunctions(functionsFlag, silent)

				// Handle hard mode
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

				// Handle custom ruleset
				if rulesetFlag != "" {
					var rsErr error
					selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, nil)
					if rsErr != nil {
						if !silent {
							message := fmt.Sprintf("Unable to load ruleset '%s': %s", rulesetFlag, rsErr.Error())
							style := createResultBoxStyle(cui.RGBRed, cui.RGBDarkRed)
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

				// Display linting info
				if !silent {
					fmt.Printf(" %svacuuming file '%s' against %d rules: %s%s\n\n",
						cui.ASCIIBlue, displayFileName, len(selectedRS.Rules), selectedRS.DocumentationURI, cui.ASCIIReset)
				}

				// Apply rules with proper filename
				result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
					RuleSet:           selectedRS,
					Spec:              specBytes,
					SpecFileName:      displayFileName, // THIS IS THE KEY FIX
					CustomFunctions:   customFuncs,
					Base:              baseFlag,
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

				// Filter ignored results
				result.Results = utils.FilterIgnoredResults(result.Results, ignoredItems)

				// Output any buffered logs
				RenderBufferedLogs(bufferedLogger, false)

				// Check for errors
				if len(result.Errors) > 0 {
					if !silent {
						// Create error box for each error
						for _, err := range result.Errors {
							message := fmt.Sprintf("Unable to process spec '%s': %s", displayFileName, err.Error())
							style := createResultBoxStyle(cui.RGBRed, cui.RGBDarkRed)
							messageStyle := lipgloss.NewStyle().Padding(1, 1)
							fmt.Println(style.Render(messageStyle.Render(message)))
						}
					}
					return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
				}

				// Process results
				resultSet = model.NewRuleResultSet(result.Results)
				resultSet.SortResultsByLineNumber()
			}

			// Check if we have results
			if resultSet == nil || len(resultSet.Results) == 0 {
				if !silent {
					renderResultBox(0, 0, 0) // Perfect score
				}
				return nil
			}

			// Launch the new interactive table view
			if !silent {
				// Create info box for launching dashboard
				message := "Launching interactive vacuum dashboard..."
				style := createResultBoxStyle(cui.RGBBlue, cui.RGBDarkBlue)
				messageStyle := lipgloss.NewStyle().Padding(1, 1)
				fmt.Println(style.Render(messageStyle.Render(message)))
			}

			err = cui.ShowViolationTableView(resultSet.Results, displayFileName, specBytes)
			if err != nil {
				if !silent {
					message := fmt.Sprintf("Failed to show dashboard: %v", err)
					style := createResultBoxStyle(cui.RGBRed, cui.RGBDarkRed)
					messageStyle := lipgloss.NewStyle().Padding(1, 1)
					fmt.Println(style.Render(messageStyle.Render(message)))
				}
				return err
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().String("ignore-file", "", "Path to ignore file")

	return cmd
}
