// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"strings"
)

// NoRefSiblings will check for anything placed next to a $ref (like a description) and will throw some shade if
// something is found. This rule is there to prevent us from  adding useless properties to a $ref child.
type NoRefSiblings struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the NoRefSiblings rule.
func (nrs NoRefSiblings) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "no_ref_siblings",
	}
}

// RunRule will execute the NoRefSiblings rule, based on supplied context and a supplied []*yaml.Node slice.
func (rfs NoRefSiblings) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// look through paths first
	ymlPath, _ := yamlpath.NewPath("$.paths")
	pathNodes, _ := ymlPath.Find(nodes[0])

	search := &utils.KeyNodeSearch{
		Key:     "$ref",
		Results: []*utils.KeyNodeResult{},
	}
	utils.FindAllKeyNodesWithPath(search, nil, pathNodes, nil, 0)
	results = append(results, rfs.checkNodes("paths", search, results)...)

	// look through components next
	ymlPath, _ = yamlpath.NewPath("$.components")
	compNodes, _ := ymlPath.Find(nodes[0])

	if len(compNodes) > 0 {
		search.Results = []*utils.KeyNodeResult{}
		utils.FindAllKeyNodesWithPath(search, nil, compNodes, nil, 0)
		results = append(results, rfs.checkNodes("components", search, results)...)
	}
	// look through parameters
	ymlPath, _ = yamlpath.NewPath("$.parameters")
	paramNodes, _ := ymlPath.Find(nodes[0])

	if len(paramNodes) > 0 {
		search.Results = []*utils.KeyNodeResult{}
		utils.FindAllKeyNodesWithPath(search, nil, paramNodes, nil, 0)
		results = append(results, rfs.checkNodes("parameters", search, results)...)
	}

	// look through definitions (swagger)
	ymlPath, _ = yamlpath.NewPath("$.definitions")
	defNodes, _ := ymlPath.Find(nodes[0])

	if len(defNodes) > 0 {
		search.Results = []*utils.KeyNodeResult{}
		utils.FindAllKeyNodesWithPath(search, nil, defNodes, nil, 0)
		results = append(results, rfs.checkNodes("definitions", search, results)...)
	}
	return results

}

func (rfs NoRefSiblings) checkNodes(prefix string, search *utils.KeyNodeSearch, results []model.RuleFunctionResult) []model.RuleFunctionResult {
	for _, res := range search.Results {
		if len(res.Parent.Content) > 2 {

			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("a $ref cannot be placed next to any other properties"),
				StartNode: res.KeyNode,
				EndNode:   res.ValueNode,
				Path:      rfs.createJSONPathFromFoundNodeArray(prefix, res.Path),
			})
		}
	}
	return results
}

func (rfs NoRefSiblings) createJSONPathFromFoundNodeArray(prefix string, nodes []yaml.Node) string {
	nodeSegments := make([]string, len(nodes))
	for i, seg := range nodes {
		nodeSegments[i] = seg.Value
	}
	return fmt.Sprintf("$.%s.%s", prefix, strings.Join(nodeSegments, "."))
}
