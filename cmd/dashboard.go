// Copyright 2020-2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pterm/pterm"
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
			// Check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				pterm.Error.Println(errText)
				pterm.Println()
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
				pterm.Error.Printf("Failed to load file: %v\n\n", err)
				return err
			}

			var resultSet *model.RuleResultSet
			var specBytes []byte
			displayFileName := reportOrSpec.FileName

			if reportOrSpec.IsReport {
				// Using a pre-compiled report
				if !silent {
					pterm.Info.Printf("Loading pre-compiled vacuum report from '%s'\n", args[0])
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

				// Setup logging
				handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
					Level: slog.LevelError,
				})
				logger := slog.New(handler)

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
						pterm.Info.Printf("🚨 HARD MODE ENABLED 🚨\n")
					}
				}

				// Handle custom ruleset
				if rulesetFlag != "" {
					var rsErr error
					selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, nil)
					if rsErr != nil {
						pterm.Error.Printf("Unable to load ruleset '%s': %s\n", rulesetFlag, rsErr.Error())
						return rsErr
					}
					if hardModeFlag {
						MergeOWASPRulesToRuleSet(selectedRS, true)
					}
				}

				// Display linting info
				if !silent {
					pterm.Info.Printf("Linting file '%s' against %d rules: %s\n",
						displayFileName, len(selectedRS.Rules), selectedRS.DocumentationURI)
				}

				// Apply rules with proper filename
				result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
					RuleSet:           selectedRS,
					Spec:              specBytes,
					SpecFileName:      displayFileName,  // THIS IS THE KEY FIX
					CustomFunctions:   customFuncs,
					Base:              baseFlag,
					AllowLookup:       remoteFlag,
					SkipDocumentCheck: skipCheckFlag,
					Logger:            logger,
					Timeout:           time.Duration(timeoutFlag) * time.Second,
					HTTPClientConfig:  utils.HTTPClientConfig{
						CertFile: certFile,
						KeyFile:  keyFile,
						CAFile:   caFile,
						Insecure: insecure,
					},
				})

				// Filter ignored results
				result.Results = utils.FilterIgnoredResults(result.Results, ignoredItems)

				// Check for errors
				if len(result.Errors) > 0 {
					for _, err := range result.Errors {
						pterm.Error.Printf("Unable to process spec '%s': %s\n", displayFileName, err.Error())
					}
					return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
				}

				// Process results
				resultSet = model.NewRuleResultSet(result.Results)
				resultSet.SortResultsByLineNumber()
			}

			// Check if we have results
			if resultSet == nil || len(resultSet.Results) == 0 {
				pterm.Println()
				pterm.Success.Println("There is nothing to see, no results found - well done!")
				pterm.Println()
				return nil
			}

			// Launch the new interactive table view
			if !silent {
				pterm.Info.Println("Launching interactive dashboard...")
			}

			err = cui.ShowViolationTableView(resultSet.Results, displayFileName, specBytes)
			if err != nil {
				pterm.Error.Printf("Failed to show dashboard: %v\n", err)
				return err
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	
	return cmd
}