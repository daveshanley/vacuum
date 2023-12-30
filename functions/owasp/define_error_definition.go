package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/base"
	v3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"slices"
	"strings"
)

type DefineErrorDefinition struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (cd DefineErrorDefinition) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "define_error_definition"}
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd DefineErrorDefinition) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.Options == nil {
		return results
	}

	// iterate through all paths looking for responses
	codes := utils.ExtractValueFromInterfaceMap("codes", context.Options).([]string)

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
	for pathPairs := drDoc.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
		for opPairs := pathPairs.Value().GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
			opValue := opPairs.Value()

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
					Message:   fmt.Sprintf("missing one of `%s` response codes", code),
					StartNode: node,
					EndNode:   node,
					Path:      fmt.Sprintf("%s.%s", opValue.GenerateJSONPath(), "responses"),
					Rule:      context.Rule,
				}
				opValue.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
