// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
	"regexp"
	"strings"
)

// Pattern is a rule that will match or not match (or both) a regular expression.
// This struct is stateless - all state is passed through function parameters.
type Pattern struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Pattern rule.
func (p Pattern) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "pattern",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "match",
				Description: "'pattern' requires a match",
			},
			{
				Name:        "notMatch",
				Description: "'pattern' needs something to not match against",
			},
		},
		MinProperties: 1,
		MaxProperties: 2,
		ErrorMessage:  "'pattern' needs 'match' or 'notMatch' function options being set to operate",
	}
}

// GetCategory returns the category of the Pattern rule.
func (p Pattern) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Pattern rule, based on supplied context and a supplied []*yaml.Node slice.
func (p Pattern) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	// extract match/notMatch from props - these are local variables, not struct state
	props := context.GetOptionsStringMap()
	match := props["match"]
	notMatch := props["notMatch"]

	// make sure we have something to look at
	if match == "" && notMatch == "" {
		return nil
	}

	// compile regexes once at the start - no caching needed since we compile per-invocation
	var matchRx, notMatchRx *regexp.Regexp
	var matchErr, notMatchErr error

	if match != "" {
		if context.Rule.PrecompiledPattern != nil {
			matchRx = context.Rule.PrecompiledPattern
		} else {
			matchRx, matchErr = regexp.Compile(match)
		}
	}

	if notMatch != "" {
		// PrecompiledPattern is typically for 'match', compile notMatch separately
		notMatchRx, notMatchErr = regexp.Compile(notMatch)
	}

	var results []model.RuleFunctionResult

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	message := context.Rule.Message
	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	// check if field contains nested path syntax
	fieldHasNestedPath := context.RuleAction.Field != "" && strings.IndexAny(context.RuleAction.Field, ".[]\\") != -1

	// if multiple patterns are being pulled in, unpack them
	if len(nodes) == 1 && len(nodes[0].Content) > 0 {
		nodes = nodes[0].Content
	}

	// handle nested field paths using FindFieldPath
	if fieldHasNestedPath {
		for _, node := range nodes {
			if utils.IsNodeMap(node) {
				result := vacuumUtils.FindFieldPath(context.RuleAction.Field, node.Content, vacuumUtils.FieldPathOptions{})
				if result.Found && result.ValueNode != nil {
					results = append(results, p.validatePatternOnNode(
						result.ValueNode, pathValue, message, ruleMessage,
						match, notMatch, matchRx, notMatchRx, matchErr, notMatchErr, context)...)
				}
			}
		}
		return results
	}

	// iterate through key-value pairs
	var currentField string
	for x, node := range nodes {
		if utils.IsNodeMap(node) {
			continue
		}
		if x%2 == 0 {
			currentField = node.Value
			if context.RuleAction.Field != "" {
				continue
			}
		}
		if context.RuleAction.Field != "" && currentField != context.RuleAction.Field {
			continue
		}

		results = append(results, p.validatePatternOnNode(
			node, pathValue, message, ruleMessage,
			match, notMatch, matchRx, notMatchRx, matchErr, notMatchErr, context)...)
	}

	return results
}

// validatePatternOnNode checks both match and notMatch patterns on a node value.
// All parameters are passed explicitly - no struct state is used.
func (p Pattern) validatePatternOnNode(
	node *yaml.Node,
	pathValue, message, ruleMessage string,
	match, notMatch string,
	matchRx, notMatchRx *regexp.Regexp,
	matchErr, notMatchErr error,
	context model.RuleFunctionContext,
) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// check match pattern
	if match != "" {
		if matchErr != nil {
			results = append(results, p.buildRegexErrorResult(node, pathValue, message, ruleMessage, match, matchErr, context))
		} else if !matchRx.MatchString(node.Value) {
			results = append(results, p.buildPatternMismatchResult(node, pathValue, message, ruleMessage, match, node.Value, context))
		}
	}

	// check notMatch pattern
	if notMatch != "" {
		if notMatchErr != nil {
			results = append(results, p.buildRegexErrorResult(node, pathValue, message, ruleMessage, notMatch, notMatchErr, context))
		} else if notMatchRx.MatchString(node.Value) {
			results = append(results, p.buildPatternMatchedResult(node, pathValue, message, ruleMessage, notMatch, context))
		}
	}

	return results
}

// buildRegexErrorResult creates a result for regex compilation errors.
func (p Pattern) buildRegexErrorResult(
	node *yaml.Node,
	pathValue, message, ruleMessage, pattern string,
	err error,
	context model.RuleFunctionContext,
) model.RuleFunctionResult {
	locatedPath, allPaths, locatedObjects := p.locateNode(node, pathValue, context)

	result := model.RuleFunctionResult{
		Message: vacuumUtils.SuppliedOrDefault(message,
			model.GetStringTemplates().BuildRegexCompileErrorMessage(ruleMessage, pattern, err.Error())),
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	p.attachResultToModel(locatedObjects, &result)
	return result
}

// buildPatternMismatchResult creates a result when a value doesn't match the expected pattern.
func (p Pattern) buildPatternMismatchResult(
	node *yaml.Node,
	pathValue, message, ruleMessage, pattern, value string,
	context model.RuleFunctionContext,
) model.RuleFunctionResult {
	locatedPath, allPaths, locatedObjects := p.locateNode(node, pathValue, context)

	result := model.RuleFunctionResult{
		Message: vacuumUtils.SuppliedOrDefault(message,
			model.GetStringTemplates().BuildPatternMessage(ruleMessage, value, pattern)),
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	p.attachResultToModel(locatedObjects, &result)
	return result
}

// buildPatternMatchedResult creates a result when a value matches a notMatch pattern.
func (p Pattern) buildPatternMatchedResult(
	node *yaml.Node,
	pathValue, message, ruleMessage, pattern string,
	context model.RuleFunctionContext,
) model.RuleFunctionResult {
	locatedPath, allPaths, locatedObjects := p.locateNode(node, pathValue, context)

	result := model.RuleFunctionResult{
		Message: vacuumUtils.SuppliedOrDefault(message,
			model.GetStringTemplates().BuildPatternMatchMessage(ruleMessage, pattern)),
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	p.attachResultToModel(locatedObjects, &result)
	return result
}

// locateNode finds the location information for a node using DrDocument.
func (p Pattern) locateNode(node *yaml.Node, pathValue string, context model.RuleFunctionContext) (string, []string, []v3.Foundational) {
	locatedPath := pathValue
	var allPaths []string
	var locatedObjects []v3.Foundational

	if context.DrDocument != nil {
		var err error
		locatedObjects, err = context.DrDocument.LocateModel(node)
		if err == nil && locatedObjects != nil {
			for i, obj := range locatedObjects {
				if i == 0 {
					locatedPath = obj.GenerateJSONPath()
				}
				allPaths = append(allPaths, obj.GenerateJSONPath())
			}
		}
	}

	return locatedPath, allPaths, locatedObjects
}

// attachResultToModel attaches the result to the first located model if it accepts results.
func (p Pattern) attachResultToModel(locatedObjects []v3.Foundational, result *model.RuleFunctionResult) {
	if len(locatedObjects) > 0 {
		if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
			arr.AddRuleFunctionResult(v3.ConvertRuleResult(result))
		}
	}
}
