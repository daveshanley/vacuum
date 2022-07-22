// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// UnusedComponent will check if a component or definition has been created, but it's not used anywhere by anything.
type UnusedComponent struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the UnusedComponent rule.
func (uc UnusedComponent) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "unused_component",
	}
}

// RunRule will execute the UnusedComponent rule, based on supplied context and a supplied []*yaml.Node slice.
func (uc UnusedComponent) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// extract all references, and every single component
	allRefs := context.Index.GetAllReferences()
	schemas := context.Index.GetAllSchemas()
	responses := context.Index.GetAllResponses()
	parameters := context.Index.GetAllParameters()
	examples := context.Index.GetAllExamples()
	requestBodies := context.Index.GetAllRequestBodies()
	headers := context.Index.GetAllHeaders()
	securitySchemes := context.Index.GetAllSecuritySchemes()
	links := context.Index.GetAllLinks()
	callbacks := context.Index.GetAllCallbacks()

	// if a component does not exist in allRefs, it was not referenced anywhere.
	notUsed := make(map[string]*index.Reference)

	// make this simple to iterate.
	mapsToSearch := []map[string]*index.Reference{
		schemas,
		responses,
		parameters,
		examples,
		requestBodies,
		headers,
		securitySchemes,
		links,
		callbacks,
	}

	// find everything that was never referenced.
	for _, resultMap := range mapsToSearch {
		for key, ref := range resultMap {
			if allRefs[key] == nil {
				// nothing is using this!
				notUsed[key] = ref
			}
		}
	}

	// for every orphan, build a result.
	for key, ref := range notUsed {
		_, path := utils.ConvertComponentIdIntoPath(ref.Definition)

		// roll back node by one, so we have the actual start.
		rolledBack := *ref.Node
		rolledBack.Line = ref.Node.Line - 1
		results = append(results, model.RuleFunctionResult{
			Message:   fmt.Sprintf("`%s` is potentially unused or has been orphaned", key),
			StartNode: &rolledBack,
			EndNode:   utils.FindLastChildNode(ref.Node),
			Path:      path,
			Rule:      context.Rule,
		})
	}

	return results
}
