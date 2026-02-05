// Copyright 2024 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"go.yaml.in/yaml/v4"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
)

// DuplicatePaths checks for duplicate path definitions in the OpenAPI spec.
// This catches the case where YAML allows duplicate keys but only keeps the last one,
// which can lead to unintentional loss of path definitions.
type DuplicatePaths struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DuplicatePaths rule.
func (dp DuplicatePaths) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "duplicatePaths",
	}
}

// GetCategory returns the category of the DuplicatePaths rule.
func (dp DuplicatePaths) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the DuplicatePaths rule, based on supplied context and a supplied []*yaml.Node slice.
func (dp DuplicatePaths) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if len(nodes) <= 0 {
		return results
	}

	// find the paths node
	var pathsNode *yaml.Node
	for _, node := range nodes {
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			node = node.Content[0] // get the root mapping node
		}

		if node.Kind == yaml.MappingNode {
			for i := 0; i < len(node.Content); i += 2 {
				if i+1 < len(node.Content) {
					keyNode := node.Content[i]
					valueNode := node.Content[i+1]

					if keyNode.Value == "paths" {
						pathsNode = valueNode
						break
					}
				}
			}
		}

		if pathsNode != nil {
			break
		}
	}

	if pathsNode == nil || pathsNode.Kind != yaml.MappingNode {
		return results
	}

	// track seen paths
	seenPaths := make(map[string][]*yaml.Node)

	// check all path keys for duplicates
	for i := 0; i < len(pathsNode.Content); i += 2 {
		if i+1 < len(pathsNode.Content) {
			keyNode := pathsNode.Content[i]
			pathKey := keyNode.Value

			// record this path occurrence
			seenPaths[pathKey] = append(seenPaths[pathKey], keyNode)
		}
	}

	// report duplicates
	for pathKey, occurrences := range seenPaths {
		if len(occurrences) > 1 {
			// report each duplicate after the first one
			for i := 1; i < len(occurrences); i++ {
				node := occurrences[i]
				res := model.BuildFunctionResultString(
					fmt.Sprintf("duplicate path '%s' found; only the last definition will be used, previous definitions are ignored", pathKey))
				res.StartNode = node
				res.EndNode = vacuumUtils.BuildEndNode(node)
				res.Path = fmt.Sprintf("$.paths['%s']", pathKey)
				res.Rule = context.Rule
				results = append(results, res)
			}
		}
	}

	return results
}
