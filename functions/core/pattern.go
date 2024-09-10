// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
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

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)

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
			expPath := fmt.Sprintf("%s['%s']", pathValue, currentField)
			if err != nil {
				locatedObject, er := context.DrDocument.LocateModel(node)
				locatedPath := expPath
				if er == nil && locatedObject != nil {
					locatedPath = locatedObject.GenerateJSONPath()
				}
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(message,
						fmt.Sprintf("%s: `%s` cannot be compiled into a regular expression [`%s`]",
							ruleMessage, p.match, err.Error())),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      locatedPath,
					Rule:      context.Rule,
				}
				results = append(results, result)
				if arr, ok := locatedObject.(base.AcceptsRuleResults); ok {
					arr.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				}
			} else {
				if !rx.MatchString(node.Value) {
					locatedObject, er := context.DrDocument.LocateModel(node)
					locatedPath := expPath
					if er == nil && locatedObject != nil {
						locatedPath = locatedObject.GenerateJSONPath()
					}
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(message,
							fmt.Sprintf("%s: `%s` does not match the expression `%s`", ruleMessage,
								node.Value, p.match)),
						StartNode: node,
						EndNode:   vacuumUtils.BuildEndNode(node),
						Path:      locatedPath,
						Rule:      context.Rule,
					}
					results = append(results, result)
					if arr, ok := locatedObject.(base.AcceptsRuleResults); ok {
						arr.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					}
				}
			}
		}

		// not match
		if p.notMatch != "" {
			rx, err := p.getPatternFromCache(p.notMatch, context.Rule)
			expPath := fmt.Sprintf("%s['%s']", pathValue, currentField)
			if err != nil {
				locatedObject, er := context.DrDocument.LocateModel(node)
				locatedPath := expPath
				if er == nil && locatedObject != nil {
					locatedPath = locatedObject.GenerateJSONPath()
				}
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(message,
						fmt.Sprintf("%s: cannot be compiled into a regular expression [`%s`]",
							ruleMessage, err.Error())),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      locatedPath,
					Rule:      context.Rule,
				}
				results = append(results, result)
				if arr, ok := locatedObject.(base.AcceptsRuleResults); ok {
					arr.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				}
			} else {
				if rx.MatchString(node.Value) {
					locatedObject, er := context.DrDocument.LocateModel(node)
					locatedPath := expPath
					if er == nil && locatedObject != nil {
						locatedPath = locatedObject.GenerateJSONPath()
					}
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(message,
							fmt.Sprintf("%s: matches the expression `%s`", ruleMessage, p.notMatch)),
						StartNode: node,
						EndNode:   vacuumUtils.BuildEndNode(node),
						Path:      locatedPath,
						Rule:      context.Rule,
					}
					results = append(results, result)
					if arr, ok := locatedObject.(base.AcceptsRuleResults); ok {
						arr.AddRuleFunctionResult(base.ConvertRuleResult(&result))
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
