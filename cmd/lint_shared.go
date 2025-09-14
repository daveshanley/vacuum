// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

// LintFlags holds all the command line flags for lint operations
type LintFlags struct {
	GlobPattern              string
	DetailsFlag              bool
	SnippetsFlag             bool
	ErrorsFlag               bool
	CategoryFlag             string
	SilentFlag               bool
	NoStyleFlag              bool
	NoBannerFlag             bool
	NoMessageFlag            bool
	AllResultsFlag           bool
	ShowRules                bool
	PipelineOutput           bool
	FailSeverityFlag         string
	BaseFlag                 string
	RemoteFlag               bool
	SkipCheckFlag            bool
	TimeoutFlag              int
	RulesetFlag              string
	FunctionsFlag            string
	TimeFlag                 bool
	HardModeFlag             bool
	IgnoreFile               string
	NoClipFlag               bool
	ExtRefsFlag              bool
	IgnoreArrayCircleRef     bool
	IgnorePolymorphCircleRef bool
	MinScore                 int
	CertFile                 string
	KeyFile                  string
	CAFile                   string
	Insecure                 bool
	DebugFlag                bool
}

// FileProcessingConfig contains all configuration needed to process a file
type FileProcessingConfig struct {
	Flags           *LintFlags
	Logger          *slog.Logger
	BufferedLogger  *cui.BufferedLogger
	SelectedRuleset *rulesets.RuleSet
	CustomFunctions map[string]model.RuleFunction
	IgnoredItems    model.IgnoredItems
}

// ReadLintFlags reads all lint-related flags from the command
func ReadLintFlags(cmd *cobra.Command) *LintFlags {
	flags := &LintFlags{}
	flags.GlobPattern, _ = cmd.Flags().GetString("globbed-files")
	flags.DetailsFlag, _ = cmd.Flags().GetBool("details")
	flags.SnippetsFlag, _ = cmd.Flags().GetBool("snippets")
	flags.ErrorsFlag, _ = cmd.Flags().GetBool("errors")
	flags.CategoryFlag, _ = cmd.Flags().GetString("category")
	flags.SilentFlag, _ = cmd.Flags().GetBool("silent")
	flags.NoStyleFlag, _ = cmd.Flags().GetBool("no-style")
	flags.NoBannerFlag, _ = cmd.Flags().GetBool("no-banner")
	flags.NoMessageFlag, _ = cmd.Flags().GetBool("no-message")
	flags.AllResultsFlag, _ = cmd.Flags().GetBool("all-results")
	flags.ShowRules, _ = cmd.Flags().GetBool("show-rules")
	flags.PipelineOutput, _ = cmd.Flags().GetBool("pipeline-output")
	flags.FailSeverityFlag, _ = cmd.Flags().GetString("fail-severity")
	flags.BaseFlag, _ = cmd.Flags().GetString("base")
	flags.RemoteFlag, _ = cmd.Flags().GetBool("remote")
	flags.SkipCheckFlag, _ = cmd.Flags().GetBool("skip-check")
	flags.TimeoutFlag, _ = cmd.Flags().GetInt("timeout")
	flags.RulesetFlag, _ = cmd.Flags().GetString("ruleset")
	flags.FunctionsFlag, _ = cmd.Flags().GetString("functions")
	flags.TimeFlag, _ = cmd.Flags().GetBool("time")
	flags.HardModeFlag, _ = cmd.Flags().GetBool("hard-mode")
	flags.IgnoreFile, _ = cmd.Flags().GetString("ignore-file")
	flags.NoClipFlag, _ = cmd.Flags().GetBool("no-clip")
	flags.ExtRefsFlag, _ = cmd.Flags().GetBool("ext-refs")
	flags.IgnoreArrayCircleRef, _ = cmd.Flags().GetBool("ignore-array-circle-ref")
	flags.IgnorePolymorphCircleRef, _ = cmd.Flags().GetBool("ignore-polymorph-circle-ref")
	flags.MinScore, _ = cmd.Flags().GetInt("min-score")
	flags.CertFile, _ = cmd.Flags().GetString("cert-file")
	flags.KeyFile, _ = cmd.Flags().GetString("key-file")
	flags.CAFile, _ = cmd.Flags().GetString("ca-file")
	flags.Insecure, _ = cmd.Flags().GetBool("insecure")
	flags.DebugFlag, _ = cmd.Flags().GetBool("debug")
	return flags
}

