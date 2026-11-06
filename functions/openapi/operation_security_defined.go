// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"net/url"
)

// OperationSecurityDefined checks that security requirements identify declared security schemes.
type OperationSecurityDefined struct {
}

// GetCategory returns the category of the OperationSecurityDefined rule.
func (osd OperationSecurityDefined) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationSecurityDefined rule.
func (osd OperationSecurityDefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasOpSecurityDefined",
	}
}

// RunRule will execute the OperationSecurityDefined rule, based on supplied context and a supplied []*yaml.Node slice.
func (osd OperationSecurityDefined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	paths := context.Index.GetAllPaths()
	securityByIndex := map[*index.SpecIndex]map[string]*index.Reference{
		context.Index: context.Index.GetAllSecuritySchemes(),
	}

	for path, methodMap := range paths {

		for method, methodNode := range methodMap {

			_, securityNode := utils.FindKeyNode("security", methodNode.Node.Content)

			if securityNode != nil {

				basePath := fmt.Sprintf("$.paths['%s'].%s", path, method)

				results = osd.checkSecurityNode(securityNode, securityByIndex, results,
					basePath, context)
			}
		}
	}

	// look through root security if it has been set.
	rootSecurity := context.Index.GetRootSecurityNode()
	if rootSecurity != nil {
		basePath := "$"

		results = osd.checkSecurityNode(rootSecurity, securityByIndex, results,
			basePath, context)
	}

	return results
}

func (osd OperationSecurityDefined) checkSecurityNode(securityNode *yaml.Node,
	securityByIndex map[*index.SpecIndex]map[string]*index.Reference, results []model.RuleFunctionResult,
	basePath string,
	context model.RuleFunctionContext) []model.RuleFunctionResult {

	allowURI := allowsSecuritySchemeURI(context)
	securityDefinitions := securityByIndex[context.Index]
	localDefinitions := securityDefinitions
	locatedSource := false
	var documentURI *url.URL

	// Every key in a requirement object names a scheme that must be defined.
	for i, securityItem := range securityNode.Content {
		if !utils.IsNodeMap(securityItem) {
			continue
		}
		for j := 0; j+1 < len(securityItem.Content); j += 2 {
			name := securityItem.Content[j]
			if name != nil {
				if allowURI && !locatedSource && securityDefinitions[securitySchemePrefix+name.Value] == nil {
					locatedSource = true
					// Locate URI ownership once per security block; named schemes stay rooted.
					source := securitySchemeSource(context.Index, securityNode)
					if source == nil {
						allowURI = false
					} else {
						documentURI = securityDocumentURI(source)
						if version, ok := source.ResolveDocumentVersion(); ok {
							allowURI = version == 3.2
						}
						localDefinitions = securityByIndex[source]
						if localDefinitions == nil {
							localDefinitions = source.GetAllSecuritySchemes()
							securityByIndex[source] = localDefinitions
						}
					}
				}

				if securitySchemeKey(name.Value, securityDefinitions, localDefinitions, allowURI, documentURI) == "" {

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
