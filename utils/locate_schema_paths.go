// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// schemaPathResult caches the result of a LocateSchemaPropertyPaths call.
type schemaPathResult struct {
	primaryPath string
	allPaths    []string
}

// LocateSchemaPropertyPaths finds all paths where a schema property appears in the document.
// It uses DrDocument.LocateModelsByKeyAndValue to find all locations where the schema
// is referenced, not just its definition location.
// Results are cached in context.SchemaPathCache when available, so multiple OWASP rules
// checking the same schema avoid redundant LocateModelsByKeyAndValue calls.
// Returns the primary path and all paths where the schema appears.
func LocateSchemaPropertyPaths(
	context model.RuleFunctionContext,
	schema *v3.Schema,
	keyNode *yaml.Node,
	valueNode *yaml.Node,
) (primaryPath string, allPaths []string) {
	// Check cache first
	if context.SchemaPathCache != nil {
		if cached, ok := context.SchemaPathCache.Load(schema); ok {
			r := cached.(*schemaPathResult)
			return r.primaryPath, r.allPaths
		}
	}

	// Start with the schema's own path
	primaryPath = schema.GenerateJSONPath()

	lookupCompleted := false

	// Try to find all locations where this schema appears
	if context.DrDocument != nil && keyNode != nil && valueNode != nil {
		locatedObjects, err := context.DrDocument.LocateModelsByKeyAndValue(keyNode, valueNode)
		if err == nil {
			lookupCompleted = true
		}
		if err == nil && locatedObjects != nil && len(locatedObjects) > 0 {
			// Use the first located object's path as the primary path
			primaryPath = locatedObjects[0].GenerateJSONPath()

			// Collect all paths
			allPaths = make([]string, 0, len(locatedObjects))
			for _, obj := range locatedObjects {
				allPaths = append(allPaths, obj.GenerateJSONPath())
			}

			// Store in cache
			if context.SchemaPathCache != nil {
				context.SchemaPathCache.Store(schema, &schemaPathResult{
					primaryPath: primaryPath,
					allPaths:    allPaths,
				})
			}
			return primaryPath, allPaths
		}
	}

	// If we couldn't locate via LocateModelsByKeyAndValue,
	// fall back to the schema's own path
	allPaths = []string{primaryPath}

	// Only cache fallback results when a full lookup actually ran.
	// This prevents nil-node calls from poisoning the cache with incomplete paths.
	if context.SchemaPathCache != nil && lookupCompleted {
		context.SchemaPathCache.Store(schema, &schemaPathResult{
			primaryPath: primaryPath,
			allPaths:    allPaths,
		})
	}
	return primaryPath, allPaths
}
