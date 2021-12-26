package core

import (
	"fmt"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
	"regexp"
)

type Pattern struct {
	match        string
	notMatch     string
	patternCache map[string]*regexp.Regexp
}

func (p Pattern) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{}
}

func (p Pattern) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) != 1 { // there can only be a single node passed in to this function.
		return nil
	}

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

	// match
	if p.match != "" {
		rx, err := p.getPatternFromCache(p.match)
		if err != nil {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("'%s' cannot be compiled into a regular expression: %s", p.match, err.Error()),
			})
		} else {
			if !rx.MatchString(nodes[0].Value) {
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("'%s' does not match the expression '%s'", nodes[0].Value, p.match),
				})
			}
		}
	}

	// not match
	if p.notMatch != "" {
		rx, err := p.getPatternFromCache(p.notMatch)
		if err != nil {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("'%s' cannot be compiled into a regular expression: %s", p.notMatch, err.Error()),
			})
		} else {
			if rx.MatchString(nodes[0].Value) {
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("'%s' matches the expression '%s'", nodes[0].Value, p.notMatch),
				})
			}
		}
	}
	return results
}

func (p Pattern) getPatternFromCache(pattern string) (*regexp.Regexp, error) {
	if pat, ok := p.patternCache[pattern]; ok {
		return pat, nil
	} else {

		rx, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		p.patternCache[pattern] = rx
		return rx, nil
	}
}
