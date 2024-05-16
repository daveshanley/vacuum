// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

// PathParameters is a rule that checks path level and operation level parameters for correct paths. The rule is
// one of the more complex ones, so here is a little detail as to what is happening.
// -- normalize paths to replace vars with %
// -- check for duplicate paths based on param placement
// -- check for duplicate param names in paths
// -- check for any unknown params (no name)
// -- check if required is set, that it's set to true only.
// -- check no duplicate params
// -- operation paths only
// -- all params in path must be defined
// -- all defined path params must be in path.
type PathParameters struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PathParameters rule.
func (pp PathParameters) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "path_parameters",
	}
}

// GetCategory returns the category of the PathParameters rule.
func (pp PathParameters) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the PathParameters rule, based on supplied context and a supplied []*yaml.Node slice.
func (pp PathParameters) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	pathNodes := context.Index.GetPathsNode()
	paramRegex := `(\{;?\??[\.a-zA-Z0-9_-]+\*?\})`
	rx := regexp.MustCompile(paramRegex)

	// check for duplicate paths
	seenPaths := make(map[string]string)

	var currentPath string
	var currentVerb string
	pathElements := make(map[string]bool)
	topLevelParams := make(map[string]map[string][]string)

	if pathNodes == nil {
		return results
	}
	for _, pathNode := range pathNodes.Content {

		if utils.IsNodeStringValue(pathNode) {
			// replace any params with an invalid char (%) so we can perform a path
			// equality check. /hello/{fresh} and /hello/{fish} are equivalent to OpenAPI.
			currentPath = pathNode.Value
			currentPathNormalized := rx.ReplaceAllString(currentPath, "%")

			// check if it's been seen
			if seenPaths[currentPathNormalized] != "" {
				res := model.BuildFunctionResultString(
					fmt.Sprintf("paths `%s` and `%s` must not be equivalent, paths must be unique",
						seenPaths[currentPathNormalized], currentPath))
				res.StartNode = pathNode
				res.EndNode = vacuumUtils.BuildEndNode(pathNode)
				res.Path = fmt.Sprintf("$.paths.['%s']", currentPath)
				res.Rule = context.Rule
				results = append(results, res)
			} else {
				seenPaths[currentPathNormalized] = currentPath
			}

			// check if the value has been used multiple times, 100 segments seems overly cautious.
			for _, pathParam := range rx.FindAllString(currentPath, 100) {
				// strip off curly brackets
				strRx, _ := regexp.Compile(`[{}?*;]`)
				param := strRx.ReplaceAllString(pathParam, "")
				if pathElements[param] {
					res := model.BuildFunctionResultString(
						fmt.Sprintf("path `%s` must not use the parameter `%s` multiple times",
							currentPath, param))
					res.StartNode = pathNode
					res.EndNode = vacuumUtils.BuildEndNode(pathNode)
					res.Path = fmt.Sprintf("$.paths.['%s']", currentPath)
					res.Rule = context.Rule
					results = append(results, res)
				} else {
					pathElements[param] = true
				}

			}
		}
		oprefs := context.Index.GetOperationParameterReferences()
		if utils.IsNodeMap(pathNode) {

			topLevelParametersNode := oprefs[currentPath]["top"]
			//_, topLevelParametersNode := utils.FindKeyNodeTop("parameters", operationNode.Content)
			// look for top level params
			//if topLevelParametersNode != nil {
			for x, topLevelParamSlice := range topLevelParametersNode {
				for _, topLevelParam := range topLevelParamSlice {

					_, paramInNode := utils.FindKeyNode("in", topLevelParam.Node.Content)
					_, paramRequiredNode := utils.FindKeyNode("required", topLevelParam.Node.Content)
					_, paramNameNode := utils.FindKeyNode("name", topLevelParam.Node.Content)

					if currentVerb == "" {
						currentVerb = "top"
					}

					if pp.isPathParamNamedAndRequired(paramInNode, paramRequiredNode, paramNameNode,
						currentPath, currentVerb, &topLevelParams, nil, &results, context) {

						var paramData map[string][]string
						if topLevelParams["top"] != nil {
							paramData = topLevelParams["top"]
						} else {
							paramData = make(map[string][]string)
						}
						path := []string{"paths", currentPath, "parameters", fmt.Sprintf("%v", x)}
						paramData[paramNameNode.Value] = path
						topLevelParams["top"] = paramData
					}
				}
			}
			//}

			// look for verb level params.
			c := 0
			verbLevelParams := make(map[string]map[string][]string)

			for _, verbMapNode := range pathNode.Content {
				if utils.IsNodeStringValue(verbMapNode) && utils.IsHttpVerb(verbMapNode.Value) {
					currentVerb = verbMapNode.Value
				} else {
					continue
				}

				// use index to locate params.
				verbParametersNode := oprefs[currentPath][currentVerb]
				for _, verbParams := range verbParametersNode {

					if verbParams == nil {
						continue
					}

					for _, verbParam := range verbParams {
						_, paramInNode := utils.FindKeyNode("in", verbParam.Node.Content)
						_, paramRequiredNode := utils.FindKeyNode("required", verbParam.Node.Content)
						_, paramNameNode := utils.FindKeyNode("name", verbParam.Node.Content)

						if pp.isPathParamNamedAndRequired(paramInNode, paramRequiredNode, paramNameNode,
							currentPath, currentVerb, &verbLevelParams, topLevelParams["top"], &results, context) {

							path := []string{"paths", currentPath, currentVerb, "parameters",
								fmt.Sprintf("%v", c)}
							var paramData map[string][]string
							if verbLevelParams[currentVerb] != nil {
								paramData = verbLevelParams[currentVerb]
							} else {
								paramData = make(map[string][]string)
							}
							paramData[paramNameNode.Value] = path
							verbLevelParams[currentVerb] = paramData
						}
						c++
					}
				}
			}

			// blend together all our params and check they all match up!
			allPathParams := make(map[string]map[string][]string, len(topLevelParams)+len(verbLevelParams))
			if len(topLevelParams) > 0 {
				for k, v := range topLevelParams {
					allPathParams[k] = v
				}
			}
			if len(verbLevelParams) > 0 {
				for k, v := range verbLevelParams {
					allPathParams[k] = v
				}
			}

			startNode := pathNode
			r := ""
			for i, verbMapNode := range pathNode.Content {
				if i%2 == 0 {
					r = verbMapNode.Value
					continue
				}
				if isVerb(r) {
					pp.ensureAllExpectedParamsInPathAreDefined(currentPath, allPathParams,
						pathElements, &results, startNode, context, r)
				}
			}
			pp.ensureAllDefinedPathParamsAreUsedInPath(currentPath, allPathParams,
				pathElements, &results, startNode, context)

			// reset for the next run.
			pathElements = make(map[string]bool)
			topLevelParams = make(map[string]map[string][]string)
			verbLevelParams = make(map[string]map[string][]string)

		}

	}

	// include operation param errors found by the index
	errors := context.Index.GetOperationParametersIndexErrors()
	for _, err := range errors {
		idxErr := err.(*index.IndexingError)
		res := model.BuildFunctionResultString(idxErr.Error())
		res.StartNode = idxErr.Node
		res.EndNode = vacuumUtils.BuildEndNode(idxErr.Node)
		res.Path = idxErr.Path
		res.Rule = context.Rule
		results = append(results, res)
	}

	return results

}

