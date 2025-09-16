package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	v3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
	"slices"
	"strings"
)

type DefineErrorDefinition struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (cd DefineErrorDefinition) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "owaspDefineErrorDefinition",
		Required: []string{"codes"},
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "codes",
				Description: "Array of HTTP Response code to search against",
			},
		},
		ErrorMessage: "'owaspDefineErrorDefinition' function has invalid options supplied. Set the 'codes' property to a valid integer",
	}
}

// GetCategory returns the category of the DefineErrorDefinition rule.
func (cd DefineErrorDefinition) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd DefineErrorDefinition) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.Options == nil {
		return results
	}

	// iterate through all paths looking for responses
	rawCodesValue := utils.ExtractValueFromInterfaceMap("codes", context.Options)
	if rawCodesValue == nil {
		return results
	}

	rawCodes := rawCodesValue.([]interface{})

	codes := make([]string, len(rawCodes))
	for i, v := range rawCodes {
		codes[i] = v.(string)
	}

	if context.DrDocument == nil {
		return results
	}

	drDoc := context.DrDocument.V3Document
	if drDoc == nil {
		return results
	}

	return processCodes(codes, drDoc, context)
}

func processCodes(codes []string, drDoc *v3.Document, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if drDoc.Paths != nil && drDoc.Paths.PathItems != nil {
		for pathPairs := drDoc.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
			for opPairs := pathPairs.Value().GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
				opValue := opPairs.Value()

				if opValue.Responses != nil && opValue.Responses.Codes != nil {
					responses := opValue.Responses.Codes
					seen := make(map[string]bool)

					var node *yaml.Node

					for respPairs := responses.First(); respPairs != nil; respPairs = respPairs.Next() {
						respCode := respPairs.Key()
						if slices.Contains(codes, respCode) {
							seen[respCode] = true
						}
					}
					node = opValue.Value.GoLow().Responses.KeyNode

					if len(seen) <= 0 {
						code := strings.Join(codes, "`, `")
						result := model.RuleFunctionResult{
							Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
								fmt.Sprintf("missing one of `%s` response codes", code)),
							StartNode: node,
							EndNode:   vacuumUtils.BuildEndNode(node),
							Path:      fmt.Sprintf("%s.%s", opValue.GenerateJSONPath(), "responses"),
							Rule:      context.Rule,
						}
						opValue.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
						results = append(results, result)
					}
				}
			}
		}
	}
	return results
}
