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

// GetSchemaDirection determines whether a schema is used in requests, responses, both, or neither.
// Use GetSchemaDirections when checking multiple schemas in the same document.
func GetSchemaDirection(doc *v3.Document, schemaName string) DirectionType {
	if direction := GetSchemaDirections(doc)[schemaName]; direction != "" {
		return direction
	}
	return DirectionNone
}

// GetSchemaDirections indexes the request and response uses of every referenced schema.
// It avoids walking the whole document once for each schema checked by a rule.
// Unused schemas are omitted; the returned map belongs to the caller.
func GetSchemaDirections(doc *v3.Document) map[string]DirectionType {
	directions := make(map[string]DirectionType)
	if doc == nil || doc.Paths == nil || doc.Paths.PathItems == nil {
		return directions
	}

	visited := make(map[string]bool)
	mark := func(proxy *base.SchemaProxy, usage DirectionType) {
		clear(visited)
		collectSchemaDirections(proxy, visited, func(name string) {
			previous := directions[name]
			if previous != "" && previous != usage {
				directions[name] = DirectionBoth
			} else if previous == "" {
				directions[name] = usage
			}
		})
	}

	markResponse := func(resp *v3.Response) {
		if resp == nil {
			return
		}
		if resp.Content != nil {
			for pair := resp.Content.First(); pair != nil; pair = pair.Next() {
				if media := pair.Value(); media != nil {
					mark(media.Schema, DirectionResponse)
				}
			}
		}
		if resp.Headers != nil {
			for pair := resp.Headers.First(); pair != nil; pair = pair.Next() {
				if header := pair.Value(); header != nil {
					mark(header.Schema, DirectionResponse)
				}
			}
		}
	}

	for pathPair := doc.Paths.PathItems.First(); pathPair != nil; pathPair = pathPair.Next() {
		pathItem := pathPair.Value()
		if pathItem == nil {
			continue
		}
		for _, param := range pathItem.Parameters {
			if param != nil {
				mark(param.Schema, DirectionRequest)
			}
		}
		for _, op := range []*v3.Operation{
			pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Patch,
			pathItem.Delete, pathItem.Options, pathItem.Head, pathItem.Trace,
		} {
			if op == nil {
				continue
			}
			if op.RequestBody != nil && op.RequestBody.Content != nil {
				for pair := op.RequestBody.Content.First(); pair != nil; pair = pair.Next() {
					if media := pair.Value(); media != nil {
						mark(media.Schema, DirectionRequest)
					}
				}
			}
			for _, param := range op.Parameters {
				if param != nil {
					mark(param.Schema, DirectionRequest)
				}
			}
			if op.Responses != nil {
				markResponse(op.Responses.Default)
				if op.Responses.Codes != nil {
					for pair := op.Responses.Codes.First(); pair != nil; pair = pair.Next() {
						markResponse(pair.Value())
					}
				}
			}
		}
	}
	return directions
}

// collectSchemaDirections tracks reference strings to retain the existing traversal semantics.
// Inline schemas share the empty reference key within each root traversal.
func collectSchemaDirections(proxy *base.SchemaProxy, visited map[string]bool, mark func(string)) {
	if proxy == nil {
		return
	}
	reference := proxy.GetReference()
	if visited[reference] {
		return
	}
	visited[reference] = true
	schema := proxy.Schema()
	if schema == nil {
		return
	}
	if name := reference[strings.LastIndexByte(reference, '/')+1:]; name != "" {
		mark(name)
	}
	for _, child := range schema.AllOf {
		collectSchemaDirections(child, visited, mark)
	}
	for _, child := range schema.AnyOf {
		collectSchemaDirections(child, visited, mark)
	}
	for _, child := range schema.OneOf {
		collectSchemaDirections(child, visited, mark)
	}
	collectSchemaDirections(schema.Not, visited, mark)
	if schema.Properties != nil {
		for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
			collectSchemaDirections(pair.Value(), visited, mark)
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() {
		collectSchemaDirections(schema.AdditionalProperties.A, visited, mark)
	}
	if schema.PatternProperties != nil {
		for pair := schema.PatternProperties.First(); pair != nil; pair = pair.Next() {
			collectSchemaDirections(pair.Value(), visited, mark)
		}
	}
	if schema.Items != nil && schema.Items.IsA() {
		collectSchemaDirections(schema.Items.A, visited, mark)
	}
	for _, child := range schema.PrefixItems {
		collectSchemaDirections(child, visited, mark)
	}
	collectSchemaDirections(schema.Contains, visited, mark)
	collectSchemaDirections(schema.If, visited, mark)
	collectSchemaDirections(schema.Else, visited, mark)
	collectSchemaDirections(schema.Then, visited, mark)
	if schema.DependentSchemas != nil {
		for pair := schema.DependentSchemas.First(); pair != nil; pair = pair.Next() {
			collectSchemaDirections(pair.Value(), visited, mark)
		}
	}
	collectSchemaDirections(schema.PropertyNames, visited, mark)
	collectSchemaDirections(schema.UnevaluatedItems, visited, mark)
	if schema.UnevaluatedProperties != nil && schema.UnevaluatedProperties.IsA() {
		collectSchemaDirections(schema.UnevaluatedProperties.A, visited, mark)
	}
}
