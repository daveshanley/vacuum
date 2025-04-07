// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func GetLintCommand() *cobra.Command {

	validFileExtensions := []string{"yaml", "yml", "json"}
	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "lint <your-openapi-file.yaml>",
		Short:        "Lint an OpenAPI specification",
		Long:         `Lint an OpenAPI specification, the output of the response will be in the terminal`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return validFileExtensions, cobra.ShellCompDirectiveFilterFileExt
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			globPattern, _ := cmd.Flags().GetString("globbed-files")
			detailsFlag, _ := cmd.Flags().GetBool("details")
			timeFlag, _ := cmd.Flags().GetBool("time")
			snippetsFlag, _ := cmd.Flags().GetBool("snippets")
			errorsFlag, _ := cmd.Flags().GetBool("errors")
			categoryFlag, _ := cmd.Flags().GetString("category")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")
			silent, _ := cmd.Flags().GetBool("silent")
			functionsFlag, _ := cmd.Flags().GetString("functions")
			failSeverityFlag, _ := cmd.Flags().GetString("fail-severity")
			noStyleFlag, _ := cmd.Flags().GetBool("no-style")
			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			debugFlag, _ := cmd.Flags().GetBool("debug")
			noBanner, _ := cmd.Flags().GetBool("no-banner")
			noMessage, _ := cmd.Flags().GetBool("no-message")
			allResults, _ := cmd.Flags().GetBool("all-results")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			noClipFlag, _ := cmd.Flags().GetBool("no-clip")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
			ignoreArrayCircleRef, _ := cmd.Flags().GetBool("ignore-array-circle-ref")
			ignorePolymorphCircleRef, _ := cmd.Flags().GetBool("ignore-polymorph-circle-ref")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				// No color or styling needed
			}

			if !silent && !noBanner {
				PrintBanner()
			}

			filesToLint, err := getFilesToLint(globPattern, args, validFileExtensions)
			// If the user has specifically asked for --globbed-files and it throws an error, they should know about it.
			// However if they have not, then they should expect the default behavior.
			if cmd.Flags().Changed("globbed-files") && err != nil {
				fmt.Fprintf(os.Stderr, "Error: Error getting files to lint: %v\n\n", err)
				return err
			}

			// verify that there is at least one file to lint
			if len(filesToLint) < 1 {
				fmt.Fprintf(os.Stderr, "Error: Please supply an OpenAPI specification to lint\n\n")
				return fmt.Errorf("no file supplied")
			}

			var errs []error

			mf := false
			if len(filesToLint) > 1 {
				mf = true
			}

			logLevel := slog.LevelError
			if debugFlag {
				logLevel = slog.LevelDebug
			}

			// setup logging
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: logLevel,
			}))

			defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
			selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
			customFunctions, _ := LoadCustomFunctions(functionsFlag, silent)

			// HARD MODE
			if hardModeFlag {
				selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()

				// extract all OWASP Rules
				owaspRules := rulesets.GetAllOWASPRules()
				allRules := selectedRS.Rules
				for k, v := range owaspRules {
					allRules[k] = v
				}
				if !silent {
					fmt.Println("\n=====================================")
					fmt.Println("🚨 HARD MODE ENABLED 🚨")
					fmt.Println("=====================================")
					fmt.Println()
				}
			}

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {

				rsBytes, rsErr := os.ReadFile(rulesetFlag)
				if rsErr != nil {
					fmt.Fprintf(os.Stderr, "Error: Unable to read ruleset file '%s': %s\n\n", rulesetFlag, rsErr.Error())
					return rsErr
				}

				selectedRS, rsErr = BuildRuleSetFromUserSuppliedSet(rsBytes, defaultRuleSets)
				if rsErr != nil {
					return rsErr
				}
			}

			var printLock sync.Mutex

			doneChan := make(chan bool)

			if len(filesToLint) <= 1 {
				if !silent {
					fmt.Printf("Info: Linting file '%s' against %d rules: %s\n\n", filesToLint[0], len(selectedRS.Rules),
						selectedRS.DocumentationURI)
				}
			}

			if len(filesToLint) > 1 {
				if !silent {
					fmt.Printf("Info: Linting %d files against %d rules: %s\n\n", len(filesToLint), len(selectedRS.Rules),
						selectedRS.DocumentationURI)
				}
			}

			if len(ignoreFile) > 1 {
				if !silent {
					fmt.Printf("Info: Using ignore file '%s'\n\n", ignoreFile)
				}
			}

			ignoredItems := model.IgnoredItems{}
			if ignoreFile != "" {
				raw, ferr := os.ReadFile(ignoreFile)
				if ferr != nil {
					return fmt.Errorf("failed to read ignore file: %w", ferr)
				}
				ferr = yaml.Unmarshal(raw, &ignoredItems)
				if ferr != nil {
					return fmt.Errorf("failed to read ignore file: %w", ferr)
				}
			}

			start := time.Now()

			var filesProcessedSize int64
			var filesProcessed int
			var size int64
			for i, fileName := range filesToLint {

				go func(c chan bool, i int, fileName string) {
					// get size
					s, _ := os.Stat(fileName)
					if s != nil {
						size = size + s.Size()
					}

					lfr := utils.LintFileRequest{
						FileName:                 fileName,
						BaseFlag:                 baseFlag,
						Remote:                   remoteFlag,
						MultiFile:                mf,
						SkipCheckFlag:            skipCheckFlag,
						Silent:                   silent,
						DetailsFlag:              detailsFlag,
						TimeFlag:                 timeFlag,
						FailSeverityFlag:         failSeverityFlag,
						CategoryFlag:             categoryFlag,
						SnippetsFlag:             snippetsFlag,
						ErrorsFlag:               errorsFlag,
						NoMessageFlag:            noMessage,
						AllResultsFlag:           allResults,
						TotalFiles:               len(filesToLint),
						FileIndex:                i,
						DefaultRuleSets:          defaultRuleSets,
						SelectedRS:               selectedRS,
						Functions:                customFunctions,
						Lock:                     &printLock,
						Logger:                   logger,
						TimeoutFlag:              timeoutFlag,
						NoClip:                   noClipFlag,
						IgnoreArrayCircleRef:     ignoreArrayCircleRef,
						IgnorePolymorphCircleRef: ignorePolymorphCircleRef,
						IgnoredResults:           ignoredItems,
						ExtensionRefs:            extensionRefsFlag,
					}
					fs, fp, err := lintFile(lfr)

					filesProcessedSize = filesProcessedSize + fs + size
					filesProcessed = filesProcessed + fp + 1

					errs = append(errs, err)
					doneChan <- true
				}(doneChan, i, fileName)
			}

			completed := 0
			for completed < len(filesToLint) {
				<-doneChan
				completed++
			}

			if !detailsFlag && !silent {
				fmt.Println()
				fmt.Println("Info: To see full details of linting report, use the '-d' flag.")
				fmt.Println()
			}

			duration := time.Since(start)

			RenderTimeAndFiles(timeFlag, duration, filesProcessedSize, filesProcessed)

			if len(errs) > 0 {
				return errors.Join(errs...)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("details", "d", false, "Show full details of linting report")
	cmd.Flags().BoolP("snippets", "s", false, "Show code snippets where issues are found")
	cmd.Flags().BoolP("errors", "e", false, "Show errors only")
	cmd.Flags().StringP("category", "c", "", "Show a single category of results")
	cmd.Flags().BoolP("silent", "x", false, "Show nothing except the result.")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().BoolP("no-banner", "b", false, "Disable the banner / header output")
	cmd.Flags().BoolP("no-message", "m", false, "Hide the message output when using -d to show details")
	cmd.Flags().BoolP("all-results", "a", false, "Render out all results, regardless of the number when using -d")
	cmd.Flags().StringP("fail-severity", "n", model.SeverityError, "Results of this level or above will trigger a failure exit code (e.g. 'info', 'warn', 'error')")
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().Bool("no-clip", false, "Do not truncate messages or paths (no '...')")
	// TODO: Add globbed-files flag to other commands as well
	cmd.Flags().String("globbed-files", "", "Glob pattern of files to lint")

	if regErr := cmd.RegisterFlagCompletionFunc("category", cobra.FixedCompletions([]string{
		model.CategoryAll,
		model.CategoryDescriptions,
		model.CategoryExamples,
		model.CategoryInfo,
		model.CategoryOperations,
		model.CategorySchemas,
		model.CategorySecurity,
		model.CategoryTags,
		model.CategoryValidation,
	}, cobra.ShellCompDirectiveNoFileComp)); regErr != nil {
		panic(regErr)
	}
	if regErr := cmd.RegisterFlagCompletionFunc("fail-severity", cobra.FixedCompletions([]string{
		model.SeverityInfo,
		model.SeverityWarn,
		model.SeverityError,
	}, cobra.ShellCompDirectiveNoFileComp)); regErr != nil {
		panic(regErr)
	}
	if regErr := cmd.RegisterFlagCompletionFunc("globbed-files", cobra.NoFileCompletions); regErr != nil {
		panic(regErr)
	}

	return cmd
}

