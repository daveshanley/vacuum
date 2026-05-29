// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestGetSchemaCommand(t *testing.T) {
	cmd := GetSchemaCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "schema <input...>", cmd.Use)
	assert.Contains(t, cmd.Short, "JSON Schema")
	assert.Contains(t, cmd.Example, "vacuum schema my-schema.json")
	assert.Contains(t, cmd.Example, "--include")
}

func TestSchemaCommand_MissingInputShowsExamples(t *testing.T) {
	cmd := GetSchemaCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--no-banner"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "please supply a JSON Schema document to lint")
	assert.Contains(t, err.Error(), "vacuum schema my-schema.json")
	assert.Contains(t, err.Error(), `vacuum schema --globbed-files "schemas/**/*.json"`)
	assert.Contains(t, err.Error(), `vacuum schema ./schemas`)
	assert.Contains(t, err.Error(), `vacuum schema ./schemas --include "**/*.schema" --exclude "**/*.test.json"`)
}

func TestSchemaCommand_JSONOutputUsesSchemaFormat(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "customer.schema-anything")
	writeTestFile(t, schemaPath, `
$schema: https://json-schema.org/draft/2020-12/schema
title: Customer
description: Customer record
type: object
required: [id, missing]
properties:
  id:
    type: string
    enum: [1]
`)

	cmd := GetSchemaCommand()
	var out bytes.Buffer
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{schemaPath, "--format", "json", "--fail-severity", "none"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.NotContains(t, out.String(), "OpenAPI")

	var report map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	specInfo := report["specInfo"].(map[string]any)
	assert.Equal(t, "json-schema-2020-12", specInfo["format"])
	resultSet := report["resultSet"].(map[string]any)
	assert.NotZero(t, resultSet["errorCount"])
}

func TestSchemaCommand_JSONOutputStillReturnsViolationExitError(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	writeTestFile(t, schemaPath, `
title: Customer
description: Customer record
type: object
required: [missing]
properties:
  id:
    type: string
`)

	cmd := GetSchemaCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{schemaPath, "--format", "json"})

	err := cmd.Execute()
	require.Error(t, err)
	var exitErr *ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, ExitCodeViolations, exitErr.Code)
	assert.Contains(t, out.String(), `"format": "json-schema-2020-12"`)
}

func TestSchemaCommand_StdinIsExclusive(t *testing.T) {
	cmd := GetSchemaCommand()
	var out bytes.Buffer
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(`{"type":"object"}`))
	cmd.SetArgs([]string{"--stdin", "schema.yaml"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--stdin cannot be combined")
}

func TestSchemaCommand_FolderDefaultsToJSONAndYAML(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "root.json"), `{"type":"object"}`)
	writeTestFile(t, filepath.Join(dir, "nested", "schema.yaml"), `type: object`)
	writeTestFile(t, filepath.Join(dir, "nested", "schema.yml"), `type: object`)
	writeTestFile(t, filepath.Join(dir, "schema.not-json"), `type: object`)

	inputs, err := collectSchemaInputs(GetSchemaCommand(), []string{dir}, nil, nil, nil, false, "", "lint")
	require.NoError(t, err)
	require.Len(t, inputs, 3)
	paths := []string{inputs[0].Path, inputs[1].Path, inputs[2].Path}
	assert.Contains(t, paths, filepath.Join(dir, "root.json"))
	assert.Contains(t, paths, filepath.Join(dir, "nested", "schema.yaml"))
	assert.Contains(t, paths, filepath.Join(dir, "nested", "schema.yml"))
	assert.NotContains(t, paths, filepath.Join(dir, "schema.not-json"))
}

