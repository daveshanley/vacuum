// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	doctorV3 "github.com/pb33f/doctor/model/high/v3"
	press "github.com/pb33f/doctor/printingpress"
	ppconfig "github.com/pb33f/doctor/printingpress/config"
	ppmodel "github.com/pb33f/doctor/printingpress/model"
	ppserve "github.com/pb33f/doctor/printingpress/serve"
	"github.com/pb33f/doctor/terminal"
)

type docsTerminal struct {
	stdout  io.Writer
	stderr  io.Writer
	palette terminal.Palette
	mode    terminal.ActivityRenderMode
	logger  *terminal.BuildLoggerSession
}

func newDocsTerminal(stdout, stderr io.Writer, debug bool) *docsTerminal {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	palette := terminal.PaletteForTheme(terminal.ThemeDark)
	mode := terminal.SelectActivityRenderMode(stderr, debug)
	return &docsTerminal{
		stdout:  stdout,
		stderr:  stderr,
		palette: palette,
		mode:    mode,
		logger:  terminal.ConfigureBuildLogger(stderr, palette, mode),
	}
}

func (t *docsTerminal) finish(err error) {
	if t == nil || t.logger == nil {
		return
	}
	t.logger.Finish(err)
}

func (t *docsTerminal) activityRenderer(totalStages int) terminal.ActivityRenderer {
	if t == nil {
		return terminal.NewActivityRenderer(terminal.ActivityRenderModePlain, os.Stderr, terminal.PaletteForTheme(terminal.ThemeDark), totalStages, nil)
	}
	var logger *slog.Logger
	if t.logger != nil {
		logger = t.logger.Logger
	}
	return terminal.NewActivityRenderer(t.mode, t.stderr, t.palette, totalStages, logger)
}

func buildDocsOutputStageCount(opts *docsOptions, diagnostics bool) int {
	if opts == nil {
		return 1
	}
	return terminal.BuildStageCount(terminal.OutputSelection{
		HTML:        !opts.noHTML,
		LLM:         !opts.noLLM,
		JSON:        !opts.noJSON,
		Diagnostics: diagnostics,
	})
}

