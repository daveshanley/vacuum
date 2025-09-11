// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"log/slog"
	"os"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
)

// FileProcessingConfig contains all configuration needed to process a file
type FileProcessingConfig struct {
	Flags           *LintFlags
	Logger          *slog.Logger
	BufferedLogger  *BufferedLogger
	SelectedRuleset *rulesets.RuleSet
	CustomFunctions map[string]model.RuleFunction
	IgnoredItems    model.IgnoredItems
}

// ProcessSingleFileOptimized processes a single file using pre-loaded configuration
func ProcessSingleFileOptimized(fileName string, config *FileProcessingConfig) *FileProcessingResult {
	var fileSize int64
	fileInfo, err := os.Stat(fileName)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	var logger *slog.Logger
	var bufferedLogger *BufferedLogger

	if config.Logger != nil {
		logger = config.Logger
		bufferedLogger = config.BufferedLogger
	} else if config.BufferedLogger != nil {
		// Use the provided BufferedLogger
		bufferedLogger = config.BufferedLogger
		handler := NewBufferedLogHandler(bufferedLogger)
		logger = slog.New(handler)
	} else {
		// Create a new BufferedLogger
		bufferedLogger = NewBufferedLogger()
		handler := NewBufferedLogHandler(bufferedLogger)
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
