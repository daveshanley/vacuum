// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"net/url"
	"strings"

	"github.com/daveshanley/vacuum/model"
	santhoshjsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v4"
)

const (
	SchemaURL2020 = "https://json-schema.org/draft/2020-12/schema"
	SchemaURL2019 = "https://json-schema.org/draft/2019-09/schema"
	SchemaURL07   = "http://json-schema.org/draft-07/schema#"
)

type Dialect struct {
	Format string
	URL    string
	Draft  *santhoshjsonschema.Draft
}

func DetectDialect(root *yaml.Node) Dialect {
	root = RootNode(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return dialectForFormat(model.JSONSchemaDraft2020)
	}
	schemaURL := strings.TrimSpace(mappingScalarValue(root, "$schema"))
	if schemaURL == "" {
		return dialectForFormat(model.JSONSchemaDraft2020)
	}
	normalized := normalizeSchemaURL(schemaURL)
	switch {
	case strings.Contains(normalized, "draft/2020-12/schema"):
		return dialectForFormat(model.JSONSchemaDraft2020)
	case strings.Contains(normalized, "draft/2019-09/schema"):
		return dialectForFormat(model.JSONSchemaDraft2019)
	case strings.Contains(normalized, "draft-07/schema"):
		return dialectForFormat(model.JSONSchemaDraft07)
	default:
		return Dialect{Format: model.JSONSchema, URL: schemaURL, Draft: santhoshjsonschema.Draft2020}
	}
}

func IsSupportedDialect(format string) bool {
	return format == model.JSONSchemaDraft2020 ||
		format == model.JSONSchemaDraft2019 ||
		format == model.JSONSchemaDraft07
}

func HasSchemaKeyword(root *yaml.Node) bool {
	return mappingValueNode(RootNode(root), "$schema") != nil
}

func EnsureRootSchema(root *yaml.Node, schemaURL string) {
	root = RootNode(root)
	if root == nil || root.Kind != yaml.MappingNode || HasSchemaKeyword(root) {
		return
	}
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "$schema", Line: root.Line, Column: root.Column}
	val := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: schemaURL, Line: root.Line, Column: root.Column}
	root.Content = append([]*yaml.Node{key, val}, root.Content...)
}

func dialectForFormat(format string) Dialect {
	switch format {
	case model.JSONSchemaDraft2019:
		return Dialect{Format: format, URL: SchemaURL2019, Draft: santhoshjsonschema.Draft2019}
	case model.JSONSchemaDraft07:
		return Dialect{Format: format, URL: SchemaURL07, Draft: santhoshjsonschema.Draft7}
	default:
		return Dialect{Format: model.JSONSchemaDraft2020, URL: SchemaURL2020, Draft: santhoshjsonschema.Draft2020}
	}
}

func normalizeSchemaURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "#")
	}
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "#")
}
