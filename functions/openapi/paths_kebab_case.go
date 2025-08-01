// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
    "github.com/daveshanley/vacuum/model"
    "github.com/daveshanley/vacuum/model/reports"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
    "regexp"
    "strings"
)

// PathsKebabCase Checks to ensure each segment of a path is using kebab case.
type PathsKebabCase struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the VerbsInPath rule.
func (vp PathsKebabCase) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "pathsKebabCase"}
}

// GetCategory returns the category of the VerbsInPath rule.
func (vp PathsKebabCase) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the PathsKebabCase rule, based on supplied context and a supplied []*yaml.Node slice.
func (vp PathsKebabCase) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	paths := context.DrDocument.V3Document.Paths

	if paths != nil {

		for k, v := range paths.PathItems.FromOldest() {
			if v != nil {
				notKebab, segments := checkPathCase(k)
				if notKebab {
					n := v.Value.GoLow().KeyNode
					endNode := vacuumUtils.BuildEndNode(n)
					result := model.RuleFunctionResult{
						Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, model.GetStringTemplates().BuildKebabCaseMessage(strings.Join(segments, "`, `"))),
						StartNode: n,
						EndNode:   endNode,
						Path:      v.GenerateJSONPath(),
						Rule:      context.Rule,
						Range: reports.Range{
							Start: reports.RangeItem{
								Line: n.Line,
								Char: n.Column,
							},
							End: reports.RangeItem{
								Line: endNode.Line,
								Char: endNode.Column,
							},
						},
					}
					v.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
					results = append(results, result)
				}
			}
		}
	}
	return results
}

var pathKebabCaseRegex, _ = regexp.Compile(`^[{}a-z\d-.]+$`)
var variableRegex, _ = regexp.Compile(`^\{(\w.*)}\.?.*$`)

func checkPathCase(path string) (bool, []string) {
	segs := strings.Split(path, "/")[1:]
	var found []string
	for _, seg := range segs {
		if !pathKebabCaseRegex.MatchString(seg) {
			// check if it's a variable, if so, skip
			if seg == "" {
				continue
			}
			// if this is a variable, or a variable at the end of a path then skip
			if variableRegex.MatchString(seg) {
				continue
			}
			found = append(found, seg)
		}
	}
	if len(found) > 0 {
		return true, found
	}
	return false, nil
}
