// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/utils"
)

// Truthy is a rule that will determine if something is seen as 'true' (could be a 1 or "pizza", or actually 'true')
type Truthy struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Truthy rule.
func (t Truthy) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "truthy",
	}
}

// GetCategory returns the category of the Truthy rule.
func (t Truthy) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Truthy rule, based on supplied context and a supplied []*yaml.Node slice.
// If no field is specified, the function checks if the matched node itself is falsy (and reports if so).
// If a field is specified, the function checks if that field within the matched node is falsy.
// A matched scalar has no fields, so it is checked itself, like Spectral.
func (t *Truthy) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	isArray := false
	if len(nodes) == 1 && utils.IsNodeArray(nodes[0]) && len(nodes[0].Content) > 0 && context.Contract != model.OpenLintV1Draft {
		nodes = nodes[0].Content
		isArray = true
	}

	ruleMessage := context.Rule.Description
	message := context.Rule.Message

	for x, node := range nodes {
		if context.Contract == model.OpenLintV1Draft && node == nil {
			results = append(results, missingOpenLintValue(context))
			continue
		}

		if node.Kind == yaml.DocumentNode {
			node = node.Content[0]
		}

		var targetNode *yaml.Node
		var fieldNode *yaml.Node
		var fieldName string
		var fieldResult vacuumUtils.FieldPathResult

		// Like Spectral, a matched scalar has no fields, so 'field' does not apply to it.
		checkNode := context.RuleAction.Field == "" || (!isArray && node.Kind == yaml.ScalarNode)

		if checkNode {
			// no field to look up - check the matched node itself
			targetNode = node
			fieldName = "value"
		} else {
			// field specified - find it within the node (supports nested paths like "properties.data")
			fieldResult = vacuumUtils.FindFieldPath(context.RuleAction.Field, node.Content, fieldLookupOptions(context, false))
			fieldNode, targetNode = fieldResult.KeyNode, fieldResult.ValueNode
			fieldName = context.RuleAction.Field
		}

		// check if the target is falsy (which means truthy check fails)
		if falsyForContract(targetNode, context.Contract) {
			if isArray {
				pathValue = model.GetStringTemplates().BuildArrayPath(pathValue, x)
			}

			// skip if target is a complex type (map or array with content)
			if targetNode != nil && (utils.IsNodeMap(targetNode) || utils.IsNodeArray(targetNode)) && len(targetNode.Content) > 0 {
				continue
			}

			var locatedObjects []v3.Foundational
			var allPaths []string
			locatedPath := pathValue
			if context.DrDocument != nil || context.Arazzo != nil || context.AsyncAPI != nil {
				if checkNode {
					locatedPath, allPaths, locatedObjects = locateNodePaths(&context, node)
				} else if fieldNode == nil {
					basePath, basePaths, baseObjects := locateNodePaths(&context, node)
					locatedPath, allPaths, locatedObjects = appendFieldPathToLocatedPaths(
						basePath, basePaths, baseObjects, context.RuleAction.Field,
					)
				} else {
					locatedPath, allPaths, locatedObjects = locateExistingFieldPaths(&context, node, context.RuleAction.Field, fieldResult)
				}
			}
			result := model.RuleFunctionResult{
				Message: vacuumUtils.SuppliedOrDefault(message,
					model.GetStringTemplates().BuildFieldValidationMessage(ruleMessage, fieldName, "set")),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      locatedPath,
				Rule:      context.Rule,
			}
			// Retain the authored owner for nested missing fields. Display ranges
			// remain unchanged; repair planners must not infer a target from text.
			if !checkNode {
				result.RepairNode = truthyFieldOwner(node, context.RuleAction.Field, fieldLookupOptions(context, false))
				// Keep the full field expression: two missing nested fields can
				// share an owner while still being different diagnostics.
				result.DiagnosticCode = context.RuleAction.Field + "-missing"
			} else {
				result.RepairNode = node
			}
			if len(allPaths) > 1 {
				result.Paths = allPaths
			}
			results = append(results, result)
			if len(locatedObjects) > 0 {
				if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
					arr.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				}
			}
		}
	}
	return results
}

// isFalsyNode checks if a YAML node represents a falsy value.
// Null, missing values and empty collections are falsy.
func isFalsyNode(node *yaml.Node) bool {
	if node == nil {
		return true
	}
	if node.Tag == "!!null" {
		return true
	}
	if node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode {
		return len(node.Content) == 0
	}
	// node with content (map or array) is truthy, not falsy
	if len(node.Content) > 0 {
		return false
	}
	// scalar values: check for falsy values
	if node.Value == "" || node.Value == "false" || node.Value == "0" {
		return true
	}
	return false
}

func falsyForContract(node *yaml.Node, contract model.Contract) bool {
	if contract != model.OpenLintV1Draft {
		return isFalsyNode(node)
	}
	if node == nil || node.Tag == "!!null" {
		return true
	}
	if node.Kind != yaml.ScalarNode {
		return false
	}
	switch node.Tag {
	case "!!str":
		return node.Value == ""
	case "!!bool":
		return node.Value == "false"
	case "!!int", "!!float":
		var n float64
		return node.Decode(&n) == nil && n == 0
	}
	return false
}

func missingOpenLintValue(context model.RuleFunctionContext) model.RuleFunctionResult {
	return model.RuleFunctionResult{Rule: context.Rule, Path: givenPathValue(context.Given), Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message, "value must be defined")}
}

// truthyFieldOwner follows the same parsed field path and combinator rules as
// the validation lookup. An absent intermediate owner is not a repair target.
func truthyFieldOwner(root *yaml.Node, field string, opts vacuumUtils.FieldPathOptions) *yaml.Node {
	segments, err := vacuumUtils.ParseFieldPath(field)
	if err != nil || len(segments) == 0 {
		return nil
	}
	if len(segments) == 1 {
		return root
	}
	var prefix strings.Builder
	for i, segment := range segments[:len(segments)-1] {
		if segment.Type == vacuumUtils.SegmentArrayIndex {
			prefix.WriteString("[" + strconv.Itoa(segment.Index) + "]")
		} else {
			if i > 0 {
				prefix.WriteByte('.')
			}
			prefix.WriteString(strings.NewReplacer(`\`, `\\`, `.`, `\.`).Replace(segment.Key))
		}
	}
	return vacuumUtils.FindFieldPath(prefix.String(), root.Content, opts).ValueNode
}
