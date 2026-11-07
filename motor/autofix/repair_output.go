// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/motor/autofix/runtime"

	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/bundler"
)

// FixOutputOptions selects the representation and destination of a fix preview.
// The destination is checked for reference portability; no files are written.
type FixOutputOptions struct {
	BundleMode string // Empty or keep preserves references; inline or composed bundles OpenAPI 3.
	Format     string // Empty preserves the input format; otherwise yaml or json.
	OutputPath string // Empty preserves the input location.
}

func prepareRepairOutput(execution *motor.RuleSetExecution, candidate []byte, options *motor.ExecutionOptions, before *motor.RuleSetExecutionResult, sources *runtime.Sources, output FixOutputOptions) ([]byte, error) {
	switch output.BundleMode {
	case "", "keep", "inline", "composed":
	default:
		return nil, fmt.Errorf("unsupported fix bundle mode %q", output.BundleMode)
	}
	format := output.Format
	switch format {
	case "":
		format = "yaml"
		if json.Valid(execution.Spec) {
			format = "json"
		}
	case "yaml", "json":
	default:
		return nil, fmt.Errorf("unsupported fix output format %q", format)
	}
	var err error
	if output.BundleMode == "inline" || output.BundleMode == "composed" {
		candidate, err = bundleRepairOutput(candidate, options, before, sources, output.BundleMode)
		if err != nil {
			return nil, err
		}
	}
	if err := repairCanceled(options); err != nil {
		return nil, err
	}
	if format == "json" && !json.Valid(candidate) {
		candidate, err = repair.RenderJSON(candidate)
	} else if format == "yaml" && json.Valid(candidate) {
		var root *yaml.Node
		root, err = repair.ParseDocument(candidate)
		if err == nil {
			// JSON collections carry flow style. YAML output uses the normal block view.
			var blockCollections func(*yaml.Node)
			blockCollections = func(n *yaml.Node) {
				if n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode {
					n.Style = 0
				}
				for _, c := range n.Content {
					blockCollections(c)
				}
			}
			blockCollections(root)
			candidate, err = yaml.Marshal(root)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("cannot render fix output as %s: %w", format, err)
	}
	if err := validateRepairOutputLocation(execution.SpecFileName, output.OutputPath, candidate); err != nil {
		return nil, err
	}
	return candidate, repairCanceled(options)
}

func bundleRepairOutput(candidate []byte, options *motor.ExecutionOptions, before *motor.RuleSetExecutionResult, sources *runtime.Sources, mode string) ([]byte, error) {
	root, err := repair.ParseDocument(candidate)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(fixScalar(root, "openapi"), "3.") {
		return nil, fmt.Errorf("fix output bundling supports OpenAPI 3 documents only; keep references for this document")
	}
	if err := repairCanceled(options); err != nil {
		return nil, err
	}
	config := motor.CloneDocumentConfiguration(before.DocumentConfig)
	indexedSources, err := newRepairIndexedSources(sources)
	if err != nil {
		return nil, err
	}
	config.LocalFS, config.RemoteFS = indexedSources, sources
	// Composed bundling consumes the index's ordered reference list, as the
	// bundle command does. Ordinary lint does not need to collect that list.
	config.ExtractRefsSequentially = true
	ctx := context.Background()
	if options != nil && options.Context != nil {
		ctx = options.Context
	}
	config.RemoteURLHandler = motor.RemoteFSHandler(ctx, sources)
	// Rebuild from the patched snapshot. Composed bundling mutates its model.
	doc, err := libopenapi.NewDocumentWithConfiguration(candidate, config)
	if err != nil {
		return nil, fmt.Errorf("cannot build fix output bundle: %w", err)
	}
	defer func() {
		if rolo := doc.GetRolodex(); rolo != nil && rolo.GetRootIndex() != nil {
			rolo.GetRootIndex().Release()
		}
		doc.Release()
	}()
	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("cannot bundle an incomplete document: %w", err)
	}
	if model == nil {
		return nil, fmt.Errorf("cannot bundle an unavailable document model")
	}
	if err := repairCanceled(options); err != nil {
		return nil, err
	}
	if mode == "composed" {
		return bundler.BundleDocumentComposed(&model.Model, nil)
	}
	keepLocal := false
	return bundler.BundleDocumentWithConfig(&model.Model, &bundler.BundleInlineConfig{InlineLocalRefs: &keepLocal})
}

func validateRepairOutputLocation(source, output string, candidate []byte) error {
	if output == "" {
		return nil
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	outputAbs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	if filepath.Dir(sourceAbs) == filepath.Dir(outputAbs) {
		return nil
	}
	root, err := repair.ParseDocument(candidate)
	if err != nil {
		return err
	}
	if locationDependentRepairInput(root, "") {
		return fmt.Errorf("fix output still has relative references: choose a bundle that removes them or save in the source directory")
	}
	return nil
}

// Only location-independent references can survive a change of directory.
// This check is deliberately conservative, including extension object values.
func locationDependentRepairInput(n *yaml.Node, parent string) bool {
	relative := func(value string) bool {
		if value == "" || strings.HasPrefix(value, "#") {
			return false
		}
		u, err := url.Parse(value)
		return err != nil || !u.IsAbs()
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, value := n.Content[i].Value, n.Content[i+1]
			switch key {
			case "$ref", "$dynamicRef", "$recursiveRef", "$id", "$self", "$schema", "externalValue", "operationRef", "termsOfService", "authorizationUrl", "tokenUrl", "refreshUrl":
				if value.Kind == yaml.ScalarNode && relative(value.Value) {
					return true
				}
			case "url":
				if (parent == "servers" || parent == "server" || parent == "externalDocs" || parent == "contact" || parent == "license") && value.Kind == yaml.ScalarNode && relative(value.Value) {
					return true
				}
			case "sourceDescriptions":
				// Arazzo source URLs are not bundled by the OpenAPI bundler.
				if value.Kind == yaml.SequenceNode {
					for _, source := range value.Content {
						if relative(fixScalar(source, "url")) {
							return true
						}
					}
				}
			case "mapping":
				if parent == "discriminator" && value.Kind == yaml.MappingNode {
					for j := 1; j < len(value.Content); j += 2 {
						mapping := value.Content[j].Value
						if strings.ContainsAny(mapping, "/.\\") && relative(mapping) {
							return true
						}
					}
				}
			}
			if locationDependentRepairInput(value, key) {
				return true
			}
		}
		return false
	}
	for _, child := range n.Content {
		if locationDependentRepairInput(child, parent) {
			return true
		}
	}
	return false
}
