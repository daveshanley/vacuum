// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/libopenapi/index"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

func GetLintPreviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "lint-preview <your-openapi-file.yaml>",
		Short:         "Preview lint results with enhanced table formatting",
		Long:          `Lint an OpenAPI specification and display results in a formatted table view`,
		RunE:          runLintPreview,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add all flags
	cmd.Flags().BoolP("details", "d", false, "Show full details of linting report")
	cmd.Flags().BoolP("snippets", "s", false, "Show code snippets where issues are found")
	cmd.Flags().BoolP("errors", "e", false, "Show errors only")
	cmd.Flags().StringP("category", "c", "", "Show a single category of results")
	cmd.Flags().BoolP("silent", "x", false, "Show nothing except the result")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output")
	cmd.Flags().BoolP("no-banner", "b", false, "Disable the banner output")
	cmd.Flags().BoolP("no-message", "m", false, "Hide message output when using -d")
	cmd.Flags().BoolP("all-results", "a", false, "Render all results when using -d")
	cmd.Flags().StringP("fail-severity", "n", model.SeverityError, "Results of this level or above will trigger a failure exit code")
	cmd.Flags().StringP("base", "p", "", "Base URL or path for resolving references")
	cmd.Flags().BoolP("remote", "u", true, "Allow remote references")
	cmd.Flags().BoolP("skip-check", "k", false, "Skip OpenAPI document validation")
	cmd.Flags().IntP("timeout", "g", 5, "Timeout in seconds for each rule")
	cmd.Flags().StringP("ruleset", "r", "", "Path to custom ruleset")
	cmd.Flags().StringP("functions", "f", "", "Path to custom functions")
	cmd.Flags().BoolP("time", "t", false, "Show execution time")
	cmd.Flags().BoolP("hard-mode", "z", false, "Enable hard mode (all rules)")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().Bool("no-clip", false, "Do not truncate messages or paths")
	cmd.Flags().Bool("ext-refs", false, "Enable $ref lookups for extension objects")
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")

	return cmd
}

