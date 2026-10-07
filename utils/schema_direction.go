package utils

import (
	"strings"

	"github.com/pb33f/go-yaml"

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
	walkSchemaDirections(doc, func(proxy *base.SchemaProxy, schema *base.Schema, usage DirectionType) {
		reference := proxy.GetReference()
		if name := reference[strings.LastIndexByte(reference, '/')+1:]; name != "" {
			directions[name] = mergeSchemaDirection(directions[name], usage)
		}
	})
	return directions
}

// GetSchemaNodeDirections indexes request and response uses by schema identity.
// Inline schemas and schemas with equal names remain distinct. Each schema is
// traversed at most once per direction; unused schemas are omitted.
func GetSchemaNodeDirections(doc *v3.Document) map[*yaml.Node]DirectionType {
	directions := make(map[*yaml.Node]DirectionType)
	walkSchemaDirections(doc, func(_ *base.SchemaProxy, schema *base.Schema, usage DirectionType) {
		if low := schema.GoLow(); low != nil && low.RootNode != nil {
			directions[low.RootNode] = mergeSchemaDirection(directions[low.RootNode], usage)
		}
	})
	return directions
}

func mergeSchemaDirection(previous, usage DirectionType) DirectionType {
	if previous != "" && previous != usage {
		return DirectionBoth
	}
	return usage
}

func walkSchemaDirections(doc *v3.Document, visit func(*base.SchemaProxy, *base.Schema, DirectionType)) {
	if doc == nil {
		return
	}
	visited := make(map[*yaml.Node]DirectionType)
	mark := func(proxy *base.SchemaProxy, usage DirectionType) {
		collectSchemaDirections(proxy, usage, visited, visit)
	}
	markParameter := func(param *v3.Parameter) {
		if param == nil {
			return
		}
		mark(param.Schema, DirectionRequest)
		if param.Content != nil {
			for _, media := range param.Content.FromOldest() {
				if media != nil {
					mark(media.Schema, DirectionRequest)
				}
			}
		}
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
					if header.Content != nil {
						for _, media := range header.Content.FromOldest() {
							if media != nil {
								mark(media.Schema, DirectionResponse)
							}
						}
					}
				}
			}
		}
	}

	seenPaths := make(map[*v3.PathItem]bool)
	var markPath func(*v3.PathItem)
	markPath = func(pathItem *v3.PathItem) {
		if pathItem == nil || seenPaths[pathItem] {
			return
		}
		seenPaths[pathItem] = true
		for _, param := range pathItem.Parameters {
			markParameter(param)
		}
		for _, op := range pathItem.GetOperations().FromOldest() {
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
				markParameter(param)
			}
			if op.Callbacks != nil {
				for _, callback := range op.Callbacks.FromOldest() {
					if callback != nil && callback.Expression != nil {
						for _, path := range callback.Expression.FromOldest() {
							markPath(path)
						}
					}
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
	if doc.Paths != nil && doc.Paths.PathItems != nil {
		for _, path := range doc.Paths.PathItems.FromOldest() {
			markPath(path)
		}
	}
	if doc.Webhooks != nil {
		for _, path := range doc.Webhooks.FromOldest() {
			markPath(path)
		}
	}
}

func collectSchemaDirections(proxy *base.SchemaProxy, usage DirectionType, visited map[*yaml.Node]DirectionType, mark func(*base.SchemaProxy, *base.Schema, DirectionType)) {
	if proxy == nil {
		return
	}
	schema := proxy.Schema()
	if schema == nil {
		return
	}
	mark(proxy, schema, usage)
	if low := schema.GoLow(); low != nil && low.RootNode != nil {
		previous := visited[low.RootNode]
		if previous == usage || previous == DirectionBoth {
			return
		}
		visited[low.RootNode] = mergeSchemaDirection(previous, usage)
	}

	for _, child := range schema.AllOf {
		collectSchemaDirections(child, usage, visited, mark)
	}
	for _, child := range schema.AnyOf {
		collectSchemaDirections(child, usage, visited, mark)
	}
	for _, child := range schema.OneOf {
		collectSchemaDirections(child, usage, visited, mark)
	}
	collectSchemaDirections(schema.Not, usage, visited, mark)
	if schema.Properties != nil {
		for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
			collectSchemaDirections(pair.Value(), usage, visited, mark)
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() {
		collectSchemaDirections(schema.AdditionalProperties.A, usage, visited, mark)
	}
	if schema.PatternProperties != nil {
		for pair := schema.PatternProperties.First(); pair != nil; pair = pair.Next() {
			collectSchemaDirections(pair.Value(), usage, visited, mark)
		}
	}
	if schema.Items != nil && schema.Items.IsA() {
		collectSchemaDirections(schema.Items.A, usage, visited, mark)
	}
	for _, child := range schema.PrefixItems {
		collectSchemaDirections(child, usage, visited, mark)
	}
	collectSchemaDirections(schema.Contains, usage, visited, mark)
	collectSchemaDirections(schema.If, usage, visited, mark)
	collectSchemaDirections(schema.Else, usage, visited, mark)
	collectSchemaDirections(schema.Then, usage, visited, mark)
	if schema.DependentSchemas != nil {
		for pair := schema.DependentSchemas.First(); pair != nil; pair = pair.Next() {
			collectSchemaDirections(pair.Value(), usage, visited, mark)
		}
	}
	collectSchemaDirections(schema.PropertyNames, usage, visited, mark)
	collectSchemaDirections(schema.UnevaluatedItems, usage, visited, mark)
	if schema.UnevaluatedProperties != nil && schema.UnevaluatedProperties.IsA() {
		collectSchemaDirections(schema.UnevaluatedProperties.A, usage, visited, mark)
	}
}
