// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// UnusedComponent will check if a component or definition has been created, but it's not used anywhere by anything.
type UnusedComponent struct {
}

type refResult struct {
	ref        string
	refDefName string
	node       *yaml.Node
	notFound   bool
	path       string
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

	// check components, parameters and definitions (swagger)
	for _, node := range nodes {

		var refsToCheck []*refResult
		var seenResults []*refResult

		foundRefs, _ := utils.FindNodesWithoutDeserializing(node, "$..[?(@.$ref)]")
		for _, component := range foundRefs {
			for i, ref := range component.Content {
				if i%2 != 0 {
					refSegs := strings.Split(ref.Value, "/")
					refsToCheck = append(refsToCheck, &refResult{ref: ref.Value, node: ref, refDefName: refSegs[len(refSegs)-1]})
				}
			}
		}

		checkRefsExist(node, refsToCheck)

		for _, ref := range refsToCheck {
			if ref.notFound {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("$ref '%s'  does not exist in the document (cannot be found)", ref.ref),
					StartNode: ref.node,
					EndNode:   ref.node,
					Path:      ref.path,
				})
			}
		}

		// now lets check if everything we find in foundRefs, parameters and definitions (swagger) matches up with
		// all the refs that have been defined.
		foundComponents, _ := utils.FindNodesWithoutDeserializing(node, "$.components")
		foundParameters, _ := utils.FindNodesWithoutDeserializing(node, "$.parameters")
		foundDefinitions, _ := utils.FindNodesWithoutDeserializing(node, "$.definitions")

		sc := processSeenComponents(foundComponents, seenResults)
		sp := processSeenParams(foundParameters, seenResults, "parameters")
		sd := processSeenParams(foundDefinitions, seenResults, "definitions")

		seenResults = append(seenResults, sc...)
		seenResults = append(seenResults, sp...)
		seenResults = append(seenResults, sd...)

		unusedComponents := checkForUnusedComponents(seenResults, refsToCheck)

		// for everything that's not referenced, create a new result.
		for _, comp := range unusedComponents {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("the definition '%s' is potentially unused or has been orphaned", comp.refDefName),
				StartNode: comp.node,
				EndNode:   comp.node,
				Path:      comp.path,
			})
		}
	}

	return results
}

func processSeenComponents(foundComponents []*yaml.Node, seenResults []*refResult) []*refResult {
	if len(foundComponents) > 0 {
		var compType string
		for i, fc := range foundComponents[0].Content {
			if i%2 != 0 {
				for c, nameNode := range fc.Content {
					if c%2 == 0 {
						seenResults = append(seenResults, &refResult{
							ref:        nameNode.Value,
							refDefName: nameNode.Value,
							node:       nameNode,
							path:       fmt.Sprintf("$.%s.%s.%s", "components", compType, nameNode.Value),
						})
					}
				}
			} else {
				compType = fc.Value
			}
		}
	}
	return seenResults
}

func processSeenParams(foundComponents []*yaml.Node, seenResults []*refResult, label string) []*refResult {
	if len(foundComponents) > 0 {
		for i, fc := range foundComponents[0].Content {
			if i%2 == 0 {

				seenResults = append(seenResults, &refResult{
					ref:        fc.Value,
					refDefName: fc.Value,
					node:       fc,
					path:       fmt.Sprintf("$.%s.%s", label, fc.Value),
				})
			}
		}
	}
	return seenResults
}

func checkRefsExist(rootNode *yaml.Node, results []*refResult) {
	for _, result := range results {
		path := convertRefIntoPath(result.ref)
		found, _ := utils.FindNodesWithoutDeserializing(rootNode, path)
		if len(found) <= 0 {
			result.notFound = true
		}
		result.path = path
	}
}

func convertRefIntoPath(ref string) string {
	if ref[0] == '#' {
		ref = ref[2:]
	}
	return fmt.Sprintf("$.%s", strings.ReplaceAll(ref, "/", "."))
}

func checkForUnusedComponents(seenDefs []*refResult, knownRefs []*refResult) []*refResult {
	var unused []*refResult
	for _, def := range seenDefs {

		known := false
		for _, ref := range knownRefs {
			if ref.refDefName == def.refDefName && ref.path == def.path {
				known = true
			}
		}
		if !known {
			unused = append(unused, def)
		}
	}
	return unused
}
