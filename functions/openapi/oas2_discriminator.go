// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// OAS2Discriminator checks swagger schemas are using discriminators properly.
type OAS2Discriminator struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationSingleTag rule.
func (od OAS2Discriminator) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oas2_discriminator",
	}
}

// RunRule will execute the OperationSingleTag rule, based on supplied context and a supplied []*yaml.Node slice.
func (od OAS2Discriminator) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	schemas := context.Index.GetAllSchemas()

	for id, schema := range schemas {

		discriminator, dv := utils.FindKeyNode("discriminator", schema.Node.Content)

		if discriminator != nil {

			// swagger needs this to be a string, openapi wants a map.
			_, path := utils.ConvertComponentIdIntoPath(id)

			if !utils.IsNodeStringValue(dv) {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("the schema '%s' uses a non string discriminator", id),
					StartNode: discriminator,
					EndNode:   dv,
					Path:      path,
					Rule:      context.Rule,
				})
			}

			// if there is a discriminator, required must be set and contain the discriminator.
			required, rv := utils.FindKeyNode("required", schema.Node.Content)

			if required == nil {

				// you can't use a discriminator without 'required' being set in the schema.
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("schema '%s' uses a discriminator but has no "+
						"'required' property set", id),
					StartNode: discriminator,
					EndNode:   dv,
					Path:      path,
					Rule:      context.Rule,
				})
				continue // no point going on.
			}

			reqFound := false
			for _, req := range rv.Content {
				if req.Value == dv.Value {
					reqFound = true
				}
			}
			if !reqFound {

				// required values for schema did not contain discriminator
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("schema '%s' uses a discriminator but is not "+
						"included in 'required' properties", id),
					StartNode: discriminator,
					EndNode:   dv,
					Path:      path,
					Rule:      context.Rule,
				})
			}
		}
	}
	return results
}
