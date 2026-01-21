package utils

import (
	"strings"

	base "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// DirectionType represents where a schema is used in an OpenAPI document
type DirectionType string

const (
	DirectionBoth     DirectionType = "both"
	DirectionRequest  DirectionType = "request"
	DirectionResponse DirectionType = "response"
	DirectionNone     DirectionType = "none"
)

// GetSchemaDirection determines if a schema is used in requests, responses, or both.
// Returns one of: "both", "request", "response", or "none"
func GetSchemaDirection(doc *v3.Document, schemaName string) DirectionType {
	if doc == nil || doc.Paths == nil {
		return DirectionNone
	}

	usedInRequest := false
	usedInResponse := false

	// Iterate over all paths and operations
	for pathPairs := doc.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
		pathItem := pathPairs.Value()
		if pathItem == nil {
			continue
		}

		// Check path-level parameters
		for _, param := range pathItem.Parameters {
			if param != nil && param.Schema != nil {
				if refMatches(param.Schema, schemaName) {
					usedInRequest = true
				}
			}
		}

		for _, op := range []*v3.Operation{
			pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Patch,
			pathItem.Delete, pathItem.Options, pathItem.Head, pathItem.Trace,
		} {
			if op == nil {
				continue
			}

			// Check requestBody
			if op.RequestBody != nil && op.RequestBody.Content != nil {
				for mediaPairs := op.RequestBody.Content.First(); mediaPairs != nil; mediaPairs = mediaPairs.Next() {
					media := mediaPairs.Value()
					if media.Schema != nil && refMatches(media.Schema, schemaName) {
						usedInRequest = true
					}
				}
			}

			// Check operation parameters
			for _, param := range op.Parameters {
				if param != nil && param.Schema != nil {
					if refMatches(param.Schema, schemaName) {
						usedInRequest = true
					}
				}
			}

			// Check responses
			if op.Responses != nil {
				// Check default response
				if op.Responses.Default != nil {
					checkResponse(op.Responses.Default, &usedInResponse, schemaName)
				}

				// Check coded responses
				if op.Responses.Codes != nil {
					for pairs := op.Responses.Codes.First(); pairs != nil; pairs = pairs.Next() {
						resp := pairs.Value()
						checkResponse(resp, &usedInResponse, schemaName)
					}
				}
			}
		}
	}

	switch {
	case usedInRequest && usedInResponse:
		return DirectionBoth
	case usedInRequest:
		return DirectionRequest
	case usedInResponse:
		return DirectionResponse
	default:
		return DirectionNone
	}
}

func checkResponse(resp *v3.Response, usedInResponse *bool, schemaRef string) {
	if resp == nil {
		return
	}
	// Check content
	if resp.Content != nil {
		for mediaPairs := resp.Content.First(); mediaPairs != nil; mediaPairs = mediaPairs.Next() {
			media := mediaPairs.Value()
			if media.Schema != nil && refMatches(media.Schema, schemaRef) {
				*usedInResponse = true
			}
		}
	}
	// Check headers
	if resp.Headers != nil {
		for headerPairs := resp.Headers.First(); headerPairs != nil; headerPairs = headerPairs.Next() {
			header := headerPairs.Value()
			if header != nil && header.Schema != nil {
				if refMatches(header.Schema, schemaRef) {
					*usedInResponse = true
				}
			}
		}
	}
}

// refMatches checks if the schema or any of its nested schemas matches the given ref.
func refMatches(schemaProxy *base.SchemaProxy, name string) bool {
	return refMatchesInternal(schemaProxy, name, make(map[string]bool))
}

// refMatchesInternal recursively checks if a schema matches the given name, tracking visited schemas to avoid circular references.
func refMatchesInternal(schemaProxy *base.SchemaProxy, name string, visited map[string]bool) bool {
	if schemaProxy == nil || name == "" {
		return false
	}

	reference := schemaProxy.GetReference()
	parts := strings.Split(reference, "/")
	actualName := parts[len(parts)-1]

	// Check for circular reference - if we've already visited this schema, skip it
	if visited[reference] {
		return false
	}
	visited[reference] = true

	schema := schemaProxy.Schema()
	if schema == nil {
		return false
	}

	if name == actualName {
		return true
	}

	// Check composition schemas
	for _, s := range append(append(schema.AllOf, schema.AnyOf...), schema.OneOf...) {
		if refMatchesInternal(s, name, visited) {
			return true
		}
	}

	// Check not
	if schema.Not != nil {
		if refMatchesInternal(schema.Not, name, visited) {
			return true
		}
	}

	// Check properties
	if schema.Properties != nil {
		for pairs := schema.Properties.First(); pairs != nil; pairs = pairs.Next() {
			if refMatchesInternal(pairs.Value(), name, visited) {
				return true
			}
		}
	}

	// Check additionalProperties
	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() {
			if refMatchesInternal(schema.AdditionalProperties.A, name, visited) {
				return true
			}
		}
		// Ignore bool case
	}

	// Check patternProperties
	if schema.PatternProperties != nil {
		for pairs := schema.PatternProperties.First(); pairs != nil; pairs = pairs.Next() {
			if refMatchesInternal(pairs.Value(), name, visited) {
				return true
			}
		}
	}

	// Check items
	if schema.Items != nil {
		if schema.Items.IsA() {
			if refMatchesInternal(schema.Items.A, name, visited) {
				return true
			}
		}
		// Ignore bool case
	}

	// Check prefixItems
	for _, pi := range schema.PrefixItems {
		if refMatchesInternal(pi, name, visited) {
			return true
		}
	}

	// Check contains
	if schema.Contains != nil {
		if refMatchesInternal(schema.Contains, name, visited) {
			return true
		}
	}

	// Check if, else, then
	if schema.If != nil {
		if refMatchesInternal(schema.If, name, visited) {
			return true
		}
	}
	if schema.Else != nil {
		if refMatchesInternal(schema.Else, name, visited) {
			return true
		}
	}
	if schema.Then != nil {
		if refMatchesInternal(schema.Then, name, visited) {
			return true
		}
	}

	// Check dependentSchemas
	if schema.DependentSchemas != nil {
		for pairs := schema.DependentSchemas.First(); pairs != nil; pairs = pairs.Next() {
			if refMatchesInternal(pairs.Value(), name, visited) {
				return true
			}
		}
	}

	// Check propertyNames
	if schema.PropertyNames != nil {
		if refMatchesInternal(schema.PropertyNames, name, visited) {
			return true
		}
	}

	// Check unevaluatedItems
	if schema.UnevaluatedItems != nil {
		if refMatchesInternal(schema.UnevaluatedItems, name, visited) {
			return true
		}
	}

	// Check unevaluatedProperties
	if schema.UnevaluatedProperties != nil {
		if schema.UnevaluatedProperties.IsA() {
			if refMatchesInternal(schema.UnevaluatedProperties.A, name, visited) {
				return true
			}
		}
	}

	return false
}
