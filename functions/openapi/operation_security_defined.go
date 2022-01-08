// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// OperationSecurityDefined is a rule that checks operation security against defined global schemes.
type OperationSecurityDefined struct {
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
	var results []model.RuleFunctionResult

	if len(nodes) <= 0 {
		return nil
	}

	var schemesPath string
	if context.Options == nil {
		return results
	}
	if opts := utils.ConvertInterfaceIntoStringMap(context.Options); opts != nil {
		if v, ok := opts["schemesPath"]; ok {
			schemesPath = v
		}
	} else {
		return results // can't do anything without a schemesPath to look at.
	}

	ops := GetOperationsFromRoot(nodes)

	for _, node := range nodes {
		securitySchemes, _ := utils.FindNodesWithoutDeserializing(node, schemesPath)
		var definedSchemes []string
		for _, schemeNode := range securitySchemes {
			for i, scheme := range schemeNode.Content {
				if i%2 == 0 {
					definedSchemes = append(definedSchemes, scheme.Value)
				}
			}
		}

		// now lets pull out all operations and then look for security definitions on them.
		var path string
		for _, op := range ops {
			if op.Kind == yaml.ScalarNode {
				path = op.Value
			} else {
				_, secVal := utils.FindFirstKeyNode("security", []*yaml.Node{op}, 0)

				if secVal != nil {
					for n, sec := range secVal.Content[0].Content {
						if n%2 == 0 {
							if !osd.isSecuritySchemeNameDefined(sec.Value, definedSchemes) {
								results = append(results, model.RuleFunctionResult{
									Message:   fmt.Sprintf("operation at '%s' references an undefined security schema '%s'", path, sec.Value),
									StartNode: sec,
									EndNode:   sec,
									Path:      fmt.Sprintf("$.paths.%s..security", path),
								})
							}
						}
					}
				}
			}
		}
	}

	return results
}

func (osd OperationSecurityDefined) isSecuritySchemeNameDefined(name string, defined []string) bool {
	found := false
	for _, def := range defined {
		if name == def {
			found = true
		}
	}
	return found
}
