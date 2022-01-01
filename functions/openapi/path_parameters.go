// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"regexp"
)

// PathParameters is a rule that checks path level and operation level parameters for correct paths. The rule is
// one of the more complex ones, so here is a little detail as to what is happening.
//-- normalize paths to replace vars with %
//-- check for duplicate paths based on param placement
//-- check for duplicate param names in paths
//-- check for any unknown params (no name)
//-- check if required is set, that it's set to true only.
//-- check no duplicate params
//-- operation paths only
//-- all params in path must be defined
//-- all defined path params must be in path.
type PathParameters struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PathParameters rule.
func (pp PathParameters) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "path_parameters",
	}
}

// RunRule will execute the PathParameters rule, based on supplied context and a supplied []*yaml.Node slice.
func (pp PathParameters) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	opNodes := GetOperationsFromRoot(nodes)
	paramRegex := `(\{;?\??[a-zA-Z0-9_-]+\*?\})`
	rx, _ := regexp.Compile(paramRegex)

	// check for duplicate paths
	seenPaths := make(map[string]string)

	var currentPath string
	var currentVerb string
	pathElements := make(map[string]bool)
	topLevelParams := make(map[string][]string)
	verbLevelParams := make(map[string][]string)

	for j, operationNode := range opNodes {

		if utils.IsNodeStringValue(operationNode) {
			// replace any params with an invalid char (%) so we can perform a path
			// equality check. /hello/{fresh} and /hello/{fish} are equivalent to OpenAPI.
			currentPath = operationNode.Value
			currentPathNormalized := rx.ReplaceAllString(currentPath, "%")

			// check if it's been seen
			if seenPaths[currentPathNormalized] != "" {
				res := model.BuildFunctionResultString(
					fmt.Sprintf("Paths '%s' and '%s' must not be equivalent, paths must be unique",
						seenPaths[currentPathNormalized], currentPath))
				res.StartNode = operationNode
				res.EndNode = operationNode
				res.Path = fmt.Sprintf("$.paths.%s", currentPath)
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
						fmt.Sprintf("Path '%s' must not use the parameter '%s' multiple times",
							currentPath, param))
					res.StartNode = operationNode
					res.EndNode = operationNode
					res.Path = fmt.Sprintf("$.paths.%s", currentPath)
					results = append(results, res)
				} else {
					pathElements[param] = true
				}

			}
		}
		if utils.IsNodeMap(operationNode) {

			_, topLevelParametersNode := utils.FindKeyNode("parameters", operationNode.Content)

			// look for top level params
			if topLevelParametersNode != nil {
				for x, topLevelParam := range topLevelParametersNode.Content {
					_, paramInNode := utils.FindKeyNode("in", topLevelParam.Content)
					_, paramRequiredNode := utils.FindKeyNode("required", topLevelParam.Content)
					_, paramNameNode := utils.FindKeyNode("name", topLevelParam.Content)

					if isNamedPathParamUnknown(paramInNode, paramRequiredNode, paramNameNode,
						currentPath, currentVerb, topLevelParams, &results) {
						topLevelParams[paramNameNode.Value] = []string{"paths", currentPath, "parameters",
							fmt.Sprintf("%v", x)}
					}

				}
			}

			// look for verb level params.
			c := 0
			for h, verbMapNode := range operationNode.Content {
				if utils.IsNodeStringValue(verbMapNode) && isHttpVerb(verbMapNode.Value) {

					currentVerb = verbMapNode.Value
				} else {
					continue
				}
				verbDataNode := operationNode.Content[h+1]

				_, verbParameterNode := utils.FindFirstKeyNode("parameters", verbDataNode.Content)
				if verbParameterNode != nil {
					_, paramInNode := utils.FindKeyNode("in", verbParameterNode.Content)
					_, paramRequiredNode := utils.FindKeyNode("required", verbParameterNode.Content)
					_, paramNameNode := utils.FindKeyNode("name", verbParameterNode.Content)

					if isNamedPathParamUnknown(paramInNode, paramRequiredNode, paramNameNode,
						currentPath, currentVerb, verbLevelParams, &results) {
						verbLevelParams[paramNameNode.Value] = []string{"paths", currentPath, currentVerb, "parameters",
							fmt.Sprintf("%v", c)}
					}
					c++
				}
			}

			// blend together all our params and check they all match up!
			allPathParams := make(map[string][]string, len(topLevelParams)+len(verbLevelParams))
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

			startNode := operationNode
			endNode := operationNode
			if j+1 < len(opNodes) {
				endNode = opNodes[j+1]
			}

			ensureAllDefinedPathParamsAreUsedInPath(currentPath, allPathParams, pathElements, &results, startNode, endNode)
			ensureAllExpectedParamsInPathAreDefined(currentPath, allPathParams, pathElements, &results, startNode, endNode)

			// reset for the next run.
			pathElements = make(map[string]bool)
			topLevelParams = make(map[string][]string)
			verbLevelParams = make(map[string][]string)

		}

	}

	return results

}

