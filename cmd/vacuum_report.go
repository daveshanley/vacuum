// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/spf13/cobra"
)

func GetVacuumReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "report",
		Short:        "Generate a vacuum sealed, re-playable report",
		Long: `Generate a full report of a linting run. This can be used as a result set, or can be used to replay a linting run.
The default filename is 'vacuum-report-MM-DD-YY-HH_MM_SS.json' located in the working directory.
Use the -i flag for using stdin instead of reading a file, and -o for stdout, instead of writing to a file.

For multiple files, use --globbed-files to specify a glob pattern:
  vacuum report --globbed-files "specs/*.yaml" --output-dir reports/

This generates one report per input file, named after the source spec.`,
		Example: `vacuum report my-awesome-spec.yaml report-prefix
vacuum report --globbed-files "specs/*.yaml" --output-dir reports/
vacuum report --globbed-files "api/**/*.json" -c`,
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
			baseFlag, _ := cmd.Flags().GetString("base")
			junitFlag, _ := cmd.Flags().GetBool("junit")
			junitFailOnWarn, _ := cmd.Flags().GetBool("junit-fail-on-warn")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			lookupTimeoutFlag, _ := cmd.Flags().GetInt("lookup-timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
			minScore, _ := cmd.Flags().GetInt("min-score")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			changesFlag, _ := cmd.Flags().GetString("changes")
			originalFlag, _ := cmd.Flags().GetString("original")
			globPattern, _ := cmd.Flags().GetString("globbed-files")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			breakingConfigPath, _ := cmd.Flags().GetString("breaking-config")
			warnOnChanges, _ := cmd.Flags().GetBool("warn-on-changes")
			errorOnBreaking, _ := cmd.Flags().GetBool("error-on-breaking")
			turboFlag, _ := cmd.Flags().GetBool("turbo")
			skipResolveFlag, _ := cmd.Flags().GetBool("skip-resolve")
			skipCircularCheckFlag, _ := cmd.Flags().GetBool("skip-circular-check")
			skipSchemaErrorsFlag, _ := cmd.Flags().GetBool("skip-schema-errors")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				color.DisableColors()
			}

			if !stdIn && !stdOut {
				PrintBanner()
			}

			// Load and apply breaking rules config early, before any change comparison
			breakingConfig, breakingConfigErr := utils.LoadBreakingRulesConfig(breakingConfigPath)
			if breakingConfigErr != nil {
				var validationErr *utils.ConfigValidationError
				if errors.As(breakingConfigErr, &validationErr) {
					tui.RenderErrorString("Breaking config validation error in %s:", validationErr.FilePath)
					fmt.Print(validationErr.FormatValidationErrors())
					return breakingConfigErr
				}
				tui.RenderErrorString("Error loading breaking config: %v", breakingConfigErr)
				return breakingConfigErr
			}
			if breakingConfig != nil {
				utils.ApplyBreakingRulesConfig(breakingConfig)
				defer utils.ResetBreakingRulesConfig()
			}

			// Get files to process (handles glob patterns and direct args)
			filesToProcess, globErr := GetFilesToProcess(globPattern, args)
			if globErr != nil {
				tui.RenderErrorString("Error resolving files: %s", globErr.Error())
				return globErr
			}

			// check for file args
			if !stdIn && len(filesToProcess) == 0 {
				errText := "please supply an OpenAPI specification to generate a report, or use the -i flag to use stdin"
				tui.RenderErrorString("%s", errText)
				return errors.New(errText)
			}

			// Ensure output directory exists for multi-file mode
			if outputDir != "" {
				if err := EnsureOutputDir(outputDir); err != nil {
					tui.RenderErrorString("Failed to create output directory '%s': %s", outputDir, err.Error())
					return err
				}
			}

			timeFlag, _ := cmd.Flags().GetBool("time")
			noPretty, _ := cmd.Flags().GetBool("no-pretty")
			compress, _ := cmd.Flags().GetBool("compress")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			// Certificate/TLS configuration
			certFile, _ := cmd.Flags().GetString("cert-file")
			keyFile, _ := cmd.Flags().GetString("key-file")
			caFile, _ := cmd.Flags().GetString("ca-file")
			insecure, _ := cmd.Flags().GetBool("insecure")
			allowPrivateNetworks, _ := cmd.Flags().GetBool("allow-private-networks")
			allowHTTP, _ := cmd.Flags().GetBool("allow-http")
			fetchTimeout, _ := cmd.Flags().GetInt("fetch-timeout")

			httpFlags := &LintFlags{
				CertFile:             certFile,
				KeyFile:              keyFile,
				CAFile:               caFile,
				Insecure:             insecure,
				AllowPrivateNetworks: allowPrivateNetworks,
				AllowHTTP:            allowHTTP,
				FetchTimeout:         fetchTimeout,
			}
			httpClientConfig, cfgErr := GetHTTPClientConfig(httpFlags)
			if cfgErr != nil {
				return fmt.Errorf("failed to resolve TLS configuration: %w", cfgErr)
			}

			fetchConfig, fetchCfgErr := GetFetchConfig(httpFlags)
			if fetchCfgErr != nil {
				return fmt.Errorf("failed to resolve fetch configuration: %w", fetchCfgErr)
			}

			extension := ".json"

			reportOutput := "vacuum-report"

			if len(args) > 1 {
				reportOutput = args[1]
			}

			ignoredItems, err := LoadIgnoreFile(ignoreFile, stdIn || stdOut, stdOut, noStyleFlag)
			if err != nil {
				return err
			}

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
				httpClient, clientErr := utils.CreateHTTPClientIfNeeded(httpClientConfig)
				if clientErr != nil {
					tui.RenderErrorString("Failed to create custom HTTP client: %s", clientErr.Error())
					return clientErr
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

			if turboFlag {
				rulesets.FilterRulesForTurbo(selectedRS)
			}

			if !stdIn && !stdOut {
				tui.RenderInfo("Linting against %d rules: %s", len(selectedRS.Rules), selectedRS.DocumentationURI)
			}

			deepGraph := false
			if ignoreFile != "" {
				deepGraph = true
			}

			// Multi-file mode: process each file and generate individual reports
			isMultiFile := len(filesToProcess) > 1 || globPattern != ""

			if isMultiFile && !stdIn && !stdOut {
				tui.RenderInfo("Processing %d files...", len(filesToProcess))
				// Warn if change filtering flags are used with multi-file mode
				if changesFlag != "" || originalFlag != "" {
					tui.RenderInfo("Note: --changes and --original flags are ignored in multi-file mode")
				}
			}

			// Track lowest score across all files for threshold check
			var lowestScore int = 100
			var processedFiles int

			// Process files - for stdin mode, we only process once
			filesToIterate := filesToProcess
			if stdIn {
				filesToIterate = []string{"stdin"}
			}

			for _, specFile := range filesToIterate {
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
					specBytes, fileError = os.ReadFile(specFile)
				}

				if fileError != nil {
					tui.RenderErrorString("Unable to read file '%s': %s", specFile, fileError.Error())
					if isMultiFile {
						continue // Skip this file and continue with others
					}
					return fileError
				}

				// Resolve base path for this specific file
				var resolvedBase string
				var baseErr error
				if stdIn {
					// For stdin input, use the provided base flag or current directory as fallback
					if baseFlag != "" {
						resolvedBase = baseFlag
					} else {
						resolvedBase, baseErr = filepath.Abs(".")
						if baseErr != nil {
							return fmt.Errorf("failed to resolve current directory as base path: %w", baseErr)
						}
					}
				} else {
					resolvedBase, baseErr = ResolveBasePathForFile(specFile, baseFlag)
					if baseErr != nil {
						tui.RenderErrorString("Failed to resolve base path for '%s': %s", specFile, baseErr.Error())
						if isMultiFile {
							continue
						}
						return fmt.Errorf("failed to resolve base path: %w", baseErr)
					}
				}

				ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
					RuleSet:                         selectedRS,
					Spec:                            specBytes,
					CustomFunctions:                 customFunctions,
					SilenceLogs:                     true,
					Base:                            resolvedBase,
					AllowLookup:                     remoteFlag,
					SkipDocumentCheck:               skipCheckFlag,
					BuildDeepGraph:                  deepGraph,
					Timeout:                         time.Duration(timeoutFlag) * time.Second,
					NodeLookupTimeout:               time.Duration(lookupTimeoutFlag) * time.Millisecond,
					ExtractReferencesFromExtensions: extensionRefsFlag,
					HTTPClientConfig:                httpClientConfig,
					FetchConfig:                     fetchConfig,
					TurboMode:                       turboFlag,
					SkipResolve:                     skipResolveFlag,
					SkipCircularCheck:               skipCircularCheckFlag,
					SkipSchemaErrors:                skipSchemaErrorsFlag,
				})

				resultSet := model.NewRuleResultSet(ruleset.Results)
				resultSet.SortResultsByLineNumber()

				resultSet.Results = utils.FilterIgnoredResultsPtr(resultSet.Results, ignoredItems)

				// Apply change-based filtering if --changes or --original is specified
				// Note: change filtering only makes sense for single-file mode
				var documentChanges *wcModel.DocumentChanges
				if !isMultiFile && ruleset != nil && ruleset.RuleSetExecution != nil {
					// Load changes first so we can use them for both filtering and violations
					if originalFlag != "" {
						changeResult, changeErr := utils.GenerateChangeReportWithTree(originalFlag, specBytes, specFile)
						if changeErr != nil {
							if !stdIn && !stdOut {
								tui.RenderErrorString("Warning: Failed to generate change report: %v. Proceeding without change filtering.", changeErr)
							}
						} else if changeResult != nil {
							documentChanges = changeResult.DocumentChanges
						}
					} else if changesFlag != "" {
						var loadErr error
						documentChanges, loadErr = utils.LoadChangeReportFromFile(changesFlag)
						if loadErr != nil {
							if !stdIn && !stdOut {
								tui.RenderErrorString("Warning: Failed to load change report: %v. Proceeding without change filtering.", loadErr)
							}
						}
					}

					// Apply change filtering
					if documentChanges != nil {
						changeFilter := utils.NewChangeFilter(documentChanges, ruleset.RuleSetExecution.DrDocument)
						resultSet.Results = changeFilter.FilterResults(resultSet.Results)
					}

					// Inject change violations if requested
					if documentChanges != nil && (warnOnChanges || errorOnBreaking) {
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

				duration := time.Since(start)

				// if we want jUnit output, then build the report and be done with it.
				if junitFlag {
					junitConfig := vacuum_report.JUnitConfig{FailOnWarn: junitFailOnWarn}
					junitXML := vacuum_report.BuildJUnitReportWithConfig(resultSet, start, []string{specFile}, junitConfig)
					if stdOut {
						fmt.Print(string(junitXML))
						return nil
					}

					timestamp := time.Now().Format("01-02-06-15_04_05")
					var reportOutputName string
					if isMultiFile {
						reportOutputName = GenerateReportFileName(specFile, outputDir, reportOutput, timestamp, ".xml")
					} else {
						reportOutputName = fmt.Sprintf("%s-%s%s", reportOutput, timestamp, ".xml")
					}

					if err = os.WriteFile(reportOutputName, junitXML, 0664); err != nil {
						tui.RenderErrorString("Unable to write junit report file: '%s': %s", reportOutputName, err.Error())
						if isMultiFile {
							continue
						}
						return err
					}

					tui.RenderSuccess("JUnit Report generated for '%s', written to '%s'", specFile, reportOutputName)
					processedFiles++
					continue
				}

				// pre-render
				resultSet.PrepareForSerialization(ruleset.SpecInfo)

				var data []byte

				// generate statistics
				stats := statistics.CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

				// Track lowest score for threshold check
				if stats != nil && stats.OverallScore < lowestScore {
					lowestScore = stats.OverallScore
				}

				// Extract all unique rules used in the results
				usedRules := make(map[string]*model.Rule)
				for _, result := range resultSet.Results {
					if result.Rule != nil && result.RuleId != "" {
						usedRules[result.RuleId] = result.Rule
					}
				}

				// create vacuum report
				vr := vacuum_report.VacuumReport{
					Generated:  time.Now(),
					SpecInfo:   ruleset.SpecInfo,
					ResultSet:  resultSet,
					Statistics: stats,
					Rules:      usedRules,
				}

				if noPretty || compress {
					data, _ = json.Marshal(vr)
				} else {
					data, _ = json.MarshalIndent(vr, "", "    ")
				}

				reportData := data

				if stdOut {
					fmt.Print(string(reportData))
					if minScore > 10 && stats != nil && stats.OverallScore < minScore {
						return fmt.Errorf("score threshold failed, overall score is %d, and the threshold is %d",
							stats.OverallScore, minScore)
					}
					return nil
				}

				fileExtension := extension
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
					fileExtension = ".json.gz"
				}

				timestamp := vr.Generated.Format("01-02-06-15_04_05")
				var reportOutputName string
				if isMultiFile {
					reportOutputName = GenerateReportFileName(specFile, outputDir, reportOutput, timestamp, fileExtension)
				} else {
					reportOutputName = fmt.Sprintf("%s-%s%s", reportOutput, timestamp, fileExtension)
				}

				err = os.WriteFile(reportOutputName, reportData, 0664)
				if err != nil {
					tui.RenderErrorString("Unable to write report file: '%s': %s", reportOutputName, err.Error())
					if isMultiFile {
						continue
					}
					return err
				}

				tui.RenderSuccess("Report generated for '%s', written to '%s'", specFile, reportOutputName)
				if !stdIn {
					fi, _ := os.Stat(specFile)
					if fi != nil {
						RenderTime(timeFlag, duration, fi.Size())
					}
				}
				processedFiles++
			}

			// Summary for multi-file mode
			if isMultiFile && !stdOut {
				tui.RenderInfo("Processed %d files successfully", processedFiles)
			}

			// Check threshold against lowest score across all files
			if minScore > 10 && lowestScore < minScore {
				return fmt.Errorf("score threshold failed, lowest overall score is %d, and the threshold is %d",
					lowestScore, minScore)
			}

			return nil
		},
	}
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("junit", "j", false, "Generate report in JUnit format (cannot be compressed)")
	cmd.Flags().Bool("junit-fail-on-warn", false, "Treat warnings as failures in JUnit report (default: only errors are failures)")
	cmd.Flags().BoolP("compress", "c", false, "Compress results using gzip")
	cmd.Flags().BoolP("no-pretty", "n", false, "Render JSON with no formatting")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().Int("min-score", 10, "Throw an error return code if the score is below this value")
	cmd.Flags().String("globbed-files", "", "Glob pattern of files to process (e.g., 'specs/*.yaml')")
	cmd.Flags().String("output-dir", "", "Directory to write report files to (default: current directory)")
	return cmd
}
