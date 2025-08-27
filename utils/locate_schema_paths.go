// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/v3"
	"gopkg.in/yaml.v3"
)

// LocateSchemaPropertyPaths finds all paths where a schema property appears in the document.
// It uses DrDocument.LocateModelsByKeyAndValue to find all locations where the schema
// is referenced, not just its definition location.
// Returns the primary path and all paths where the schema appears.
func LocateSchemaPropertyPaths(
	context model.RuleFunctionContext,
	schema *v3.Schema,
	keyNode *yaml.Node,
	valueNode *yaml.Node,
) (primaryPath string, allPaths []string) {
	// Start with the schema's own path
	primaryPath = schema.GenerateJSONPath()
	
	// Try to find all locations where this schema appears
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
	// fall back to the schema's own path
	allPaths = []string{primaryPath}
	return primaryPath, allPaths
}