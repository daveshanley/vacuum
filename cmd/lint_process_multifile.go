// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// FileProcessingResult contains the results of processing a single file
type FileProcessingResult struct {
	Results  []*model.RuleFunctionResult
	Errors   int
	Warnings int
	Informs  int
	FileSize int64
	Logs     []string
	Error    error
}

// ProcessSingleFileWithLogs processes a single file and captures logs
func ProcessSingleFileWithLogs(cmd *cobra.Command, fileName string) *FileProcessingResult {

	var fileSize int64
	fileInfo, err := os.Stat(fileName)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	// read flags we need
	baseFlag, _ := cmd.Flags().GetString("base")
	remoteFlag, _ := cmd.Flags().GetBool("remote")
	skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
	rulesetFlag, _ := cmd.Flags().GetString("ruleset")
	functionsFlag, _ := cmd.Flags().GetString("functions")
	hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
	ignoreFile, _ := cmd.Flags().GetString("ignore-file")
	extRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
	ignoreArrayCircleRef, _ := cmd.Flags().GetBool("ignore-array-circle-ref")
	ignorePolymorphCircleRef, _ := cmd.Flags().GetBool("ignore-polymorph-circle-ref")
	certFile, _ := cmd.Flags().GetString("cert-file")
	keyFile, _ := cmd.Flags().GetString("key-file")
	caFile, _ := cmd.Flags().GetString("ca-file")
	insecure, _ := cmd.Flags().GetBool("insecure")
	debugFlag, _ := cmd.Flags().GetBool("debug")
	silentFlag, _ := cmd.Flags().GetBool("silent")
	timeoutFlag, _ := cmd.Flags().GetInt("timeout")

	var logBuffer strings.Builder
	logLevel := slog.LevelError
	if debugFlag {
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)

	// load ignore file
	ignoredItems := model.IgnoredItems{}
	if ignoreFile != "" {
		raw, ferr := os.ReadFile(ignoreFile)
		if ferr == nil {
			yaml.Unmarshal(raw, &ignoredItems)
		}
	}

	// load spec
	specBytes, err := os.ReadFile(fileName)
	if err != nil {
		return &FileProcessingResult{
			FileSize: fileSize,
			Error:    err,
		}
	}

	// build ruleset
	defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	customFuncs, _ := LoadCustomFunctions(functionsFlag, true) // always silent for multi-file

	// hard mode
	if hardModeFlag {
		selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
		owaspRules := rulesets.GetAllOWASPRules()
		for k, v := range owaspRules {
			selectedRS.Rules[k] = v
		}
	}

	// handle custom ruleset
	if rulesetFlag != "" {
		var httpClient *http.Client
		httpClientConfig := utils.HTTPClientConfig{
			CertFile: certFile,
			KeyFile:  keyFile,
			CAFile:   caFile,
			Insecure: insecure,
		}
		if utils.ShouldUseCustomHTTPClient(httpClientConfig) {
			httpClient, _ = utils.CreateCustomHTTPClient(httpClientConfig)
		}
		rs, err := BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, httpClient)
		if err == nil {
			selectedRS = rs
			MergeOWASPRulesToRuleSet(selectedRS, hardModeFlag)
		}
	}

	// apply rules
	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:                         selectedRS,
		Spec:                            specBytes,
		SpecFileName:                    fileName,
		CustomFunctions:                 customFuncs,
		Base:                            baseFlag,
		AllowLookup:                     remoteFlag,
		SkipDocumentCheck:               skipCheckFlag,
		SilenceLogs:                     silentFlag,
		Timeout:                         time.Duration(timeoutFlag) * time.Second,
		IgnoreCircularArrayRef:          ignoreArrayCircleRef,
		IgnoreCircularPolymorphicRef:    ignorePolymorphCircleRef,
		BuildDeepGraph:                  len(ignoredItems) > 0,
		ExtractReferencesFromExtensions: extRefsFlag,
		Logger:                          logger,
	})

	if len(result.Errors) > 0 {
		// capture logs
		var logs []string
		if logBuffer.Len() > 0 {
			logs = strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
		}
		return &FileProcessingResult{
			FileSize: fileSize,
			Logs:     logs,
			Error:    result.Errors[0],
		}
	}

	var results []*model.RuleFunctionResult
	var errors, warnings, informs int

	for _, r := range result.Results {
		resultCopy := r // make a copy
		results = append(results, &resultCopy)

		switch r.Rule.Severity {
		case "error":
			errors++
		case "warn":
			warnings++
		case "info":
			informs++
		}
	}

	var logs []string
	if logBuffer.Len() > 0 {
		logs = strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
	}

	return &FileProcessingResult{
		Results:  results,
		Errors:   errors,
		Warnings: warnings,
		Informs:  informs,
		FileSize: fileSize,
		Logs:     logs,
		Error:    nil,
	}
}

