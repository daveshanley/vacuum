// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/logging"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v4"
)

// ResolveBasePathForFile determines the base path to use for a given spec file.
// If baseFlag is explicitly set (not empty), it returns that value unchanged.
// If baseFlag is empty, it returns the absolute directory of the spec file.
func ResolveBasePathForFile(specFilePath string, baseFlag string) (string, error) {
	// If base is explicitly set, use it as-is
	if baseFlag != "" {
		return baseFlag, nil
	}

	// Auto-detect base from spec file location
	absPath, err := filepath.Abs(specFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for %s: %w", specFilePath, err)
	}

	return filepath.Dir(absPath), nil
}

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
	AllowPrivateNetworks     bool
	AllowHTTP                bool
	FetchTimeout             int
	DebugFlag                bool
	LookupTimeoutFlag        int
	FixFlag                  bool
	FixFileFlag              string
	ChangesFlag              string // --changes: path to JSON change report
	OriginalFlag             string // --original: path to original spec for inline comparison
	ChangesSummaryFlag       bool   // --changes-summary: show filtered results summary
	BreakingConfigPath       string // --breaking-config: path to breaking rules config
	WarnOnChanges            bool   // --warn-on-changes: inject warnings for API changes
	ErrorOnBreaking          bool   // --error-on-breaking: inject errors for breaking changes
	TurboMode                bool   // --turbo: faster linting, trades some checks for speed
	SkipResolve              bool   // --skip-resolve: skip second-pass reference resolution
	SkipCircularCheck        bool   // --skip-circular-check: skip circular reference detection
	SkipSchemaErrors         bool   // --skip-schema-errors: skip schema build error injection
	SkipStats                bool   // --skip-stats: skip report statistics generation
	MaxResultsPerRule        int    // --max-results-per-rule: max results per rule (0 = unlimited)
	MaxTotalResults          int    // --max-total-results: max total results (0 = unlimited)
}

