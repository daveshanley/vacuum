// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/color"
	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

const (
	schemaOutputText = "text"
	schemaOutputJSON = "json"

	schemaLintExamples = `Examples:
  vacuum schema my-schema.json
  vacuum schema -d my-schema.yaml`
)

func GetSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "schema <input...>",
		Short:         "Lint JSON Schema documents",
		Long:          "Lint JSON Schema documents, run JSON Schema rulesets, and bundle external schema references.",
		Example:       schemaLintExamples,
		Args:          cobra.ArbitraryArgs,
		RunE:          runSchemaLint,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	addSchemaLintFlags(cmd)

	lintCmd := &cobra.Command{
		Use:           "lint <input...>",
		Short:         "Lint JSON Schema documents",
		Example:       schemaLintExamples,
		Args:          cobra.ArbitraryArgs,
		RunE:          runSchemaLint,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	addSchemaLintFlags(lintCmd)

	cmd.AddCommand(lintCmd)
	cmd.AddCommand(getSchemaBundleCommand())
	return cmd
}

func addSchemaLintFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("details", "d", false, "Show full details of schema linting report")
	cmd.Flags().BoolP("snippets", "s", false, "Show code snippets where issues are found")
	cmd.Flags().BoolP("errors", "e", false, "Show errors only")
	cmd.Flags().BoolP("silent", "x", false, "Show nothing except the result")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output")
	cmd.Flags().BoolP("no-banner", "b", false, "Disable the banner output")
	cmd.Flags().BoolP("no-message", "m", false, "Hide message output when using -d")
	cmd.Flags().BoolP("all-results", "a", false, "Render all results when using -d")
	cmd.Flags().StringP("fail-severity", "n", model.SeverityError, "Results of this level or above will trigger a failure exit code (error, warn, info, hint, none)")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	cmd.Flags().Bool("no-clip", false, "Do not truncate messages or paths")
	cmd.Flags().Bool("show-rules", false, "Show which rules are being used when linting")
	cmd.Flags().StringArray("globbed-files", nil, "Glob pattern of schema files to lint; may be repeated")
	cmd.Flags().StringArray("include", nil, "Include glob for folder inputs; defaults to **/*.json, **/*.yaml, and **/*.yml and may be repeated")
	cmd.Flags().StringArray("exclude", nil, "Exclude glob for folder inputs; may be repeated")
	cmd.Flags().BoolP("stdin", "i", false, "Read a JSON Schema document from stdin")
	cmd.Flags().String("format", schemaOutputText, "Output format: text or json")
	cmd.Flags().StringP("output", "o", "", "Write JSON output to a file")
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")
	cmd.Flags().BoolP("abs-paths", "", false, "If --details(-d) flag is active then output absolute paths")
	cmd.Flags().Bool("bundle", false, "Bundle a single JSON Schema document in memory before linting")
}

func readSchemaLintFlags(cmd *cobra.Command) (*schemaLintFlags, error) {
	flags := &schemaLintFlags{}
	flags.GlobPatterns, _ = cmd.Flags().GetStringArray("globbed-files")
	flags.Includes, _ = cmd.Flags().GetStringArray("include")
	flags.Excludes, _ = cmd.Flags().GetStringArray("exclude")
	flags.Stdin, _ = cmd.Flags().GetBool("stdin")
	flags.Format, _ = cmd.Flags().GetString("format")
	flags.Output, _ = cmd.Flags().GetString("output")
	flags.Details, _ = cmd.Flags().GetBool("details")
	flags.Snippets, _ = cmd.Flags().GetBool("snippets")
	flags.ErrorsOnly, _ = cmd.Flags().GetBool("errors")
	flags.Silent, _ = cmd.Flags().GetBool("silent")
	flags.NoStyle, _ = cmd.Flags().GetBool("no-style")
	flags.NoBanner, _ = cmd.Flags().GetBool("no-banner")
	flags.NoMessage, _ = cmd.Flags().GetBool("no-message")
	flags.AllResults, _ = cmd.Flags().GetBool("all-results")
	flags.NoClip, _ = cmd.Flags().GetBool("no-clip")
	flags.ShowRules, _ = cmd.Flags().GetBool("show-rules")
	flags.FailSeverity, _ = cmd.Flags().GetString("fail-severity")
	flags.IgnoreFile, _ = cmd.Flags().GetString("ignore-file")
	flags.Base, _ = cmd.Flags().GetString("base")
	flags.Remote, _ = cmd.Flags().GetBool("remote")
	flags.Timeout, _ = cmd.Flags().GetInt("timeout")
	flags.LookupTimeout, _ = cmd.Flags().GetInt("lookup-timeout")
	flags.Ruleset, _ = cmd.Flags().GetString("ruleset")
	flags.Functions, _ = cmd.Flags().GetString("functions")
	flags.Time, _ = cmd.Flags().GetBool("time")
	flags.Debug, _ = cmd.Flags().GetBool("debug")
	flags.ExtRefs, _ = cmd.Flags().GetBool("ext-refs")
	flags.CertFile, _ = cmd.Flags().GetString("cert-file")
	flags.KeyFile, _ = cmd.Flags().GetString("key-file")
	flags.CAFile, _ = cmd.Flags().GetString("ca-file")
	flags.Insecure, _ = cmd.Flags().GetBool("insecure")
	flags.AllowPrivateNetworks, _ = cmd.Flags().GetBool("allow-private-networks")
	flags.AllowHTTP, _ = cmd.Flags().GetBool("allow-http")
	flags.FetchTimeout, _ = cmd.Flags().GetInt("fetch-timeout")
	flags.IgnoreArrayCircleRef, _ = cmd.Flags().GetBool("ignore-array-circle-ref")
	flags.IgnorePolyCircleRef, _ = cmd.Flags().GetBool("ignore-polymorph-circle-ref")
	flags.ResolveAllRefs, _ = cmd.Flags().GetBool("resolve-all-refs")
	flags.NestedRefsDocContext, _ = cmd.Flags().GetBool("nested-refs-doc-context")
	flags.OutputAbsPaths, _ = cmd.Flags().GetBool("abs-paths")
	flags.Bundle, _ = cmd.Flags().GetBool("bundle")
	flags.Format = strings.ToLower(strings.TrimSpace(flags.Format))
	if flags.Format == "" {
		flags.Format = schemaOutputText
	}
	if flags.Output != "" && flags.Format == schemaOutputText {
		flags.Format = schemaOutputJSON
	}
	if flags.Format != schemaOutputText && flags.Format != schemaOutputJSON {
		return nil, fmt.Errorf("invalid schema output format %q, expected text or json", flags.Format)
	}
	return flags, nil
}

