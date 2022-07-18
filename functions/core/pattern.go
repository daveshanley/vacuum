// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
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
		ErrorMessage:  "'pattern' needs 'match' or 'notMatch' properties being set to operate",
	}
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

	// if multiple patterns are being pulled in, unpack them
	if len(nodes) == 1 && len(nodes[0].Content) > 0 {
		nodes = nodes[0].Content
	}

	for x, node := range nodes {
		if utils.IsNodeMap(node) {
			continue
		}
		if p.match != "" {
			rx, err := p.getPatternFromCache(p.match, context.Rule)
			if err != nil {
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("%s: '%s' cannot be compiled into a regular expression: %s",
						context.Rule.Description, p.match, err.Error()),
					StartNode: node,
					EndNode:   node,
					Path:      pathValue,
					Rule:      context.Rule,
				})
			} else {

				// if a field is supplied, use that, if not then use the raw node value.
				matchValue := node.Value
				if context.RuleAction.Field != "" && x+1 <= len(nodes) {
					if x < len(nodes)-1 {
						_, fieldValue := utils.FindKeyNode(context.RuleAction.Field, nodes[x+1].Content)
						if fieldValue != nil {
							matchValue = fieldValue.Value
							pathValue = fmt.Sprintf("%s.%s", pathValue, context.RuleAction.Field)
						}
					}
				}

				if !rx.MatchString(matchValue) {
					results = append(results, model.RuleFunctionResult{
						Message: fmt.Sprintf("%s: '%s' does not match the expression '%s'", context.Rule.Description,
							matchValue, p.match),
						StartNode: node,
						EndNode:   node,
						Path:      utils.BuildPath(pathValue, []string{node.Value}),
						Rule:      context.Rule,
					})
				}
			}
		}

		// not match
		if p.notMatch != "" {
			rx, err := p.getPatternFromCache(p.notMatch, context.Rule)
			if err != nil {
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("%s: cannot be compiled into a regular expression: %s",
						context.Rule.Description, err.Error()),
					StartNode: node,
					EndNode:   node,
					Path:      pathValue,
					Rule:      context.Rule,
				})
			} else {
				if rx.MatchString(node.Value) {
					results = append(results, model.RuleFunctionResult{
						Message:   fmt.Sprintf("%s: matches the expression '%s'", context.Rule.Description, p.notMatch),
						StartNode: node,
						EndNode:   node,
						Path:      utils.BuildPath(pathValue, []string{node.Value}),
						Rule:      context.Rule,
					})
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