// FileProcessingConfig contains all configuration needed to process a file
type FileProcessingConfig struct {
	Flags           *LintFlags
	Logger          *slog.Logger
	BufferedLogger  *logging.BufferedLogger
	SelectedRuleset *rulesets.RuleSet
	CustomFunctions map[string]model.RuleFunction
	IgnoredItems    model.IgnoredItems
	FetchConfig     *utils.FetchConfig
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
	if flags.BaseFlag == "" && viper.IsSet("lint.base") {
		flags.BaseFlag = viper.GetString("lint.base")
	}
	flags.RemoteFlag, _ = cmd.Flags().GetBool("remote")
	if !cmd.Flags().Changed("remote") && viper.IsSet("lint.remote") {
		flags.RemoteFlag = viper.GetBool("lint.remote")
	}
	flags.SkipCheckFlag, _ = cmd.Flags().GetBool("skip-check")
	if !cmd.Flags().Changed("skip-check") && viper.IsSet("lint.skip-check") {
		flags.SkipCheckFlag = viper.GetBool("lint.skip-check")
	}
	flags.TimeoutFlag, _ = cmd.Flags().GetInt("timeout")
	if !cmd.Flags().Changed("timeout") && viper.IsSet("lint.timeout") {
		flags.TimeoutFlag = viper.GetInt("lint.timeout")
	}
	flags.LookupTimeoutFlag, _ = cmd.Flags().GetInt("lookup-timeout")
	if !cmd.Flags().Changed("lookup-timeout") && viper.IsSet("lint.lookup-timeout") {
		flags.LookupTimeoutFlag = viper.GetInt("lint.lookup-timeout")
	}
	flags.RulesetFlag, _ = cmd.Flags().GetString("ruleset")
	// Fallback to lint-scoped config if no ruleset was provided via flag/env/root-config
	if flags.RulesetFlag == "" && viper.IsSet("lint.ruleset") {
		flags.RulesetFlag = viper.GetString("lint.ruleset")
	}
	flags.FunctionsFlag, _ = cmd.Flags().GetString("functions")
	if flags.FunctionsFlag == "" && viper.IsSet("lint.functions") {
		flags.FunctionsFlag = viper.GetString("lint.functions")
	}
	flags.TimeFlag, _ = cmd.Flags().GetBool("time")
	if !cmd.Flags().Changed("time") && viper.IsSet("lint.time") {
		flags.TimeFlag = viper.GetBool("lint.time")
	}
	flags.HardModeFlag, _ = cmd.Flags().GetBool("hard-mode")
	if !cmd.Flags().Changed("hard-mode") && viper.IsSet("lint.hard-mode") {
		flags.HardModeFlag = viper.GetBool("lint.hard-mode")
	}
	flags.IgnoreFile, _ = cmd.Flags().GetString("ignore-file")
	flags.NoClipFlag, _ = cmd.Flags().GetBool("no-clip")
	flags.ExtRefsFlag, _ = cmd.Flags().GetBool("ext-refs")
	if !cmd.Flags().Changed("ext-refs") && viper.IsSet("lint.ext-refs") {
		flags.ExtRefsFlag = viper.GetBool("lint.ext-refs")
	}
	flags.IgnoreArrayCircleRef, _ = cmd.Flags().GetBool("ignore-array-circle-ref")
	flags.IgnorePolymorphCircleRef, _ = cmd.Flags().GetBool("ignore-polymorph-circle-ref")
	flags.MinScore, _ = cmd.Flags().GetInt("min-score")
	flags.CertFile, _ = cmd.Flags().GetString("cert-file")
	if flags.CertFile == "" && viper.IsSet("lint.cert-file") {
		flags.CertFile = viper.GetString("lint.cert-file")
	}
	flags.KeyFile, _ = cmd.Flags().GetString("key-file")
	if flags.KeyFile == "" && viper.IsSet("lint.key-file") {
		flags.KeyFile = viper.GetString("lint.key-file")
	}
	flags.CAFile, _ = cmd.Flags().GetString("ca-file")
	if flags.CAFile == "" && viper.IsSet("lint.ca-file") {
		flags.CAFile = viper.GetString("lint.ca-file")
	}
	flags.Insecure, _ = cmd.Flags().GetBool("insecure")
	if !cmd.Flags().Changed("insecure") && viper.IsSet("lint.insecure") {
		flags.Insecure = viper.GetBool("lint.insecure")
	}
	flags.AllowPrivateNetworks, _ = cmd.Flags().GetBool("allow-private-networks")
	if !cmd.Flags().Changed("allow-private-networks") && viper.IsSet("lint.allow-private-networks") {
		flags.AllowPrivateNetworks = viper.GetBool("lint.allow-private-networks")
	}
	flags.AllowHTTP, _ = cmd.Flags().GetBool("allow-http")
	if !cmd.Flags().Changed("allow-http") && viper.IsSet("lint.allow-http") {
		flags.AllowHTTP = viper.GetBool("lint.allow-http")
	}
	flags.FetchTimeout, _ = cmd.Flags().GetInt("fetch-timeout")
	if !cmd.Flags().Changed("fetch-timeout") && viper.IsSet("lint.fetch-timeout") {
		flags.FetchTimeout = viper.GetInt("lint.fetch-timeout")
	}
	flags.DebugFlag, _ = cmd.Flags().GetBool("debug")
	if !cmd.Flags().Changed("debug") && viper.IsSet("lint.debug") {
		flags.DebugFlag = viper.GetBool("lint.debug")
	}
	flags.FixFlag, _ = cmd.Flags().GetBool("fix")
	flags.FixFileFlag, _ = cmd.Flags().GetString("fix-file")
	flags.ChangesFlag, _ = cmd.Flags().GetString("changes")
	flags.OriginalFlag, _ = cmd.Flags().GetString("original")
	flags.ChangesSummaryFlag, _ = cmd.Flags().GetBool("changes-summary")
	flags.BreakingConfigPath, _ = cmd.Flags().GetString("breaking-config")
	flags.WarnOnChanges, _ = cmd.Flags().GetBool("warn-on-changes")
	flags.ErrorOnBreaking, _ = cmd.Flags().GetBool("error-on-breaking")
	flags.TurboMode, _ = cmd.Flags().GetBool("turbo")
	if !cmd.Flags().Changed("turbo") && viper.IsSet("lint.turbo") {
		flags.TurboMode = viper.GetBool("lint.turbo")
	}
	flags.SkipResolve, _ = cmd.Flags().GetBool("skip-resolve")
	flags.SkipCircularCheck, _ = cmd.Flags().GetBool("skip-circular-check")
	flags.SkipSchemaErrors, _ = cmd.Flags().GetBool("skip-schema-errors")
	flags.SkipStats, _ = cmd.Flags().GetBool("skip-stats")
	flags.MaxResultsPerRule, _ = cmd.Flags().GetInt("max-results-per-rule")
	flags.MaxTotalResults, _ = cmd.Flags().GetInt("max-total-results")
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
		color.DisableColors()
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

	originalPath := ignoreFile
	resolvedPath, err := ResolveConfigPath(ignoreFile)
	if err != nil {
		if !silent {
			fmt.Printf("%sError: Failed to resolve ignore file path '%s': %v%s\n\n",
				color.ASCIIRed, ignoreFile, err, color.ASCIIReset)
		}
		return ignoredItems, fmt.Errorf("failed to resolve ignore file path: %w", err)
	}

	raw, err := os.ReadFile(resolvedPath)
	if err != nil {
		if !os.IsNotExist(err) || originalPath == resolvedPath {
			if !silent {
				fmt.Printf("%sError: Failed to read ignore file '%s': %v%s\n\n",
					color.ASCIIRed, resolvedPath, err, color.ASCIIReset)
			}
			return ignoredItems, fmt.Errorf("failed to read ignore file: %w", err)
		}
		// fallback to original path if resolution-based path not found
		raw, err = os.ReadFile(originalPath)
		if err != nil {
			if !silent {
				fmt.Printf("%sError: Failed to read ignore file '%s': %v%s\n\n",
					color.ASCIIRed, originalPath, err, color.ASCIIReset)
			}
			return ignoredItems, fmt.Errorf("failed to read ignore file: %w", err)
		}
		resolvedPath = originalPath
	}

	err = yaml.Unmarshal(raw, &ignoredItems)
	if err != nil {
		if !silent {
			fmt.Printf("%sError: Failed to parse ignore file '%s': %v%s\n\n",
				color.ASCIIRed, resolvedPath, err, color.ASCIIReset)
		}
		return ignoredItems, fmt.Errorf("failed to parse ignore file: %w", err)
	}

	if !silent && !pipeline {
		renderInfoMessage(fmt.Sprintf("Using ignore file '%s'", resolvedPath), noStyle)
		renderIgnoredItems(ignoredItems, noStyle)
	}

	return ignoredItems, nil
}

