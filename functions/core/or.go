// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	openapiUtils "github.com/pb33f/libopenapi/utils"
)

const orOptionsError = "'or' requires at least two property names in 'properties', for example: [title, summary, description]"

// Or requires at least one named property to exist on an object, regardless of its value.
type Or struct{}

// GetSchema describes the options accepted by Or.
func (o Or) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "or",
		Required: []string{"properties"},
		Properties: []model.RuleFunctionProperty{{
			Name:        "properties",
			Description: "At least two property names, as a string array or comma-separated string.",
		}},
		ErrorMessage: orOptionsError,
	}
}

// GetCategory returns the core function category.
func (o Or) GetCategory() string { return model.FunctionCategoryCore }

// ValidateOptions requires at least two string property names.
func (o Or) ValidateOptions(options any) error {
	if len(orProperties(options)) < 2 {
		return errors.New(orOptionsError)
	}
	return nil
}

// RunRule checks each object independently and reports objects with no listed property.
// Non-object inputs are ignored, as in Spectral.
func (o Or) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) == 0 {
		return nil
	}
	pathValue := givenPathValue(context.Given)
	field := ""
	if context.RuleAction != nil {
		field = context.RuleAction.Field
	}

	properties := orProperties(context.Options)
	if len(properties) < 2 {
		return []model.RuleFunctionResult{{Message: orOptionsError, Rule: context.Rule, Path: pathValue, StartNode: &yaml.Node{}, EndNode: &yaml.Node{}}}
	}

	var results []model.RuleFunctionResult
	var message string
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			node = node.Content[0]
		}
		objectNode := openapiUtils.NodeMerge([]*yaml.Node{node})
		if objectNode == nil {
			continue
		}
		if field != "" {
			selected := vacuumUtils.FindFieldPath(field, objectNode.Content, fieldLookupOptions(context, false))
			node = selected.ValueNode
			if node == nil {
				continue
			}
			objectNode = openapiUtils.NodeMerge([]*yaml.Node{node})
		}
		if objectNode == nil || objectNode.Kind != yaml.MappingNode {
			continue
		}
		found := false
		for i := 0; i+1 < len(objectNode.Content); i += 2 {
			if slices.Contains(properties, objectNode.Content[i].Value) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		if message == "" {
			if context.Rule != nil {
				message = context.Rule.Message
			}
			if message == "" {
				message = orMissingMessage(properties)
			}
		}
		fallbackPath := pathValue
		if field != "" {
			fallbackPath = joinJSONPath(pathValue, field)
		}
		nodeContext := context
		nodeContext.Given = fallbackPath
		locatedPath, allPaths, locatedObjects := locateNodePaths(nodeContext, node)
		// Use the selected object's exact path; Doctor can locate its parent or an alias.
		if context.Index != nil {
			paths := vacuumUtils.NodePathIndexForContext(context, context.Index.GetRootNode())
			if exactPath, ok := paths.Lookup(node); ok {
				locatedPath, allPaths = exactPath, nil
			}
		}

		result := model.RuleFunctionResult{
			Message:           message,
			StartNode:         node,
			EndNode:           vacuumUtils.BuildEndNode(node),
			Path:              locatedPath,
			Rule:              context.Rule,
			PathFromRuleGiven: locatedPath == fallbackPath,
		}
		if len(allPaths) > 1 {
			result.Paths = allPaths
		}
		results = append(results, result)
		addResultToLocatedModel(locatedObjects, &result)
	}
	return results
}

func orProperties(options any) []string {
	var value any
	switch opts := options.(type) {
	case map[string]any:
		value = opts["properties"]
	case map[string]string:
		value = opts["properties"]
	}
	switch properties := value.(type) {
	case []string:
		return properties
	case []any:
		names := make([]string, len(properties))
		for i, property := range properties {
			name, ok := property.(string)
			if !ok {
				return nil
			}
			names[i] = name
		}
		return names
	case string:
		names := strings.Split(properties, ",")
		count := 0
		for _, name := range names {
			if name = strings.TrimSpace(name); name != "" {
				names[count] = name
				count++
			}
		}
		return names[:count]
	}
	return nil
}

func orMissingMessage(properties []string) string {
	if len(properties) > 4 {
		return `At least one of "` + strings.Join(properties[:3], `" or "`) + `" or ` +
			strconv.Itoa(len(properties)-3) + " other properties must be defined"
	}
	return `At least one of "` + strings.Join(properties, `" or "`) + `" must be defined`
}