func runSchemaLint(cmd *cobra.Command, args []string) error {
	flags, err := readSchemaLintFlags(cmd)
	if err != nil {
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	setupSchemaOutput(flags.Silent, flags.NoBanner, flags.NoStyle, flags.Format)

	inputs, err := collectSchemaInputs(cmd, args, flags.GlobPatterns, flags.Includes, flags.Excludes, flags.Stdin, flags.Base, "lint")
	if err != nil {
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	if len(inputs) == 0 {
		err = errors.New("please supply a JSON Schema document to lint\n\n" + schemaLintExamples)
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	if flags.Bundle {
		if len(inputs) != 1 {
			err = errors.New("schema lint --bundle requires exactly one file input or --stdin")
			tui.RenderErrorString("%s", err.Error())
			return err
		}
		if err = ensureSchemaStdinBaseForBundle(inputs[0], flags.Base); err != nil {
			tui.RenderErrorString("%s", err.Error())
			return err
		}
	}

	start := time.Now()
	httpClientConfig, cfgErr := schemaHTTPClientConfig(flags.CertFile, flags.KeyFile, flags.CAFile, flags.Insecure)
	if cfgErr != nil {
		return fmt.Errorf("failed to resolve TLS configuration: %w", cfgErr)
	}
	fetchConfig, fetchErr := schemaFetchConfig(flags, httpClientConfig)
	if fetchErr != nil {
		return fmt.Errorf("failed to resolve fetch configuration: %w", fetchErr)
	}
	selectedRS, err := loadSchemaRuleset(flags, httpClientConfig)
	if err != nil {
		return err
	}
	customFuncs, err := LoadCustomFunctions(flags.Functions, flags.Silent)
	if err != nil {
		return err
	}
	ignoredItems, err := LoadIgnoreFile(flags.IgnoreFile, flags.Silent, false, flags.NoStyle)
	if err != nil {
		return err
	}
	logger, bufferedLogger := createLogger(flags.Debug)

	if flags.ShowRules && !flags.Silent && flags.Format == schemaOutputText {
		renderRulesList(selectedRS.Rules)
	}
	if !flags.Silent && flags.Format == schemaOutputText {
		fmt.Printf(" %slinting %d JSON Schema document(s) against %d rules: %s%s\n\n",
			color.ASCIIBlue, len(inputs), len(selectedRS.Rules), selectedRS.DocumentationURI, color.ASCIIReset)
	}

	var runs []schemaLintRun
	var aggregate []model.RuleFunctionResult
	var firstSpecInfo *datamodel.SpecInfo
	var totalSize int64
	for _, input := range inputs {
		totalSize += int64(len(input.Bytes))
		run, runErr := lintSchemaInput(input, selectedRS, customFuncs, ignoredItems, flags, logger, httpClientConfig, fetchConfig)
		if runErr != nil {
			return runErr
		}
		runs = append(runs, run)
		if firstSpecInfo == nil {
			firstSpecInfo = run.SpecInfo
		}
		for _, res := range run.ResultSet.Results {
			aggregate = append(aggregate, *res)
		}
		if len(run.Errors) > 0 {
			for _, execErr := range run.Errors {
				fmt.Fprintf(cmd.ErrOrStderr(), "Unable to process schema '%s': %s\n", input.Display, execErr.Error())
			}
			return NewInputError("schema linting failed due to %d issues", len(run.Errors))
		}
	}

	if flags.Debug && flags.Format == schemaOutputText {
		RenderBufferedLogs(bufferedLogger, flags.NoStyle)
	}
	resultSet := model.NewRuleResultSet(aggregate)
	resultSet.SortResultsByLineNumber()
	prepareSchemaResults(resultSet)

	if flags.Format == schemaOutputJSON {
		if err := writeSchemaJSONReport(cmd, flags.Output, resultSet, firstSpecInfo, selectedRS); err != nil {
			return err
		}
		return CheckFailureSeverity(flags.FailSeverity, resultSet.GetErrorCount(), resultSet.GetWarnCount(), resultSet.GetInfoCount(), resultSet.GetHintCount())
	}

	if flags.Details && len(resultSet.Results) > 0 {
		for _, run := range runs {
			if len(run.ResultSet.Results) == 0 {
				continue
			}
			renderFixedDetails(RenderDetailsOptions{
				Results:     run.ResultSet.Results,
				SpecData:    strings.Split(string(run.Input.Bytes), "\n"),
				Snippets:    flags.Snippets,
				Errors:      flags.ErrorsOnly,
				Silent:      flags.Silent,
				NoMessage:   flags.NoMessage,
				AllResults:  flags.AllResults,
				NoClip:      flags.NoClip,
				FileName:    run.Input.Display,
				NoStyle:     flags.NoStyle,
				ShowAbsPath: flags.OutputAbsPaths,
			})
		}
	}

	renderFixedSummary(RenderSummaryOptions{
		RuleResultSet:  resultSet,
		RuleCategories: model.RuleCategoriesOrdered,
		Filename:       schemaSummaryName(inputs),
		Silent:         flags.Silent,
		NoStyle:        flags.NoStyle,
		ShowRules:      flags.ShowRules,
	})
	if flags.Time {
		RenderTimeAndFiles(true, time.Since(start), totalSize, len(inputs))
	}
	return CheckFailureSeverity(flags.FailSeverity, resultSet.GetErrorCount(), resultSet.GetWarnCount(), resultSet.GetInfoCount(), resultSet.GetHintCount())
}

func lintSchemaInput(
	input schemaInput,
	selectedRS *rulesets.RuleSet,
	customFuncs map[string]model.RuleFunction,
	ignoredItems model.IgnoredItems,
	flags *schemaLintFlags,
	logger *slog.Logger,
	httpClientConfig utils.HTTPClientConfig,
	fetchConfig *utils.FetchConfig,
) (schemaLintRun, error) {
	run := schemaLintRun{Input: input}
	if flags.Bundle {
		httpClient, clientErr := utils.CreateHTTPClientIfNeeded(httpClientConfig)
		if clientErr != nil {
			return run, fmt.Errorf("failed to create HTTP client: %w", clientErr)
		}
		bundled, warnings, bundleErr := bundleSchemaInput(input, &schemaBundleFlags{
			Delimiter: "__",
			Base:      flags.Base,
			Remote:    flags.Remote,
		}, httpClient)
		for _, warning := range warnings {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
		}
		if bundleErr != nil {
			return run, bundleErr
		}
		rendered, renderErr := renderSchemaBundleOutput(bundled, detectSchemaInputFormat(input.Path, input.Bytes))
		if renderErr != nil {
			return run, renderErr
		}
		input.Bytes = rendered
		run.Input = input
	}
	var root yaml.Node
	if err := yaml.Unmarshal(input.Bytes, &root); err != nil {
		return run, fmt.Errorf("unable to parse JSON Schema '%s': %w", input.Display, err)
	}
	dialect := schemautil.DetectDialect(&root)
	if !schemautil.IsSupportedDialect(dialect.Format) {
		fmt.Fprintf(os.Stderr, "Warning: schema '%s' declares unsupported dialect %q; running generic JSON Schema rules only.\n", input.Display, dialect.URL)
	}
	specPath, err := ResolveSpecPathForExecution(input.Path)
	if err != nil {
		return run, fmt.Errorf("failed to resolve schema path: %w", err)
	}
	execution := &motor.RuleSetExecution{
		RuleSet:                         selectedRS,
		Spec:                            input.Bytes,
		SpecFileName:                    specPath,
		CustomFunctions:                 customFuncs,
		Base:                            input.Base,
		AllowLookup:                     flags.Remote,
		SkipDocumentCheck:               true,
		Logger:                          logger,
		Timeout:                         time.Duration(flags.Timeout) * time.Second,
		NodeLookupTimeout:               time.Duration(flags.LookupTimeout) * time.Millisecond,
		IgnoreCircularArrayRef:          flags.IgnoreArrayCircleRef,
		IgnoreCircularPolymorphicRef:    flags.IgnorePolyCircleRef,
		ExtractReferencesFromExtensions: flags.ExtRefs,
		HTTPClientConfig:                httpClientConfig,
		FetchConfig:                     fetchConfig,
		SpecFormat:                      dialect.Format,
	}
	result := motor.ApplyRulesToRuleSetWithOptions(execution, &motor.ExecutionOptions{
		ResolveAllRefs:       flags.ResolveAllRefs,
		NestedRefsDocContext: flags.NestedRefsDocContext,
	})
	result.Results = utils.FilterIgnoredResultsWithOptions(
		result.Results,
		ignoredItems,
		buildIgnoreFilterOptions(input.Bytes, result, flags.LookupTimeout),
	)
	run.Errors = result.Errors
	run.SpecInfo = result.SpecInfo
	run.ResultSet = model.NewRuleResultSet(result.Results)
	run.ResultSet.SortResultsByLineNumber()
	prepareSchemaResults(run.ResultSet)
	return run, nil
}

func setupSchemaOutput(silent, noBanner, noStyle bool, format string) {
	if format == schemaOutputJSON {
		color.DisableColors()
		return
	}
	if noStyle {
		color.DisableColors()
	}
	if !silent && !noBanner {
		PrintBanner(noStyle)
	}
}

func loadSchemaRuleset(flags *schemaLintFlags, httpClientConfig utils.HTTPClientConfig) (*rulesets.RuleSet, error) {
	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateJSONSchemaRecommendedRuleSet()
	if flags.Ruleset != "" && flags.Ruleset != rulesets.VacuumJSONSchemaRecommended && flags.Ruleset != "json-schema-recommended" {
		httpClient, err := utils.CreateHTTPClientIfNeeded(httpClientConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create custom HTTP client: %w", err)
		}
		rs, rsErr := BuildRuleSetFromUserSuppliedLocation(flags.Ruleset, defaultRuleSets, flags.Remote, httpClient)
		if rsErr != nil {
			tui.RenderErrorString("Unable to load ruleset '%s': %s", flags.Ruleset, rsErr.Error())
			return nil, rsErr
		}
		selectedRS = rs
	}
	return selectedRS, nil
}

func schemaSummaryName(inputs []schemaInput) string {
	if len(inputs) == 1 {
		return inputs[0].Display
	}
	return fmt.Sprintf("%d JSON Schema documents", len(inputs))
}

func writeSchemaJSONReport(cmd *cobra.Command, output string, resultSet *model.RuleResultSet, specInfo *datamodel.SpecInfo, selectedRS *rulesets.RuleSet) error {
	report := vacuum_report.VacuumReport{
		Generated: time.Now(),
		SpecInfo:  specInfo,
		ResultSet: resultSet,
		Rules:     selectedRS.Rules,
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if output == "" {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if err := os.WriteFile(output, raw, 0664); err != nil {
		return err
	}
	return nil
}

func prepareSchemaResults(resultSet *model.RuleResultSet) {
	if resultSet == nil {
		return
	}
	for _, result := range resultSet.Results {
		if result.Rule != nil {
			result.RuleId = result.Rule.Id
			result.RuleSeverity = result.Rule.Severity
		}
		if result.StartNode != nil {
			result.Range.Start = reports.RangeItem{Line: result.StartNode.Line, Char: result.StartNode.Column}
		}
		if result.EndNode != nil {
			result.Range.End = reports.RangeItem{Line: result.EndNode.Line, Char: result.EndNode.Column}
		}
	}
}

func schemaHTTPClientConfig(certFile, keyFile, caFile string, insecure bool) (utils.HTTPClientConfig, error) {
	resolvedCert, err := ResolveConfigPath(certFile)
	if err != nil {
		return utils.HTTPClientConfig{}, err
	}
	resolvedKey, err := ResolveConfigPath(keyFile)
	if err != nil {
		return utils.HTTPClientConfig{}, err
	}
	resolvedCA, err := ResolveConfigPath(caFile)
	if err != nil {
		return utils.HTTPClientConfig{}, err
	}
	return utils.HTTPClientConfig{CertFile: resolvedCert, KeyFile: resolvedKey, CAFile: resolvedCA, Insecure: insecure}, nil
}

func schemaFetchConfig(flags *schemaLintFlags, httpClientConfig utils.HTTPClientConfig) (*utils.FetchConfig, error) {
	if flags.FetchTimeout < 0 {
		return nil, fmt.Errorf("fetch-timeout cannot be negative: %d", flags.FetchTimeout)
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