// SetupVacuumEnvironment configures the environment based on flags
func SetupVacuumEnvironment(flags *LintFlags) {
	if !flags.NoStyleFlag && !flags.PipelineOutput {
		fileInfo, _ := os.Stdout.Stat()
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			flags.NoStyleFlag = true
		}
	}

	if flags.NoStyleFlag && !flags.PipelineOutput {
		cui.DisableColors()
	}

	if !flags.SilentFlag && !flags.NoBannerFlag && !flags.PipelineOutput {
		PrintBanner(flags.NoStyleFlag)
	}
}

// LoadIgnoreFile loads and parses the ignore file if specified
func LoadIgnoreFile(ignoreFile string, silent, pipeline, noStyle bool) (model.IgnoredItems, error) {
	ignoredItems := model.IgnoredItems{}
	if ignoreFile == "" {
		return ignoredItems, nil
	}

	raw, err := os.ReadFile(ignoreFile)
	if err != nil {
		if !silent {
			fmt.Printf("%sError: Failed to read ignore file '%s': %v%s\n\n",
				cui.ASCIIRed, ignoreFile, err, cui.ASCIIReset)
		}
		return ignoredItems, fmt.Errorf("failed to read ignore file: %w", err)
	}

	err = yaml.Unmarshal(raw, &ignoredItems)
	if err != nil {
		if !silent {
			fmt.Printf("%sError: Failed to parse ignore file '%s': %v%s\n\n",
				cui.ASCIIRed, ignoreFile, err, cui.ASCIIReset)
		}
		return ignoredItems, fmt.Errorf("failed to parse ignore file: %w", err)
	}

	if !silent && !pipeline {
		renderInfoMessage(fmt.Sprintf("Using ignore file '%s'", ignoreFile), noStyle)
		renderIgnoredItems(ignoredItems, noStyle)
	}

	return ignoredItems, nil
}

// CreateHTTPClientFromFlags creates an HTTP client based on certificate flags
func CreateHTTPClientFromFlags(flags *LintFlags) (*http.Client, error) {
	httpClientConfig := utils.HTTPClientConfig{
		CertFile: flags.CertFile,
		KeyFile:  flags.KeyFile,
		CAFile:   flags.CAFile,
		Insecure: flags.Insecure,
	}

	if !utils.ShouldUseCustomHTTPClient(httpClientConfig) {
		return nil, nil
	}

	httpClient, err := utils.CreateCustomHTTPClient(httpClientConfig)
	if err != nil {
		fmt.Printf("\033[31mFailed to create custom HTTP client: %s\033[0m\n", err.Error())
		return nil, err
	}

	return httpClient, nil
}

// LoadRulesetWithConfig loads and configures the ruleset based on flags
func LoadRulesetWithConfig(flags *LintFlags, logger *slog.Logger) (*rulesets.RuleSet, error) {
	defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	if flags.HardModeFlag {
		selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
		owaspRules := rulesets.GetAllOWASPRules()
		for k, v := range owaspRules {
			selectedRS.Rules[k] = v
		}
		if !flags.SilentFlag && !flags.PipelineOutput {
			if flags.RulesetFlag == "" {
				renderHardModeBox(HardModeEnabled, flags.NoStyleFlag)
			}
		}
	}

	if flags.RulesetFlag != "" {
		httpClient, err := CreateHTTPClientFromFlags(flags)
		if err != nil {
			return nil, err
		}

		var rsErr error
		selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(
			flags.RulesetFlag, defaultRuleSets, flags.RemoteFlag, httpClient)
		if rsErr != nil {
			fmt.Printf("\033[31mUnable to load ruleset '%s': %s\033[0m\n",
				flags.RulesetFlag, rsErr.Error())
			return nil, rsErr
		}

		if !flags.SilentFlag && !flags.PipelineOutput {
			if flags.NoStyleFlag {
				fmt.Printf(" using ruleset '%s' (containing %d rules)\n",
					flags.RulesetFlag, len(selectedRS.Rules))
			} else {
				fmt.Printf(" %susing ruleset %s'%s'%s %s(containing %s%d%s rules)%s\n",
					cui.ASCIIGrey,
					cui.ASCIIBold+cui.ASCIIItalic, flags.RulesetFlag, cui.ASCIIReset+cui.ASCIIGrey,
					cui.ASCIIGrey,
					cui.ASCIIBold+cui.ASCIIItalic, len(selectedRS.Rules), cui.ASCIIReset+cui.ASCIIGrey,
					cui.ASCIIReset)
			}
		}

		if flags.HardModeFlag {
			if MergeOWASPRulesToRuleSet(selectedRS, true) {
				if !flags.SilentFlag && !flags.PipelineOutput {
					renderHardModeBox(HardModeWithCustomRuleset, flags.NoStyleFlag)
				}
			}
		}
	}

	if flags.ShowRules && !flags.PipelineOutput && !flags.SilentFlag {
		renderRulesList(selectedRS.Rules)
	}

	return selectedRS, nil
}

