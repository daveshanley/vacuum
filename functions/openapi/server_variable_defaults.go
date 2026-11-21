// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/openapi"
	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/go-yaml"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	libutils "github.com/pb33f/libopenapi/utils"
)

// ServerVariableDefaults checks the advisory enum membership of OpenAPI 3.0 server defaults.
type ServerVariableDefaults struct{}

// GetSchema identifies the server variable default rule function.
func (ServerVariableDefaults) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasServerVariableDefaults"}
}

// GetCategory returns the OpenAPI function category.
func (ServerVariableDefaults) GetCategory() string { return model.FunctionCategoryOpenAPI }

// RunRule checks Server Objects in the document, including those on operations and links.
func (ServerVariableDefaults) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	return runDocumentCheck(nodes, ctx, func(doc *v3.Document, _ model.RuleFunctionContext, add reportRelationship) {
		openapi.OperationVisitor{Server: checkServerDefaults("should", true, add)}.Walk(doc)
	})
}

// checkServerDefaults reports defaults outside their enums. With checkEmpty, an empty enum is
// reported too; OpenAPI 3.1 and later leave that to the schema, which requires at least one item.
func checkServerDefaults(obligation string, checkEmpty bool, add reportRelationship) func(*v3.Server, string) {
	seen := make(map[*yaml.Node]bool)
	return func(s *v3.Server, at string) {
		for name, variable := range s.Variables.FromOldest() {
			if variable == nil || variable.GoLow() == nil {
				continue
			}
			low := variable.GoLow()
			n := low.Default.ValueNode
			if n == nil || seen[low.KeyNode] {
				continue
			}
			if len(low.Enum) == 0 {
				if !checkEmpty {
					continue
				}
				enum := libutils.NodeAlias(findMappingValue(low.RootNode, "enum"))
				if enum == nil || enum.Kind != yaml.SequenceNode {
					continue
				}
			}
			seen[low.KeyNode] = true
			found := false
			for _, entry := range low.Enum {
				if entry.Value == low.Default.Value {
					found = true
					break
				}
			}
			if !found {
				add(n, at+"/variables/"+repair.EscapeToken(name)+"/default", fmt.Sprintf("server variable %q default %q %s be a member of its enum", name, low.Default.Value, obligation))
			}
		}
	}
}
