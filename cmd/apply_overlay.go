// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"github.com/daveshanley/vacuum/color"
	openapitransform "github.com/daveshanley/vacuum/openapi/transform"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
	"github.com/pb33f/libopenapi/overlay"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// WarningsExitCode is the exit code when warnings are present with --fail-on-warnings
const WarningsExitCode = 2

// warningsError is a custom error type for warnings that should exit with code 2
type warningsError struct {
	count int
}

func (e *warningsError) Error() string {
	return fmt.Sprintf("apply-overlay produced %d warning(s) and --fail-on-warnings is set", e.count)
}

func GetApplyOverlayCommand() *cobra.Command {
	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "apply-overlay <spec> [overlay] <output>",
		Short:         "Apply an Overlay to an OpenAPI specification",
		Long: `Apply an OpenAPI Overlay document to modify a specification without changing the original.

The overlay file can be a local file path or a remote URL (http/https).
It can be omitted when filtering operations or pruning unused content.
All references in the overlay are resolved using JSONPath expressions.

The resulting document is written with all Overlay actions applied. Optional
filters then select operations by case-sensitive tags and operation IDs.
All active inclusion filters must match; exclusions take precedence.
The any/all tag mode applies separately to each tag list. Component and tag
pruning runs last. These transforms require
a bundled, self-contained document. YAML aliases and merge keys are expanded
when filtering, preserving mapping order, comments, and styles. Pruning rejects
schema $id scopes. Link Objects targeting filtered operations are retained and
reported as warnings. Pruning removes unused top-level Tag Objects and retains
the parents of used OpenAPI 3.2 tags. JSON input retains JSON output.`,
		Example: `  vacuum apply-overlay openapi.yaml overlay.yaml modified.yaml
  vacuum apply-overlay openapi.yaml public.json --exclude-tag internal --prune-unused
  vacuum apply-overlay openapi.yaml selected.yaml --include-operation-id listPets
  vacuum apply-overlay spec.yaml https://example.com/overlay.yaml output.yaml
  vacuum apply-overlay internal.yaml public.overlay.yaml public.yaml --include-tag public --tag-match any --prune-unused
  cat spec.yaml | vacuum apply-overlay -i overlay.yaml output.yaml
  vacuum apply-overlay spec.yaml overlay.yaml -o > modified.yaml
  cat spec.yaml | vacuum apply-overlay -i overlay.yaml -o > modified.yaml`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) < 3 {
				return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: runApplyOverlay,
	}
	cmd.Flags().BoolP("stdin", "i", false, "Read spec from stdin instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Write output to stdout instead of a file")
	cmd.Flags().BoolP("fail-on-warnings", "W", false, "Treat overlay warnings as errors (exit code 2)")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().StringArray("include-tag", nil, "Keep operations containing this exact tag; may be repeated")
	cmd.Flags().StringArray("exclude-tag", nil, "Remove operations containing this exact tag; may be repeated")
	cmd.Flags().StringArray("include-operation-id", nil, "Keep operations with this exact operationId; may be repeated")
	cmd.Flags().StringArray("exclude-operation-id", nil, "Remove operations with this exact operationId; may be repeated")
	cmd.Flags().String("tag-match", string(openapitransform.MatchAny), "Tag matching mode for each include/exclude list: any or all")
	cmd.Flags().Bool("prune-unused", false, "Remove unused reusable components and top-level tag definitions")
	return cmd
}