func lintFile(req utils.LintFileRequest) (int64, int, error) {
	// read file.
	specBytes, ferr := os.ReadFile(req.FileName)

	// split up file into an array with lines.
	specStringData := strings.Split(string(specBytes), "\n")

	if ferr != nil {

		fmt.Fprintf(os.Stderr, "Error: Unable to read file '%s': %s\n\n", req.FileName, ferr.Error())
		return 0, 0, ferr

	}

	deepGraph := false
	if req.IgnoredResults != nil && len(req.IgnoredResults) > 0 {
		deepGraph = true
	}

	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:                         req.SelectedRS,
		Spec:                            specBytes,
		SpecFileName:                    req.FileName,
		CustomFunctions:                 req.Functions,
		Base:                            req.BaseFlag,
		AllowLookup:                     req.Remote,
		SkipDocumentCheck:               req.SkipCheckFlag,
		Logger:                          req.Logger,
		BuildDeepGraph:                  deepGraph,
		Timeout:                         time.Duration(req.TimeoutFlag) * time.Second,
		IgnoreCircularArrayRef:          req.IgnoreArrayCircleRef,
		IgnoreCircularPolymorphicRef:    req.IgnorePolymorphCircleRef,
		ExtractReferencesFromExtensions: req.ExtensionRefs,
	})

	result.Results = filterIgnoredResults(result.Results, req.IgnoredResults)

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "Error: unable to process spec '%s', error: %s\n\n", req.FileName, err.Error())
		}
		return result.FileSize, result.FilesProcessed, fmt.Errorf("linting failed due to %d issues", len(result.Errors))
	}

	resultSet := model.NewRuleResultSet(result.Results)
	resultSet.SortResultsByLineNumber()
	warnings := resultSet.GetWarnCount()
	errs := resultSet.GetErrorCount()
	informs := resultSet.GetInfoCount()
	req.Lock.Lock()
	defer req.Lock.Unlock()
	if !req.DetailsFlag {
		RenderSummary(resultSet, req.Silent, req.TotalFiles, req.FileIndex, req.FileName, req.FailSeverityFlag)
		return result.FileSize, result.FilesProcessed, CheckFailureSeverity(req.FailSeverityFlag, errs, warnings, informs)
	}

	abs, _ := filepath.Abs(req.FileName)

	if len(resultSet.Results) > 0 {
		processResults(
			resultSet.Results,
			specStringData,
			req.SnippetsFlag,
			req.ErrorsFlag,
			req.Silent,
			req.NoMessageFlag,
			req.AllResultsFlag,
			req.NoClip,
			abs,
			req.FileName,
			req.CategoryFlag)
	}

	RenderSummary(resultSet, req.Silent, req.TotalFiles, req.FileIndex, req.FileName, req.FailSeverityFlag)

	return result.FileSize, result.FilesProcessed, CheckFailureSeverity(req.FailSeverityFlag, errs, warnings, informs)
}

