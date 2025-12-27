// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
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
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/spf13/cobra"
)

func GetSpectralReportCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "spectral-report",
		Short:        "Generate a Spectral compatible JSON report",
		Long: `Generate a JSON report using the same model as Spectral. Default output filename is 'vacuum-spectral-report.json' located in the working directory.
Use the -i flag for using stdin instead of reading a file, and -o for stdout, instead of writing to a file.

For multiple files, use --globbed-files to specify a glob pattern:
  vacuum spectral-report --globbed-files "specs/*.yaml" --output-dir reports/

This generates one Spectral-format report per input file, named after the source spec.`,
		Example: `vacuum spectral-report my-awesome-spec.yaml vacuum-spectral-report.json
vacuum spectral-report --globbed-files "specs/*.yaml" --output-dir reports/
vacuum spectral-report --globbed-files "api/**/*.json" -n`,
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
			lookupTimeoutFlag, _ := cmd.Flags().GetInt("lookup-timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")
			changesFlag, _ := cmd.Flags().GetString("changes")
			originalFlag, _ := cmd.Flags().GetString("original")
			globPattern, _ := cmd.Flags().GetString("globbed-files")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			breakingConfigPath, _ := cmd.Flags().GetString("breaking-config")
			warnOnChanges, _ := cmd.Flags().GetBool("warn-on-changes")
			errorOnBreaking, _ := cmd.Flags().GetBool("error-on-breaking")

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
				errText := "please supply an OpenAPI specification to generate a spectral report, or use " +
					"the -i flag to use stdin"
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

			// Certificate/TLS configuration
			certFile, _ := cmd.Flags().GetString("cert-file")
			keyFile, _ := cmd.Flags().GetString("key-file")
			caFile, _ := cmd.Flags().GetString("ca-file")
			insecure, _ := cmd.Flags().GetBool("insecure")
			allowPrivateNetworks, _ := cmd.Flags().GetBool("allow-private-networks")
			fetchTimeout, _ := cmd.Flags().GetInt("fetch-timeout")

			lintFlags := &LintFlags{
				CertFile:             certFile,
				KeyFile:              keyFile,
				CAFile:               caFile,
				Insecure:             insecure,
				AllowPrivateNetworks: allowPrivateNetworks,
				FetchTimeout:         fetchTimeout,
			}

			httpClientConfig, cfgErr := GetHTTPClientConfig(lintFlags)
			if cfgErr != nil {
				return fmt.Errorf("failed to resolve TLS configuration: %w", cfgErr)
			}

			fetchConfig, fetchCfgErr := GetFetchConfig(lintFlags)
			if fetchCfgErr != nil {
				return fmt.Errorf("failed to resolve fetch configuration: %w", fetchCfgErr)
			}

			reportOutput := "vacuum-spectral-report.json"

			if len(args) > 1 {
				reportOutput = args[1]
			}

			ignoredItems, err := LoadIgnoreFile(ignoreFile, true, stdOut, noStyleFlag)
			if err != nil {
				return err
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

			if !stdIn && !stdOut {
				tui.RenderInfo("Linting against %d rules: %s", len(selectedRS.Rules), selectedRS.DocumentationURI)
			}

			// Multi-file mode detection
			isMultiFile := len(filesToProcess) > 1 || globPattern != ""

			if isMultiFile && !stdIn && !stdOut {
				tui.RenderInfo("Processing %d files...", len(filesToProcess))
				// Warn if change filtering flags are used with multi-file mode
				if changesFlag != "" || originalFlag != "" {
					tui.RenderInfo("Note: --changes and --original flags are ignored in multi-file mode")
				}
			}

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
						continue
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
					Timeout:                         time.Duration(timeoutFlag) * time.Second,
					NodeLookupTimeout:               time.Duration(lookupTimeoutFlag) * time.Millisecond,
					ExtractReferencesFromExtensions: extensionRefsFlag,
					HTTPClientConfig:                httpClientConfig,
					FetchConfig:                     fetchConfig,
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

				var source string
				if stdIn {
					source = "stdin"
				} else {
					source = specFile
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

				// Determine output filename
				var outputFile string
				if isMultiFile {
					timestamp := time.Now().Format("01-02-06-15_04_05")
					outputFile = GenerateReportFileName(specFile, outputDir, "spectral-report", timestamp, ".json")
				} else {
					outputFile = reportOutput
				}

				if err = os.WriteFile(outputFile, data, 0664); err != nil {
					tui.RenderErrorString("Unable to write report file: '%s': %s", outputFile, err.Error())
					if isMultiFile {
						continue
					}
					return err
				}

				tui.RenderSuccess("Report generated for '%s', written to '%s'", specFile, outputFile)

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

			return nil
		},
	}
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("no-pretty", "n", false, "Render JSON with no formatting")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().String("globbed-files", "", "Glob pattern of files to process (e.g., 'specs/*.yaml')")
	cmd.Flags().String("output-dir", "", "Directory to write report files to (default: current directory)")
	return cmd

}