func ensureAllDefinedPathParamsAreUsedInPath(path string, allPathParams map[string][]string,
	pathElements map[string]bool, results *[]model.RuleFunctionResult, startNode, endNode *yaml.Node) {

	for k := range allPathParams {
		foundInElements := false
		for e := range pathElements {
			if k == e {
				foundInElements = true
			}
		}
		if !foundInElements {
			err := fmt.Sprintf("parameter '%s' must be used in path '%s'", k, path)
			res := model.BuildFunctionResultString(err)
			res.StartNode = startNode
			res.EndNode = endNode
			res.Path = fmt.Sprintf("$.paths.%s", path)
			*results = append(*results, res)
		}
	}
}

func ensureAllExpectedParamsInPathAreDefined(path string, allPathParams map[string][]string,
	pathElements map[string]bool, results *[]model.RuleFunctionResult, startNode, endNode *yaml.Node) {

	for k := range pathElements {
		foundInParams := false
		for e := range allPathParams {
			if k == e {
				foundInParams = true
			}
		}
		if !foundInParams {
			err := fmt.Sprintf("Operation must define parameter '%s' as expected by path '%s'", k, path)
			res := model.BuildFunctionResultString(err)
			res.StartNode = startNode
			res.EndNode = endNode
			res.Path = fmt.Sprintf("$.paths.%s", path)
			*results = append(*results, res)
		}
	}
}

func isHttpVerb(verb string) bool {
	verbs := []string{"get", "post", "put", "patch", "delete", "options", "trace", "head"}
	for _, v := range verbs {
		if verb == v {
			return true
		}
	}
	return false
}

func isPathParamNamed(in, name *yaml.Node) bool {
	if in == nil || name == nil {
		return false
	}
	if in.Value != "path" {
		return false
	}
	return true
}

func isNamedPathParamUnknown(in, required, name *yaml.Node, currentPath, currentVerb string,
	seenNodes map[string][]string, results *[]model.RuleFunctionResult) bool {
	if !isPathParamNamed(in, name) {
		return false
	}
	// check if required is set, if so that it's also a bool
	if required != nil {
		errMsg := fmt.Sprintf("%s %s must have 'required' parameter that is set to 'true'",
			currentPath, currentVerb)

		res := model.BuildFunctionResultString(errMsg)
		res.StartNode = required
		res.EndNode = required
		res.Path = fmt.Sprintf("$.paths.%s.%s.parameters", currentPath, currentVerb)

		if utils.IsNodeBoolValue(required) {
			if required.Value != "true" {
				*results = append(*results, res)
			}
		} else {
			*results = append(*results, res)
		}
	}

	// check if name is defined and if it's been defined multiple times.
	if name != nil {
		if seenNodes[name.Value] != nil {
			res := model.BuildFunctionResultString(
				fmt.Sprintf("%s %s contains has a parameter '%s' defined multiple times'",
					currentPath, currentVerb, name.Value))
			res.StartNode = name
			res.EndNode = name
			res.Path = fmt.Sprintf("$.paths.%s.%s.parameters", currentPath, currentVerb)
			*results = append(*results, res)
			return false
		}
	}
	return true
}
