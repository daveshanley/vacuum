// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// OASSchema  will check that the document is a valid OpenAPI schema.
type OASSchema struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OASSchema rule.
func (os OASSchema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oas_schema",
	}
}

// RunRule will execute the OASSchema rule, based on supplied context and a supplied []*yaml.Node slice.
func (os OASSchema) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// grab the original bytes and the spec info from context.
	info := context.SpecInfo

	// rule cannot proceed until JSON parsing is complete. Wait on channel to signal all clear.
	<-info.GetJSONParsingChannel()

	if info.SpecType == "" {
		// spec type is un-known, there is no point in running this rule.
		return results
	}

	specBytes := *info.SpecJSONBytes

	// create loader from original bytes.
	doc := gojsonschema.NewStringLoader(string(specBytes))

	res, err := gojsonschema.Validate(info.APISchema, doc)

	if err != nil {
		results = append(results, model.RuleFunctionResult{
			Message:   fmt.Sprintf("OpenAPI specification cannot be validated: %s", err.Error()),
			StartNode: nodes[0],
			EndNode:   nodes[0],
			Path:      "$",
			Rule:      context.Rule,
		})
		return results
	}

	// if the spec is not valid, run through all the issues and return.
	if !res.Valid() {
		for _, err := range res.Errors() {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("OpenAPI specification is invalid: %s", err.Description()),
				StartNode: nodes[0],
				EndNode:   nodes[0],
				Path:      "$",
				Rule:      context.Rule,
			})
		}
	}
	return results
}