// runMultipleFiles processes multiple files for lint-preview
func runMultipleFiles(cmd *cobra.Command, filesToLint []string) error {

	silentFlag, _ := cmd.Flags().GetBool("silent")
	timeFlag, _ := cmd.Flags().GetBool("time")
	failSeverityFlag, _ := cmd.Flags().GetString("fail-severity")
	pipelineOutput, _ := cmd.Flags().GetBool("pipeline-output")
	noStyleFlag, _ := cmd.Flags().GetBool("no-style")
	detailsFlag, _ := cmd.Flags().GetBool("details")

	if !silentFlag && !pipelineOutput {
		fmt.Printf(" vacuuming %s%d%s files...\n\n", cui.ASCIIGreenBold, len(filesToLint), cui.ASCIIReset)
	}

	var totalErrors, totalWarnings, totalInforms int
	var totalSize int64
	start := time.Now()

	fileResults := make([]fileResult, len(filesToLint))
	stopSpinner := make(chan bool)
	currentFile := make(chan string, 1)
	progressChan := make(chan float64, 1)

	if !silentFlag && !pipelineOutput && !noStyleFlag {
		go func() {
			spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			spinnerIndex := 0
			barWidth := 30
			currentFileName := ""
			currentProgress := 0.0
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-stopSpinner:
					// final clear
					fmt.Printf("\r%s\r", strings.Repeat(" ", 150))
					return
				case file := <-currentFile:
					currentFileName = file
				case prog := <-progressChan:
					currentProgress = prog
				case <-ticker.C:
					// animate spinner
					spinner := spinners[spinnerIndex%len(spinners)]
					spinnerIndex++

					// create progress bar
					filledWidth := int(currentProgress * float64(barWidth))
					bar := ""
					for j := 0; j < barWidth; j++ {
						if j < filledWidth {
							bar += cui.ASCIIBlue + "█"
						} else {
							bar += cui.ASCIIGrey + "░"
						}
					}
					bar += cui.ASCIIReset

					// clear line and redraw
					fmt.Printf("\r%s", strings.Repeat(" ", 150))
					if currentFileName != "" {
						processed := int(currentProgress*float64(len(filesToLint))) + 1
						fmt.Printf("\r %s%s%s %s%s[%d/%d]%s %s %s%s%s%s",
							cui.ASCIIPink, spinner, cui.ASCIIReset,
							cui.ASCIIPink, cui.ASCIIBold, processed, len(filesToLint), cui.ASCIIReset,
							bar,
							cui.ASCIIGrey, cui.ASCIIItalic, currentFileName, cui.ASCIIReset)
					}
				}
			}
		}()
	}

	// process all files
	for i, fileName := range filesToLint {
		// update progress display
		if !silentFlag && !pipelineOutput {
			if !noStyleFlag {
				currentFile <- fileName
				progressChan <- float64(i) / float64(len(filesToLint))
			} else {
				// plain text progress for no-style mode
				fmt.Printf("[%d/%d] vacuuming %s...\n", i+1, len(filesToLint), fileName)
			}
		}

		result := ProcessSingleFileWithLogs(cmd, fileName)

		fileResults[i] = fileResult{
			fileName: fileName,
			results:  result.Results,
			errors:   result.Errors,
			warnings: result.Warnings,
			informs:  result.Informs,
			size:     result.FileSize,
			logs:     result.Logs,
			err:      result.Error,
		}

		// accumulate totals
		totalErrors += result.Errors
		totalWarnings += result.Warnings
		totalInforms += result.Informs
		totalSize += result.FileSize
	}

	// stop spinner and clear line properly
	if !silentFlag && !pipelineOutput && !noStyleFlag {
		stopSpinner <- true
		time.Sleep(150 * time.Millisecond) // give spinner time to clear
	}

	// render all results
	if !silentFlag && !pipelineOutput {
		// get terminal width and calculate table width
		termWidth := getTerminalWidth()
		widths := calculateColumnWidths(termWidth)

		// calculate actual table width (matching the summary table)
		// for full width: rule (40) + violation (12) + impact (50) + separators (4 spaces) + leading space (1) = 107
		tableWidth := widths.rule + widths.violation + widths.impact + 4 + 1
		if termWidth < 100 {
			// for smaller terminals, adjust table width accordingly
			tableWidth = termWidth - 13 // leave some margin
		}

		for _, fr := range fileResults {
			// only print header if we're not showing details (details prints its own header)
			if !(detailsFlag && len(fr.results) > 0 && fr.err == nil) {
				if !noStyleFlag {
					fmt.Printf("\n %s%s>%s %s%s%s\n", cui.ASCIIPink, cui.ASCIIBold, cui.ASCIIReset, cui.ASCIIBlue, fr.fileName, cui.ASCIIReset)
					fmt.Printf(" %s%s%s\n\n", cui.ASCIIPink, strings.Repeat("-", tableWidth-1), cui.ASCIIReset)
				} else {
					fmt.Printf("\n > %s\n", fr.fileName)
					fmt.Printf(" %s\n\n", strings.Repeat("-", tableWidth-1))
				}
			}

			if fr.err != nil {
				// for errors, we need to print the header since details won't be shown
				if detailsFlag && len(fr.results) > 0 {
					if !noStyleFlag {
						fmt.Printf("\n %s%s>%s %s%s%s\n", cui.ASCIIBlue, cui.ASCIIBold, cui.ASCIIReset, cui.ASCIIBlue, fr.fileName, cui.ASCIIReset)
						fmt.Printf(" %s%s%s\n\n", cui.ASCIIPink, strings.Repeat("-", tableWidth-1), cui.ASCIIReset)
					} else {
						fmt.Printf("\n > %s\n", fr.fileName)
						fmt.Printf(" %s\n\n", strings.Repeat("-", tableWidth-1))
					}
				}
				if !noStyleFlag {
					fmt.Printf("%sError: %v%s\n", cui.ASCIIRed, fr.err, cui.ASCIIReset)
				} else {
					fmt.Printf("Error: %v\n", fr.err)
				}
			} else {
				// show details if requested
				if detailsFlag && len(fr.results) > 0 {
					// get spec data for snippets
					specBytes, _ := os.ReadFile(fr.fileName)
					specStringData := strings.Split(string(specBytes), "\n")
					renderFixedDetails(fr.results, specStringData, false, false, silentFlag,
						false, false, false, fr.fileName, noStyleFlag)
				}

				// create result set and render summary
				resultSet := model.NewRuleResultSetPointer(fr.results)
				renderFixedSummary(resultSet, model.RuleCategoriesOrdered, nil, fr.fileName, silentFlag,
					noStyleFlag, pipelineOutput, false)
			}

			// show logs if any
			if len(fr.logs) > 0 {
				if !noStyleFlag {
					fmt.Printf("\n%sLogs:%s\n", cui.ASCIIGrey, cui.ASCIIReset)
				} else {
					fmt.Println("\nLogs:")
				}
				for _, log := range fr.logs {
					if !noStyleFlag {
						fmt.Printf("%s  %s%s\n", cui.ASCIIGrey, log, cui.ASCIIReset)
					} else {
						fmt.Printf("  %s\n", log)
					}
				}
			}
		}
	}

	// show overall summary
	if !silentFlag && !pipelineOutput {
		fmt.Printf("\n%s=== Overall Summary for %d files ===%s\n", cui.ASCIIPink, len(filesToLint), cui.ASCIIReset)
		fmt.Printf("Total issues: %s%d errors%s, %s%d warnings%s, %s%d info%s\n",
			cui.ASCIIRed, totalErrors, cui.ASCIIReset,
			cui.ASCIIYellow, totalWarnings, cui.ASCIIReset,
			cui.ASCIIBlue, totalInforms, cui.ASCIIReset)
	}

	// show timing
	if timeFlag && !pipelineOutput && !silentFlag {
		duration := time.Since(start)
		RenderTimeAndFiles(timeFlag, duration, totalSize, len(filesToLint))
	}

	return CheckFailureSeverity(failSeverityFlag, totalErrors, totalWarnings, totalInforms)
}
