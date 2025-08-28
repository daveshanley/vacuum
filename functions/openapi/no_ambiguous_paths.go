// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

// AmbiguousPaths will determine if paths can be confused by a compiler.
type AmbiguousPaths struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the AmbiguousPaths rule.
func (ap AmbiguousPaths) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "noAmbiguousPaths"}
}

// GetCategory returns the category of the AmbiguousPaths rule.
func (ap AmbiguousPaths) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the AmbiguousPaths rule, based on supplied context and a supplied []*yaml.Node slice.
func (ap AmbiguousPaths) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	var seen []string

	ops := context.Index.GetPathsNode()

	var opPath string

	if ops != nil {
		var opNode *yaml.Node
		for i, op := range ops.Content {
			if i%2 == 0 {
				opPath = op.Value
				opNode = op
				continue
			}
			path := fmt.Sprintf("$.paths['%s']", opPath)
			for _, p := range seen {
				ambiguous := checkPaths(p, opPath)
				if ambiguous {

					results = append(results, model.RuleFunctionResult{
						Message:   fmt.Sprintf("paths are ambiguous with one another: `%s` and `%s`", p, opPath),
						StartNode: opNode,
						EndNode:   vacuumUtils.BuildEndNode(opNode),
						Path:      path,
						Rule:      context.Rule,
					})

				}
			}
			seen = append(seen, opPath)
		}
	}
	return results
}

var reggie, _ = regexp.Compile(`^{.+?}$`)

func checkPaths(pA, pB string) bool {
	segsA := strings.Split(pA, "/")[1:]
	segsB := strings.Split(pB, "/")[1:]

	if len(segsA) != len(segsB) {
		return false
	}

	// Two paths are considered ambiguous if they could match the same request URL
	// This can happen in two scenarios:
	// 1. Both paths have variables in the same positions and literals match (original behavior)
	//    e.g., /{id}/ambiguous and /{entity}/ambiguous
	// 2. One path has a variable where the other has a literal (issue #644)
	//    e.g., /foo/{x} and /foo/bar
	
	hasConflict := false
	for i := range segsA {
		aVar := reggie.MatchString(segsA[i])
		bVar := reggie.MatchString(segsB[i])
		
		if aVar && bVar {
			// Both are variables - continue checking other segments
			continue
		} else if !aVar && !bVar {
			// Both are literals - they must match
			if segsA[i] != segsB[i] {
				return false
			}
		} else {
			// One is a variable, one is a literal - this creates ambiguity
			// Mark that we found a conflict, but keep checking other segments
			hasConflict = true
		}
	}
	
	// Paths are ambiguous if we found at least one position where a variable 
	// conflicts with a literal while all other segments are compatible
	return hasConflict
}