// RenderBufferedLogs renders the buffered logs with proper formatting and spacing
func RenderBufferedLogs(bufferedLogger *cui.BufferedLogger, noStyle bool) {
	if bufferedLogger == nil {
		return
	}

	logOutput := bufferedLogger.RenderTree(noStyle)
	if logOutput != "" {
		fmt.Print(logOutput)
		fmt.Println() // Add spacing after logs
	}
}

// GetHTTPClientConfig creates HTTPClientConfig from flags
func GetHTTPClientConfig(flags *LintFlags) utils.HTTPClientConfig {
	return utils.HTTPClientConfig{
		CertFile: flags.CertFile,
		KeyFile:  flags.KeyFile,
		CAFile:   flags.CAFile,
		Insecure: flags.Insecure,
	}
}

// ProcessSingleFileOptimized processes a single file using pre-loaded configuration
func ProcessSingleFileOptimized(fileName string, config *FileProcessingConfig) *FileProcessingResult {
	var fileSize int64
	fileInfo, err := os.Stat(fileName)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	var logger *slog.Logger
	var bufferedLogger *cui.BufferedLogger

	if config.Logger != nil {
		logger = config.Logger
		bufferedLogger = config.BufferedLogger
	} else if config.BufferedLogger != nil {
		// Use the provided BufferedLogger
		bufferedLogger = config.BufferedLogger
		handler := cui.NewBufferedLogHandler(bufferedLogger)
		logger = slog.New(handler)
	} else {
		// Create a new BufferedLogger
		bufferedLogger = cui.NewBufferedLogger()
		handler := cui.NewBufferedLogHandler(bufferedLogger)
		logger = slog.New(handler)
	}

	specBytes, err := os.ReadFile(fileName)
	if err != nil {
		return &FileProcessingResult{
			FileSize: fileSize,
			Error:    err,
		}
	}

	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:                         config.SelectedRuleset,
		Spec:                            specBytes,
		SpecFileName:                    fileName,
		CustomFunctions:                 config.CustomFunctions,
		Base:                            config.Flags.BaseFlag,
		AllowLookup:                     config.Flags.RemoteFlag,
		SkipDocumentCheck:               config.Flags.SkipCheckFlag,
		SilenceLogs:                     config.Flags.SilentFlag,
		Timeout:                         time.Duration(config.Flags.TimeoutFlag) * time.Second,
		IgnoreCircularArrayRef:          config.Flags.IgnoreArrayCircleRef,
		IgnoreCircularPolymorphicRef:    config.Flags.IgnorePolymorphCircleRef,
		BuildDeepGraph:                  len(config.IgnoredItems) > 0,
		ExtractReferencesFromExtensions: config.Flags.ExtRefsFlag,
		Logger:                          logger,
		HTTPClientConfig:                GetHTTPClientConfig(config.Flags),
	})

	if len(result.Errors) > 0 {
		var logs []string
		if bufferedLogger != nil {
			// Render the buffered logs as a tree
			treeOutput := bufferedLogger.RenderTree(config.Flags.NoStyleFlag)
			if treeOutput != "" {
				// Store the entire rendered tree output as a single log entry
				// This preserves the spacing that RenderTree carefully added
				logs = append(logs, treeOutput)
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
		if shouldIgnoreResult(r, config.IgnoredItems) {
			continue
		}

		resultCopy := r
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
	if bufferedLogger != nil {
		// Render the buffered logs as a tree
		treeOutput := bufferedLogger.RenderTree(config.Flags.NoStyleFlag)
		if treeOutput != "" {
			// Store the entire rendered tree output as a single log entry
			// This preserves the spacing that RenderTree carefully added
			logs = append(logs, treeOutput)
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

// shouldIgnoreResult checks if a result should be ignored based on ignore rules
func shouldIgnoreResult(result model.RuleFunctionResult, ignoredItems model.IgnoredItems) bool {
	if len(ignoredItems) == 0 {
		return false
	}

	// Check if this rule/path combination should be ignored
	if paths, exists := ignoredItems[result.Rule.Id]; exists {
		for _, ignorePath := range paths {
			if result.Path == ignorePath {
				return true
			}
		}
	}

	return false
}