func runDocsSingle(source *docsSource, opts *docsOptions, diagnostics *docsDiagnosticsContext, term *docsTerminal) error {
	buildStart := time.Now()
	renderer := term.activityRenderer(buildDocsOutputStageCount(opts, diagnostics != nil && diagnostics.enabled))
	defer renderer.Close()

	outputDir, err := normalizeDocsOutputDir(opts.outputDir)
	if err != nil {
		return err
	}

	var lintResults []*doctorRuleResult
	if diagnostics != nil && diagnostics.enabled {
		diagnosticsStart := time.Now()
		renderer.UpdateManual("diagnostics", "linting diagnostics", "running", 0.05, 0, nil)
		lintResults, err = diagnostics.lintSpec(source.specBytes, source.specPath)
		if err != nil {
			renderer.UpdateManual("diagnostics", "diagnostics lint failed", "failed", 0, time.Since(diagnosticsStart), err)
			return fmt.Errorf("diagnostics lint failed: %w", err)
		}
		renderer.UpdateManual("diagnostics", fmt.Sprintf("linted %d diagnostics", len(lintResults)), "completed", 1, time.Since(diagnosticsStart), nil)
	}

	pp, err := press.CreatePrintingPressFromBytes(source.specBytes, &press.PrintingPressConfig{
		Title:                              opts.title,
		BaseURL:                            opts.baseURL,
		BasePath:                           source.basePath,
		SpecPath:                           source.specPath,
		SpecURL:                            source.specURL,
		OutputDir:                          outputDir,
		AssetMode:                          docsAssetMode(opts),
		DeveloperMode:                      diagnostics != nil && diagnostics.enabled,
		ArchiveExportURL:                   docsArchiveExportURLForServe(opts),
		LintResults:                        lintResults,
		Footer:                             buildDocsFooterConfig(opts),
		MaxPatternRepeatBudget:             opts.maxPatternRepeatBudget,
		MaxGeneratedStringBytes:            opts.maxGeneratedStringBytes,
		MaxGeneratedMockBytes:              opts.maxGeneratedMockBytes,
		LLMAggregateSpecSizeThresholdBytes: opts.llmAggregateSpecSizeThresholdBytes,
		LLMMaxAggregateFileBytes:           opts.llmMaxAggregateFileBytes,
		LLMGenerateMonoliths:               opts.llmGenerateMonoliths,
	})
	if err != nil {
		return fmt.Errorf("unable to create printing press: %w", err)
	}

	var htmlStats *press.PressStatistics
	var llmStats *press.PressStatistics
	var site *ppmodel.Site
	if !opts.noHTML {
		htmlStats, err = terminal.RunWithActivity(pp, renderer, pp.PrintHTML)
		if err != nil {
			return fmt.Errorf("html render failed: %w", err)
		}
	}
	if !opts.noLLM {
		llmStats, err = terminal.RunWithActivity(pp, renderer, pp.PrintLLM)
		if err != nil {
			return fmt.Errorf("llm render failed: %w", err)
		}
	}
	if !opts.noJSON {
		if htmlStats == nil && llmStats == nil {
			site, err = terminal.RunWithActivity(pp, renderer, pp.PressModel)
			if err != nil {
				return fmt.Errorf("model build failed: %w", err)
			}
		} else {
			site, err = pp.PressModel()
			if err != nil {
				return fmt.Errorf("model build failed: %w", err)
			}
		}
		jsonStart := time.Now()
		renderer.UpdateManual("json", "writing json artifacts", "running", 0.2, 0, nil)
		if err := press.PrintJSONArtifacts(site, ""); err != nil {
			renderer.UpdateManual("json", "json artifact write failed", "failed", 0, time.Since(jsonStart), err)
			return fmt.Errorf("json artifact write failed: %w", err)
		}
		renderer.UpdateManual("json", "json artifacts complete", "completed", 1, time.Since(jsonStart), nil)
	}
	if site == nil {
		site, err = pp.PressModel()
		if err != nil {
			return fmt.Errorf("model build failed: %w", err)
		}
	}

	fileCount, totalBytes, err := terminal.ScanOutputDir(site.OutputDir)
	if err != nil {
		return fmt.Errorf("unable to scan output directory: %w", err)
	}
	renderer.Close()
	terminal.PrintSummary(term.stdout, term.palette, site, htmlStats, llmStats, time.Since(buildStart), fileCount, totalBytes)
	if opts.serve {
		return serveDocsSingle(source, opts, site, lintResults)
	}
	return nil
}