func runApplyOverlay(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Read command-specific flags
	stdIn, _ := cmd.Flags().GetBool("stdin")
	stdOut, _ := cmd.Flags().GetBool("stdout")
	failOnWarnings, _ := cmd.Flags().GetBool("fail-on-warnings")
	noStyleFlag, _ := cmd.Flags().GetBool("no-style")
	includeTags, _ := cmd.Flags().GetStringArray("include-tag")
	excludeTags, _ := cmd.Flags().GetStringArray("exclude-tag")
	includeIDs, _ := cmd.Flags().GetStringArray("include-operation-id")
	excludeIDs, _ := cmd.Flags().GetStringArray("exclude-operation-id")
	tagMatch, _ := cmd.Flags().GetString("tag-match")
	pruneUnused, _ := cmd.Flags().GetBool("prune-unused")

	filterOptions := openapitransform.TagFilterOptions{
		IncludeTags: includeTags, ExcludeTags: excludeTags,
		IncludeOperationIDs: includeIDs, ExcludeOperationIDs: excludeIDs,
		MatchStrategy: openapitransform.MatchStrategy(tagMatch),
	}
	flagErr := validateOverlayTransformFlags(filterOptions, cmd.Flags().Changed("tag-match"))
	if flagErr != nil {
		renderApplyOverlayError(stdOut, flagErr)
		return NewInputError("%s", flagErr)
	}

	// Read global flags
	timeFlag, _ := cmd.Flags().GetBool("time")
	certFile, _ := cmd.Flags().GetString("cert-file")
	keyFile, _ := cmd.Flags().GetString("key-file")
	caFile, _ := cmd.Flags().GetString("ca-file")
	insecure, _ := cmd.Flags().GetBool("insecure")

	// Disable color and styling for CI/CD use
	if noStyleFlag {
		color.DisableColors()
	}

	// Show banner unless piping
	if !stdIn && !stdOut {
		PrintBanner()
	}

	// Validate arguments
	// Without stdin: need 3 args (spec, overlay, output)
	// With stdin, without stdout: need 2 args (overlay, output)
	// Without stdin, with stdout: need 2 args (spec, overlay)
	// With both stdin and stdout: need 1 arg (overlay)
	requiredArgs := 3
	if stdIn {
		requiredArgs--
	}
	if stdOut {
		requiredArgs--
	}
	withoutOverlay := (filterOptions.Enabled() || pruneUnused) && len(args) == requiredArgs-1
	if withoutOverlay {
		requiredArgs--
	}
	if len(args) > requiredArgs {
		return NewInputError("too many arguments for apply-overlay")
	}

	if len(args) < requiredArgs {
		var errText string
		switch {
		case !stdIn && len(args) == 0:
			errText = "please supply input OpenAPI specification, or use -i flag for stdin"
		case !stdIn && len(args) == 1:
			errText = "please supply overlay file path or URL"
		case stdIn && len(args) == 0:
			errText = "please supply overlay file path or URL"
		case !stdOut && len(args) < requiredArgs:
			errText = "please supply output file path, or use -o flag for stdout"
		default:
			errText = "insufficient arguments provided"
		}
		renderApplyOverlayError(stdOut, errors.New(errText))
		return NewInputError("%s", errText)
	}

	// Determine argument positions based on flags
	var specPath, overlayPath, outputPath string
	argIdx := 0

	if !stdIn {
		specPath = args[argIdx]
		argIdx++
	}
	if !withoutOverlay {
		overlayPath = args[argIdx]
		argIdx++
	}
	if !stdOut {
		outputPath = args[argIdx]
	}

	// Read spec bytes
	var specBytes []byte
	var fileError error

	if stdIn {
		inputReader := cmd.InOrStdin()
		buf := &bytes.Buffer{}
		_, fileError = buf.ReadFrom(inputReader)
		specBytes = buf.Bytes()
	} else {
		specBytes, fileError = os.ReadFile(specPath)
	}

	if fileError != nil {
		if !stdIn {
			tui.RenderErrorString("Unable to read specification file '%s': %s", specPath, fileError.Error())
		} else {
			tui.RenderErrorString("Unable to read specification from stdin: %s", fileError.Error())
		}
		return fileError
	}
	if err := rejectAsyncAPIForOpenAPICommand("apply-overlay", specBytes); err != nil {
		if stdOut {
			fmt.Fprintln(os.Stderr, err.Error())
		} else {
			tui.RenderErrorString("%s", err.Error())
		}
		return err
	}

	// Build HTTP client config for remote overlay fetching
	resolvedCertFile, _ := ResolveConfigPath(certFile)
	resolvedKeyFile, _ := ResolveConfigPath(keyFile)
	resolvedCAFile, _ := ResolveConfigPath(caFile)

	httpClientConfig := utils.HTTPClientConfig{
		CertFile: resolvedCertFile,
		KeyFile:  resolvedKeyFile,
		CAFile:   resolvedCAFile,
		Insecure: insecure,
	}

	// Read overlay bytes (file or remote URL)
	var overlayBytes []byte
	var result *libopenapi.OverlayResult
	var applyErr error
	if withoutOverlay {
		var document libopenapi.Document
		document, applyErr = libopenapi.NewDocument(specBytes)
		result = &libopenapi.OverlayResult{Bytes: specBytes, OverlayDocument: document}
	} else {
		var overlayErr error
		overlayBytes, overlayErr = fetchOverlay(overlayPath, httpClientConfig)
		if overlayErr != nil {
			renderApplyOverlayError(stdOut, overlayErr)
			return overlayErr
		}
		result, applyErr = libopenapi.ApplyOverlayFromBytesToSpecBytes(specBytes, overlayBytes)
	}
	if applyErr != nil {
		renderApplyOverlayError(stdOut, applyErr)
		return NewInputError("unable to prepare OpenAPI document: %s", applyErr)
	}
	defer result.OverlayDocument.Release()

	var filterStats openapitransform.FilterStats
	var pruneStats openapitransform.PruneStats
	var transformWarnings []openapitransform.Warning
	transformsEnabled := filterOptions.Enabled() || pruneUnused
	if transformsEnabled {
		specInfo := result.OverlayDocument.GetSpecInfo()
		if specInfo == nil || specInfo.RootNode == nil {
			transformErr := errors.New("applied Overlay did not produce an OpenAPI document root")
			renderApplyOverlayError(stdOut, transformErr)
			return NewInputError("%s", transformErr)
		}
		root := specInfo.RootNode
		var transformErr error
		filterStats, pruneStats, transformErr = openapitransform.Transform(root, result.OverlayDocument.GetVersion(), openapitransform.Options{
			TagFilter:   filterOptions,
			PruneUnused: pruneUnused,
		})
		if transformErr != nil {
			renderApplyOverlayError(stdOut, transformErr)
			return NewInputError("%s", transformErr)
		}
		transformWarnings = filterStats.Warnings
		if filterOptions.Enabled() && !openapitransform.HasReachableOperations(root, result.OverlayDocument.GetVersion()) {
			transformWarnings = append(transformWarnings, openapitransform.Warning{
				Path:    "$",
				Message: "operation filtering kept zero operations",
			})
		}
		result.Bytes, transformErr = yaml.Marshal(root)
		if transformErr != nil {
			renderApplyOverlayError(stdOut, transformErr)
			return NewInputError("unable to render transformed OpenAPI document: %s", transformErr)
		}
	}

	// The overlay engine emits YAML. Preserve JSON input for both file and
	// stdout output, and honor an explicitly named JSON destination.
	if json.Valid(specBytes) || (!stdOut && strings.EqualFold(filepath.Ext(outputPath), ".json")) {
		var renderErr error
		result.Bytes, renderErr = renderBundledOutput(result.Bytes, bundleOutputFormatJSON)
		if renderErr != nil {
			renderApplyOverlayError(stdOut, renderErr)
			return NewInputError("unable to render JSON output: %s", renderErr)
		}
	}

	// Parse overlay document to get actions for the table display
	var overlayDoc *highoverlay.Overlay
	var overlayDocErr error
	if !withoutOverlay {
		overlayDoc, overlayDocErr = libopenapi.NewOverlayDocument(overlayBytes)
	}
	if overlayDocErr != nil {
		// Non-fatal: we still applied the overlay, just can't show the table
		tui.RenderWarning("Could not parse overlay for display: %s", overlayDocErr.Error())
	}

	// Count warnings
	warningCount := len(result.Warnings) + len(transformWarnings)

	// Display actions table (unless stdout mode or overlay parsing failed)
	if !stdOut && overlayDoc != nil && len(overlayDoc.Actions) > 0 {
		renderOverlayActionsTable(overlayDoc.Actions, result.Warnings, overlayPath, noStyleFlag)
	}

	// Display individual warnings (only when not showing table, or for stderr in stdout mode)
	if stdOut && warningCount > 0 {
		for _, warning := range result.Warnings {
			// Write warnings to stderr when using stdout for output
			fmt.Fprintf(os.Stderr, "Warning: target '%s' - %s\n", warning.Target, warning.Message)
		}
		for _, warning := range transformWarnings {
			fmt.Fprintf(os.Stderr, "Warning: %s - %s\n", warning.Path, warning.Message)
		}
	} else if !stdOut {
		for _, warning := range transformWarnings {
			tui.RenderWarning("%s - %s", warning.Path, warning.Message)
		}
	}

	// Write output (always write, even with warnings, so users can inspect the result)
	if stdOut {
		fmt.Print(string(result.Bytes))
	} else {
		writeErr := os.WriteFile(outputPath, result.Bytes, 0664)
		if writeErr != nil {
			tui.RenderErrorString("Unable to write output file '%s': %s", outputPath, writeErr.Error())
			return writeErr
		}

		if warningCount > 0 {
			tui.RenderSuccess("Overlay applied with %d warning(s), output written to '%s'", warningCount, outputPath)
		} else {
			tui.RenderSuccess("Overlay applied successfully, output written to '%s'", outputPath)
		}
		renderOverlayTransformSummary(filterStats, pruneStats, filterOptions.Enabled(), pruneUnused)
	}

	// Show timing if requested
	if timeFlag && !stdOut {
		duration := time.Since(startTime)
		RenderTime(true, duration, int64(len(result.Bytes)))
	}

	// Handle --fail-on-warnings exit code (after output is written)
	if failOnWarnings && warningCount > 0 {
		wErr := &warningsError{count: warningCount}
		if !stdOut {
			tui.RenderErrorString("%s", wErr.Error())
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", wErr.Error())
		}
		return &ExitError{Code: WarningsExitCode, Message: wErr.Error()}
	}

	return nil
}

