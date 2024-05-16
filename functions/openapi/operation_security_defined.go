// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// OperationSecurityDefined is a rule that checks operation security against defined global schemes.
type OperationSecurityDefined struct {
}

// GetCategory returns the category of the OperationSecurityDefined rule.
func (osd OperationSecurityDefined) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationSecurityDefined rule.
func (osd OperationSecurityDefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "operation_security_defined",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "schemesPath",
				Description: "operation_security_defined requires a schemesPath in which to look up security definitions",
			},
		},
		MinProperties: 1,
		MaxProperties: 1,
		ErrorMessage:  "operation_security_defined requires a 'schemesPath'",
	}
}

// RunRule will execute the OperationSecurityDefined rule, based on supplied context and a supplied []*yaml.Node slice.
func (osd OperationSecurityDefined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	paths := context.Index.GetAllPaths()
	securityDefinitions := context.Index.GetAllSecuritySchemes()

	for path, methodMap := range paths {

		for method, methodNode := range methodMap {

			_, securityNode := utils.FindKeyNode("security", methodNode.Node.Content)

			if securityNode != nil {

				basePath := fmt.Sprintf("$.paths['%s'].%s", path, method)

				results = osd.checkSecurityNode(securityNode, securityDefinitions, results,
					basePath, methodNode.Node, context)
			}
		}
	}

	// look through root security if it has been set.
	rootSecurity := context.Index.GetRootSecurityNode()
	if rootSecurity != nil {
		basePath := "$"

		results = osd.checkSecurityNode(rootSecurity, securityDefinitions, results,
			basePath, rootSecurity, context)
	}

	return results
}

func (osd OperationSecurityDefined) checkSecurityNode(securityNode *yaml.Node,
	securityDefinitions map[string]*index.Reference, results []model.RuleFunctionResult,
	basePath string, startNode *yaml.Node,
	context model.RuleFunctionContext) []model.RuleFunctionResult {

	// look through each security item and check it exists in the global security index.
	for i, securityItem := range securityNode.Content {
		if len(securityItem.Content) > 0 {
			// name is key and role scope an array value.
			name := securityItem.Content[0]
			if name != nil {

				// lookup in security definitions
				lookup := fmt.Sprintf("#/components/securitySchemes/%s", name.Value)
				if securityDefinitions[lookup] == nil {

					results = append(results, model.RuleFunctionResult{
						Message: fmt.Sprintf("Security definition points a non-existent "+
							"securityScheme `%s`", name.Value),
						StartNode: name,
						EndNode:   vacuumUtils.BuildEndNode(name),
						Path:      fmt.Sprintf("%s.security[%d]", basePath, i),
						Rule:      context.Rule,
					})
				}
			}
		}
	}
	return results
}
