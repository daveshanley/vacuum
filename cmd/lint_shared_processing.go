// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
)

// FileProcessingConfig contains all configuration needed to process a file
type FileProcessingConfig struct {
	Flags           *LintFlags
	Logger          *slog.Logger
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

	var logBuffer bytes.Buffer
	var logger *slog.Logger

	if config.Logger != nil {
		logger = config.Logger
	} else {
		charmLogger := log.New(&logBuffer)
		if config.Flags.DebugFlag {
			charmLogger.SetLevel(log.DebugLevel)
		} else {
			charmLogger.SetLevel(log.ErrorLevel)
		}

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
		charmLogger.SetStyles(styles)
		charmLogger.SetReportCaller(false)
		charmLogger.SetReportTimestamp(false)

		logger = slog.New(charmLogger)
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
		if logBuffer.Len() > 0 {
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
	if logBuffer.Len() > 0 {
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