func validateOverlayTransformFlags(options openapitransform.TagFilterOptions, matchExplicit bool) error {
	if matchExplicit && len(options.IncludeTags)+len(options.ExcludeTags) == 0 {
		return errors.New("--tag-match requires at least one --include-tag or --exclude-tag")
	}
	return openapitransform.ValidateTagFilterOptions(options)
}

func renderApplyOverlayError(stdout bool, err error) {
	if stdout {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return
	}
	tui.RenderErrorString("%s", err)
}

func renderOverlayTransformSummary(filter openapitransform.FilterStats, prune openapitransform.PruneStats, filtered, pruned bool) {
	if filtered {
		tui.RenderInfo("Filtered %d operations: %d kept, %d removed; removed %d empty operation containers",
			filter.OperationsSeen, filter.OperationsKept, filter.OperationsRemoved,
			filter.PathItemsRemoved+filter.WebhooksRemoved+filter.CallbackItemsRemoved)
	}
	if pruned {
		summary := fmt.Sprintf("Pruned %d unused components", prune.ComponentsRemoved)
		if prune.ComponentsRemoved > 0 {
			sections := make([]string, 0, len(prune.RemovedBySection))
			for section, count := range prune.RemovedBySection {
				sections = append(sections, fmt.Sprintf("%d %s", count, section))
			}
			sort.Strings(sections)
			summary += ": " + strings.Join(sections, ", ")
		}
		tui.RenderInfo("%s", summary)
		if prune.TagsRemoved > 0 {
			tui.RenderInfo("Pruned %d unused tags", prune.TagsRemoved)
		}
	}
}

