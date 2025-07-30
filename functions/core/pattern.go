// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
    "regexp"
)

// Pattern is a rule that will match or not match (or both) a regular expression.
type Pattern struct {
	match        string
	notMatch     string
	patternCache map[string]*regexp.Regexp
}

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

	// check supplied type - use cached options to avoid repeated interface conversions
	props := context.GetOptionsStringMap()

	if props["match"] != "" {
		p.match = props["match"] // TODO: there should be no state in here, clean this up.
	}

	if props["notMatch"] != "" {
		p.notMatch = props["notMatch"]
	}

	// make sure we have something to look at.
	if p.match == "" && p.notMatch == "" {
		return nil
	}

	var results []model.RuleFunctionResult

	if p.patternCache == nil {
		p.patternCache = make(map[string]*regexp.Regexp)
	}

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	message := context.Rule.Message

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	// if multiple patterns are being pulled in, unpack them
	if len(nodes) == 1 && len(nodes[0].Content) > 0 {
		nodes = nodes[0].Content
	}

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
			continue // not what we're looking for.
		}
		if p.match != "" {
			rx, err := p.getPatternFromCache(p.match, context.Rule)
			expPath := model.GetStringTemplates().BuildQuotedPath(pathValue, currentField)
			if err != nil {
				locatedObjects, lErr := context.DrDocument.LocateModel(node)
				locatedPath := expPath
				var allPaths []string
				if lErr == nil && locatedObjects != nil {
					for d, obj := range locatedObjects {
						if d == 0 {
							locatedPath = obj.GenerateJSONPath()
						}
						allPaths = append(allPaths, obj.GenerateJSONPath())
					}
				}
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(message,
						model.GetStringTemplates().BuildRegexCompileErrorMessage(ruleMessage, p.match, err.Error())),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      locatedPath,
					Rule:      context.Rule,
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
			} else {
				if !rx.MatchString(node.Value) {
					locatedObjects, lErr := context.DrDocument.LocateModel(node)
					locatedPath := pathValue
					var allPaths []string
					if lErr == nil && locatedObjects != nil {
						for s, obj := range locatedObjects {
							if s == 0 {
								locatedPath = obj.GenerateJSONPath()
							}
							allPaths = append(allPaths, obj.GenerateJSONPath())
						}
					}
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(message,
							model.GetStringTemplates().BuildPatternMessage(ruleMessage, node.Value, p.match)),
						StartNode: node,
						EndNode:   vacuumUtils.BuildEndNode(node),
						Path:      locatedPath,
						Rule:      context.Rule,
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
		}

		// not match
		if p.notMatch != "" {
			rx, err := p.getPatternFromCache(p.notMatch, context.Rule)
			expPath := model.GetStringTemplates().BuildQuotedPath(pathValue, currentField)
			if err != nil {
				locatedObjects, lErr := context.DrDocument.LocateModel(node)
				locatedPath := expPath
				var allPaths []string
				if lErr == nil && locatedObjects != nil {
					for s, obj := range locatedObjects {
						if s == 0 {
							locatedPath = obj.GenerateJSONPath()
						}
						allPaths = append(allPaths, obj.GenerateJSONPath())
					}
				}
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(message,
						model.GetStringTemplates().BuildRegexCompileErrorMessage(ruleMessage, p.notMatch, err.Error())),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      locatedPath,
					Rule:      context.Rule,
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
			} else {
				if rx.MatchString(node.Value) {

					locatedObjects, lErr := context.DrDocument.LocateModel(node)
					locatedPath := pathValue
					var allPaths []string
					if lErr == nil && locatedObjects != nil {
						for s, obj := range locatedObjects {
							if s == 0 {
								locatedPath = obj.GenerateJSONPath()
							}
							allPaths = append(allPaths, obj.GenerateJSONPath())
						}
					}

					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(message,
							model.GetStringTemplates().BuildPatternMatchMessage(ruleMessage, p.notMatch)),
						StartNode: node,
						EndNode:   vacuumUtils.BuildEndNode(node),
						Path:      locatedPath,
						Rule:      context.Rule,
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
		}
	}
	return results
}

func (p Pattern) getPatternFromCache(pattern string, rule *model.Rule) (*regexp.Regexp, error) {
	if pat, ok := p.patternCache[pattern]; ok {
		return pat, nil
	}
	var rx *regexp.Regexp
	var err error

	// if we're using a built-in rule, we should have already compiled this.
	if rule.PrecompiledPattern != nil {
		rx = rule.PrecompiledPattern
	} else {
		rx, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}
	p.patternCache[pattern] = rx
	return rx, nil
}
