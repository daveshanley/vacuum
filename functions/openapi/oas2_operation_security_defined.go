// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// OAS2OperationSecurityDefined will check to make sure operation security has been defined correctly for swagger docs.
type OAS2OperationSecurityDefined struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the UniqueOperationId rule.
func (sd OAS2OperationSecurityDefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oas2_operation_security_defined",
	}
}

// RunRule will execute the OAS2OperationSecurityDefined rule, based on supplied context and a supplied []*yaml.Node slice.
func (sd OAS2OperationSecurityDefined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	paths := context.Index.GetAllPaths()
	securityDefinitions := context.Index.GetAllSecuritySchemes()

	for path, methodMap := range paths {

		for method, methodNode := range methodMap {

			_, securityNode := utils.FindKeyNode("security", methodNode.Node.Content)
			lastNode := utils.FindLastChildNode(methodNode.Node)

			if securityNode != nil {

				results = sd.checkSecurityNode(securityNode, securityDefinitions, results,
					method, path, methodNode, lastNode, context)
			}
		}
	}
	return results
}

func (sd OAS2OperationSecurityDefined) checkSecurityNode(securityNode *yaml.Node,
	securityDefinitions map[string]*model.Reference, results []model.RuleFunctionResult,
	method string, path string, methodNode *model.Reference, lastNode *yaml.Node,
	context model.RuleFunctionContext) []model.RuleFunctionResult {

	// look through each security item and check it exists in the global security index.
	for i, securityItem := range securityNode.Content {

		// name is key and role scope an array value.
		name := securityItem.Content[0]
		if name != nil {

			// lookup in security definitions
			lookup := fmt.Sprintf("#/securityDefinitions/%s", name.Value)
			if securityDefinitions[lookup] == nil {

				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("the '%s' operation at '%s' references a non-existent "+
						"security definition '%s'", method, path, name.Value),
					StartNode: methodNode.Node,
					EndNode:   lastNode,
					Path:      fmt.Sprintf("$.paths.%s.%s.security[%d]", path, method, i),
					Rule:      context.Rule,
				})
			}
		}
	}
	return results
}
