// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// OpenAPISchema is a rule that creates a schema check against a field value
type OpenAPISchema struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationParameters rule.
func (sch OpenAPISchema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oas_schema",
	}
}

// RunRule will execute the OpenAPISchema function
func (sch OpenAPISchema) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	schema := utils.ExtractValueFromInterfaceMap("schema", context.Options).(parser.Schema)

	for x, node := range nodes {

		// find field from rule
		_, field := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if field != nil {

			// validate using schema provided.
			res, _ := parser.ValidateNodeAgainstSchema(&schema, field, false)

			for _, resError := range res.Errors() {

				r := model.BuildFunctionResultString(fmt.Sprintf("%s: %s", context.Rule.Description, resError.Description()))
				r.StartNode = field
				r.EndNode = field
				if p, ok := context.Given.(string); ok {
					r.Path = fmt.Sprintf("%s[%d]", p, x)
				}
				results = append(results, r)
			}
		}
	}
	return results
}
