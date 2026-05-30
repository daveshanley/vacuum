// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/color"
	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

func getSchemaBundleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "bundle <input> [output]",
		Short:         "Bundle JSON Schema documents with external references",
		Long:          "Bundle JSON Schema documents by rewriting ordinary external $ref values into root $defs. Dynamic and recursive references are preserved.",
		RunE:          runSchemaBundle,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Flags().BoolP("stdin", "i", false, "Read a JSON Schema document from stdin")
	cmd.Flags().BoolP("stdout", "o", false, "Write bundled schema to stdout")
	cmd.Flags().String("output", "", "Write bundled schema to a file")
	cmd.Flags().String("format", "", "Output format for bundled documents: yaml or json")
	cmd.Flags().StringP("delimiter", "d", "__", "Delimiter used to separate clashing $defs names")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output")
	return cmd
}

func readSchemaBundleFlags(cmd *cobra.Command) (*schemaBundleFlags, error) {
	flags := &schemaBundleFlags{}
	flags.Stdin, _ = cmd.Flags().GetBool("stdin")
	flags.Stdout, _ = cmd.Flags().GetBool("stdout")
	flags.Output, _ = cmd.Flags().GetString("output")
	flags.Format, _ = cmd.Flags().GetString("format")
	flags.Delimiter, _ = cmd.Flags().GetString("delimiter")
	flags.NoStyle, _ = cmd.Flags().GetBool("no-style")
	flags.Base, _ = cmd.Flags().GetString("base")
	flags.Remote, _ = cmd.Flags().GetBool("remote")
	flags.CertFile, _ = cmd.Flags().GetString("cert-file")
	flags.KeyFile, _ = cmd.Flags().GetString("key-file")
	flags.CAFile, _ = cmd.Flags().GetString("ca-file")
	flags.Insecure, _ = cmd.Flags().GetBool("insecure")
	if flags.Delimiter == "" {
		flags.Delimiter = "__"
	}
	return flags, nil
}

