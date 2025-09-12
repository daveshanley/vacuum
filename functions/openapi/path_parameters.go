// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/helpers"
	doctorModel "github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"regexp"
	"slices"
	"strings"
)

// PathParameters is a rule that checks path level and operation level parameters for correct paths.
// this rule has been refactored to use the doctor model. The logic is now much simpler, but does not work against
// swagger
type PathParameters struct {
	rx *regexp.Regexp
}

const paramRegex = `(\{;?\??[\.a-zA-Z0-9_-]+\*?\})`

var rxPathParameters *regexp.Regexp

func init() {
	rxPathParameters = regexp.MustCompile(paramRegex)
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PathParameters rule.
func (pp PathParameters) GetSchema() model.RuleFunctionSchema {
	pp.rx = regexp.MustCompile(paramRegex)
	return model.RuleFunctionSchema{
		Name: "oasPathParam",
	}
}

// GetCategory returns the category of the PathParameters rule.
func (pp PathParameters) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the PathParameters rule, based on supplied context and a supplied []*yaml.Node slice.
func (pp PathParameters) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if pp.rx == nil {
		pp.rx = rxPathParameters
	}

	var results []model.RuleFunctionResult

	// get doctor model.
	if context.DrDocument == nil {
		return results
	}

	// get paths.
	paths := context.DrDocument.V3Document.Paths
	if paths == nil {
		return results
	}

	// check for duplicate paths
	seenPaths := make(map[string]string)

	// iterate through each path and do the things.
	for pathKey, pathValue := range paths.PathItems.FromOldest() {

		//if utils.IsNodeStringValue(pathNode) {
		// replace any params with an invalid char (%) so we can perform a path
		// equality check. /hello/{fresh} and /hello/{fish} are equivalent to OpenAPI.
		//currentPath = pathKey
		currentPathNormalized := pp.rx.ReplaceAllString(pathKey, "%")

		// check if it's been seen
		if seenPaths[currentPathNormalized] != "" {
			res := model.BuildFunctionResultString(
				fmt.Sprintf("paths `%s` and `%s` must not be equivalent, paths must be unique",
					seenPaths[currentPathNormalized], pathKey))
			res.StartNode = pathValue.KeyNode
			res.EndNode = vacuumUtils.BuildEndNode(pathValue.KeyNode)
			res.Path = pathValue.GenerateJSONPath()
			res.Rule = context.Rule
			results = append(results, res)
		} else {
			seenPaths[currentPathNormalized] = pathKey
		}

		pathElements := make(map[string]bool)

		var params []string

		// check if the value has been used multiple times, 100 segments seems overly cautious.
		for _, pathParamStringValue := range pp.rx.FindAllString(pathKey, 100) {
			// strip off curly brackets
			strRx, _ := regexp.Compile(`[{}?*;]`)
			param := strRx.ReplaceAllString(pathParamStringValue, "")
			params = append(params, param)
			if pathElements[param] {
				res := model.BuildFunctionResultString(
					fmt.Sprintf("path `%s` must not use the parameter `%s` multiple times",
						pathKey, param))
				res.StartNode = pathValue.KeyNode
				res.EndNode = vacuumUtils.BuildEndNode(pathValue.KeyNode)
				res.Path = pathValue.GenerateJSONPath()
				res.Rule = context.Rule
				results = append(results, res)
			} else {
				pathElements[param] = true
			}

			// for each param, check if it's been named in the path params or the operation params.
			// check path level parameters to determine if they are all good.
			pathParams := pathValue.Parameters
			var foundPathParam *doctorModel.Parameter

			for _, pathParam := range pathParams {
				if pathParam.Value.In == "path" {
					if pathParam.Value.Name == param {
						foundPathParam = pathParam
					}
				}
			}

			for opVerb, operation := range pathValue.GetOperations().FromOldest() {

				foundOperationParams := make(map[string]*doctorModel.Parameter)
				missingParams := make(map[string]string)

				// check if the operation has a parameter with the same name as the path param.
				// if so, check if it's in the path.
				operationParams := operation.Parameters
				for _, operationParam := range operationParams {
					if operationParam.Value.In == "path" {
						if operationParam.Value.Name == param {
							foundOperationParams[param] = operationParam
						}
					}
				}
				if foundOperationParams[param] == nil {
					missingParams[opVerb] = param
				}

				// if no path param or operation param was found, then it's an unknown param.
				if foundPathParam == nil && len(missingParams) > 0 {

					var verbs []string
					for v, _ := range missingParams {
						verbs = append(verbs, strings.ToUpper(v))
					}

					err := fmt.Sprintf("parameter named `%s` must be defined as part of the path `%s` definition, or in the %s operation(s)",
						param, pathKey, helpers.WrapBackticksString(verbs))
					res := model.BuildFunctionResultString(err)
					res.StartNode = pathValue.KeyNode
					res.EndNode = vacuumUtils.BuildEndNode(pathValue.KeyNode)
					res.Path = pathValue.GenerateJSONPath()
					res.Rule = context.Rule
					results = append(results, res)
				}
			}
		}

		// check if the path param is defined in the path params.
		pathParamNames := make(map[string]*doctorModel.Parameter)
		duplicateParamNames := make(map[string]bool)
		for _, pathParam := range pathValue.Parameters {
			if pathParam.Value.In == "path" {

				// if param does not have required set to true, then it's an error.
				if pathParam.Value.Required != nil && !*pathParam.Value.Required {
					err := fmt.Sprintf("path parameter named `%s` at `%s` must have `required` set to `true`", pathParam.Value.Name, pathKey)
					res := model.BuildFunctionResultString(err)
					res.StartNode = pathParam.KeyNode
					res.EndNode = vacuumUtils.BuildEndNode(pathParam.KeyNode)
					res.Path = pathParam.GenerateJSONPath()
					res.Rule = context.Rule
					results = append(results, res)
				}

				if _, kk := duplicateParamNames[pathParam.Value.Name]; !kk {
					duplicateParamNames[pathParam.Value.Name] = true
				} else {
					// duplicate param name
					err := fmt.Sprintf("path parameter named `%s` at `%s` is a duplicate of another parameter with the same name", pathParam.Value.Name, pathKey)
					res := model.BuildFunctionResultString(err)
					res.StartNode = pathParam.KeyNode
					res.EndNode = vacuumUtils.BuildEndNode(pathParam.KeyNode)
					res.Path = pathParam.GenerateJSONPath()
					res.Rule = context.Rule
					results = append(results, res)
				}
				pathParamNames[pathParam.Value.Name] = pathParam
			}
		}

		for paramName, param := range pathParamNames {
			if !slices.Contains(params, paramName) {
				// unknown param
				err := fmt.Sprintf("path parameter named `%s` does not exist in path `%s`", paramName, pathKey)
				res := model.BuildFunctionResultString(err)
				res.StartNode = param.KeyNode
				res.EndNode = vacuumUtils.BuildEndNode(param.KeyNode)
				res.Path = param.GenerateJSONPath()
				res.Rule = context.Rule
				results = append(results, res)
			}
		}

		for verb, op := range pathValue.GetOperations().FromOldest() {

			duplicateParamNames = make(map[string]bool)
			for _, param := range op.Parameters {
				if param.Value.In != "path" {
					continue
				}
				if !slices.Contains(params, param.Value.Name) {
					// unknown param
					err := fmt.Sprintf("`%s` parameter named `%s` does not exist in path `%s`",
						strings.ToUpper(verb), param.Value.Name, pathKey)
					res := model.BuildFunctionResultString(err)
					res.StartNode = param.KeyNode
					res.EndNode = vacuumUtils.BuildEndNode(param.KeyNode)
					res.Path = param.GenerateJSONPath()
					res.Rule = context.Rule
					results = append(results, res)
				}

				// required is set to false
				if param.Value.Required != nil && !*param.Value.Required {
					err := fmt.Sprintf("`%s` `%s` parameter named `%s` must have `required` set to `true`", pathKey,
						strings.ToUpper(verb), param.Value.Name)
					res := model.BuildFunctionResultString(err)
					res.StartNode = param.KeyNode
					res.EndNode = vacuumUtils.BuildEndNode(param.KeyNode)
					res.Path = param.GenerateJSONPath()
					res.Rule = context.Rule
					results = append(results, res)
				}
				if _, kk := duplicateParamNames[param.Value.Name]; !kk {
					duplicateParamNames[param.Value.Name] = true
				} else {
					// duplicate param name
					err := fmt.Sprintf("`%s` parameter named `%s` at `%s` is a duplicate of another parameter with the same name",
						strings.ToUpper(verb), param.Value.Name, pathKey)
					res := model.BuildFunctionResultString(err)
					res.StartNode = param.KeyNode
					res.EndNode = vacuumUtils.BuildEndNode(param.KeyNode)
					res.Path = param.GenerateJSONPath()
					res.Rule = context.Rule
					results = append(results, res)
				}
			}
		}
	}
	return results
}
