// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
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
				Message:   fmt.Sprintf("Swagger specification cannot be validated: %v", validateErr.Error()),
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
					Message:   fmt.Sprintf("Swagger specification is invalid: %s", resErr.Description()),
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

		if failure, ok := validationError.(*jsonschema.ValidationError); ok {
			diveIntoFailure(failure.Causes, &results, nodes[0], context.Rule)
		}
		if failure, ok := validationError.(*jsonschema.InvalidJSONTypeError); ok {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("OpenAPI specification has `invalid` data: %v", failure.Error()),
				StartNode: nodes[0],
				EndNode:   nodes[0],
				Path:      "$",
				Rule:      context.Rule,
			})
		}
	}
	return results
}

func diveIntoFailure(validationErrors []*jsonschema.ValidationError,
	results *[]model.RuleFunctionResult,
	root *yaml.Node,
	rule *model.Rule) {
	for x := range validationErrors {
		if len(validationErrors[x].Causes) > 0 {
			diveIntoFailure(validationErrors[x].Causes, results, root, rule)
		}
		_, path := utils.ConvertComponentIdIntoFriendlyPathSearch(validationErrors[x].InstanceLocation)

		// try and find node using path.
		searchPath, err := yamlpath.NewPath(path)
		var foundNode *yaml.Node
		if err == nil {
			foundNodesFromPath, pErr := searchPath.Find(root)
			if pErr != nil {
				foundNode = root
			} else {
				foundNode = foundNodesFromPath[0]
			}
		}
		*results = append(*results, model.RuleFunctionResult{
			Message: fmt.Sprintf("OpenAPI specification is `invalid`: %s %v",
				validationErrors[x].InstanceLocation,
				validationErrors[x].Message),
			StartNode: foundNode,
			EndNode:   foundNode,
			Path:      path,
			Rule:      rule,
		})
	}
}