func runDocsAggregate(scanRoot string, opts *docsOptions, fileConfig *ppconfig.File, diagnostics *docsDiagnosticsContext, term *docsTerminal) error {
	buildStart := time.Now()
	scanStages := 1
	if diagnostics != nil && diagnostics.enabled {
		scanStages = 2
	}
	scanRenderer := term.activityRenderer(scanStages)
	defer scanRenderer.Close()

	outputDir, err := normalizeDocsAggregateOutputDir(opts.outputDir)
	if err != nil {
		return err
	}
	cfg := buildDocsAggregateConfig(scanRoot, outputDir, docsAssetMode(opts), opts, fileConfig)
	if diagnostics != nil {
		cfg.EntryConfigFingerprint = diagnostics.fingerprint
	}

	ap, err := press.CreateAggregatePrintingPressFromPath(scanRoot, cfg)
	if err != nil {
		return fmt.Errorf("unable to create aggregate printing press: %w", err)
	}
	catalog, err := runDocsAggregateCatalogStage(scanRenderer, ap)
	if err != nil {
		return fmt.Errorf("catalog discovery failed: %w", err)
	}

	var specLintResults map[string][]*doctorRuleResult
	if diagnostics != nil && diagnostics.enabled {
		specLintResults, err = runDocsAggregateDiagnosticsStage(scanRenderer, diagnostics, catalog)
		if err != nil {
			return fmt.Errorf("catalog diagnostics failed: %w", err)
		}
	}
	scanRenderer.Close()

	poolLogger := slog.Default()
	if term != nil && term.logger != nil && term.logger.Logger != nil {
		poolLogger = term.logger.Logger
	}
	poolRenderer := terminal.NewAggregatePoolRenderer(term.mode, term.stderr, term.palette, poolLogger)
	defer poolRenderer.Close()
	var metricsMonitor *terminal.RuntimeMetricsMonitor
	if opts.metrics {
		metricsMonitor = terminal.StartRuntimeMetricsMonitor(buildStart, terminal.DefaultRuntimeMetricsInterval, poolRenderer.ReportRuntimeMetrics)
		defer metricsMonitor.Close()
	}

	stats, err := ap.PrintSelectedOutputs(press.AggregateRenderOptions{
		HTML: !opts.noHTML,
		LLM:  !opts.noLLM,
		JSON: !opts.noJSON,
		ProgressReporter: press.AggregateProgressReporterFunc(func(update press.AggregateProgressUpdate) {
			poolRenderer.Report(update)
		}),
		DeveloperMode:   diagnostics != nil && diagnostics.enabled,
		SpecLintResults: specLintResults,
	})
	if err != nil {
		return fmt.Errorf("aggregate render failed: %w", err)
	}
	if metricsMonitor != nil {
		metricsMonitor.Close()
	}

	fileCount, totalBytes, err := terminal.ScanOutputDir(catalog.OutputDir)
	if err != nil {
		return fmt.Errorf("unable to scan output directory: %w", err)
	}
	poolRenderer.Close()
	terminal.PrintAggregateSummary(term.stdout, term.palette, catalog, stats, nil, nil, time.Since(buildStart), fileCount, totalBytes)
	if opts.serve {
		return serveDocsDirectory(opts, catalog.OutputDir, catalog.BaseURL)
	}
	return nil
}

type doctorRuleResult = doctorV3.RuleFunctionResult

func docsAssetMode(opts *docsOptions) string {
	if opts != nil && (opts.publish || opts.serve) {
		return press.HTMLAssetModeServed
	}
	return press.HTMLAssetModePortable
}

func docsArchiveExportURLForServe(opts *docsOptions) string {
	if opts == nil || !opts.serve || opts.disableExport {
		return ""
	}
	return ppserve.ArchiveExportPathForBaseURL(opts.baseURL)
}

func serveDocsSingle(source *docsSource, opts *docsOptions, site *ppmodel.Site, lintResults []*doctorRuleResult) error {
	serveOpts := ppserve.Config{
		Dir:           site.OutputDir,
		BaseURL:       site.BaseURL,
		DisableExport: opts.disableExport,
	}
	if !opts.disableExport {
		archiveDirs, err := ppserve.RenderArchiveVariants(ppserve.ArchiveRenderOptions{
			Title:                              opts.title,
			BasePath:                           source.basePath,
			SpecPath:                           source.specPath,
			SpecURL:                            source.specURL,
			SpecBytes:                          source.specBytes,
			LintResults:                        lintResults,
			Footer:                             buildDocsFooterConfig(opts),
			MaxPatternRepeatBudget:             opts.maxPatternRepeatBudget,
			MaxGeneratedStringBytes:            opts.maxGeneratedStringBytes,
			MaxGeneratedMockBytes:              opts.maxGeneratedMockBytes,
			LLMAggregateSpecSizeThresholdBytes: opts.llmAggregateSpecSizeThresholdBytes,
			LLMMaxAggregateFileBytes:           opts.llmMaxAggregateFileBytes,
			LLMGenerateMonoliths:               opts.llmGenerateMonoliths,
			IncludeLLM:                         !opts.noLLM,
			NoHTML:                             opts.noHTML,
		})
		if err != nil {
			return fmt.Errorf("unable to render served archive export: %w", err)
		}
		defer archiveDirs.Cleanup()
		if archiveDirs != nil {
			serveOpts.ArchiveDir = archiveDirs.Plain
			serveOpts.DiagnosticsArchiveDir = archiveDirs.Diagnostics
			serveOpts.LLMArchiveDir = archiveDirs.LLM
			serveOpts.DiagnosticsLLMArchiveDir = archiveDirs.DiagnosticsLLM
		}
	}
	return serveDocsDirectoryWithConfig(opts, serveOpts)
}