// maxOverlaySize is the maximum size for remote overlay downloads (1 GiB)
const maxOverlaySize = 1024 * 1024 * 1024

// defaultHTTPTimeout is the timeout for HTTP requests when no custom client is configured
const defaultHTTPTimeout = 30 * time.Second

// fetchOverlay reads overlay content from a local file or remote URL.
// Remote URLs are limited to http/https schemes and 1 GiB maximum size.
func fetchOverlay(urlOrPath string, httpClientConfig utils.HTTPClientConfig) ([]byte, error) {
	// Check if it's a remote URL (only http/https allowed)
	if strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://") {
		var client *http.Client
		var err error

		if utils.ShouldUseCustomHTTPClient(httpClientConfig) {
			client, err = utils.CreateCustomHTTPClient(httpClientConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create HTTP client: %w", err)
			}
		} else {
			client = &http.Client{Timeout: defaultHTTPTimeout}
		}

		resp, err := client.Get(urlOrPath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch remote overlay from %s: %w", urlOrPath, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("remote overlay %s returned status %d", urlOrPath, resp.StatusCode)
		}

		// Limit read size to prevent memory exhaustion
		limitedReader := io.LimitReader(resp.Body, maxOverlaySize+1)
		data, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read overlay response: %w", err)
		}

		if len(data) > maxOverlaySize {
			return nil, fmt.Errorf("overlay exceeds maximum size of %d bytes", maxOverlaySize)
		}

		return data, nil
	}

	// Local file - resolve path
	resolvedPath, err := ResolveConfigPath(urlOrPath)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(resolvedPath)
}