func runLintPreview(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide an OpenAPI specification file to lint")
	}

	fileName := args[0]

	// Read all flags
	detailsFlag, _ := cmd.Flags().GetBool("details")
	snippetsFlag, _ := cmd.Flags().GetBool("snippets")
	errorsFlag, _ := cmd.Flags().GetBool("errors")
	categoryFlag, _ := cmd.Flags().GetString("category")
	silentFlag, _ := cmd.Flags().GetBool("silent")
	noStyleFlag, _ := cmd.Flags().GetBool("no-style")
	noBannerFlag, _ := cmd.Flags().GetBool("no-banner")
	noMessageFlag, _ := cmd.Flags().GetBool("no-message")
	allResultsFlag, _ := cmd.Flags().GetBool("all-results")
	failSeverityFlag, _ := cmd.Flags().GetString("fail-severity")
	baseFlag, _ := cmd.Flags().GetString("base")
	remoteFlag, _ := cmd.Flags().GetBool("remote")
	skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
	timeoutFlag, _ := cmd.Flags().GetInt("timeout")
	rulesetFlag, _ := cmd.Flags().GetString("ruleset")
	functionsFlag, _ := cmd.Flags().GetString("functions")
	timeFlag, _ := cmd.Flags().GetBool("time")
	hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
	ignoreFile, _ := cmd.Flags().GetString("ignore-file")
	noClipFlag, _ := cmd.Flags().GetBool("no-clip")
	extRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
	ignoreArrayCircleRef, _ := cmd.Flags().GetBool("ignore-array-circle-ref")
	ignorePolymorphCircleRef, _ := cmd.Flags().GetBool("ignore-polymorph-circle-ref")

	// Show banner unless disabled
	if !silentFlag && !noBannerFlag {
		PrintBanner()
	}

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
	reportOrSpec, err := LoadFileAsReportOrSpec(fileName)
	if err != nil {
		fmt.Printf("\033[31mUnable to load file '%s': %v\033[0m\n", fileName, err)
		return err
	}

	// Get file info for timing
	fileInfo, _ := os.Stat(fileName)

	// Setup logging
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	})
	logger := slog.New(handler)

	var resultSet *model.RuleResultSet
	var specBytes []byte
	var displayFileName string
	start := time.Now()

	if reportOrSpec.IsReport {
		// Using a pre-compiled report
		if !silentFlag {
			fmt.Printf("\033[36mLoading pre-compiled vacuum report from '%s'\033[0m\n\n", fileName)
		}

		// Create a new RuleResultSet from the results to ensure proper initialization
		if reportOrSpec.ResultSet != nil && reportOrSpec.ResultSet.Results != nil {
			// Filter ignored results first
			filteredResults := utils.FilterIgnoredResultsPtr(reportOrSpec.ResultSet.Results, ignoredItems)
			// Create properly initialized RuleResultSet
			resultSet = model.NewRuleResultSetPointer(filteredResults)
		} else {
			resultSet = model.NewRuleResultSetPointer([]*model.RuleFunctionResult{})
		}

		specBytes = reportOrSpec.SpecBytes
		displayFileName = reportOrSpec.FileName
	} else {
		// Regular spec file - run linting
		specBytes = reportOrSpec.SpecBytes
		displayFileName = fileName

		// Build ruleset
		defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
		selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
		customFuncs, _ := LoadCustomFunctions(functionsFlag, silentFlag)

		// Handle hard mode
		if hardModeFlag {
			selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
			owaspRules := rulesets.GetAllOWASPRules()
			for k, v := range owaspRules {
				selectedRS.Rules[k] = v
			}
			if !silentFlag {
				fmt.Printf("\033[31m🚨 HARD MODE ENABLED 🚨\033[0m\n\n")
			}
		}

		// Handle custom ruleset
		if rulesetFlag != "" {
			var rsErr error
			selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag, nil)
			if rsErr != nil {
				fmt.Printf("\033[31mUnable to load ruleset '%s': %s\033[0m\n", rulesetFlag, rsErr.Error())
				return rsErr
			}
			if hardModeFlag {
				MergeOWASPRulesToRuleSet(selectedRS, true)
			}
		}

		// Display linting info
		if !silentFlag {
			fmt.Printf("\033[36mLinting file '%s' against %d rules: %s\033[0m\n\n",
				displayFileName, len(selectedRS.Rules), selectedRS.DocumentationURI)
		}

		// Build deep graph if we have ignored items
		deepGraph := false
		if len(ignoredItems) > 0 {
			deepGraph = true
		}

		// Apply rules
		result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:                         selectedRS,
			Spec:                            specBytes,
			SpecFileName:                    displayFileName,
			CustomFunctions:                 customFuncs,
			Base:                            baseFlag,
			AllowLookup:                     remoteFlag,
			SkipDocumentCheck:               skipCheckFlag,
			Logger:                          logger,
			BuildDeepGraph:                  deepGraph,
			Timeout:                         time.Duration(timeoutFlag) * time.Second,
			IgnoreCircularArrayRef:          ignoreArrayCircleRef,
			IgnoreCircularPolymorphicRef:    ignorePolymorphCircleRef,
			ExtractReferencesFromExtensions: extRefsFlag,
		})

		// Filter ignored results
		result.Results = utils.FilterIgnoredResults(result.Results, ignoredItems)

		// Check for errors
		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				fmt.Printf("\033[31mUnable to process spec '%s': %s\033[0m\n", displayFileName, err.Error())
			}
			return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
		}

		// Process results
		resultSet = model.NewRuleResultSet(result.Results)
	}

	specStringData := strings.Split(string(specBytes), "\n")

	// Handle category filtering
	var cats []*model.RuleCategory
	if categoryFlag != "" {
		// Category filtering logic here (same as original)
		cats = model.RuleCategoriesOrdered
	} else {
		cats = model.RuleCategoriesOrdered
	}

	resultSet.SortResultsByLineNumber()

	// Create statistics if we have the necessary data
	var stats *reports.ReportStatistics
	if reportOrSpec.IsReport && reportOrSpec.Report.Statistics != nil {
		stats = reportOrSpec.Report.Statistics
	}
	// Note: For fresh linting, we'd need the index and spec info from the result,
	// but that's not available in this flow anymore. We can skip stats for now.

	// Show detailed results if requested
	if detailsFlag && len(resultSet.Results) > 0 {
		// Always use regular detailed view (no interactive UI)
		renderFixedDetails(resultSet.Results, specStringData, snippetsFlag, errorsFlag,
			silentFlag, noMessageFlag, allResultsFlag, noClipFlag, displayFileName, noStyleFlag)
	}

	// Render summary
	renderFixedSummary(resultSet, cats, stats, displayFileName, silentFlag, noStyleFlag)

	// Show timing
	duration := time.Since(start)
	if timeFlag {
		renderFixedTiming(duration, fileInfo.Size())
	}

	// Check severity failure
	errs := resultSet.GetErrorCount()
	warnings := resultSet.GetWarnCount()
	informs := resultSet.GetInfoCount()

	// Check for failure but handle it gracefully without showing help
	failErr := CheckFailureSeverity(failSeverityFlag, errs, warnings, informs)
	if failErr != nil {
		os.Exit(1)
	}

	return nil
}

