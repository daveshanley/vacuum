// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/spf13/cobra"
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

// runMultipleFiles processes multiple files for lint-preview
func runMultipleFiles(cmd *cobra.Command, filesToLint []string) error {

	// Read all flags at once
	flags := ReadLintFlags(cmd)

	// Environment already setup by parent command

	// Create logger once for all files (we'll create BufferedLogger per file)
	logger, _ := createDebugLogger(flags.DebugFlag)

	// Load and configure ruleset once for all files
	selectedRS, err := LoadRulesetWithConfig(flags, logger)
	if err != nil {
		return err
	}

	// Load custom functions once
	customFuncs, _ := LoadCustomFunctions(flags.FunctionsFlag, flags.SilentFlag)

	// Load ignore file once
	ignoredItems, _ := LoadIgnoreFile(flags.IgnoreFile, flags.SilentFlag, flags.PipelineOutput, flags.NoStyleFlag)

	if !flags.SilentFlag && !flags.PipelineOutput {
		if !flags.NoStyleFlag {
			fmt.Printf(" vacuuming %s%d%s files...\n\n", cui.ASCIIGreenBold, len(filesToLint), cui.ASCIIReset)
		} else {
			fmt.Printf(" vacuuming %d files...\n\n", len(filesToLint))
		}
	}

	var totalErrors, totalWarnings, totalInforms int
	var totalSize int64
	start := time.Now()

	fileResults := make([]fileResult, len(filesToLint))
	stopSpinner := make(chan bool)
	currentFile := make(chan string, 1)
	progressChan := make(chan float64, 1)

	if !flags.SilentFlag && !flags.PipelineOutput && !flags.NoStyleFlag {
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
		if !flags.SilentFlag && !flags.PipelineOutput {
			if !flags.NoStyleFlag {
				currentFile <- fileName
				progressChan <- float64(i) / float64(len(filesToLint))
			} else {
				// plain text progress for no-style mode
				fmt.Printf("[%d/%d] vacuuming %s...\n", i+1, len(filesToLint), fileName)
			}
		}

		// Create a new BufferedLogger for this file with appropriate log level
		var bufferedLogger *BufferedLogger
		if flags.DebugFlag {
			bufferedLogger = NewBufferedLoggerWithLevel(cui.LogLevelDebug)
		} else {
			bufferedLogger = NewBufferedLoggerWithLevel(cui.LogLevelError)
		}

		// Create processing config with the BufferedLogger for this file
		processingConfig := &FileProcessingConfig{
			Flags:           flags,
			BufferedLogger:  bufferedLogger,
			SelectedRuleset: selectedRS,
			CustomFunctions: customFuncs,
			IgnoredItems:    ignoredItems,
		}

		result := ProcessSingleFileOptimized(fileName, processingConfig)

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
	if !flags.SilentFlag && !flags.PipelineOutput && !flags.NoStyleFlag {
		stopSpinner <- true
		time.Sleep(150 * time.Millisecond) // give spinner time to clear
	}

	// render all results
	if !flags.SilentFlag && !flags.PipelineOutput {
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
			if !(flags.DetailsFlag && len(fr.results) > 0 && fr.err == nil) {
				if !flags.NoStyleFlag {
					fmt.Printf("\n %s%s>%s %s%s%s\n", cui.ASCIIPink, cui.ASCIIBold, cui.ASCIIReset, cui.ASCIIBlue, fr.fileName, cui.ASCIIReset)
					fmt.Printf(" %s%s%s\n\n", cui.ASCIIPink, strings.Repeat("-", tableWidth-1), cui.ASCIIReset)
				} else {
					fmt.Printf("\n > %s\n", fr.fileName)
					fmt.Printf(" %s\n\n", strings.Repeat("-", tableWidth-1))
				}
			}

			if fr.err != nil {
				// for errors, we need to print the header since details won't be shown
				if flags.DetailsFlag && len(fr.results) > 0 {
					if !flags.NoStyleFlag {
						fmt.Printf("\n %s%s>%s %s%s%s\n", cui.ASCIIBlue, cui.ASCIIBold, cui.ASCIIReset, cui.ASCIIBlue, fr.fileName, cui.ASCIIReset)
						fmt.Printf(" %s%s%s\n\n", cui.ASCIIPink, strings.Repeat("-", tableWidth-1), cui.ASCIIReset)
					} else {
						fmt.Printf("\n > %s\n", fr.fileName)
						fmt.Printf(" %s\n\n", strings.Repeat("-", tableWidth-1))
					}
				}
				if !flags.NoStyleFlag {
					fmt.Printf("%sError: %v%s\n", cui.ASCIIRed, fr.err, cui.ASCIIReset)
				} else {
					fmt.Printf("Error: %v\n", fr.err)
				}
			} else {
				// show details if requested
				if flags.DetailsFlag && len(fr.results) > 0 {
					// get spec data for snippets
					specBytes, _ := os.ReadFile(fr.fileName)
					specStringData := strings.Split(string(specBytes), "\n")
					renderFixedDetails(fr.results, specStringData, false, false, flags.SilentFlag,
						false, false, false, fr.fileName, flags.NoStyleFlag)
				}

				// create result set and render summary
				resultSet := model.NewRuleResultSetPointer(fr.results)
				renderFixedSummary(resultSet, model.RuleCategoriesOrdered, nil, fr.fileName, flags.SilentFlag,
					flags.NoStyleFlag, flags.PipelineOutput, false)
			}

			// show logs if any with nice tree formatting
			if len(fr.logs) > 0 && len(fr.logs[0]) > 0 {
				if !flags.NoStyleFlag {
					fmt.Printf("%s※※ vacuumed logs for %s'%s%s%s%s' %s※※%s\n", cui.ASCIIGrey, cui.ASCIIReset,
						cui.ASCIIItalic, cui.ASCIIGreenBold, fr.fileName, cui.ASCIIReset, cui.ASCIIGrey, cui.ASCIIReset)
				} else {
					fmt.Println("vacuumed logs:")
				}

				// Print the already-formatted logs from BufferedLogger
				// The log is the complete rendered tree with proper spacing
				fmt.Print(fr.logs[0])
				fmt.Println() // Add spacing after logs
			}
		}
	}

	// show timing
	if flags.TimeFlag && !flags.PipelineOutput && !flags.SilentFlag {
		duration := time.Since(start)
		RenderTimeAndFiles(flags.TimeFlag, duration, totalSize, len(filesToLint))
	}

	return CheckFailureSeverity(flags.FailSeverityFlag, totalErrors, totalWarnings, totalInforms)
}
