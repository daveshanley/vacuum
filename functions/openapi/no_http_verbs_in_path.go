// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// VerbsInPaths Checks to make sure that no HTTP verbs have been used
type VerbsInPaths struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the VerbsInPath rule.
func (vp VerbsInPaths) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "noVerbsInPath"}
}

// RunRule will execute the VerbsInPath rule, based on supplied context and a supplied []*yaml.Node slice.
func (vp VerbsInPaths) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

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
			path := fmt.Sprintf("$.paths['%s']", opPath)
			containsVerb, verb := checkPath(opPath)
			if containsVerb {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("path `%s` contains an HTTP Verb `%s`", opPath, verb),
					StartNode: op,
					EndNode:   vacuumUtils.BuildEndNode(op),
					Path:      path,
					Rule:      context.Rule,
				})
			}
		}
	}
	return results
}

func checkPath(path string) (bool, string) {
	segs := strings.Split(path, "/")[1:]
	for _, seg := range segs {
		if utils.IsHttpVerb(strings.ToLower(seg)) {
			return true, seg
		}
	}
	return false, ""
}
