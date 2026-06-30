// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	"fmt"
	"sort"
	"strings"

	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"go.yaml.in/yaml/v4"
)

func constNodeValidForType(node *yaml.Node, schemaType string) bool {
	if node == nil {
		return false
	}
	nodeTag := node.ShortTag()
	switch schemaType {
	case "string":
		return nodeTag == "!!str"
	case "integer":
		if nodeTag == "!!int" {
			return true
		}
		if nodeTag == "!!float" {
			return isFloatWhole(node.Value)
		}
		return false
	case "number":
		return nodeTag == "!!int" || nodeTag == "!!float"
	case "boolean":
		return nodeTag == "!!bool"
	case "null":
		return nodeTag == "!!null"
	case "array":
		return nodeTag == "!!seq"
	case "object":
		return nodeTag == "!!map"
	}
	return false
}

func isFloatWhole(value string) bool {
	if !strings.Contains(value, ".") {
		return true
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	for _, char := range parts[1] {
		if char != '0' {
			return false
		}
	}
	return true
}

func containsType(types []string, target string) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}

func stableNodeValue(node *yaml.Node) string {
	value, err := schemautil.NodeToInterface(node)
	if err != nil {
		return node.Value
	}
	return stableValue(value)
}

func stableValue(value any) string {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("map[")
		for i, key := range keys {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(key)
			b.WriteString(":")
			b.WriteString(stableValue(v[key]))
		}
		b.WriteString("]")
		return b.String()
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, stableValue(item))
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return fmt.Sprintf("%#v", value)
	}
}

func sequenceContainsEquivalent(seq *yaml.Node, target *yaml.Node) bool {
	targetValue := stableNodeValue(target)
	for _, item := range seq.Content {
		if stableNodeValue(item) == targetValue {
			return true
		}
	}
	return false
}