// renderOverlayActionsTable renders a table showing all overlay actions and their status
func renderOverlayActionsTable(actions []*highoverlay.Action, warnings []*overlay.Warning, overlayPath string, noStyle bool) {
	// Get terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width == 0 {
		width = 120 // default
	}

	// Make overlay path relative to working directory if possible
	displayPath := overlayPath
	if wd, wdErr := os.Getwd(); wdErr == nil {
		if relPath, relErr := filepath.Rel(wd, overlayPath); relErr == nil {
			displayPath = relPath
		}
	}

	// Build table data
	columns, rows := tui.BuildOverlayActionsTable(actions, warnings, displayPath, width)

	if len(rows) == 0 {
		return
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)+1),
		table.WithWidth(width-2),
	)

	// Apply styling (unless no-style mode)
	if !noStyle {
		color.ApplyOverlayTableStyles(&t)
	} else {
		// Apply plain styles to avoid default table styling
		color.ApplyPlainTableStyles(&t)
	}

	// Get table output and colorize paths and status
	tableOutput := t.View()
	if !noStyle {
		// Sort rows by target length (longest first) to avoid partial matches
		// Target is column 1 (after Position)
		sortedRows := make([]table.Row, len(rows))
		copy(sortedRows, rows)
		sort.Slice(sortedRows, func(i, j int) bool {
			return len(sortedRows[i][1]) > len(sortedRows[j][1])
		})

		// Colorize the target paths and status in each row
		lines := strings.Split(tableOutput, "\n")
		for i, line := range lines {
			// Colorize paths (Target is now column 1, after Position)
			for _, row := range sortedRows {
				if len(row) > 1 && row[1] != "" && strings.Contains(line, row[1]) {
					colorizedPath := color.ColorizePath(row[1])
					lines[i] = strings.Replace(line, row[1], colorizedPath, 1)
					break
				}
			}
			// Colorize position (file:line:col format)
			for _, row := range rows {
				if len(row) > 0 && row[0] != "" && row[0] != "-" && strings.Contains(lines[i], row[0]) {
					colorizedLocation := color.ColorizeLocation(row[0])
					lines[i] = strings.Replace(lines[i], row[0], colorizedLocation, 1)
					break
				}
			}
			// Colorize status - OK (green) and WARN (yellow)
			lines[i] = strings.Replace(lines[i], "OK", color.StyleStatusOK.Render("OK"), 1)
			lines[i] = strings.Replace(lines[i], "WARN", color.StyleStatusWarn.Render("WARN"), 1)
			// Colorize action prefixes - [~] grey, [-] red
			lines[i] = strings.Replace(lines[i], "[~]", color.StylePathGrey.Render("[~]"), 1)
			lines[i] = strings.Replace(lines[i], "[-]", color.StyleActionRemove.Render("[-]"), 1)
		}
		tableOutput = strings.Join(lines, "\n")
	}

	// Print table
	fmt.Println()
	fmt.Println(tableOutput)
	fmt.Println()

	// Print summary
	summary := tui.FormatOverlaySummary(actions, len(warnings))
	if noStyle {
		fmt.Println(summary)
	} else {
		tui.RenderInfo("%s", summary)
	}
	fmt.Println()
}
