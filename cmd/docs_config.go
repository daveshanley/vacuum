// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"

	ppconfig "github.com/pb33f/doctor/printingpress/config"
	"github.com/spf13/cobra"
)

type docsOptions struct {
	outputDir                          string
	title                              string
	catalogTitle                       string
	description                        string
	baseURL                            string
	basePath                           string
	docsConfigPath                     string
	buildMode                          string
	maxPools                           int
	workersPerPool                     int
	maxPatternRepeatBudget             int
	maxGeneratedStringBytes            int
	maxGeneratedMockBytes              int
	llmAggregateSpecSizeThresholdBytes int64
	llmMaxAggregateFileBytes           int64
	llmGenerateMonoliths               string
	disableSkippedRendering            bool
	footerURL                          string
	footerLinkTitle                    string
	footerContent                      string
	noLogo                             bool
	noFooter                           bool
	disableExport                      bool
	noHTML                             bool
	noLLM                              bool
	noJSON                             bool
	noDiagnostics                      bool
	publish                            bool
	serve                              bool
	metrics                            bool
	port                               int
}

func loadDocsConfig(configPath, inputArg string) (*ppconfig.File, error) {
	return ppconfig.Load(configPath, inputArg)
}

func applyDocsConfigToOptions(cmd *cobra.Command, opts *docsOptions, fileConfig *ppconfig.File) {
	if cmd == nil || opts == nil || fileConfig == nil {
		return
	}

	applyDocsStringFlag(cmd, "output", &opts.outputDir, fileConfig.Output)
	applyDocsStringFlag(cmd, "title", &opts.title, fileConfig.Title)
	applyDocsStringFlag(cmd, "base-url", &opts.baseURL, fileConfig.BaseURL)
	applyDocsStringFlag(cmd, "base-path", &opts.basePath, fileConfig.BasePath)
	applyDocsStringFlag(cmd, "build-mode", &opts.buildMode, fileConfig.Build.Mode)
	applyDocsIntFlag(cmd, "max-pools", &opts.maxPools, fileConfig.Build.MaxPools)
	applyDocsIntFlag(cmd, "workers-per-pool", &opts.workersPerPool, fileConfig.Build.WorkersPerPool)
	applyDocsIntFlag(cmd, "max-pattern-repeat-budget", &opts.maxPatternRepeatBudget, fileConfig.Build.MaxPatternRepeatBudget)
	applyDocsIntFlag(cmd, "max-generated-string-bytes", &opts.maxGeneratedStringBytes, fileConfig.Build.MaxGeneratedStringBytes)
	applyDocsIntFlag(cmd, "max-generated-mock-bytes", &opts.maxGeneratedMockBytes, fileConfig.Build.MaxGeneratedMockBytes)
	applyDocsInt64Flag(cmd, "llm-aggregate-spec-size-threshold-bytes", &opts.llmAggregateSpecSizeThresholdBytes, fileConfig.Build.LLMAggregateSpecSizeThresholdBytes)
	applyDocsInt64Flag(cmd, "llm-max-aggregate-file-bytes", &opts.llmMaxAggregateFileBytes, fileConfig.Build.LLMMaxAggregateFileBytes)
	applyDocsStringFlag(cmd, "llm-generate-monoliths", &opts.llmGenerateMonoliths, fileConfig.Build.LLMGenerateMonoliths)
	applyDocsBoolFlag(cmd, "disable-skipped-rendering", &opts.disableSkippedRendering, fileConfig.Build.DisableSkippedRendering)
	applyDocsStringFlag(cmd, "footer-url", &opts.footerURL, fileConfig.Footer.URL)
	applyDocsStringFlag(cmd, "footer-link-title", &opts.footerLinkTitle, fileConfig.Footer.LinkTitle)
	applyDocsStringFlag(cmd, "footer-content", &opts.footerContent, fileConfig.Footer.Content)
	applyDocsBoolFlag(cmd, "no-logo", &opts.noLogo, fileConfig.NoLogo)
	applyDocsBoolFlag(cmd, "disable-export", &opts.disableExport, fileConfig.DisableExport)
	applyDocsBoolFlag(cmd, "no-html", &opts.noHTML, fileConfig.NoHTML)
	applyDocsBoolFlag(cmd, "no-llm", &opts.noLLM, fileConfig.NoLLM)
	applyDocsBoolFlag(cmd, "no-json", &opts.noJSON, fileConfig.NoJSON)
	applyDocsBoolFlag(cmd, "publish", &opts.publish, fileConfig.Publish)
	applyDocsBoolFlag(cmd, "serve", &opts.serve, fileConfig.Serve)
	applyDocsBoolFlag(cmd, "metrics", &opts.metrics, fileConfig.Metrics)
	applyDocsIntFlag(cmd, "port", &opts.port, fileConfig.Port)

	opts.description = strings.TrimSpace(fileConfig.Description)
	if fileConfig.Footer.Enabled != nil && !cmd.Flags().Changed("no-footer") {
		opts.noFooter = !*fileConfig.Footer.Enabled
	}
}

func applyDocsStringFlag(cmd *cobra.Command, name string, dest *string, value string) {
	if dest == nil || strings.TrimSpace(value) == "" || cmd.Flags().Changed(name) {
		return
	}
	*dest = value
}

func applyDocsBoolFlag(cmd *cobra.Command, name string, dest *bool, value bool) {
	if dest == nil || !value || cmd.Flags().Changed(name) {
		return
	}
	*dest = true
}

func applyDocsIntFlag(cmd *cobra.Command, name string, dest *int, value int) {
	if dest == nil || value == 0 || cmd.Flags().Changed(name) {
		return
	}
	*dest = value
}

func applyDocsInt64Flag(cmd *cobra.Command, name string, dest *int64, value int64) {
	if dest == nil || value == 0 || cmd.Flags().Changed(name) {
		return
	}
	*dest = value
}
