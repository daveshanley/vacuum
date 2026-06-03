// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	asyncapi_context "github.com/daveshanley/vacuum/asyncapi"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	ppconfig "github.com/pb33f/doctor/printingpress/config"
	"github.com/spf13/cobra"
)

type docsInputMode int

const (
	docsInputSingle docsInputMode = iota
	docsInputAggregate
)

type docsSource struct {
	specBytes []byte
	basePath  string
	specPath  string
	specURL   string
}

// GetDocsCommand returns the OpenAPI documentation generation command.
func GetDocsCommand() *cobra.Command {
	opts := &docsOptions{
		port: 9090,
	}

	cmd := &cobra.Command{
		Use:           "docs <openapi-file-url-or-directory>",
		Short:         "Generate Agentic AI and Human OpenAPI documentation via the printing press",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := ""
			if len(args) > 0 {
				input = args[0]
			}
			err := runDocs(cmd, input, opts)
			if err != nil {
				tui.RenderError(err)
			}
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", "", "Output directory for rendered docs")
	cmd.Flags().StringVar(&opts.title, "title", "", "Override the API title")
	cmd.Flags().StringVar(&opts.catalogTitle, "catalog-title", "", "Override the API catalog title")
	cmd.Flags().StringVar(&opts.docsConfigPath, "docs-config", "", "Path to a printing press docs config file")
	cmd.Flags().StringVar(&opts.baseURL, "base-url", "", "Base URL to use in generated HTML")
	cmd.Flags().StringVar(&opts.basePath, "base-path", "", "Base path for resolving local file references")
	cmd.Flags().StringVar(&opts.buildMode, "build-mode", "", "Aggregate build mode: full, fast, or watch")
	cmd.Flags().IntVar(&opts.maxPools, "max-pools", 0, "Aggregate max concurrent render pools")
	cmd.Flags().IntVar(&opts.workersPerPool, "workers-per-pool", 0, "Aggregate core budget per render pool")
	cmd.Flags().IntVar(&opts.maxPatternRepeatBudget, "max-pattern-repeat-budget", 0, "Maximum regex repeat budget for generated mock strings")
	cmd.Flags().IntVar(&opts.maxGeneratedStringBytes, "max-generated-string-bytes", 0, "Maximum bytes for each generated mock string")
	cmd.Flags().IntVar(&opts.maxGeneratedMockBytes, "max-generated-mock-bytes", 0, "Maximum bytes for each serialized generated mock payload")
	cmd.Flags().Int64Var(&opts.llmAggregateSpecSizeThresholdBytes, "llm-aggregate-spec-size-threshold-bytes", 0, "Root spec byte threshold for generating monolithic LLM aggregate files")
	cmd.Flags().Int64Var(&opts.llmMaxAggregateFileBytes, "llm-max-aggregate-file-bytes", 0, "Target maximum bytes for each sharded LLM aggregate file")
	cmd.Flags().StringVar(&opts.llmGenerateMonoliths, "llm-generate-monoliths", "", "LLM monolithic aggregate mode: auto, always, or never")
	cmd.Flags().BoolVar(&opts.disableSkippedRendering, "disable-skipped-rendering", false, "Hide skipped-render warnings from aggregate catalog pages")
	cmd.Flags().StringVar(&opts.footerURL, "footer-url", "", "Footer link URL for generated HTML")
	cmd.Flags().StringVar(&opts.footerLinkTitle, "footer-link-title", "", "Footer link text/title for generated HTML")
	cmd.Flags().StringVar(&opts.footerContent, "footer-content", "", "Footer trailing content text for generated HTML")
	cmd.Flags().BoolVarP(&opts.noLogo, "no-logo", "b", false, "Disable the vacuum banner")
	cmd.Flags().BoolVar(&opts.noFooter, "no-footer", false, "Disable the generated HTML footer")
	cmd.Flags().BoolVar(&opts.disableExport, "disable-export", false, "Disable local preview archive export controls")
	cmd.Flags().BoolVar(&opts.noHTML, "no-html", false, "Skip HTML output")
	cmd.Flags().BoolVar(&opts.noLLM, "no-llm", false, "Skip LLM output")
	cmd.Flags().BoolVar(&opts.noJSON, "no-json", false, "Skip JSON artifact output")
	cmd.Flags().BoolVar(&opts.noDiagnostics, "no-diagnostics", false, "Skip lint diagnostics in generated docs")
	cmd.Flags().BoolVar(&opts.publish, "publish", false, "Build hosted/served HTML assets without starting a local server")
	cmd.Flags().BoolVar(&opts.serve, "serve", false, "Serve the rendered output after building")
	cmd.Flags().BoolVar(&opts.metrics, "metrics", false, "Show live aggregate runtime metrics while rendering")
	cmd.Flags().IntVar(&opts.port, "port", 9090, "Port to use with --serve")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")

	return cmd
}

func runDocs(cmd *cobra.Command, input string, opts *docsOptions) (err error) {
	fileConfig, err := loadDocsConfig(opts.docsConfigPath, input)
	if err != nil {
		return fmt.Errorf("unable to load docs config: %w", err)
	}
	applyDocsConfigToOptions(cmd, opts, fileConfig)
	renderDocsBanner(opts)

	resolvedInput, err := resolveDocsInput(input, fileConfig)
	if err != nil {
		return err
	}

	lintFlags := docsLintFlags(cmd)
	term := newDocsTerminal(cmd.OutOrStdout(), cmd.ErrOrStderr(), lintFlags.DebugFlag)
	defer func() {
		term.finish(err)
	}()

	if opts.noHTML && opts.noLLM && opts.noJSON {
		return fmt.Errorf("all output types are disabled; leave at least one of HTML, LLM, or JSON enabled")
	}

	httpClient, httpClientConfig, err := docsHTTPClient(lintFlags)
	if err != nil {
		return err
	}

	mode, inputPath, err := detectDocsInputMode(resolvedInput)
	if err != nil {
		return err
	}

	if mode == docsInputAggregate {
		if err := rejectAsyncAPIForDocsAggregate(inputPath); err != nil {
			return err
		}
		fetchConfig, err := GetFetchConfig(lintFlags)
		if err != nil {
			return err
		}
		diagnostics, err := newDocsDiagnosticsContext(lintFlags, httpClientConfig, fetchConfig, !opts.noDiagnostics)
		if err != nil {
			return err
		}
		return runDocsAggregate(inputPath, opts, fileConfig, diagnostics, term)
	}

	source, err := loadDocsSource(inputPath, opts, httpClient)
	if err != nil {
		return err
	}
	if err := rejectAsyncAPIForOpenAPICommand("docs", source.specBytes); err != nil {
		return err
	}
	fetchConfig, err := GetFetchConfig(lintFlags)
	if err != nil {
		return err
	}
	diagnostics, err := newDocsDiagnosticsContext(lintFlags, httpClientConfig, fetchConfig, !opts.noDiagnostics)
	if err != nil {
		return err
	}
	return runDocsSingle(source, opts, diagnostics, term)
}

func renderDocsBanner(opts *docsOptions) {
	flags := &LintFlags{}
	if opts != nil {
		flags.NoBannerFlag = opts.noLogo
	}
	SetupVacuumEnvironment(flags)
}

func resolveDocsInput(input string, fileConfig *ppconfig.File) (string, error) {
	if strings.TrimSpace(input) != "" {
		return input, nil
	}
	if fileConfig != nil && strings.TrimSpace(fileConfig.Scan.Root) != "" {
		return strings.TrimSpace(fileConfig.Scan.Root), nil
	}
	return "", fmt.Errorf(`Supply an OpenAPI spec path, URL, or directory to generate the most fly, modern and agentic ready API documentation you have ever seen.
usage:
  vacuum docs ./openapi.yaml
  vacuum docs https://example.com/openapi.yaml
  vacuum docs ./apis --output ./api-docs
  vacuum docs ./openapi.yaml --serve --port 9090
hint: use --docs-config printing-press.yaml to load printing press docs settings`)
}

func docsLintFlags(cmd *cobra.Command) *LintFlags {
	flags := ReadLintFlags(cmd)
	flags.SilentFlag = true
	flags.NoStyleFlag = true
	flags.PipelineOutput = true
	return flags
}

func docsHTTPClient(flags *LintFlags) (*http.Client, utils.HTTPClientConfig, error) {
	httpClientConfig, err := GetHTTPClientConfig(flags)
	if err != nil {
		return nil, utils.HTTPClientConfig{}, err
	}
	httpClient, err := utils.CreateHTTPClientIfNeeded(httpClientConfig)
	if err != nil {
		return nil, utils.HTTPClientConfig{}, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	return httpClient, httpClientConfig, nil
}

func detectDocsInputMode(input string) (docsInputMode, string, error) {
	if isDocsRemoteInput(input) {
		return docsInputSingle, input, nil
	}
	absPath, err := filepath.Abs(input)
	if err != nil {
		return docsInputSingle, input, fmt.Errorf("resolve input path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return docsInputSingle, absPath, fmt.Errorf("inspect input path: %w", err)
	}
	if info.IsDir() {
		return docsInputAggregate, absPath, nil
	}
	return docsInputSingle, absPath, nil
}

func isDocsRemoteInput(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func loadDocsSource(input string, opts *docsOptions, httpClient *http.Client) (*docsSource, error) {
	reportOrSpec, err := LoadFileAsReportOrSpecWithClient(input, httpClient)
	if err != nil {
		return nil, fmt.Errorf("unable to load specification input: %w", err)
	}
	if len(reportOrSpec.SpecBytes) == 0 {
		return nil, fmt.Errorf("specification input did not contain OpenAPI bytes")
	}

	if isDocsRemoteInput(input) {
		basePath, err := normalizeDocsBasePath(opts.basePath)
		if err != nil {
			return nil, err
		}
		return &docsSource{
			specBytes: reportOrSpec.SpecBytes,
			basePath:  basePath,
			specPath:  input,
			specURL:   input,
		}, nil
	}

	basePath := opts.basePath
	if basePath == "" {
		basePath = filepath.Dir(input)
	}
	normalizedBasePath, err := normalizeDocsBasePath(basePath)
	if err != nil {
		return nil, err
	}
	return &docsSource{
		specBytes: reportOrSpec.SpecBytes,
		basePath:  normalizedBasePath,
		specPath:  input,
	}, nil
}

func normalizeDocsBasePath(basePath string) (string, error) {
	if strings.TrimSpace(basePath) == "" {
		return "", nil
	}
	if strings.Contains(basePath, "://") {
		return basePath, nil
	}
	abs, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("resolve base path: %w", err)
	}
	return abs, nil
}

var errDocsAggregateAsyncAPI = errors.New("asyncapi document found in docs aggregate input")

func rejectAsyncAPIForDocsAggregate(root string) error {
	var asyncPath string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isDocsSpecCandidate(path) {
			return nil
		}
		specBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		isAsyncAPI, detectErr := asyncapi_context.IsDocument(specBytes)
		if detectErr != nil && !asyncapi_context.HasMarker(specBytes) {
			return nil
		}
		if detectErr == nil && !isAsyncAPI {
			return nil
		}
		asyncPath = path
		return errDocsAggregateAsyncAPI
	})
	if errors.Is(err, errDocsAggregateAsyncAPI) {
		return fmt.Errorf("`vacuum docs` only supports OpenAPI documents; AsyncAPI document found in aggregate input: %s", asyncPath)
	}
	return err
}

func isDocsSpecCandidate(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