func isVerb(verb string) bool {
	switch strings.ToLower(verb) {
	case v3.GetLabel, v3.PostLabel, v3.PutLabel, v3.PatchLabel, v3.DeleteLabel, v3.HeadLabel, v3.OptionsLabel, v3.TraceLabel:
		return true
	}
	return false
}

func (pp PathParameters) ensureAllDefinedPathParamsAreUsedInPath(path string, allPathParams map[string]map[string][]string,
	pathElements map[string]bool, results *[]model.RuleFunctionResult, startNode *yaml.Node,
	context model.RuleFunctionContext) {

	for _, item := range allPathParams {

		for param := range item {
			foundInElements := false
			for e := range pathElements {
				if param == e {
					foundInElements = true
				}
			}
			if !foundInElements {
				err := fmt.Sprintf("parameter `%s` must be used in path `%s`", param, path)
				res := model.BuildFunctionResultString(err)
				res.StartNode = startNode
				res.EndNode = vacuumUtils.BuildEndNode(startNode)
				res.Path = fmt.Sprintf("$.paths['%s']", path)
				res.Rule = context.Rule
				*results = append(*results, res)
			}
		}
	}
}

func (pp PathParameters) ensureAllExpectedParamsInPathAreDefined(path string, allPathParams map[string]map[string][]string,
	pathElements map[string]bool, results *[]model.RuleFunctionResult, startNode *yaml.Node,
	context model.RuleFunctionContext, verb string) {
	var topParams map[string][]string
	var verbParams map[string][]string
	if allPathParams != nil {
		topParams = allPathParams["top"]
		verbParams = allPathParams[verb]
	}
	// For each expected path parameter, check the top and verb-level defined parameters
	for p := range pathElements {
		if !pp.segmentExistsInPathParams(p, verbParams, topParams) {
			err := fmt.Sprintf("`%s` must define parameter `%s` as expected by path `%s`", strings.ToUpper(verb), p, path)
			res := model.BuildFunctionResultString(err)
			res.StartNode = startNode
			res.EndNode = vacuumUtils.BuildEndNode(startNode)
			res.Path = fmt.Sprintf("$.paths['%s']", path)
			res.Rule = context.Rule
			*results = append(*results, res)
		}
	}
}

func (pp PathParameters) segmentExistsInPathParams(segment string, params, top map[string][]string) bool {
	for k := range params {
		if k == segment {
			return true
		}
	}
	for k := range top {
		if k == segment {
			return true
		}
	}
	return false
}

func (pp PathParameters) isPathParamNamed(in, name *yaml.Node) bool {
	if in == nil || name == nil {
		return false
	}
	if in.Value != "path" {
		return false
	}
	return true
}

func (pp PathParameters) isPathParamNamedAndRequired(in, required, name *yaml.Node, currentPath, currentVerb string,
	seenNodes *map[string]map[string][]string, topNodes map[string][]string, results *[]model.RuleFunctionResult,
	context model.RuleFunctionContext) bool {
	if !pp.isPathParamNamed(in, name) {
		return false
	}
	// check if required is set, if so that it's also a bool
	if required != nil {

		var errMsg string
		if currentVerb == "top" {
			errMsg = fmt.Sprintf("%s must have `required` parameter that is set to `true`",
				currentPath)
		} else {
			errMsg = fmt.Sprintf("%s %s must have `required` parameter that is set to `true`",
				currentPath, currentVerb)
		}

		res := model.BuildFunctionResultString(errMsg)
		res.StartNode = required
		res.EndNode = required
		res.Path = fmt.Sprintf("$.paths['%s'].%s.parameters", currentPath, currentVerb)
		res.Rule = context.Rule

		if utils.IsNodeBoolValue(required) {
			if required.Value != "true" {
				*results = append(*results, res)
			}
		} else {
			*results = append(*results, res)
		}
	}
	return true
}