func serveDocsDirectory(opts *docsOptions, outputDir, baseURL string) error {
	return serveDocsDirectoryWithConfig(opts, ppserve.Config{
		Dir:           outputDir,
		BaseURL:       baseURL,
		DisableExport: opts.disableExport,
	})
}

func serveDocsDirectoryWithConfig(opts *docsOptions, serveOpts ppserve.Config) error {
	fmt.Printf("serving http://127.0.0.1:%d from %s\n", opts.port, serveOpts.Dir)
	return serveOpts.ListenAndServe(context.Background(), fmt.Sprintf(":%d", opts.port))
}

func runDocsAggregateCatalogStage(renderer terminal.ActivityRenderer, ap *press.AggregatePrintingPress) (*ppmodel.CatalogSite, error) {
	start := time.Now()
	renderer.UpdateManual("scan", "discovering specs", "running", 0.05, 0, nil)
	catalog, err := ap.PressModel()
	if err != nil {
		renderer.UpdateManual("scan", "spec discovery failed", "failed", 0, time.Since(start), err)
		return nil, err
	}
	renderer.UpdateManual("scan", fmt.Sprintf("discovered %d services across %d specs", len(catalog.Services), countDocsCatalogSpecs(catalog)), "completed", 1, time.Since(start), nil)
	return catalog, nil
}

func runDocsAggregateDiagnosticsStage(renderer terminal.ActivityRenderer, diagnostics *docsDiagnosticsContext, catalog *ppmodel.CatalogSite) (map[string][]*doctorRuleResult, error) {
	start := time.Now()
	total := len(docsCatalogLintJobs(catalog))
	if total == 0 {
		renderer.UpdateManual("diagnostics", "no specs to lint", "completed", 1, 0, nil)
		return map[string][]*doctorRuleResult{}, nil
	}
	renderer.UpdateManual("diagnostics", fmt.Sprintf("linting %d specs", total), "running", 0.05, 0, nil)
	results, err := diagnostics.lintCatalog(catalog, func(completed, total int, currentSpec string, elapsed time.Duration) {
		if completed == 0 {
			return
		}
		task := fmt.Sprintf("linted %d/%d specs", completed, total)
		if strings.TrimSpace(currentSpec) != "" {
			task = task + " · " + currentSpec
		}
		renderer.UpdateManual("diagnostics", task, "running", float64(completed)/float64(total), elapsed, nil)
	})
	if err != nil {
		renderer.UpdateManual("diagnostics", "diagnostics lint failed", "failed", 0, time.Since(start), err)
		return nil, err
	}
	renderer.UpdateManual("diagnostics", fmt.Sprintf("linted diagnostics for %d specs", total), "completed", 1, time.Since(start), nil)
	return results, nil
}

func countDocsCatalogSpecs(catalog *ppmodel.CatalogSite) int {
	if catalog == nil {
		return 0
	}
	total := 0
	for _, service := range catalog.Services {
		if service == nil {
			continue
		}
		total += service.SpecCount
	}
	return total
}