// CreateHTTPClientFromFlags creates an HTTP client based on certificate flags
func CreateHTTPClientFromFlags(flags *LintFlags) (*http.Client, error) {
	httpClientConfig, err := GetHTTPClientConfig(flags)
	if err != nil {
		return nil, err
	}

	httpClient, err := utils.CreateHTTPClientIfNeeded(httpClientConfig)
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
					color.ASCIIGrey,
					color.ASCIIBold+color.ASCIIItalic, flags.RulesetFlag, color.ASCIIReset+color.ASCIIGrey,
					color.ASCIIGrey,
					color.ASCIIBold+color.ASCIIItalic, len(selectedRS.Rules), color.ASCIIReset+color.ASCIIGrey,
					color.ASCIIReset)
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

	// Apply turbo mode rule filtering
	if flags.TurboMode {
		if flags.HardModeFlag && !flags.SilentFlag && !flags.PipelineOutput {
			fmt.Printf(" %s⚡ turbo mode active — some hard-mode rules will be excluded for speed%s\n",
				color.ASCIIYellow, color.ASCIIReset)
		}
		removed := rulesets.FilterRulesForTurbo(selectedRS)
		if !flags.SilentFlag && !flags.PipelineOutput {
			fmt.Printf(" %s⚡ turbo mode: removed %d expensive rules (%d rules remaining)%s\n",
				color.ASCIIYellow, removed, len(selectedRS.Rules), color.ASCIIReset)
		}
	}

	if flags.ShowRules && !flags.PipelineOutput && !flags.SilentFlag {
		renderRulesList(selectedRS.Rules)
	}

	return selectedRS, nil
}

// RenderBufferedLogs renders the buffered logs with proper formatting and spacing
func RenderBufferedLogs(bufferedLogger *logging.BufferedLogger, noStyle bool) {
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
func GetHTTPClientConfig(flags *LintFlags) (utils.HTTPClientConfig, error) {
	certFile, err := ResolveConfigPath(flags.CertFile)
	if err != nil {
		return utils.HTTPClientConfig{}, err
	}

	keyFile, err := ResolveConfigPath(flags.KeyFile)
	if err != nil {
		return utils.HTTPClientConfig{}, err
	}

	caFile, err := ResolveConfigPath(flags.CAFile)
	if err != nil {
		return utils.HTTPClientConfig{}, err
	}

	return utils.HTTPClientConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
		Insecure: flags.Insecure,
	}, nil
}

