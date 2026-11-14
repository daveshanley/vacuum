// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// Includes adaptations of @openlint/ruleset-openapi 0.1.0.
// Copyright 2018 Stoplight, Inc. and 2026 OpenLint.
// See rulesets/schemas/OPENLINT-OPENAPI-LICENSE.
// SPDX-License-Identifier: MIT AND Apache-2.0

package openapi

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
)

type openLintValueFunction string

func (f openLintValueFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "openlint:openapi/" + string(f)}
}

func (openLintValueFunction) GetCategory() string { return model.FunctionCategoryOpenAPI }

func (f openLintValueFunction) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) != 1 || nodes[0] == nil {
		return nil
	}
	path, _ := ctx.Given.(string)
	v := packagedOperand{nodes[0], path}
	var results packagedDiagnostics
	switch f {
	case "refSiblings":
		// The packaged selector uses the JSONPath parent operator. Each
		// match is the mapping containing the $ref, including at the root.
		if v.field("$ref").node == nil {
			return nil
		}
		for key, sibling := range v.fields() {
			if key != "$ref" {
				results.add(sibling, "$ref must not be placed next to any other properties")
			}
		}
	case "uniquenessTags":
		seen := make(map[string]bool, len(v.node.Content))
		for tag := range v.items() {
			name := tag.field("name")
			if !name.isText() {
				// The published function only accepts arrays of named tags.
				return nil
			}
			if seen[name.text()] {
				results.add(name, fmt.Sprintf(`"tags" object contains duplicate tag name "%s".`, name.text()))
			}
			seen[name.text()] = true
		}
	case "typedEnum":
		return openLintEnumFindings(v, ctx)
	}
	return results
}

func openLintEnumFindings(schema packagedOperand, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	values, typ := schema.field("enum"), schema.field("type")
	if !values.is(yaml.SequenceNode) || typ.node == nil {
		return nil
	}
	var types []string
	switch typ.node.ShortTag() {
	case "!!str":
		types = []string{typ.text()}
	case "!!seq":
		for item := range typ.items() {
			if !item.isText() {
				return nil
			}
			types = append(types, item.text())
		}
	default:
		return nil
	}
	root := ctx.ExecutionRoot
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
		root = root.Content[0]
	}
	nullableKey := ""
	if version := findMappingValue(root, "openapi"); version != nil && (version.Value == "3.0" || strings.HasPrefix(version.Value, "3.0.")) {
		nullableKey = "nullable"
	} else if version := findMappingValue(root, "swagger"); version != nil && strings.HasPrefix(version.Value, "2") {
		nullableKey = "x-nullable"
	}
	if nullable := findMappingValue(schema.node, nullableKey); nullableKey != "" && nullable != nil && nullable.ShortTag() == "!!bool" && strings.EqualFold(nullable.Value, "true") {
		types = append(types, "null")
	}

	checkInteger := slices.Contains(types, "integer")
	var results packagedDiagnostics
	for item := range values.items() {
		value := item.node
		kind := "string"
		switch value.ShortTag() {
		case "!!map":
			kind = "object"
		case "!!seq":
			kind = "array"
		case "!!null":
			kind = "null"
		case "!!bool":
			kind = "boolean"
		case "!!float", "!!int":
			kind = "number"
			if checkInteger {
				var number float64
				if value.Decode(&number) == nil && math.Trunc(number) == number {
					kind = "integer"
				}
			}
		}
		if slices.Contains(types, kind) {
			continue
		}
		var printed string
		switch kind {
		case "array":
			printed = "Array[]"
		case "object":
			printed = "Object{}"
		default:
			var scalar any
			if err := value.Decode(&scalar); err != nil {
				return nil
			}
			data, err := json.Marshal(scalar)
			if err != nil {
				return nil
			}
			printed = string(data)
		}
		results.add(item, fmt.Sprintf(`Enum value %s must be "%s".`, printed, strings.Join(types, ",")))
	}
	return results
}
