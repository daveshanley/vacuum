// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	v3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"net/http"
	"strconv"
	"strings"
)

// OperationDescription will check if an operation has a description, and if the description is useful
type OperationDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationDescription rule.
func (od OperationDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasDescriptions",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "minWords",
				Description: "Minimum number of words required in a description, defaults to '0'",
			},
		},
		ErrorMessage: "'oasDescriptions' function has invalid options supplied. Set the 'minWords' property to a valid integer",
	}
}

// GetCategory returns the category of the OperationDescription rule.
func (od OperationDescription) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the OperationDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (od OperationDescription) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)

	minWordsString := props["minWords"]
	minWords, _ := strconv.Atoi(minWordsString)

	if context.DrDocument == nil {
		return results
	}

	paths := context.DrDocument.V3Document.Paths

	buildResult := func(message, path string, node *yaml.Node, component base.AcceptsRuleResults) model.RuleFunctionResult {
		endNode := vacuumUtils.BuildEndNode(node)
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   endNode,
			Path:      path,
			Rule:      context.Rule,
			Range: reports.Range{
				Start: reports.RangeItem{
					Line: node.Line,
					Char: node.Column,
				},
				End: reports.RangeItem{
					Line: endNode.Line,
					Char: endNode.Column,
				},
			},
		}
		component.AddRuleFunctionResult(base.ConvertRuleResult(&result))
		return result
	}

	checkDescription := func(description string, method, location, missing, JSONPath string, node *yaml.Node, component base.AcceptsRuleResults) {
		if description == "" {
			results = append(results,
				buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					fmt.Sprintf("operation method `%s` %s is missing a `%s`", method, location, missing)),
					JSONPath, node, component))
		} else {
			words := strings.Split(description, " ")
			if len(words) < minWords {
				results = append(results, buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					fmt.Sprintf("operation method `%s` %s has a `%s` that must be at least `%d` words long",
						method, location, missing, minWords)), location, node, component))
			}
		}
	}

	checkOperation := func(desc, summary, path, method, location, reqLocation, jsonPath string, node *yaml.Node,
		requestBody *v3.RequestBody, responses *v3.Responses, op base.AcceptsRuleResults) {

		checkDescription(desc, method, location, "description", jsonPath, node, op)
		checkDescription(summary, method, location, "summary", jsonPath, node, op)

		// check request body
		if requestBody != nil {
			checkDescription(requestBody.Value.Description, method, reqLocation,
				"description", requestBody.GenerateJSONPath(), requestBody.Value.GoLow().KeyNode, op)
		}

		// check responses
		if responses != nil {
			for responsePairs := responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
				code := responsePairs.Key()
				response := responsePairs.Value()
				checkDescription(response.Value.Description, method, fmt.Sprintf("response code `%s` `responseBody` at path `%s`", code, path),
					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			}
		}
	}

	if paths != nil {
		for pathPairs := paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {

			path := pathPairs.Key()
			operation := pathPairs.Value()

			atPath := fmt.Sprintf("at path `%s`", path)
			atRequest := fmt.Sprintf("`requestBody` at path `%s`", path)

			if operation.Get != nil {
				checkOperation(operation.Get.Value.Description, operation.Get.Value.Summary, path, http.MethodGet,
					atPath, atRequest, operation.Get.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Get.RequestBody, operation.Get.Responses, operation.Get)
			}

			if operation.Post != nil {
				checkOperation(operation.Post.Value.Description, operation.Post.Value.Summary, path, http.MethodPost,
					atPath, atRequest, operation.Post.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Post.RequestBody, operation.Post.Responses, operation.Post)
			}

			if operation.Put != nil {
				checkOperation(operation.Put.Value.Description, operation.Put.Value.Summary, path, http.MethodPut,
					atPath, atRequest, operation.Put.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Put.RequestBody, operation.Put.Responses, operation.Put)
			}

			if operation.Delete != nil {
				checkOperation(operation.Delete.Value.Description, operation.Delete.Value.Summary, path, http.MethodDelete,
					atPath, atRequest, operation.Delete.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Delete.RequestBody, operation.Delete.Responses, operation.Delete)
			}

			if operation.Head != nil {
				checkOperation(operation.Head.Value.Description, operation.Head.Value.Summary, path, http.MethodHead,
					atPath, atRequest, operation.Head.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Head.RequestBody, operation.Head.Responses, operation.Head)
			}

			if operation.Patch != nil {
				checkOperation(operation.Patch.Value.Description, operation.Patch.Value.Summary, path, http.MethodPatch,
					atPath, atRequest, operation.Patch.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Patch.RequestBody, operation.Patch.Responses, operation.Patch)
			}

			if operation.Options != nil {
				checkOperation(operation.Options.Value.Description, operation.Options.Value.Summary, path, http.MethodOptions,
					atPath, atRequest, operation.Options.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Options.RequestBody, operation.Options.Responses, operation.Options)
			}

			if operation.Trace != nil {
				checkOperation(operation.Trace.Value.Description, operation.Trace.Value.Summary, path, http.MethodTrace,
					atPath, atRequest, operation.Trace.GenerateJSONPath(), operation.Value.GoLow().KeyNode,
					operation.Trace.RequestBody, operation.Trace.Responses, operation.Trace)
			}

		}
	}
	return results
}