func runSchemaBundle(cmd *cobra.Command, args []string) error {
	flags, err := readSchemaBundleFlags(cmd)
	if err != nil {
		return err
	}
	if flags.NoStyle {
		color.DisableColors()
	}
	if !flags.Stdin && !flags.Stdout && flags.Output == "" {
		PrintBanner(flags.NoStyle)
	}
	bundleArgs, outputArg, err := splitSchemaBundleArgs(args, flags)
	if err != nil {
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	inputs, err := collectSchemaInputs(cmd, bundleArgs, nil, nil, nil, flags.Stdin, flags.Base, "bundle")
	if err != nil {
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	if len(inputs) == 0 {
		err = errors.New("please supply a JSON Schema document to bundle")
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	if len(inputs) != 1 {
		err = errors.New("schema bundle requires exactly one file input or --stdin")
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	if flags.Output != "" {
		outputArg = flags.Output
	}
	if flags.Stdin && !flags.Stdout && outputArg == "" {
		err = errors.New("schema bundle --stdin requires --stdout or --output")
		tui.RenderErrorString("%s", err.Error())
		return err
	}
	if err = ensureSchemaStdinBaseForBundle(inputs[0], flags.Base); err != nil {
		tui.RenderErrorString("%s", err.Error())
		return err
	}

	httpClientConfig, cfgErr := schemaHTTPClientConfig(flags.CertFile, flags.KeyFile, flags.CAFile, flags.Insecure)
	if cfgErr != nil {
		return fmt.Errorf("failed to resolve TLS configuration: %w", cfgErr)
	}
	httpClient, clientErr := utils.CreateHTTPClientIfNeeded(httpClientConfig)
	if clientErr != nil {
		return fmt.Errorf("failed to create HTTP client: %w", clientErr)
	}

	for _, input := range inputs {
		outputFormat, formatErr := resolveSchemaBundleOutputFormat(flags.Format, outputArg, input.Path, input.Bytes)
		if formatErr != nil {
			tui.RenderErrorString("%s", formatErr.Error())
			return formatErr
		}
		bundled, warnings, bundleErr := bundleSchemaInput(input, flags, httpClient)
		for _, warning := range warnings {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", warning)
		}
		if bundleErr != nil {
			tui.RenderError(bundleErr)
			return bundleErr
		}
		rendered, renderErr := renderSchemaBundleOutput(bundled, outputFormat)
		if renderErr != nil {
			tui.RenderErrorString("Unable to render bundled schema as %s: %s", outputFormat, renderErr.Error())
			return renderErr
		}
		if flags.Stdout {
			_, _ = cmd.OutOrStdout().Write(rendered)
			if len(rendered) == 0 || rendered[len(rendered)-1] != '\n' {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}
			continue
		}
		outPath := outputArg
		if outPath == "" {
			err = schemaBundleMissingOutputError(input.Path)
			tui.RenderErrorString("%s", err.Error())
			return err
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0775); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, rendered, 0664); err != nil {
			tui.RenderErrorString("Unable to write bundled schema: '%s': %s", outPath, err.Error())
			return err
		}
		tui.RenderSuccess("Bundled JSON Schema document written to '%s'", outPath)
	}
	return nil
}

func schemaBundleMissingOutputError(inputPath string) error {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" || inputPath == "stdin" {
		return errors.New("schema bundle requires an output path or --stdout")
	}
	return fmt.Errorf("schema bundle requires an output path or --stdout\n\nTry one of these:\n  vacuum schema bundle %q bundled.schema.json\n  vacuum schema bundle %q --stdout", inputPath, inputPath)
}

func splitSchemaBundleArgs(args []string, flags *schemaBundleFlags) ([]string, string, error) {
	if flags.Stdin {
		if len(args) > 0 {
			return nil, "", errors.New("schema bundle --stdin cannot be combined with file inputs")
		}
		return nil, flags.Output, nil
	}
	if len(args) == 0 {
		return nil, "", errors.New("please supply a JSON Schema document to bundle")
	}
	if len(args) > 2 {
		return nil, "", errors.New("schema bundle accepts one input and one optional output")
	}
	info, err := os.Stat(args[0])
	if err != nil {
		return nil, "", err
	}
	if info.IsDir() {
		return nil, "", errors.New("schema bundle cannot bundle folders; provide a single JSON Schema file")
	}
	if flags.Output != "" && len(args) == 2 {
		return nil, "", errors.New("schema bundle output was supplied twice; use either an output argument or --output")
	}
	if flags.Output != "" || len(args) == 1 {
		return args[:1], flags.Output, nil
	}
	return args[:1], args[1], nil
}

func resolveSchemaBundleOutputFormat(formatFlag, outputPath, inputPath string, inputBytes []byte) (string, error) {
	if formatFlag != "" {
		switch strings.ToLower(formatFlag) {
		case bundleOutputFormatYAML, "yml":
			return bundleOutputFormatYAML, nil
		case bundleOutputFormatJSON:
			return bundleOutputFormatJSON, nil
		default:
			return "", fmt.Errorf("invalid schema bundle output format %q, expected yaml or json", formatFlag)
		}
	}
	return detectSchemaInputFormat(inputPath, inputBytes), nil
}

func detectSchemaInputFormat(inputPath string, inputBytes []byte) string {
	switch strings.ToLower(filepath.Ext(inputPath)) {
	case ".json":
		return bundleOutputFormatJSON
	case ".yaml", ".yml":
		return bundleOutputFormatYAML
	}
	trimmed := bytes.TrimSpace(inputBytes)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return bundleOutputFormatJSON
	}
	return bundleOutputFormatYAML
}

func renderSchemaBundleOutput(root *yaml.Node, format string) ([]byte, error) {
	switch format {
	case bundleOutputFormatJSON:
		return schemautil.ToJSON(root, true)
	case bundleOutputFormatYAML:
		return yaml.Marshal(root)
	default:
		return nil, fmt.Errorf("unsupported schema bundle output format %q", format)
	}
}

func bundleSchemaInput(input schemaInput, flags *schemaBundleFlags, httpClient *http.Client) (*yaml.Node, []string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(input.Bytes, &doc); err != nil {
		return nil, nil, fmt.Errorf("unable to parse JSON Schema '%s': %w", input.Display, err)
	}
	root := schemautil.RootNode(&doc)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("schema '%s' must be a mapping/object root to bundle", input.Display)
	}
	if !schemautil.HasSchemaKeyword(root) {
		schemautil.EnsureRootSchema(root, schemautil.SchemaURL2020)
	}
	rootFormat := schemautil.DetectDialect(root).Format
	ctx := &schemaBundleContext{
		rootFormat:  rootFormat,
		rootDefs:    ensureSchemaDefs(root),
		delimiter:   flags.Delimiter,
		cache:       make(map[string]string),
		usedKeys:    existingDefKeys(root),
		httpClient:  httpClient,
		allowRemote: flags.Remote,
	}
	currentDir := input.Base
	if !input.FromStdin && !strings.Contains(input.Path, "://") {
		currentDir = filepath.Dir(input.Path)
	}
	if err := rewriteExternalRefs(ctx, root, root, currentDir); err != nil {
		return nil, ctx.warnings, err
	}
	return root, ctx.warnings, nil
}

