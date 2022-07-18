// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
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

				basePath := fmt.Sprintf("$.paths.%s.%s", path, method)

				results = sd.checkSecurityNode(securityNode, securityDefinitions, results,
					basePath, methodNode.Node, lastNode, context)
			}
		}
	}

	// look through root security if it has been set.
	rootSecurity := context.Index.GetRootSecurityNode()
	if rootSecurity != nil {
		basePath := "$"
		lastNode := utils.FindLastChildNode(rootSecurity)

		results = sd.checkSecurityNode(rootSecurity, securityDefinitions, results,
			basePath, rootSecurity, lastNode, context)
	}

	return results
}

func (sd OAS2OperationSecurityDefined) checkSecurityNode(securityNode *yaml.Node,
	securityDefinitions map[string]*index.Reference, results []model.RuleFunctionResult,
	basePath string, startNode *yaml.Node, endNode *yaml.Node,
	context model.RuleFunctionContext) []model.RuleFunctionResult {

	// look through each security item and check it exists in the global security index.
	for i, securityItem := range securityNode.Content {

		// name is key and role scope an array value.
		if len(securityItem.Content) == 0 {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("Security definition is empty, no reference found"),
				StartNode: startNode,
				EndNode:   endNode,
				Path:      fmt.Sprintf("%s.security[%d]", basePath, i),
				Rule:      context.Rule,
			})
			continue
		}

		name := securityItem.Content[0]
		if name != nil {

			// lookup in security definitions
			lookup := fmt.Sprintf("#/securityDefinitions/%s", name.Value)
			if securityDefinitions[lookup] == nil {

				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("Security definition points a non-existent "+
						"securityDefinition '%s'", name.Value),
					StartNode: startNode,
					EndNode:   endNode,
					Path:      fmt.Sprintf("%s.security[%d]", basePath, i),
					Rule:      context.Rule,
				})
			}
		}
	}
	return results
}
