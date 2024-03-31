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
	return model.RuleFunctionSchema{Name: "operation_description"}
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

			//	if operation.Put != nil {
			//		checkDescription(operation.Put.Value.Description, http.MethodPut, fmt.Sprintf("at path `%s`", path),
			//			"description", operation.Put.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Put)
			//
			//		checkDescription(operation.Put.Value.Summary, http.MethodPut, fmt.Sprintf("at path `%s`", path),
			//			"summary", operation.Put.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Put)
			//
			//		// check request body
			//		if operation.Put.RequestBody != nil {
			//			checkDescription(operation.Put.Value.RequestBody.Description, http.MethodPut, fmt.Sprintf("`requestBody` at path `%s`", path),
			//				"description", operation.Put.GenerateJSONPath(),
			//				operation.Put.RequestBody.Value.GoLow().KeyNode, operation.Put)
			//		}
			//
			//		// check responses
			//		if operation.Put.Responses != nil {
			//			for responsePairs := operation.Put.Responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			//				code := responsePairs.Key()
			//				response := responsePairs.Value()
			//				checkDescription(response.Value.Description, http.MethodPut, fmt.Sprintf("code `%s` `responseBody` at path `%s`", code, path),
			//					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			//			}
			//
			//		}
			//	}
			//
			//	if operation.Post != nil {
			//		checkDescription(operation.Post.Value.Description, http.MethodPost, fmt.Sprintf("at path `%s`", path),
			//			"description", operation.Post.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Post)
			//
			//		checkDescription(operation.Post.Value.Summary, http.MethodPost, fmt.Sprintf("at path `%s`", path),
			//			"summary", operation.Post.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Post)
			//
			//		// check request body
			//		if operation.Post.RequestBody != nil {
			//			checkDescription(operation.Post.Value.RequestBody.Description, http.MethodPost, fmt.Sprintf("`requestBody` at path `%s`", path),
			//				"description", operation.Post.GenerateJSONPath(),
			//				operation.Post.RequestBody.Value.GoLow().KeyNode, operation.Post)
			//		}
			//
			//		// check responses
			//		if operation.Post.Responses != nil {
			//			for responsePairs := operation.Post.Responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			//				code := responsePairs.Key()
			//				response := responsePairs.Value()
			//				checkDescription(response.Value.Description, http.MethodPost, fmt.Sprintf("code `%s` `responseBody` at path `%s`", code, path),
			//					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			//			}
			//
			//		}
			//	}
			//
			//	if operation.Delete != nil {
			//		checkDescription(operation.Delete.Value.Description, http.MethodDelete, fmt.Sprintf("at path `%s`", path),
			//			"description", operation.Delete.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Delete)
			//
			//		checkDescription(operation.Delete.Value.Summary, http.MethodDelete, fmt.Sprintf("at path `%s`", path),
			//			"summary", operation.Delete.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Delete)
			//
			//		// check request body
			//		if operation.Delete.RequestBody != nil {
			//			checkDescription(operation.Delete.Value.RequestBody.Description, http.MethodDelete, fmt.Sprintf("`requestBody` at path `%s`", path),
			//				"description", operation.Delete.GenerateJSONPath(),
			//				operation.Delete.RequestBody.Value.GoLow().KeyNode, operation.Delete)
			//		}
			//
			//		// check responses
			//		if operation.Delete.Responses != nil {
			//			for responsePairs := operation.Delete.Responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			//				code := responsePairs.Key()
			//				response := responsePairs.Value()
			//				checkDescription(response.Value.Description, http.MethodDelete, fmt.Sprintf("code `%s` `responseBody` at path `%s`", code, path),
			//					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			//			}
			//
			//		}
			//	}
			//
			//	if operation.Options != nil {
			//		checkDescription(operation.Options.Value.Description, http.MethodOptions, fmt.Sprintf("at path `%s`", path),
			//			"description", operation.Options.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Options)
			//
			//		checkDescription(operation.Options.Value.Summary, http.MethodOptions, fmt.Sprintf("at path `%s`", path),
			//			"summary", operation.Options.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Options)
			//
			//		// check request body
			//		if operation.Options.RequestBody != nil {
			//			checkDescription(operation.Options.Value.RequestBody.Description, http.MethodOptions, fmt.Sprintf("`requestBody` at path `%s`", path),
			//				"description", operation.Options.GenerateJSONPath(),
			//				operation.Options.RequestBody.Value.GoLow().KeyNode, operation.Options)
			//		}
			//
			//		// check responses
			//		if operation.Options.Responses != nil {
			//			for responsePairs := operation.Options.Responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			//				code := responsePairs.Key()
			//				response := responsePairs.Value()
			//				checkDescription(response.Value.Description, http.MethodOptions, fmt.Sprintf("code `%s` `responseBody` at path `%s`", code, path),
			//					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			//			}
			//
			//		}
			//	}
			//
			//	if operation.Head != nil {
			//		checkDescription(operation.Head.Value.Description, http.MethodHead, fmt.Sprintf("at path `%s`", path),
			//			"description", operation.Head.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Head)
			//
			//		checkDescription(operation.Head.Value.Summary, http.MethodHead, fmt.Sprintf("at path `%s`", path),
			//			"summary", operation.Head.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Head)
			//
			//		// check request body
			//		if operation.Head.RequestBody != nil {
			//			checkDescription(operation.Head.Value.RequestBody.Description, http.MethodHead, fmt.Sprintf("`requestBody` at path `%s`", path),
			//				"description", operation.Head.GenerateJSONPath(),
			//				operation.Head.RequestBody.Value.GoLow().KeyNode, operation.Head)
			//		}
			//
			//		// check responses
			//		if operation.Head.Responses != nil {
			//			for responsePairs := operation.Head.Responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			//				code := responsePairs.Key()
			//				response := responsePairs.Value()
			//				checkDescription(response.Value.Description, http.MethodHead, fmt.Sprintf("code `%s` `responseBody` at path `%s`", code, path),
			//					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			//			}
			//
			//		}
			//	}
			//
			//	if operation.Patch != nil {
			//		checkDescription(operation.Patch.Value.Description, http.MethodPatch, fmt.Sprintf("at path `%s`", path),
			//			"description", operation.Patch.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Patch)
			//
			//		checkDescription(operation.Patch.Value.Summary, http.MethodPatch, fmt.Sprintf("at path `%s`", path),
			//			"summary", operation.Patch.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Patch)
			//
			//		// check request body
			//		if operation.Patch.RequestBody != nil {
			//			checkDescription(operation.Patch.Value.RequestBody.Description, http.MethodPatch, fmt.Sprintf("`requestBody` at path `%s`", path),
			//				"description", operation.Patch.GenerateJSONPath(),
			//				operation.Patch.RequestBody.Value.GoLow().KeyNode, operation.Patch)
			//		}
			//
			//		// check responses
			//		if operation.Patch.Responses != nil {
			//			for responsePairs := operation.Patch.Responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			//				code := responsePairs.Key()
			//				response := responsePairs.Value()
			//				checkDescription(response.Value.Description, http.MethodPatch, fmt.Sprintf("code `%s` `responseBody` at path `%s`", code, path),
			//					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			//			}
			//
			//		}
			//	}
			//
			//	if operation.Trace != nil {
			//		checkDescription(operation.Trace.Value.Description, http.MethodTrace, fmt.Sprintf("at path `%s`", path),
			//			"description", operation.Trace.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Trace)
			//
			//		checkDescription(operation.Trace.Value.Summary, http.MethodTrace, fmt.Sprintf("at path `%s`", path),
			//			"summary", operation.Trace.GenerateJSONPath(), operation.Value.GoLow().KeyNode, operation.Trace)
			//
			//		// check request body
			//		if operation.Trace.RequestBody != nil {
			//			checkDescription(operation.Trace.Value.RequestBody.Description, http.MethodTrace, fmt.Sprintf("`requestBody` at path `%s`", path),
			//				"description", operation.Trace.GenerateJSONPath(),
			//				operation.Trace.RequestBody.Value.GoLow().KeyNode, operation.Trace)
			//		}
			//
			//		// check responses
			//		if operation.Trace.Responses != nil {
			//			for responsePairs := operation.Trace.Responses.Codes.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			//				code := responsePairs.Key()
			//				response := responsePairs.Value()
			//				checkDescription(response.Value.Description, http.MethodTrace, fmt.Sprintf("code `%s` `responseBody` at path `%s`", code, path),
			//					"description", response.GenerateJSONPath(), response.Value.GoLow().KeyNode, response)
			//			}
			//
			//		}
			//	}
		}
	}

	//
	//if context.Index.GetPathsNode() == nil {
	//	return results
	//}
	//ops := context.Index.GetPathsNode().Content
	//
	//var opPath, opMethod string
	//for i, op := range ops {
	//	if i%2 == 0 {
	//		opPath = op.Value
	//		continue
	//	}
	//
	//	skip := false
	//	for m, method := range op.Content {
	//
	//		if m%2 == 0 {
	//			opMethod = method.Value
	//			if skip {
	//				skip = false
	//			}
	//			continue
	//		}
	//		// skip non-operations
	//		switch opMethod {
	//		case
	//			// No v2.*Label here, they're duplicates
	//			v3.GetLabel, v3.PutLabel, v3.PostLabel, v3.DeleteLabel, v3.OptionsLabel, v3.HeadLabel, v3.PatchLabel, v3.TraceLabel:
	//			// Ok, an operation
	//		default:
	//			skip = true
	//			continue
	//		}
	//		if skip {
	//			skip = false
	//			continue
	//		}
	//
	//		basePath := fmt.Sprintf("$.paths['%s'].%s", opPath, opMethod)
	//		descKey, descNode := utils.FindKeyNodeTop("description", method.Content)
	//		_, summNode := utils.FindKeyNodeTop("summary", method.Content)
	//		requestBodyKey, requestBodyNode := utils.FindKeyNodeTop("requestBody", method.Content)
	//		_, responsesNode := utils.FindKeyNode("responses", method.Content)
	//
	//		if descNode == nil {
	//
	//			// if there is no summary either, then report
	//			if summNode == nil {
	//				res := createDescriptionResult(fmt.Sprintf("operation `%s` at path `%s` is missing a description and a summary",
	//					opMethod, opPath), basePath, method, method)
	//				res.Rule = context.Rule
	//				results = append(results, res)
	//			}
	//
	//		} else {
	//
	//			// check if description is above a certain length of words
	//			words := strings.Split(descNode.Value, " ")
	//			if len(words) < minWords {
	//
	//				res := createDescriptionResult(fmt.Sprintf("operation `%s` description at path `%s` must be "+
	//					"at least %d words long, (%d is not enough)", opMethod, opPath, minWords, len(words)), basePath, descKey, descKey)
	//				res.Rule = context.Rule
	//				results = append(results, res)
	//			}
	//		}
	//		// check operation request body
	//		if requestBodyNode != nil {
	//
	//			descKey, descNode = utils.FindKeyNodeTop("description", requestBodyNode.Content)
	//			_, summNode = utils.FindKeyNodeTop("summary", requestBodyNode.Content)
	//
	//			if descNode == nil {
	//
	//				// if there is no summary either, then report
	//				if summNode == nil {
	//					res := createDescriptionResult(fmt.Sprintf("field `requestBody` for operation `%s` at path `%s` "+
	//						"is missing a description and a summary", opMethod, opPath),
	//						utils.BuildPath(basePath, []string{"requestBody"}), requestBodyKey, requestBodyKey)
	//					res.Rule = context.Rule
	//					results = append(results, res)
	//				}
	//
	//			} else {
	//
	//				// check if request body description is above a certain length of words
	//				words := strings.Split(descNode.Value, " ")
	//				if len(words) < minWords {
	//
	//					res := createDescriptionResult(fmt.Sprintf("field `requestBody` for operation `%s` description "+
	//						"at path `%s` must be at least %d words long, (%d is not enough)", opMethod, opPath,
	//						minWords, len(words)), basePath, descKey, descKey)
	//					res.Rule = context.Rule
	//					results = append(results, res)
	//				}
	//			}
	//		}
	//
	//		// check operation responses
	//		if responsesNode != nil {
	//
	//			// run through each response.
	//			var opCode string
	//			var opCodeNode *yaml.Node
	//			for z, response := range responsesNode.Content {
	//				if z%2 == 0 {
	//					opCode = response.Value
	//					opCodeNode = response
	//					continue
	//				}
	//				if strings.HasPrefix(opCode, "x-") {
	//					continue
	//				}
	//
	//				descKey, descNode = utils.FindKeyNodeTop("description", response.Content)
	//				_, summNode = utils.FindKeyNodeTop("summary", response.Content)
	//
	//				if descNode == nil {
	//
	//					// if there is no summary either, then report
	//					if summNode == nil {
	//						res := createDescriptionResult(fmt.Sprintf("operation `%s` response `%s` "+
	//							"at path `%s` is missing a description and a summary", opMethod, opCode, opPath),
	//							utils.BuildPath(basePath, []string{"requestBody"}), opCodeNode, opCodeNode)
	//						res.Rule = context.Rule
	//						results = append(results, res)
	//					}
	//				} else {
	//
	//					// check if response description is above a certain length of words
	//					words := strings.Split(descNode.Value, " ")
	//					if len(words) < minWords {
	//
	//						res := createDescriptionResult(fmt.Sprintf("operation `%s` response `%s` "+
	//							"description at path `%s` must be at least %d words long, (%d is not enough)", opMethod, opCode, opPath,
	//							minWords, len(words)), basePath, descKey, descKey)
	//						res.Rule = context.Rule
	//						results = append(results, res)
	//					}
	//				}
	//			}
	//		}
	//	}
	//}
	return results
}