// filterIgnoredResultsPtr filters the given results slice, taking out any (RuleID, Path) combos that were listed in the
// ignore file
func filterIgnoredResultsPtr(results []*model.RuleFunctionResult, ignored model.IgnoredItems) []*model.RuleFunctionResult {
	var filteredResults []*model.RuleFunctionResult

	for _, r := range results {

		var found bool
		for _, i := range ignored[r.Rule.Id] {
			if len(r.Paths) > 0 {
				for _, p := range r.Paths {
					if p == i {
						found = true
						break
					}
				}
			}
			if r.Path == i {
				found = true
				break
			}

		}
		if !found {
			filteredResults = append(filteredResults, r)
		}
	}

	return filteredResults
}

// filterIgnoredResults does the filtering of ignored results on non-pointer result elements
func filterIgnoredResults(results []model.RuleFunctionResult, ignored model.IgnoredItems) []model.RuleFunctionResult {
	resultsPtrs := make([]*model.RuleFunctionResult, 0, len(results))
	for _, r := range results {
		r := r // prevent loop memory aliasing
		resultsPtrs = append(resultsPtrs, &r)
	}
	resultsFiltered := make([]model.RuleFunctionResult, 0, len(results))
	for _, r := range filterIgnoredResultsPtr(resultsPtrs, ignored) {
		resultsFiltered = append(resultsFiltered, *r)
	}
	return resultsFiltered
}

