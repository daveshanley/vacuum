// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// BuildResult creates a schema rule result using Doctor and libopenapi low-node locations.
func BuildResult(message, path, violationProperty string, segment int, schema *drV3.Schema, node *yaml.Node,
	context *model.RuleFunctionContext,
) model.RuleFunctionResult {
	if node == nil {
		node = schemaNode(schema)
	}
	if node == nil {
		node = &yaml.Node{Line: 1, Column: 1}
	}

	locatedPath := path
	var allPaths []string
	if schema != nil {
		locatedPath, allPaths = vacuumUtils.LocateSchemaPropertyPaths(*context, schema, node, node)
	}
	if locatedPath == "" {
		locatedPath = path
	}
	if locatedPath == "" {
		locatedPath = "$"
	}

	if violationProperty != "" {
		if segment >= 0 {
			locatedPath = model.GetStringTemplates().BuildPropertyArrayPath(locatedPath, violationProperty, segment)
		} else {
			locatedPath = model.GetStringTemplates().BuildJSONPath(locatedPath, violationProperty)
		}

		if len(allPaths) > 1 {
			updatedPaths := make([]string, len(allPaths))
			for i, p := range allPaths {
				if segment >= 0 {
					updatedPaths[i] = model.GetStringTemplates().BuildPropertyArrayPath(p, violationProperty, segment)
				} else {
					updatedPaths[i] = model.GetStringTemplates().BuildJSONPath(p, violationProperty)
				}
			}
			allPaths = updatedPaths
		}
	}

	result := model.RuleFunctionResult{
		Message:   message,
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	if schema != nil {
		schema.AddRuleFunctionResult(drV3.ConvertRuleResult(&result))
	}
	return result
}

func schemaNode(schema *drV3.Schema) *yaml.Node {
	if schema == nil {
		return nil
	}
	if schema.GetKeyNode() != nil {
		return schema.GetKeyNode()
	}
	if schema.Value != nil && schema.Value.GoLow() != nil {
		return schema.Value.GoLow().RootNode
	}
	return nil
}