func ensureSchemaDefs(root *yaml.Node) *yaml.Node {
	defs := schemautil.MappingValueNode(root, "$defs")
	if defs != nil && defs.Kind == yaml.MappingNode {
		return defs
	}
	if defs != nil {
		defs.Kind = yaml.MappingNode
		defs.Tag = "!!map"
		defs.Content = nil
		return defs
	}
	defs = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "$defs"}, defs)
	return defs
}

func existingDefKeys(root *yaml.Node) map[string]bool {
	used := make(map[string]bool)
	defs := schemautil.MappingValueNode(root, "$defs")
	if defs == nil || defs.Kind != yaml.MappingNode {
		return used
	}
	for i := 0; i+1 < len(defs.Content); i += 2 {
		used[defs.Content[i].Value] = true
	}
	return used
}

func rewriteExternalRefs(ctx *schemaBundleContext, documentRoot, node *yaml.Node, currentDir string) error {
	node = schemautil.RootNode(node)
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if key.Value == "$dynamicRef" || key.Value == "$recursiveRef" {
				continue
			}
			if key.Value == "$ref" && value.Kind == yaml.ScalarNode {
				rewritten, err := rewriteSchemaRef(ctx, documentRoot, value.Value, currentDir)
				if err != nil {
					return err
				}
				if rewritten != "" {
					value.Value = rewritten
				}
				continue
			}
			if err := rewriteExternalRefs(ctx, documentRoot, value, currentDir); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := rewriteExternalRefs(ctx, documentRoot, child, currentDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func rewriteSchemaRef(ctx *schemaBundleContext, documentRoot *yaml.Node, ref, currentDir string) (string, error) {
	external, fragment := splitSchemaRef(ref)
	if external == "" {
		keyName, suffix, ok := missingLocalDefsReference(documentRoot, fragment)
		if !ok {
			return "", nil
		}
		key, err := ctx.bundleMissingLocalDef(keyName, currentDir)
		if err != nil {
			return "", err
		}
		return "#/$defs/" + escapeJSONPointerSegment(key) + suffix, nil
	}
	key, err := ctx.bundleExternalSchema(external, currentDir)
	if err != nil {
		return "", err
	}
	if fragment == "" || fragment == "#" {
		return "#/$defs/" + escapeJSONPointerSegment(key), nil
	}
	return "#/$defs/" + escapeJSONPointerSegment(key) + strings.TrimPrefix(fragment, "#"), nil
}

func missingLocalDefsReference(documentRoot *yaml.Node, fragment string) (keyName, suffix string, ok bool) {
	if !strings.HasPrefix(fragment, "#/$defs/") {
		return "", "", false
	}
	remaining := strings.TrimPrefix(fragment, "#/$defs/")
	encodedKey, suffix, _ := strings.Cut(remaining, "/")
	keyName = unescapeJSONPointerSegment(encodedKey)
	if keyName == "" {
		return "", "", false
	}
	defs := schemautil.MappingValueNode(documentRoot, "$defs")
	if defs != nil && defs.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(defs.Content); i += 2 {
			if defs.Content[i].Value == keyName {
				return "", "", false
			}
		}
	}
	if suffix != "" {
		suffix = "/" + suffix
	}
	return keyName, suffix, true
}

func (ctx *schemaBundleContext) bundleMissingLocalDef(keyName, currentDir string) (string, error) {
	candidates := schemaDefinitionFilenames(keyName)
	var attempts []string
	var lastErr error
	for _, candidate := range candidates {
		attempts = append(attempts, candidate)
		key, err := ctx.bundleExternalSchema(candidate, currentDir)
		if err == nil {
			return key, nil
		}
		lastErr = err
		if !strings.Contains(err.Error(), "unable to resolve external schema reference") {
			return "", err
		}
	}
	return "", fmt.Errorf("cannot resolve local $defs reference %q; tried %s: %w", keyName, strings.Join(attempts, ", "), lastErr)
}

func schemaDefinitionFilenames(keyName string) []string {
	return []string{
		keyName + ".schema.json",
		keyName + ".json",
		keyName + ".schema.yaml",
		keyName + ".yaml",
		keyName + ".schema.yml",
		keyName + ".yml",
	}
}

func schemaHasRelativeExternalRefs(raw []byte) bool {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return false
	}
	return nodeHasRelativeExternalRefs(schemautil.RootNode(&doc))
}