func TestSchemaCommand_FolderIncludeAllowsArbitraryExtension(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.not-json"), `
title: Thing
description: Thing record
type: object
properties:
  id:
    type: string
`)

	cmd := GetSchemaCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{dir, "--include", "*.not-json", "--format", "json", "--fail-severity", "none"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), `"format": "json-schema-2020-12"`)
}

func TestSchemaCommand_FolderRejectsShellExpandedIncludeFiles(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestFile(t, filepath.Join("schemas", "schema.json"), `{"type":"object"}`)
	writeTestFile(t, "one.json", `{"type":"object"}`)
	writeTestFile(t, "two.json", `{"type":"object"}`)

	cmd := GetSchemaCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"schemas", "--include", "one.json", "two.json", "--format", "json", "--fail-severity", "none"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder inputs cannot be combined with file inputs")
	assert.Contains(t, err.Error(), `--include "**/*.json"`)
}

func TestSchemaCommand_LintSubcommand(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	writeTestFile(t, schemaPath, `
title: Thing
description: Thing record
type: object
`)

	cmd := GetSchemaCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"lint", schemaPath, "--format", "json", "--fail-severity", "none"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), `"format": "json-schema-2020-12"`)
}

func TestSchemaCommand_CustomRulesetUsesCoreFunctions(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.yaml")
	rulesetPath := filepath.Join(dir, "ruleset.yaml")
	writeTestFile(t, schemaPath, `
title: Account
description: Account record
type: object
`)
	writeTestFile(t, rulesetPath, `
extends: [[vacuum:json-schema, off]]
rules:
  schema-title-is-customer:
    description: Schema title must be Customer
    severity: error
    formats: [json-schema]
    given: $
    then:
      field: title
      function: pattern
      functionOptions:
        match: ^Customer$
`)

	cmd := GetSchemaCommand()
	var out bytes.Buffer
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{schemaPath, "--ruleset", rulesetPath, "--format", "json", "--fail-severity", "none"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, out.String(), "schema-title-is-customer")
	assert.NotContains(t, out.String(), "operation")
}

func TestSchemaBundle_RewritesExternalRefsAndPreservesDynamicRefs(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	outPath := filepath.Join(dir, "bundle.yaml")
	writeTestFile(t, rootPath, `
$schema: https://json-schema.org/draft/2020-12/schema
$dynamicAnchor: node
type: object
properties:
  child:
    $ref: child.yaml#/$defs/Child
  recursive:
    $dynamicRef: "#node"
`)
	writeTestFile(t, filepath.Join(dir, "child.yaml"), `
$schema: https://json-schema.org/draft/2020-12/schema
$defs:
  Child:
    type: object
    properties:
      name:
        type: string
`)

	cmd := getSchemaBundleCommand()
	cmd.PersistentFlags().StringP("base", "p", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{rootPath, outPath, "--no-style"})

	err := cmd.Execute()
	require.NoError(t, err)

	raw, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "$dynamicRef")
	assert.Contains(t, string(raw), "$defs")

	var bundled yaml.Node
	require.NoError(t, yaml.Unmarshal(raw, &bundled))
	root := bundled.Content[0]
	properties := findYAMLMapValue(root, "properties")
	child := findYAMLMapValue(properties, "child")
	ref := findYAMLMapValue(child, "$ref")
	require.NotNil(t, ref)
	assert.True(t, strings.HasPrefix(ref.Value, "#/$defs/child/"))
}

func TestSchemaBundle_MissingOutputSuggestsNextCommand(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.schema.json")
	writeTestFile(t, rootPath, `{"type":"object"}`)

	cmd := getSchemaBundleCommand()
	cmd.PersistentFlags().StringP("base", "p", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{rootPath, "--no-style"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema bundle requires an output path or --stdout")
	assert.Contains(t, err.Error(), fmt.Sprintf("vacuum schema bundle %q bundled.schema.json", rootPath))
	assert.Contains(t, err.Error(), fmt.Sprintf("vacuum schema bundle %q --stdout", rootPath))
}

func TestSchemaBundle_RejectsFolderInput(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schemas", "one.schema.json"), `{"type":"object"}`)

	cmd := getSchemaBundleCommand()
	cmd.PersistentFlags().StringP("base", "p", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{filepath.Join(dir, "schemas"), "--no-style"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot bundle folders")
}

func TestSchemaBundle_StdoutPreservesJSONAndRewritesExtractedDefs(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.schema.json")
	writeTestFile(t, rootPath, `{
  "type": "object",
  "properties": {
    "barrier": {
      "$ref": "_defs/BarrierFeature.schema.json"
    }
  }
}`)
	writeTestFile(t, filepath.Join(dir, "_defs", "BarrierFeature.schema.json"), `{
  "type": "object",
  "properties": {
    "barrierType": {
      "$ref": "#/$defs/BarrierType"
    }
  }
}`)
	writeTestFile(t, filepath.Join(dir, "_defs", "BarrierType.schema.json"), `{
  "type": "string",
  "enum": ["KnockIn", "KnockOut"]
}`)

	cmd := getSchemaBundleCommand()
	cmd.PersistentFlags().StringP("base", "p", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{rootPath, "--stdout", "--no-style"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(out.String()), "{"))

	var bundled map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &bundled))
	defs := bundled["$defs"].(map[string]any)
	assert.Contains(t, defs, "BarrierFeature")
	assert.Contains(t, defs, "BarrierType")
	feature := defs["BarrierFeature"].(map[string]any)
	properties := feature["properties"].(map[string]any)
	barrierType := properties["barrierType"].(map[string]any)
	assert.Equal(t, "#/$defs/BarrierType", barrierType["$ref"])
}

func TestSchemaBundle_StdinRelativeRefsRequireBase(t *testing.T) {
	cmd := getSchemaBundleCommand()
	cmd.PersistentFlags().StringP("base", "p", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.SetIn(strings.NewReader(`{"$ref":"defs/Thing.schema.json"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--stdin", "--stdout", "--no-style"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provide --base")
}

func findYAMLMapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
