// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

type CheckErrorResponse struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (er CheckErrorResponse) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "owaspCheckErrorResponse",
		Required: []string{"code"},
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "code",
				Description: "HTTP Response code to search against",
			},
		},
		ErrorMessage: "'owaspCheckErrorResponse' function has invalid options supplied. Set the 'code' property to a valid integer",
	}
}

// GetCategory returns the category of the CheckErrorResponse rule.
func (er CheckErrorResponse) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (er CheckErrorResponse) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	// iterate through all paths looking for responses
	codeValue := utils.ExtractValueFromInterfaceMap("code", context.Options)
	if codeValue == nil {
		return nil
	}
	code := fmt.Sprintf("%v", codeValue)

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	doc := context.Document
	drDoc := context.DrDocument.V3Document
	if doc == nil {
		return results
	}

	if doc.GetSpecInfo().VersionNumeric <= 2 {
		return results
	}
	if drDoc.Paths != nil && drDoc.Paths.PathItems != nil {
		for pathPairs := drDoc.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
			for opPairs := pathPairs.Value().GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
				opValue := opPairs.Value()
				opType := opPairs.Key()

				if opValue.Responses != nil && opValue.Responses.Codes != nil {
					responses := opValue.Responses.Codes
					found := false
					schemaMissing := true
					var node *yaml.Node

					for respPairs := responses.First(); respPairs != nil; respPairs = respPairs.Next() {
						resp := respPairs.Value()
						respCode := respPairs.Key()
						if respCode == code {
							found = true
							node = resp.Value.GoLow().Content.KeyNode
							if resp.Content.First() != nil {
								if resp.Content.First().Value().Value.Schema != nil {
									schemaMissing = false
								}
							}
						}
					}
					if node == nil {
						n := responses.GetOrZero(code)
						if n != nil {
							node = n.Value.GoLow().RootNode
						} else {
							node = opValue.Responses.Value.GoLow().KeyNode
						}
					}
					if !found {
						result := model.RuleFunctionResult{
							Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
								fmt.Sprintf("missing response code `%s` for `%s`", code, strings.ToUpper(opType))),
							StartNode: node,
							EndNode:   vacuumUtils.BuildEndNode(node),
							Path:      fmt.Sprintf("$.paths['%s'].%s.responses", pathPairs.Key(), opType),
							Rule:      context.Rule,
						}
						opValue.AddRuleFunctionResult(base.ConvertRuleResult(&result))
						results = append(results, result)
					}
					if schemaMissing && found {
						result := model.RuleFunctionResult{
							Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
								fmt.Sprintf("missing schema for `%s` response on `%s`", code, strings.ToUpper(opType))),
							StartNode: node,
							EndNode:   vacuumUtils.BuildEndNode(node),
							Path:      fmt.Sprintf("$.paths['%s'].%s.responses['%s']", pathPairs.Key(), opType, code),
							Rule:      context.Rule,
						}
						opValue.AddRuleFunctionResult(base.ConvertRuleResult(&result))
						results = append(results, result)
					}
				}
			}
		}
	}
	return results
}
