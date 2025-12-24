// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// LocateComponentPaths finds all paths where a component appears in the document.
// It uses DrDocument.LocateModelsByKeyAndValue to find all locations where the component
// is referenced, not just its definition location.
// This is a generic version that works with any component type that has GenerateJSONPath.
// Returns the primary path and all paths where the component appears.
func LocateComponentPaths(
	context model.RuleFunctionContext,
	component v3.Foundational,
	keyNode *yaml.Node,
	valueNode *yaml.Node,
) (primaryPath string, allPaths []string) {
	// Start with the component's own path
	primaryPath = component.GenerateJSONPath()

	// Try to find all locations where this component appears
	if context.DrDocument != nil && keyNode != nil && valueNode != nil {
		locatedObjects, err := context.DrDocument.LocateModelsByKeyAndValue(keyNode, valueNode)
		if err == nil && locatedObjects != nil && len(locatedObjects) > 0 {
			// Use the first located object's path as the primary path
			primaryPath = locatedObjects[0].GenerateJSONPath()

			// Collect all paths
			for _, obj := range locatedObjects {
				allPaths = append(allPaths, obj.GenerateJSONPath())
			}
			return primaryPath, allPaths
		}
	}

	// If we couldn't locate via LocateModelsByKeyAndValue,
	// fall back to the component's own path
	allPaths = []string{primaryPath}
	return primaryPath, allPaths
}