func processResults(results []*model.RuleFunctionResult,
	specData []string,
	snippets,
	errors,
	silent,
	noMessage,
	allResults bool,
	noClip bool,
	abs, filename string,
	categoryFlag string) {

	if allResults && len(results) > 1000 {
		fmt.Printf("Warning: Formatting %s results - this could take a moment to render out in the terminal\n\n",
			humanize.Comma(int64(len(results))))
	}

	if !silent {
		fmt.Printf("\n%s\n", abs)
		underline := make([]string, len(abs))
		for x := range abs {
			underline[x] = "-"
		}
		fmt.Println(strings.Join(underline, ""))
	}

	// if snippets are being used, we render a single table for a result and then a snippet, if not
	// we just render the entire table, all rows.
	var tableData [][]string
	if !snippets {
		tableData = [][]string{{"Location", "Severity", "Message", "Rule", "Category", "Path"}}
	}
	if noMessage {
		tableData = [][]string{{"Location", "Severity", "Rule", "Category", "Path"}}
	}

	// width, height, err := terminal.GetSize(0)
	// TODO: determine the terminal size and render the linting results in a table that fits the screen.

	for i, r := range results {

		if i > 1000 && !allResults {
			tableData = append(tableData, []string{"", "", fmt.Sprintf("...%d more violations not rendered.", len(results)-1000), ""})
			break
		}

		startLine := 0
		startCol := 0
		if r.StartNode != nil {
			startLine = r.StartNode.Line
		}
		if r.StartNode != nil {
			startCol = r.StartNode.Column
		}

		f := filename
		if r.Origin != nil {
			f = r.Origin.AbsoluteLocation
			startLine = r.Origin.Line
			startCol = r.Origin.Column
		}
		start := fmt.Sprintf("%s:%v:%v", f, startLine, startCol)
		m := r.Message
		p := r.Path

		if !noClip {
			if len(r.Path) > 60 {
				p = fmt.Sprintf("%s...", r.Path[:60])
			}

			if len(r.Message) > 100 {
				m = fmt.Sprintf("%s...", r.Message[:80])
			}
		}
		sev := "info"
		if r.Rule != nil {
			sev = r.Rule.Severity
		}

		switch sev {
		case model.SeverityError:
			sev = "error"
		case model.SeverityWarn:
			sev = "warning"
		case model.SeverityInfo:
			sev = "info"
		}

		if errors && r.Rule.Severity != model.SeverityError {
			continue // only show errors
		}

		if !noMessage {
			tableData = append(tableData, []string{start, sev, m, r.Rule.Id, r.Rule.RuleCategory.Name, p})
		} else {
			tableData = append(tableData, []string{start, sev, r.Rule.Id, r.Rule.RuleCategory.Name, p})
		}
		if snippets && !silent {
			fmt.Println("Table data would be rendered here")
			renderCodeSnippet(r, specData)
		}
	}

	if !snippets && !silent {
		fmt.Println("Table data would be rendered here")
	}

}

func renderCodeSnippet(r *model.RuleFunctionResult, specData []string) {
	// render out code snippet
	if r.StartNode.Line-3 >= 0 {
		fmt.Printf("%d | %s\n", r.StartNode.Line-3, specData[r.StartNode.Line-3])
	} else {
		fmt.Printf("\n")
	}

	if r.StartNode.Line-2 >= 1 {
		fmt.Printf("%d | %s\n", r.StartNode.Line-2, specData[r.StartNode.Line-2])
	}
	if r.StartNode.Line-1 >= 2 {
		fmt.Printf("%d | %s\n", r.StartNode.Line-1, specData[r.StartNode.Line-1])
	}
	fmt.Printf("%d | %s\n", r.StartNode.Line, specData[r.StartNode.Line])

	if r.StartNode.Line+1 <= len(specData) {
		fmt.Printf("%d | %s\n\n", r.StartNode.Line+1, specData[r.StartNode.Line+1])
	}
}

