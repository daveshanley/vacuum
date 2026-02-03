// Copyright 2020-2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/logging"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	drModel "github.com/pb33f/doctor/model"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/spf13/cobra"
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
			lookupTimeoutFlag, _ := cmd.Flags().GetInt("lookup-timeout")
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
			allowPrivateNetworks, _ := cmd.Flags().GetBool("allow-private-networks")
			allowHTTP, _ := cmd.Flags().GetBool("allow-http")
			fetchTimeout, _ := cmd.Flags().GetInt("fetch-timeout")
			watchFlag, _ := cmd.Flags().GetBool("watch")
			changesFlag, _ := cmd.Flags().GetString("changes")
			originalFlag, _ := cmd.Flags().GetString("original")
			breakingConfigPath, _ := cmd.Flags().GetString("breaking-config")
			warnOnChanges, _ := cmd.Flags().GetBool("warn-on-changes")
			errorOnBreaking, _ := cmd.Flags().GetBool("error-on-breaking")

			// Load and apply breaking rules config early, before any change comparison
			breakingConfig, breakingConfigErr := utils.LoadBreakingRulesConfig(breakingConfigPath)
			if breakingConfigErr != nil {
				var validationErr *utils.ConfigValidationError
				if errors.As(breakingConfigErr, &validationErr) {
					message := fmt.Sprintf("Breaking config validation error in %s:", validationErr.FilePath)
					style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
					messageStyle := lipgloss.NewStyle().Padding(1, 1)
					fmt.Println(style.Render(messageStyle.Render(message)))
					fmt.Print(validationErr.FormatValidationErrors())
					return breakingConfigErr
				}
				message := fmt.Sprintf("Error loading breaking config: %v", breakingConfigErr)
				style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
				messageStyle := lipgloss.NewStyle().Padding(1, 1)
				fmt.Println(style.Render(messageStyle.Render(message)))
				return breakingConfigErr
			}
			if breakingConfig != nil {
				utils.ApplyBreakingRulesConfig(breakingConfig)
				defer utils.ResetBreakingRulesConfig()
			}

			ignoredItems, err := LoadIgnoreFile(ignoreFile, silent, false, false)
			if err != nil {
				return err
			}

			// Create HTTP client for URL support (with TLS config)
			httpClientConfig := utils.HTTPClientConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   caFile,
				Insecure: insecure,
			}
			var httpClient *http.Client
			if utils.ShouldUseCustomHTTPClient(httpClientConfig) {
				httpClient, err = utils.CreateCustomHTTPClient(httpClientConfig)
				if err != nil {
					return fmt.Errorf("failed to create HTTP client: %w", err)
				}
			}

			reportOrSpec, err := LoadFileAsReportOrSpecWithClient(args[0], httpClient)
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
			var drDocument *drModel.DrDocument // To hold DrDocument for change filtering
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

				tempLintFlags := &LintFlags{
					CertFile:             certFile,
					KeyFile:              keyFile,
					CAFile:               caFile,
					Insecure:             insecure,
					AllowPrivateNetworks: allowPrivateNetworks,
					AllowHTTP:            allowHTTP,
					FetchTimeout:         fetchTimeout,
				}
				httpConfig, err := GetHTTPClientConfig(tempLintFlags)
				if err != nil {
					return fmt.Errorf("failed to resolve TLS configuration: %w", err)
				}

				fetchConfig, fetchCfgErr := GetFetchConfig(tempLintFlags)
				if fetchCfgErr != nil {
					return fmt.Errorf("failed to resolve fetch configuration: %w", fetchCfgErr)
				}

				if rulesetFlag != "" {
					httpClient, clientErr := utils.CreateHTTPClientIfNeeded(httpConfig)
					if clientErr != nil {
						return fmt.Errorf("failed to create custom HTTP client: %w", clientErr)
					}

					var rsErr error
					selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, httpClient)
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
					SpecFileName:      displayFileName,
					CustomFunctions:   customFuncs,
					Base:              resolvedBase,
					AllowLookup:       remoteFlag,
					SkipDocumentCheck: skipCheckFlag,
					Logger:            logger,
					Timeout:           time.Duration(timeoutFlag) * time.Second,
					NodeLookupTimeout: time.Duration(lookupTimeoutFlag) * time.Millisecond,
					HTTPClientConfig:  httpConfig,
					FetchConfig:       fetchConfig,
				})

				result.Results = utils.FilterIgnoredResults(result.Results, ignoredItems)

				// Store DrDocument for change filtering
				if result.RuleSetExecution != nil {
					drDocument = result.RuleSetExecution.DrDocument
				}

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

			// Apply change-based filtering if --changes or --original is specified
			var changeStats *utils.ChangeStats
			var filterStats *utils.ChangeFilterStats
			if changesFlag != "" || originalFlag != "" {
				var documentChanges *wcModel.DocumentChanges
				var changesErr error

				if originalFlag != "" {
					documentChanges, changesErr = utils.GenerateChangeReport(originalFlag, specBytes, displayFileName)
				} else {
					documentChanges, changesErr = utils.LoadChangeReportFromFile(changesFlag)
				}

				if changesErr != nil {
					if !silent {
						message := fmt.Sprintf("Warning: Failed to load changes: %v. Proceeding without change filtering.", changesErr)
						style := createResultBoxStyle(color.RGBRed, color.RGBDarkRed)
						messageStyle := lipgloss.NewStyle().Padding(1, 1)
						fmt.Println(style.Render(messageStyle.Render(message)))
					}
				} else if documentChanges != nil {
					changeStats = utils.ExtractChangeStats(documentChanges)

					changeFilter := utils.NewChangeFilter(documentChanges, drDocument)
					if changeFilter != nil {
						resultSet.Results, filterStats = changeFilter.FilterResultsWithStats(resultSet.Results)
					}

					// Inject change violations if requested
					if warnOnChanges || errorOnBreaking {
						changeViolations := utils.GenerateChangeViolations(documentChanges, utils.ChangeViolationOptions{
							WarnOnChanges:   warnOnChanges,
							ErrorOnBreaking: errorOnBreaking,
						})
						for _, v := range changeViolations {
							if v != nil {
								resultSet.Results = append(resultSet.Results, v)
							}
						}
					}
				}
			}

			if resultSet == nil || len(resultSet.Results) == 0 {
				if !silent {
					renderResultBox(0, 0, 0, 0) // Perfect score
				}
				// If not in watch mode, exit early since there's nothing to show
				if !watchFlag {
					return nil
				}
				// In watch mode, continue to dashboard with empty results so user can
				// see violations appear as they edit the file (issue #797)
				resultSet = model.NewRuleResultSetPointer([]*model.RuleFunctionResult{})
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
				Enabled:           watchFlag,
				BaseFlag:          baseFlag,
				SkipCheckFlag:     skipCheckFlag,
				TimeoutFlag:       timeoutFlag,
				HardModeFlag:      hardModeFlag,
				RemoteFlag:        remoteFlag,
				IgnoreFile:        ignoreFile,
				FunctionsFlag:     functionsFlag,
				RulesetFlag:       rulesetFlag,
				CertFile:          certFile,
				KeyFile:           keyFile,
				CAFile:            caFile,
				Insecure:          insecure,
				Silent:            silent,
				CustomFunctions:   customFuncs,
				OriginalSpecPath:  originalFlag,
				ChangesReportPath: changesFlag,
			}

			err = tui.ShowViolationTableView(resultSet.Results, displayFileName, specBytes, watchConfig, changeStats, filterStats)
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
