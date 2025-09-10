// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// wrapLogText wraps log text to fit within terminal width, preserving indentation for wrapped lines
func wrapLogText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var currentLine strings.Builder
	currentLine.WriteString(words[0])

	for i := 1; i < len(words); i++ {
		word := words[i]
		// check if adding this word would exceed the width
		if cui.VisibleLength(currentLine.String()+" "+word) > maxWidth {
			// save current line and start a new one
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		} else {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		}
	}

	// add the last line
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

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

	// Read all flags at once
	flags := ReadLintFlags(cmd)

	// setup charm logger with custom styles (capture to buffer for logs)
	var logBuffer bytes.Buffer
	charmLogger := log.New(&logBuffer)

	// set log level
	if flags.DebugFlag {
		charmLogger.SetLevel(log.DebugLevel)
	} else {
		charmLogger.SetLevel(log.ErrorLevel)
	}

	// customize the charm log styles to match our theme
	styles := log.DefaultStyles()
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff3366")).
		Bold(true)
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffcc00")).
		Bold(true)
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#62c4ff")).
		Bold(true)
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f83aff"))
	styles.Key = lipgloss.NewStyle().Foreground(lipgloss.Color("#62c4ff"))
	styles.Value = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	styles.Separator = lipgloss.NewStyle().Foreground(lipgloss.Color("#f83aff"))
	charmLogger.SetReportCaller(false)
	charmLogger.SetReportTimestamp(false)

	// create slog logger with charm handler for compatibility
	logger := slog.New(charmLogger)

	// load ignore file (silently for multi-file processing)
	ignoredItems := model.IgnoredItems{}
	if flags.IgnoreFile != "" {
		raw, ferr := os.ReadFile(flags.IgnoreFile)
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

	// Load custom functions
	customFuncs, _ := LoadCustomFunctions(flags.FunctionsFlag, true) // always silent for multi-file

	// Load and configure ruleset (but silently for multi-file processing)
	silentFlags := *flags // copy flags
	silentFlags.SilentFlag = true
	silentFlags.PipelineOutput = false
	selectedRS, err := LoadRulesetWithConfig(&silentFlags, logger)
	if err != nil {
		// If ruleset loading fails, use default
		defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
		selectedRS = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	}

	// apply rules
	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:                         selectedRS,
		Spec:                            specBytes,
		SpecFileName:                    fileName,
		CustomFunctions:                 customFuncs,
		Base:                            flags.BaseFlag,
		AllowLookup:                     flags.RemoteFlag,
		SkipDocumentCheck:               flags.SkipCheckFlag,
		SilenceLogs:                     flags.SilentFlag,
		Timeout:                         time.Duration(flags.TimeoutFlag) * time.Second,
		IgnoreCircularArrayRef:          flags.IgnoreArrayCircleRef,
		IgnoreCircularPolymorphicRef:    flags.IgnorePolymorphCircleRef,
		BuildDeepGraph:                  len(ignoredItems) > 0,
		ExtractReferencesFromExtensions: flags.ExtRefsFlag,
		Logger:                          logger,
		HTTPClientConfig:                GetHTTPClientConfig(flags),
	})

	if len(result.Errors) > 0 {
		// capture logs - charm logger outputs formatted lines
		var logs []string
		if logBuffer.Len() > 0 {
			// split by newline and filter empty lines
			lines := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					logs = append(logs, line)
				}
			}
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

	// capture logs - charm logger outputs formatted lines
	var logs []string
	if logBuffer.Len() > 0 {
		// split by newline and filter empty lines
		lines := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				logs = append(logs, line)
			}
		}
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

	// Read all flags at once
	flags := ReadLintFlags(cmd)
	
	// Setup environment (terminal detection, colors)
	SetupLintEnvironment(flags)

	// Create logger once for all files
	logger := createDebugLogger(flags.DebugFlag)
	
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
		fmt.Printf(" vacuuming %s%d%s files...\n\n", cui.ASCIIGreenBold, len(filesToLint), cui.ASCIIReset)
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

	// Create processing config to reuse for all files
	processingConfig := &FileProcessingConfig{
		Flags:           flags,
		Logger:          logger,
		SelectedRuleset: selectedRS,
		CustomFunctions: customFuncs,
		IgnoredItems:    ignoredItems,
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
			if len(fr.logs) > 0 {
				if !flags.NoStyleFlag {
					fmt.Printf("\n %svacuumed logs for %s'%s%s%s%s':\n", cui.ASCIIGrey, cui.ASCIIReset,
						cui.ASCIIItalic, cui.ASCIIGreenBold, fr.fileName, cui.ASCIIReset)
				} else {
					fmt.Println("\n vacuumed logs:")
				}

				// get terminal width for wrapping
				termWidth := getTerminalWidth()
				// calculate available width for log text (terminal width - prefix width)
				// prefix is " ├─ " or " └─ " which is 4 visible chars
				availableWidth := termWidth - 4
				if availableWidth < 40 {
					availableWidth = 40 // minimum width
				}

				for i, log := range fr.logs {
					isLast := i == len(fr.logs)-1
					if !flags.NoStyleFlag {
						// colorize quoted text in the log
						colorizedLog := cui.ColorizeLogEntry(strings.TrimSpace(log), cui.ASCIIGrey)

						// wrap the log text
						wrappedLines := wrapLogText(colorizedLog, availableWidth)

						for j, line := range wrappedLines {
							if j == 0 {
								// first line gets the tree character
								if isLast {
									fmt.Printf(" %s└─ %s%s%s%s\n", cui.ASCIIPink, cui.ASCIIReset, cui.ASCIIGrey, line, cui.ASCIIReset)
								} else {
									fmt.Printf(" %s├─ %s%s%s%s\n", cui.ASCIIPink, cui.ASCIIReset, cui.ASCIIGrey, line, cui.ASCIIReset)
								}
							} else {
								// continuation lines: align with the text after "├─ " or "└─ "
								// that's 1 space + 3 chars (│ and 2 spaces) = "│   " for non-last
								// or just 4 spaces for last item
								if isLast {
									fmt.Printf("    %s%s%s\n", cui.ASCIIGrey, line, cui.ASCIIReset)
								} else {
									fmt.Printf(" %s│  %s %s%s%s\n", cui.ASCIIPink, cui.ASCIIReset, cui.ASCIIGrey, line, cui.ASCIIReset)
								}
							}
						}
					} else {
						// no style mode
						wrappedLines := wrapLogText(strings.TrimSpace(log), availableWidth)

						for j, line := range wrappedLines {
							if j == 0 {
								if isLast {
									fmt.Printf(" └─ %s\n", line)
								} else {
									fmt.Printf(" ├─ %s\n", line)
								}
							} else {
								if isLast {
									fmt.Printf("    %s\n", line)
								} else {
									fmt.Printf(" │   %s\n", line)
								}
							}
						}
					}
				}
			}
		}
	}

	// show overall summary
	if !flags.SilentFlag && !flags.PipelineOutput {
		fmt.Printf("\n%s=== Overall Summary for %d files ===%s\n", cui.ASCIIPink, len(filesToLint), cui.ASCIIReset)
		fmt.Printf("Total issues: %s%d errors%s, %s%d warnings%s, %s%d info%s\n",
			cui.ASCIIRed, totalErrors, cui.ASCIIReset,
			cui.ASCIIYellow, totalWarnings, cui.ASCIIReset,
			cui.ASCIIBlue, totalInforms, cui.ASCIIReset)
	}

	// show timing
	if flags.TimeFlag && !flags.PipelineOutput && !flags.SilentFlag {
		duration := time.Since(start)
		RenderTimeAndFiles(flags.TimeFlag, duration, totalSize, len(filesToLint))
	}

	return CheckFailureSeverity(flags.FailSeverityFlag, totalErrors, totalWarnings, totalInforms)
}