// GetFetchConfig creates FetchConfig from flags for JavaScript fetch() configuration.
// The config struct is always allocated (small cost), but the expensive HTTP client
// creation is deferred until a JS function actually calls fetch() - see NewFetchModuleFromConfig.
func GetFetchConfig(flags *LintFlags) (*utils.FetchConfig, error) {
	if flags.FetchTimeout < 0 {
		return nil, fmt.Errorf("fetch-timeout cannot be negative: %d", flags.FetchTimeout)
	}

	httpClientConfig, err := GetHTTPClientConfig(flags)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(flags.FetchTimeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &utils.FetchConfig{
		HTTPClientConfig:     httpClientConfig,
		AllowPrivateNetworks: flags.AllowPrivateNetworks,
		AllowHTTP:            flags.AllowHTTP,
		Timeout:              timeout,
	}, nil
}

// ProcessSingleFileOptimized processes a single file using pre-loaded configuration
func ProcessSingleFileOptimized(fileName string, config *FileProcessingConfig) *FileProcessingResult {
	var fileSize int64
	fileInfo, err := os.Stat(fileName)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	var logger *slog.Logger
	var bufferedLogger *logging.BufferedLogger

	if config.Logger != nil {
		logger = config.Logger
		bufferedLogger = config.BufferedLogger
	} else if config.BufferedLogger != nil {
		// Use the provided BufferedLogger
		bufferedLogger = config.BufferedLogger
		handler := logging.NewBufferedLogHandler(bufferedLogger)
		logger = slog.New(handler)
	} else {
		// Create a new BufferedLogger
		bufferedLogger = logging.NewBufferedLogger()
		handler := logging.NewBufferedLogHandler(bufferedLogger)
		logger = slog.New(handler)
	}

	specBytes, err := os.ReadFile(fileName)
	if err != nil {
		return &FileProcessingResult{
			FileSize: fileSize,
			Error:    err,
		}
	}

	// Resolve base path for this specific file
	resolvedBase, baseErr := ResolveBasePathForFile(fileName, config.Flags.BaseFlag)
	if baseErr != nil {
		return &FileProcessingResult{
			FileSize: fileSize,
			Error:    fmt.Errorf("failed to resolve base path: %w", baseErr),
		}
	}

	httpClientConfig, err := GetHTTPClientConfig(config.Flags)
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
		AutoFixFunctions:                make(map[string]model.AutoFixFunction),
		Base:                            resolvedBase,
		AllowLookup:                     config.Flags.RemoteFlag,
		SkipDocumentCheck:               config.Flags.SkipCheckFlag,
		SilenceLogs:                     config.Flags.SilentFlag,
		Timeout:                         time.Duration(config.Flags.TimeoutFlag) * time.Second,
		NodeLookupTimeout:               time.Duration(config.Flags.LookupTimeoutFlag) * time.Millisecond,
		IgnoreCircularArrayRef:          config.Flags.IgnoreArrayCircleRef,
		IgnoreCircularPolymorphicRef:    config.Flags.IgnorePolymorphCircleRef,
		BuildDeepGraph:                  len(config.IgnoredItems) > 0,
		ExtractReferencesFromExtensions: config.Flags.ExtRefsFlag,
		Logger:                          logger,
		HTTPClientConfig:                httpClientConfig,
		ApplyAutoFixes:                  config.Flags.FixFlag,
		FetchConfig:                     config.FetchConfig,
		TurboMode:                       config.Flags.TurboMode,
		SkipResolve:                     config.Flags.SkipResolve,
		SkipCircularCheck:               config.Flags.SkipCircularCheck,
		SkipSchemaErrors:                config.Flags.SkipSchemaErrors,
		MaxResultsPerRule:               config.Flags.MaxResultsPerRule,
		MaxTotalResults:                 config.Flags.MaxTotalResults,
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

	// Use index-based iteration to avoid copying the struct and take direct pointer to slice element
	for i := range result.Results {
		if shouldIgnoreResult(result.Results[i], config.IgnoredItems) {
			continue
		}

		results = append(results, &result.Results[i])

		switch result.Results[i].Rule.Severity {
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