func printFixedBanner() {
	banner := `
██╗   ██╗ █████╗  ██████╗██╗   ██╗██╗   ██╗███╗   ███╗
██║   ██║██╔══██╗██╔════╝██║   ██║██║   ██║████╗ ████║
██║   ██║███████║██║     ██║   ██║██║   ██║██╔████╔██║
╚██╗ ██╔╝██╔══██║██║     ██║   ██║██║   ██║██║╚██╔╝██║
 ╚████╔╝ ██║  ██║╚██████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝`

	// Use color constants
	fmt.Printf("%s%s%s\n\n", cui.ASCIIPink, banner, cui.ASCIIReset)
	fmt.Printf("%sversion: %s | compiled: %s%s\n", cui.ASCIIGreen, Version, Date, cui.ASCIIReset)
	fmt.Printf("%s🔗 https://quobix.com/vacuum | https://github.com/daveshanley/vacuum%s\n\n", cui.ASCIIBlue, cui.ASCIIReset)
}

func renderFixedDetails(results []*model.RuleFunctionResult, specData []string,
	snippets, errors, silent, noMessage, allResults, noClip bool,
	fileName string, noStyle bool) {

	if !silent {
		// Display file header
		abs, _ := filepath.Abs(fileName)
		displayPath := abs
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, abs); err == nil {
				displayPath = relPath
			}
		}

		fmt.Printf("\n%s%s%s\n", cui.ASCIIPink, displayPath, cui.ASCIIReset)
		fmt.Println(strings.Repeat("-", len(displayPath)))
		fmt.Println()
	}

	// Get terminal width
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 120 // Default fallback
	}

	// First pass: calculate the actual maximum widths needed for each column
	maxLocationLen := len("Location") // Start with header width
	maxRuleLen := len("Rule")
	maxCategoryLen := len("Category")
	maxMessageLen := len("Message") // Track actual max message length

	for _, r := range results {
		// Build location for this result
		startLine := 0
		startCol := 0
		if r.StartNode != nil {
			startLine = r.StartNode.Line
			startCol = r.StartNode.Column
		}

		f := fileName
		if r.Origin != nil {
			f = r.Origin.AbsoluteLocation
			startLine = r.Origin.Line
			startCol = r.Origin.Column
		}

		// Make path relative
		if absPath, err := filepath.Abs(f); err == nil {
			if cwd, err := os.Getwd(); err == nil {
				if relPath, err := filepath.Rel(cwd, absPath); err == nil {
					f = relPath
				}
			}
		}

		location := fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
		if len(location) > maxLocationLen {
			maxLocationLen = len(location)
		}

		// Check rule length
		if r.Rule != nil && len(r.Rule.Id) > maxRuleLen {
			maxRuleLen = len(r.Rule.Id)
		}

		// Check category length
		if r.Rule != nil && r.Rule.RuleCategory != nil && len(r.Rule.RuleCategory.Name) > maxCategoryLen {
			maxCategoryLen = len(r.Rule.RuleCategory.Name)
		}

		// Check message length (skip if showing errors only and this isn't an error)
		if !errors || (r.Rule != nil && r.Rule.Severity == model.SeverityError) {
			if len(r.Message) > maxMessageLen {
				maxMessageLen = len(r.Message)
			}
		}
	}

	// Column width allocation based on actual content
	// Priority order: location (never truncated), rule (never truncated), category (conditionally shown), message, path

	// Fixed/dynamic widths based on content
	locWidth := maxLocationLen
	sevWidth := 9 // Fixed width for severity with icon
	ruleWidth := maxRuleLen
	catWidth := maxCategoryLen
	
	// Responsive column visibility based on terminal width
	showCategory := true
	showPath := true
	showRule := true
	
	if width >= 100 && width < 120 {
		// Very narrow terminals: hide category, path, and rule
		showCategory = false
		showPath = false
		showRule = false
		catWidth = 0
		ruleWidth = 0
		sevWidth = 2  // Just the symbol, no text
	} else if width >= 120 && width < 130 {
		// Narrow terminals: hide both category and path
		showCategory = false
		showPath = false
		catWidth = 0
	} else if width >= 130 && width < 160 {
		// Medium terminals: hide category only
		showCategory = false
		catWidth = 0
	}
	// Wide terminals (160+): show everything

	// Calculate remaining width after fixed columns
	separators := 10 // Space for column separators
	if !showRule && !showCategory && !showPath {
		separators = 4  // Only location, severity, message
	} else if !showCategory && !showPath {
		separators = 6  // Two less separators without category and path
	} else if !showCategory {
		separators = 8  // One less separator without category column
	}
	fixedWidth := locWidth + sevWidth + ruleWidth + catWidth + separators
	remainingWidth := width - fixedWidth

	// Allocate remaining space between message and path
	// Message should only be as wide as needed (plus small buffer), give rest to path
	var msgWidth, pathWidth int
	if remainingWidth > 0 {
		if showPath {
			// Use actual max message length plus a small buffer for readability
			msgWidth = maxMessageLen // Just 3 chars padding for visual comfort

			// Cap at remaining space minus minimum path width
			if msgWidth > remainingWidth-20 { // Leave at least 20 for path
				msgWidth = remainingWidth - 20
			}

			// Give ALL remaining space to path
			pathWidth = remainingWidth - msgWidth

			// Ensure minimum widths
			if msgWidth < 20 {
				msgWidth = 20
				pathWidth = remainingWidth - msgWidth
			}
			if pathWidth < 10 {
				pathWidth = 10
			}
		} else {
			// No path column - give all remaining space to message
			msgWidth = remainingWidth
			pathWidth = 0
			
			// Ensure minimum message width
			if msgWidth < 20 {
				msgWidth = 20
			}
		}
	} else {
		// If no remaining width, use minimums
		msgWidth = 20
		if showPath {
			pathWidth = 10
		} else {
			pathWidth = 0
		}
	}

	// Build and render table
	if !snippets {
		// Print header with pink color and bold (matching BubbleTea UI)
		// Apply color codes outside of the formatted strings to avoid width calculation issues
		if !noMessage {
			if !showRule && !showCategory && !showPath {
				// Very narrow terminals: only location, severity symbol, message
				fmt.Printf("%s%s%-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, "Location",
					sevWidth, "",  // No header for severity symbol
					msgWidth, "Message",
					cui.ASCIIReset)
			} else if showCategory && showPath {
				// All columns
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, "Location",
					sevWidth, "Severity",
					msgWidth, "Message",
					ruleWidth, "Rule",
					catWidth, "Category",
					pathWidth, "Path",
					cui.ASCIIReset)
			} else if !showCategory && showPath {
				// No category column for medium terminals
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, "Location",
					sevWidth, "Severity",
					msgWidth, "Message",
					ruleWidth, "Rule",
					pathWidth, "Path",
					cui.ASCIIReset)
			} else if !showCategory && !showPath {
				// No category or path for narrow terminals
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, "Location",
					sevWidth, "Severity",
					msgWidth, "Message",
					ruleWidth, "Rule",
					cui.ASCIIReset)
			}
		} else {
			// Adjust widths when no message column
			if showPath {
				pathWidth = msgWidth + pathWidth + 2
			}
			if showCategory && showPath {
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, "Location",
					sevWidth, "Severity",
					ruleWidth, "Rule",
					catWidth, "Category",
					pathWidth, "Path",
					cui.ASCIIReset)
			} else if !showCategory && showPath {
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, "Location",
					sevWidth, "Severity",
					ruleWidth, "Rule",
					pathWidth, "Path",
					cui.ASCIIReset)
			} else if !showCategory && !showPath {
				fmt.Printf("%s%s%-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, "Location",
					sevWidth, "Severity",
					ruleWidth, "Rule",
					cui.ASCIIReset)
			}
		}

		// Print separator with pink color and bold (same as header)
		// Use the same format specifiers as header to ensure alignment
		if !noMessage {
			if !showRule && !showCategory && !showPath {
				// Very narrow terminals: only location, severity symbol, message
				fmt.Printf("%s%s%-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, strings.Repeat("─", locWidth),
					sevWidth, strings.Repeat("─", sevWidth),
					msgWidth, strings.Repeat("─", msgWidth),
					cui.ASCIIReset)
			} else if showCategory && showPath {
				// All columns
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, strings.Repeat("─", locWidth),
					sevWidth, strings.Repeat("─", sevWidth),
					msgWidth, strings.Repeat("─", msgWidth),
					ruleWidth, strings.Repeat("─", ruleWidth),
					catWidth, strings.Repeat("─", catWidth),
					pathWidth, strings.Repeat("─", pathWidth),
					cui.ASCIIReset)
			} else if !showCategory && showPath {
				// No category separator for medium terminals
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, strings.Repeat("─", locWidth),
					sevWidth, strings.Repeat("─", sevWidth),
					msgWidth, strings.Repeat("─", msgWidth),
					ruleWidth, strings.Repeat("─", ruleWidth),
					pathWidth, strings.Repeat("─", pathWidth),
					cui.ASCIIReset)
			} else if !showCategory && !showPath {
				// No category or path for narrow terminals
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, strings.Repeat("─", locWidth),
					sevWidth, strings.Repeat("─", sevWidth),
					msgWidth, strings.Repeat("─", msgWidth),
					ruleWidth, strings.Repeat("─", ruleWidth),
					cui.ASCIIReset)
			}
		} else {
			if showCategory && showPath {
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, strings.Repeat("─", locWidth),
					sevWidth, strings.Repeat("─", sevWidth),
					ruleWidth, strings.Repeat("─", ruleWidth),
					catWidth, strings.Repeat("─", catWidth),
					pathWidth, strings.Repeat("─", pathWidth),
					cui.ASCIIReset)
			} else if !showCategory && showPath {
				fmt.Printf("%s%s%-*s  %-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, strings.Repeat("─", locWidth),
					sevWidth, strings.Repeat("─", sevWidth),
					ruleWidth, strings.Repeat("─", ruleWidth),
					pathWidth, strings.Repeat("─", pathWidth),
					cui.ASCIIReset)
			} else if !showCategory && !showPath {
				fmt.Printf("%s%s%-*s  %-*s  %-*s%s\n",
					cui.ASCIIPink, cui.ASCIIBold,
					locWidth, strings.Repeat("─", locWidth),
					sevWidth, strings.Repeat("─", sevWidth),
					ruleWidth, strings.Repeat("─", ruleWidth),
					cui.ASCIIReset)
			}
		}

		// Print rows
		for i, r := range results {
			if i > 1000 && !allResults {
				fmt.Printf("%s...%d more violations not rendered%s\n", cui.ASCIIRed, len(results)-1000, cui.ASCIIReset)
				break
			}

			// Skip if showing errors only
			if errors && r.Rule != nil && r.Rule.Severity != model.SeverityError {
				continue
			}

			// Build location
			startLine := 0
			startCol := 0
			if r.StartNode != nil {
				startLine = r.StartNode.Line
				startCol = r.StartNode.Column
			}

			f := fileName
			if r.Origin != nil {
				f = r.Origin.AbsoluteLocation
				startLine = r.Origin.Line
				startCol = r.Origin.Column
			}

			// Make path relative
			if absPath, err := filepath.Abs(f); err == nil {
				if cwd, err := os.Getwd(); err == nil {
					if relPath, err := filepath.Rel(cwd, absPath); err == nil {
						f = relPath
					}
				}
			}

			// Format location as file:line:col (never truncated)
			location := fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
			// Apply color formatting to location
			coloredLocation := cui.ColorizeLocation(location)

			// Truncate fields if needed
			m := r.Message
			p := r.Path
			if !noClip {
				if len(m) > msgWidth && msgWidth > 3 {
					m = m[:msgWidth-3] + "..."
				}
				if len(p) > pathWidth && pathWidth > 3 {
					p = p[:pathWidth-3] + "..."
				}
			}

			// Apply message colorization (highlights backtick-enclosed text)
			coloredMessage := cui.ColorizeMessage(m)

			// Apply path colorization (highlights single-quoted text)
			var coloredPath string
			if showPath {
				coloredPath = cui.ColorizePath(truncate(p, pathWidth))
			}

			// Format severity with color and icon (matching BubbleTea UI)
			var sevColored string
			if !showRule {
				// Very narrow mode - just show the colored symbol
				if r.Rule != nil {
					switch r.Rule.Severity {
					case model.SeverityError:
						sevColored = fmt.Sprintf("%s%-*s%s", cui.ASCIIRed, sevWidth, "✗", cui.ASCIIReset)
					case model.SeverityWarn:
						sevColored = fmt.Sprintf("%s%-*s%s", cui.ASCIIYellow, sevWidth, "▲", cui.ASCIIReset)
					case model.SeverityInfo:
						sevColored = fmt.Sprintf("%s%-*s%s", cui.ASCIIBlue, sevWidth, "●", cui.ASCIIReset)
					default:
						sevColored = fmt.Sprintf("%s%-*s%s", cui.ASCIIBlue, sevWidth, "●", cui.ASCIIReset)
					}
				} else {
					sevColored = fmt.Sprintf("%s%-*s%s", cui.ASCIIBlue, sevWidth, "●", cui.ASCIIReset)
				}
			} else {
				// Normal mode - show symbol and text
				if r.Rule != nil {
					switch r.Rule.Severity {
					case model.SeverityError:
						sevColored = fmt.Sprintf("%s%s error  %s", cui.ASCIIRed, "✗", cui.ASCIIReset)
					case model.SeverityWarn:
						sevColored = fmt.Sprintf("%s%s warning%s", cui.ASCIIYellow, "▲", cui.ASCIIReset)
					case model.SeverityInfo:
						sevColored = fmt.Sprintf("%s%s info   %s", cui.ASCIIBlue, "●", cui.ASCIIReset)
					default:
						sevColored = fmt.Sprintf("%-*s", sevWidth, r.Rule.Severity)
					}
				} else {
					sevColored = fmt.Sprintf("%s%s info   %s", cui.ASCIIBlue, "●", cui.ASCIIReset)
				}
			}

			// Get rule and category
			ruleId := ""
			category := ""
			if r.Rule != nil {
				ruleId = r.Rule.Id
				if r.Rule.RuleCategory != nil {
					category = r.Rule.RuleCategory.Name
				}
			}

			// Print row with path in grey (like BubbleTea UI)
			// Calculate padding based on visible width (excluding ANSI codes)
			locPadding := locWidth - cui.VisibleLength(coloredLocation)
			if locPadding < 0 {
				locPadding = 0
			}

			msgPadding := msgWidth - cui.VisibleLength(coloredMessage)
			if msgPadding < 0 {
				msgPadding = 0
			}

			// Calculate padding for colorized path (account for ANSI codes)
			var pathPadding int
			if showPath {
				pathPadding = pathWidth - cui.VisibleLength(coloredPath)
				if pathPadding < 0 {
					pathPadding = 0
				}
			}

			if !noMessage {
				if !showRule && !showCategory && !showPath {
					// Very narrow terminals: only location, severity symbol, message
					fmt.Printf("%s%*s  %s  %s%*s\n",
						coloredLocation, locPadding, "",
						sevColored,
						coloredMessage, msgPadding, "")
				} else if showCategory && showPath {
					// All columns
					fmt.Printf("%s%*s  %-10s  %s%*s  %-*s  %-*s  %s%s%*s%s\n",
						coloredLocation, locPadding, "",
						sevColored,
						coloredMessage, msgPadding, "",
						ruleWidth, ruleId,
						catWidth, category,
						cui.ASCIIGrey, coloredPath, pathPadding, "", cui.ASCIIReset)
				} else if !showCategory && showPath {
					// No category column for medium terminals
					fmt.Printf("%s%*s  %-10s  %s%*s  %-*s  %s%s%*s%s\n",
						coloredLocation, locPadding, "",
						sevColored,
						coloredMessage, msgPadding, "",
						ruleWidth, ruleId,
						cui.ASCIIGrey, coloredPath, pathPadding, "", cui.ASCIIReset)
				} else if !showCategory && !showPath {
					// No category or path for narrow terminals
					fmt.Printf("%s%*s  %-10s  %s%*s  %-*s\n",
						coloredLocation, locPadding, "",
						sevColored,
						coloredMessage, msgPadding, "",
						ruleWidth, ruleId)
				}
			} else {
				if showCategory && showPath {
					fmt.Printf("%s%*s  %-10s  %-*s  %-*s  %s%s%*s%s\n",
						coloredLocation, locPadding, "",
						sevColored,
						ruleWidth, ruleId,
						catWidth, category,
						cui.ASCIIGrey, coloredPath, pathPadding, "", cui.ASCIIReset)
				} else if !showCategory && showPath {
					fmt.Printf("%s%*s  %-10s  %-*s  %s%s%*s%s\n",
						coloredLocation, locPadding, "",
						sevColored,
						ruleWidth, ruleId,
						cui.ASCIIGrey, coloredPath, pathPadding, "", cui.ASCIIReset)
				} else if !showCategory && !showPath {
					fmt.Printf("%s%*s  %-10s  %-*s\n",
						coloredLocation, locPadding, "",
						sevColored,
						ruleWidth, ruleId)
				}
			}
		}
		fmt.Println()
	}
}

