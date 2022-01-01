// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
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
		p.match = props["match"]
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

	for _, node := range nodes {

		if p.match != "" {
			rx, err := p.getPatternFromCache(p.match)
			if err != nil {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("'%s' cannot be compiled into a regular expression: %s", p.match, err.Error()),
					StartNode: node,
					EndNode:   node,
					Path:      pathValue,
				})
			} else {
				if !rx.MatchString(node.Value) {
					results = append(results, model.RuleFunctionResult{
						Message:   fmt.Sprintf("'%s' does not match the expression '%s'", node.Value, p.match),
						StartNode: node,
						EndNode:   node,
						Path:      pathValue,
					})
				}
			}
		}

		// not match
		if p.notMatch != "" {
			rx, err := p.getPatternFromCache(p.notMatch)
			if err != nil {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("'%s' cannot be compiled into a regular expression: %s", p.notMatch, err.Error()),
					StartNode: node,
					EndNode:   node,
					Path:      pathValue,
				})
			} else {
				if rx.MatchString(node.Value) {
					results = append(results, model.RuleFunctionResult{
						Message:   fmt.Sprintf("'%s' matches the expression '%s'", node.Value, p.notMatch),
						StartNode: node,
						EndNode:   node,
						Path:      pathValue,
					})
				}
			}
		}
	}
	return results
}

func (p Pattern) getPatternFromCache(pattern string) (*regexp.Regexp, error) {
	if pat, ok := p.patternCache[pattern]; ok {
		return pat, nil
	}
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	p.patternCache[pattern] = rx
	return rx, nil
}
