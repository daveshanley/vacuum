// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

// UnusedComponents reports reusable AsyncAPI components that are never
// referenced from outside their own component definition.
type UnusedComponents struct{}

// GetSchema returns the AsyncAPI unused-components function schema.
func (u UnusedComponents) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiUnusedComponents"}
}

// GetCategory returns the AsyncAPI function category.
func (u UnusedComponents) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule walks every known AsyncAPI reusable component map and compares keys
// against `$ref` values found outside that component map.
func (u UnusedComponents) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	root := rootNode(context)
	_, components := mappingValue(root, "components")
	if components == nil {
		return nil
	}

	refs := make(map[string]bool)
	collectRefsOutsideComponents(root, refs)

	sweeps := []string{
		"schemas",
		"servers",
		"channels",
		"operations",
		"messages",
		"securitySchemes",
		"serverVariables",
		"parameters",
		"correlationIds",
		"replies",
		"replyAddresses",
		"operationTraits",
		"messageTraits",
		"serverBindings",
		"channelBindings",
		"operationBindings",
		"messageBindings",
	}
	if options := context.GetOptionsStringMap(); options["location"] != "" {
		sweeps = []string{options["location"]}
	}

	var results []model.RuleFunctionResult
	for _, location := range sweeps {
		_, reusableMap := mappingValue(components, location)
		for _, entry := range mappingEntries(reusableMap) {
			ref := "#/components/" + location + "/" + entry[0].Value
			if !refs[ref] {
				results = append(results, result(
					context,
					entry[0],
					nodePath(context, entry[0], ""),
					fmt.Sprintf("Potentially unused AsyncAPI component `%s` was detected.", ref),
				))
			}
		}
	}
	return results
}

func collectRefsOutsideComponents(root *yaml.Node, refs map[string]bool) {
	if root == nil || root.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		key := root.Content[i]
		value := root.Content[i+1]
		if key.Value == "components" {
			collectRefsFromComponentValues(value, refs)
			continue
		}
		collectRefs(value, refs)
	}
}

func collectRefsFromComponentValues(components *yaml.Node, refs map[string]bool) {
	for _, location := range mappingEntries(components) {
		componentType := location[0].Value
		for _, entry := range mappingEntries(location[1]) {
			ref := "#/components/" + componentType + "/" + entry[0].Value
			collectRefsExcept(entry[1], refs, ref)
		}
	}
}

func collectRefsExcept(node *yaml.Node, refs map[string]bool, excludedRef string) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if key.Value == "$ref" && value.Value != "" && value.Value != excludedRef {
				refs[value.Value] = true
			}
			collectRefsExcept(value, refs, excludedRef)
		}
		return
	}
	for _, child := range node.Content {
		collectRefsExcept(child, refs, excludedRef)
	}
}