func renderFixedSummary(rs *model.RuleResultSet, cats []*model.RuleCategory,
	stats *reports.ReportStatistics, fileName string, silent bool, noStyle bool) {

	if silent {
		return
	}

	// Build category summary table
	fmt.Printf("%s%-20s  %-10s  %-10s  %-10s%s\n", cui.ASCIIBlue, "Category", "Errors", "Warnings", "Info", cui.ASCIIReset)
	fmt.Printf("%s%s  %s  %s  %s%s\n",
		cui.ASCIIGrey,
		strings.Repeat("─", 20),
		strings.Repeat("─", 10),
		strings.Repeat("─", 10),
		strings.Repeat("─", 10),
		cui.ASCIIReset)

	for _, cat := range cats {
		errors := rs.GetErrorsByRuleCategory(cat.Id)
		warn := rs.GetWarningsByRuleCategory(cat.Id)
		info := rs.GetInfoByRuleCategory(cat.Id)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {
			fmt.Printf("%-20s  %-10s  %-10s  %-10s\n",
				cat.Name,
				humanize.Comma(int64(len(errors))),
				humanize.Comma(int64(len(warn))),
				humanize.Comma(int64(len(info))))
		}
	}
	fmt.Println()

	// Render result box
	errs := rs.GetErrorCount()
	warnings := rs.GetWarnCount()
	informs := rs.GetInfoCount()

	if errs > 0 {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIRed, cui.ASCIIReset)
		fmt.Printf("%s│  ❌ Linting failed with %d errors, %d warnings and %d informs  │%s\n",
			cui.ASCIIRed, errs, warnings, informs, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIRed, cui.ASCIIReset)
	} else if warnings > 0 {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIYellow, cui.ASCIIReset)
		fmt.Printf("%s│  ⚠️  Linting passed with %d warnings and %d informs  │%s\n",
			cui.ASCIIYellow, warnings, informs, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIYellow, cui.ASCIIReset)
	} else if informs > 0 {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIBlue, cui.ASCIIReset)
		fmt.Printf("%s│  ℹ️  Linting passed, %d informs reported  │%s\n", cui.ASCIIBlue, informs, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIBlue, cui.ASCIIReset)
	} else {
		fmt.Printf("%s╭──────────────────────────────────────────────────────────────────────╮%s\n", cui.ASCIIGreen, cui.ASCIIReset)
		fmt.Printf("%s│  ✅ Perfect score! Well done!  │%s\n", cui.ASCIIGreen, cui.ASCIIReset)
		fmt.Printf("%s╰──────────────────────────────────────────────────────────────────────────╯%s\n", cui.ASCIIGreen, cui.ASCIIReset)
	}

	// Show score if we have stats
	if stats != nil {
		fmt.Println()
		score := stats.OverallScore
		var color string
		var emoji string

		switch {
		case score >= 90:
			color = cui.ASCIIGreen
			emoji = "🏆"
		case score >= 70:
			color = cui.ASCIIBlue
			emoji = "👍"
		case score >= 50:
			color = cui.ASCIIYellow
			emoji = "⚡"
		default:
			color = cui.ASCIIRed
			emoji = "💔"
		}

		fmt.Printf("%s╔════════════════════════════╗%s\n", color, cui.ASCIIReset)
		fmt.Printf("%s║  %s Quality Score: %d/100  ║%s\n", color, emoji, score, cui.ASCIIReset)
		fmt.Printf("%s╚════════════════════════════╝%s\n", color, cui.ASCIIReset)
	}
}

func renderFixedTiming(duration time.Duration, fileSize int64) {
	fmt.Println()

	l := "milliseconds"
	d := fmt.Sprintf("%d", duration.Milliseconds())
	if duration.Milliseconds() > 1000 {
		l = "seconds"
		d = humanize.FormatFloat("##.##", duration.Seconds())
	}

	fmt.Printf("\033[36m⏱️  vacuum took %s %s to lint %s\033[0m\n",
		d, l, index.HumanFileSize(float64(fileSize)))
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