func buildDocsAggregateConfig(scanRoot, outputDir, assetMode string, opts *docsOptions, fileConfig *ppconfig.File) *press.AggregatePrintingPressConfig {
	catalogTitle := opts.title
	if override := strings.TrimSpace(opts.catalogTitle); override != "" {
		catalogTitle = override
	}
	cfg := &press.AggregatePrintingPressConfig{
		Title:                              catalogTitle,
		Description:                        opts.description,
		ScanRoot:                           scanRoot,
		OutputDir:                          outputDir,
		BaseURL:                            opts.baseURL,
		AssetMode:                          assetMode,
		BuildMode:                          opts.buildMode,
		MaxPools:                           opts.maxPools,
		WorkersPerPool:                     opts.workersPerPool,
		MaxPatternRepeatBudget:             opts.maxPatternRepeatBudget,
		MaxGeneratedStringBytes:            opts.maxGeneratedStringBytes,
		MaxGeneratedMockBytes:              opts.maxGeneratedMockBytes,
		LLMAggregateSpecSizeThresholdBytes: opts.llmAggregateSpecSizeThresholdBytes,
		LLMMaxAggregateFileBytes:           opts.llmMaxAggregateFileBytes,
		LLMGenerateMonoliths:               opts.llmGenerateMonoliths,
		DisableSkippedRendering:            opts.disableSkippedRendering,
		Footer:                             buildDocsFooterConfig(opts),
	}
	if fileConfig == nil {
		return cfg
	}

	cfg.Include = append([]string(nil), fileConfig.Scan.Include...)
	cfg.IgnoreRules = append([]string(nil), fileConfig.Scan.IgnoreRules...)
	cfg.NoiseSegments = append([]string(nil), fileConfig.Grouping.NoiseSegments...)
	cfg.ServiceOverrides = toDocsAggregateOverrides(fileConfig.Grouping.ServiceOverrides)
	cfg.DisplayNameOverrides = toDocsAggregateOverrides(fileConfig.Grouping.DisplayNameOverrides)
	cfg.VersionOverrides = toDocsAggregateOverrides(fileConfig.Grouping.VersionOverrides)
	cfg.StateNamespace = fileConfig.State.Namespace
	cfg.StateSQLitePath = fileConfig.State.SQLite.Path
	if cfg.MaxPools == 0 {
		cfg.MaxPools = fileConfig.Build.MaxPools
	}
	if cfg.WorkersPerPool == 0 {
		cfg.WorkersPerPool = fileConfig.Build.WorkersPerPool
	}
	if !cfg.DisableSkippedRendering {
		cfg.DisableSkippedRendering = fileConfig.Build.DisableSkippedRendering
	}
	return cfg
}

func toDocsAggregateOverrides(configs []ppconfig.PathOverride) []press.AggregatePathOverride {
	overrides := make([]press.AggregatePathOverride, 0, len(configs))
	for _, override := range configs {
		if override.Pattern == "" || override.Value == "" {
			continue
		}
		overrides = append(overrides, press.AggregatePathOverride{
			Pattern: override.Pattern,
			Value:   override.Value,
		})
	}
	return overrides
}

func buildDocsFooterConfig(opts *docsOptions) *ppmodel.FooterConfig {
	if opts == nil {
		return nil
	}
	footerURL := strings.TrimSpace(opts.footerURL)
	footerLinkTitle := strings.TrimSpace(opts.footerLinkTitle)
	footerContent := strings.TrimSpace(opts.footerContent)
	if !opts.noFooter && footerURL == "" && footerLinkTitle == "" && footerContent == "" {
		return nil
	}
	return &ppmodel.FooterConfig{
		Disabled:  opts.noFooter,
		URL:       footerURL,
		LinkTitle: footerLinkTitle,
		Build:     footerContent,
	}
}

func normalizeDocsOutputDir(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}
	return abs, nil
}

func normalizeDocsAggregateOutputDir(raw string) (string, error) {
	if raw != "" {
		return normalizeDocsOutputDir(raw)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return filepath.Join(cwd, "api-docs"), nil
}
