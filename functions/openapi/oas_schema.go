// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/santhosh-tekuri/jsonschema/v5"
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

	// Swagger specs are not supported with this schema checker (annoying, but you get what you pay for).
	schema, err := jsonschema.CompileString("schema.json", info.APISchema)
	if err != nil {

		// do the swagger thing.
		swaggerSchema := gojsonschema.NewStringLoader(info.APISchema)
		spec := gojsonschema.NewStringLoader(string(*info.SpecJSONBytes))
		res, validateErr := gojsonschema.Validate(swaggerSchema, spec)

		if validateErr != nil {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("OpenAPI specification cannot be validated: %v", validateErr.Error()),
				StartNode: nodes[0],
				EndNode:   nodes[0],
				Path:      "$",
				Rule:      context.Rule,
			})
			return results
		}

		// if the spec is not valid, run through all the issues and return.
		if !res.Valid() {
			for _, resErr := range res.Errors() {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("OpenAPI specification is invalid: %s", resErr.Description()),
					StartNode: nodes[0],
					EndNode:   nodes[0],
					Path:      "$",
					Rule:      context.Rule,
				})
			}
			return results
		}
		return nil
	}

	//validate using faster, more accurate resolver.
	if validationError := schema.Validate(*info.SpecJSON); validationError != nil {
		failure := validationError.(*jsonschema.ValidationError)
		for _, fail := range failure.Causes {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("OpenAPI specification is invalid: %s %v", fail.KeywordLocation,
					fail.Message),
				StartNode: nodes[0],
				EndNode:   nodes[0],
				Path:      "$",
				Rule:      context.Rule,
			})
		}
	}
	return results
}
