// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
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

// RunRule will execute the PathsKebabCase rule, based on supplied context and a supplied []*yaml.Node slice.
func (vp PathsKebabCase) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	ops := context.Index.GetPathsNode()

	var opPath string

	if ops != nil {
		for i, op := range ops.Content {
			if i%2 == 0 {
				opPath = op.Value
				continue
			}
			path := fmt.Sprintf("$.paths.%s", opPath)
			if opPath == "/" {
				continue
			}
			notKebab, segments := checkPathCase(opPath)
			if notKebab {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("Path segments `%s` do not use kebab-case", strings.Join(segments, "`, `")),
					StartNode: op,
					EndNode:   op,
					Path:      path,
					Rule:      context.Rule,
				})
			}
		}
	}
	return results
}

var pathKebabCaseRegex, _ = regexp.Compile(`^[{}a-z\d-.]+$`)

func checkPathCase(path string) (bool, []string) {
	segs := strings.Split(path, "/")[1:]
	var found []string
	for i, seg := range segs {
		if !pathKebabCaseRegex.MatchString(seg) {
			// check if it's a variable, if so, skip
			if seg == "" {
				continue
			}
			// if this is a variable, or a variable at the end of a path then skip
			if seg[0] == '{' && (strings.Contains("}", seg) || i+1 == len(segs)) {
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