func RenderSummary(rs *model.RuleResultSet, silent bool, totalFiles, fileIndex int, filename, sev string) {
	var tableData [][]string
	tableData = [][]string{{"Category", "Errors", "Warnings", "Info"}}

	for _, cat := range model.RuleCategoriesOrdered {
		errors := rs.GetErrorsByRuleCategory(cat.Id)
		warn := rs.GetWarningsByRuleCategory(cat.Id)
		info := rs.GetInfoByRuleCategory(cat.Id)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {
			tableData = append(tableData, []string{cat.Name, fmt.Sprintf("%v", humanize.Comma(int64(len(errors)))),
				fmt.Sprintf("%v", humanize.Comma(int64(len(warn)))), fmt.Sprintf("%v", humanize.Comma(int64(len(info))))})
		}
	}

	if len(rs.Results) > 0 {
		if !silent {
			fmt.Println("Table would be rendered here")
		}
	}

	errors := rs.GetErrorCount()
	warnings := rs.GetWarnCount()
	informs := rs.GetInfoCount()
	errorsHuman := humanize.Comma(int64(rs.GetErrorCount()))
	warningsHuman := humanize.Comma(int64(rs.GetWarnCount()))
	informsHuman := humanize.Comma(int64(rs.GetInfoCount()))

	if totalFiles <= 1 {

		if errors > 0 {
			fmt.Fprintf(os.Stderr, "Error: '%s' failed with %v errors, %v warnings and %v informs\n\n",
				filename, errorsHuman, warningsHuman, informsHuman)
			return
		}
		if warnings > 0 {
			msg := "passed, but with"
			switch sev {
			case model.SeverityWarn:
				msg = "failed with"
			}

			fmt.Printf("Warning: '%s' %s %v warnings and %v informs\n\n", filename, msg, warningsHuman, informsHuman)
			return
		}

		if informs > 0 {
			fmt.Printf("Success: '%s' passed, %v informs reported\n\n", filename, informsHuman)
			return
		}

		if silent {
			return
		}

		fmt.Printf("Success: '%s' passed, A perfect score! well done!\n\n", filename)

	} else {

		if errors > 0 {
			fmt.Fprintf(os.Stderr, "Error: '%s' failed with %v errors, %v warnings and %v informs\n\n",
				filename, errorsHuman, warningsHuman, informsHuman)
			return
		}
		if warnings > 0 {
			fmt.Printf("Warning: '%s' passed, but with %v warnings and %v informs\n\n", 
				filename, warningsHuman, informsHuman)
			return
		}

		if informs > 0 {
			fmt.Printf("Success: '%s' passed, %v informs reported\n\n", filename, informsHuman)
			return
		}

		fmt.Printf("Success: '%s' passed, A perfect score! well done!\n\n", filename)

	}

}

// The user may pass in filenames, a glob pattern, or both.
// We simply concatenate them together, and remove any duplicates we may find.
func getFilesToLint(globPattern string, filepaths []string, validFileExtensions []string) ([]string, error) {
	// Note that if some of the paths are absolute and the others are relative,
	// then we turn all paths into relative ones.
	if globPattern == "" {
		return deduplicate(filepaths), nil
	}

	var filesToLint = filepaths

	// Get all files that match the glob pattern
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return []string{}, err
	}
	filesToLint = append(filesToLint, matches...)

	// Remove any duplicates
	filesToLint = deduplicate(filesToLint)

	// Ensure that all files have valid file extensions
	for _, file := range filesToLint {
		if !hasValidExtension(file, validFileExtensions) {
			return []string{}, fmt.Errorf("File %q has an invalid file extension. Only %v are supported.\n", file, validFileExtensions)
		}
	}

	return filesToLint, nil
}

func deduplicate(input []string) []string {
	seen := make(map[string]bool)
	deduplicated := []string{}
	for _, val := range input {
		if _, alreadySeen := seen[val]; !alreadySeen {
			seen[val] = true
			deduplicated = append(deduplicated, val)
		}
	}
	return deduplicated
}

func hasValidExtension(filename string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}