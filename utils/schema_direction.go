package utils

import (
	"encoding/json"
	"reflect"
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
	if schemaProxy == nil || name == "" {
		return false
	}

	parts := strings.Split(schemaProxy.GetReference(), "/")
	actualName := parts[len(parts)-1]

	schema := schemaProxy.Schema()
	if schema == nil {
		return false
	}

	if name == actualName {
		return true
	}

	// Check composition schemas
	for _, s := range append(append(schema.AllOf, schema.AnyOf...), schema.OneOf...) {
		if refMatches(s, name) {
			return true
		}
	}

	// Check not
	if schema.Not != nil {
		if refMatches(schema.Not, name) {
			return true
		}
	}

	// Check properties
	if schema.Properties != nil {
		for pairs := schema.Properties.First(); pairs != nil; pairs = pairs.Next() {
			if refMatches(pairs.Value(), name) {
				return true
			}
		}
	}

	// Check additionalProperties
	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() {
			if refMatches(schema.AdditionalProperties.A, name) {
				return true
			}
		}
		// Ignore bool case
	}

	// Check patternProperties
	if schema.PatternProperties != nil {
		for pairs := schema.PatternProperties.First(); pairs != nil; pairs = pairs.Next() {
			if refMatches(pairs.Value(), name) {
				return true
			}
		}
	}

	// Check items
	if schema.Items != nil {
		if schema.Items.IsA() {
			if refMatches(schema.Items.A, name) {
				return true
			}
		}
		// Ignore bool case
	}

	// Check prefixItems
	for _, pi := range schema.PrefixItems {
		if refMatches(pi, name) {
			return true
		}
	}

	// Check contains
	if schema.Contains != nil {
		if refMatches(schema.Contains, name) {
			return true
		}
	}

	// Check if, else, then
	if schema.If != nil {
		if refMatches(schema.If, name) {
			return true
		}
	}
	if schema.Else != nil {
		if refMatches(schema.Else, name) {
			return true
		}
	}
	if schema.Then != nil {
		if refMatches(schema.Then, name) {
			return true
		}
	}

	// Check dependentSchemas
	if schema.DependentSchemas != nil {
		for pairs := schema.DependentSchemas.First(); pairs != nil; pairs = pairs.Next() {
			if refMatches(pairs.Value(), name) {
				return true
			}
		}
	}

	// Check propertyNames
	if schema.PropertyNames != nil {
		if refMatches(schema.PropertyNames, name) {
			return true
		}
	}

	// Check unevaluatedItems
	if schema.UnevaluatedItems != nil {
		if refMatches(schema.UnevaluatedItems, name) {
			return true
		}
	}

	// Check unevaluatedProperties
	if schema.UnevaluatedProperties != nil {
		if schema.UnevaluatedProperties.IsA() {
			if refMatches(schema.UnevaluatedProperties.A, name) {
				return true
			}
		}
	}

	return false
}

func SchemasEqual(s1 *base.Schema, p *base.SchemaProxy) bool {
	if p == nil {
		return s1 == nil
	}
	s2 := p.Schema()
	if s2 == nil {
		return false // Or handle build error via p.GetBuildError()
	}

	js1, err1 := json.Marshal(s1)
	if err1 != nil {
		return false
	}
	js2, err2 := json.Marshal(s2)
	if err2 != nil {
		return false
	}

	var i1, i2 interface{}
	if err := json.Unmarshal(js1, &i1); err != nil {
		return false
	}
	if err := json.Unmarshal(js2, &i2); err != nil {
		return false
	}

	return reflect.DeepEqual(i1, i2)
}