func nodeHasRelativeExternalRefs(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if key.Value == "$ref" && value.Kind == yaml.ScalarNode {
				external, _ := splitSchemaRef(value.Value)
				if external != "" && !filepath.IsAbs(external) && !strings.Contains(external, "://") {
					return true
				}
			}
			if nodeHasRelativeExternalRefs(value) {
				return true
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if nodeHasRelativeExternalRefs(child) {
				return true
			}
		}
	}
	return false
}

func ensureSchemaStdinBaseForBundle(input schemaInput, baseFlag string) error {
	if !input.FromStdin || baseFlag != "" || !schemaHasRelativeExternalRefs(input.Bytes) {
		return nil
	}
	return errors.New("schema bundle --stdin input contains relative external refs; provide --base")
}

func (ctx *schemaBundleContext) bundleExternalSchema(external, currentDir string) (string, error) {
	canonical, raw, err := loadExternalSchema(external, currentDir, ctx.allowRemote, ctx.httpClient)
	if err != nil {
		return "", err
	}
	if key, ok := ctx.cache[canonical]; ok {
		return key, nil
	}
	key := ctx.nextDefKey(external)
	ctx.cache[canonical] = key
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("unable to parse external schema '%s': %w", external, err)
	}
	root := schemautil.RootNode(&doc)
	if root == nil || root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("external schema '%s' must be a mapping/object root", external)
	}
	dialect := schemautil.DetectDialect(root)
	if schemautil.IsSupportedDialect(ctx.rootFormat) && schemautil.IsSupportedDialect(dialect.Format) && dialect.Format != ctx.rootFormat {
		ctx.warnings = append(ctx.warnings, fmt.Sprintf("mixed-dialect bundle: root is %s, %s is %s; no dialect conversion was applied",
			ctx.rootFormat, external, dialect.Format))
	}
	childDir := currentDir
	if !strings.Contains(canonical, "://") {
		childDir = filepath.Dir(canonical)
	}
	if err := rewriteExternalRefs(ctx, root, root, childDir); err != nil {
		return "", err
	}
	ctx.rootDefs.Content = append(ctx.rootDefs.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		root,
	)
	return key, nil
}

func (ctx *schemaBundleContext) nextDefKey(external string) string {
	base := external
	if parsed, _, ok := strings.Cut(external, "#"); ok {
		base = parsed
	}
	base = path.Base(filepath.ToSlash(base))
	for _, suffix := range []string{".schema.json", ".schema.yaml", ".schema.yml"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
			break
		}
	}
	if ext := path.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	key := sanitizeSchemaDefKey(base)
	if key == "" {
		key = "schema"
	}
	if !ctx.usedKeys[key] {
		ctx.usedKeys[key] = true
		return key
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%s%d", key, ctx.delimiter, i)
		if !ctx.usedKeys[candidate] {
			ctx.usedKeys[candidate] = true
			return candidate
		}
	}
}

func sanitizeSchemaDefKey(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return strings.Trim(b.String(), "_")
}

func splitSchemaRef(ref string) (external, fragment string) {
	before, after, found := strings.Cut(ref, "#")
	if !found {
		return ref, ""
	}
	return before, "#" + after
}

func loadExternalSchema(external, currentDir string, allowRemote bool, httpClient *http.Client) (string, []byte, error) {
	if strings.HasPrefix(external, "http://") || strings.HasPrefix(external, "https://") {
		if !allowRemote {
			return "", nil, fmt.Errorf("remote schema reference %q found but --remote=false", external)
		}
		raw, err := fetchRemoteSpec(external, httpClient)
		return external, raw, err
	}
	target := external
	if !filepath.IsAbs(target) {
		target = filepath.Join(currentDir, external)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", nil, err
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return "", nil, fmt.Errorf("unable to resolve external schema reference %q: %w", external, err)
	}
	return abs, raw, nil
}

func escapeJSONPointerSegment(segment string) string {
	segment = strings.ReplaceAll(segment, "~", "~0")
	segment = strings.ReplaceAll(segment, "/", "~1")
	return segment
}

func unescapeJSONPointerSegment(segment string) string {
	segment = strings.ReplaceAll(segment, "~1", "/")
	segment = strings.ReplaceAll(segment, "~0", "~")
	return segment
}
