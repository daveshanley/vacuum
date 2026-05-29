// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"go.yaml.in/yaml/v4"
)

// CheckRequiredFields checks duplicate required entries and required values that are absent from known properties.
func CheckRequiredFields(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	if schema == nil || schema.Value == nil || len(schema.Value.Required) == 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	seen := make(map[string]int)
	requiredKeyNode := requiredKeyNode(schema)

	for i, required := range schema.Value.Required {
		requiredNode := requiredItemNode(schema, i, requiredKeyNode)
		if first, ok := seen[required]; ok {
			result := BuildResult(fmt.Sprintf("required property `%s` duplicates required[%d]", required, first),
				schema.GenerateJSONPath(), "required", i, schema, requiredNode, context)
			results = append(results, result)
			continue
		}
		seen[required] = i

		propertyLookup := LookupRequiredProperty(schema, required)
		if propertyLookup.PropertiesFound && !propertyLookup.PropertyDefined {
			result := BuildResult(model.GetStringTemplates().BuildRequiredFieldMessage(required),
				schema.GenerateJSONPath(), "required", i, schema, requiredNode, context)
			results = append(results, result)
		}
	}

	return results
}

// LookupRequiredProperty checks whether a property is defined in the schema or composed object shape.
func LookupRequiredProperty(schema *drV3.Schema, required string) RequiredPropertyLookup {
	visited := make(map[*yaml.Node]struct{})
	lookup := lookupRequiredPropertyInSchema(schema, required, visited)
	current := schema

	for current != nil {
		parentProxy, ok := current.GetParent().(*drV3.SchemaProxy)
		if !ok || parentProxy.GetPathSegment() != "allOf" {
			break
		}
		parentSchema, ok := parentProxy.GetParent().(*drV3.Schema)
		if !ok || parentSchema == nil {
			break
		}
		lookup = lookup.merge(lookupRequiredPropertyInSchema(parentSchema, required, visited))
		current = parentSchema
	}

	return lookup
}

// SchemaUsesObjectKeywords reports whether a schema behaves like an object schema.
func SchemaUsesObjectKeywords(schema *drV3.Schema) bool {
	if schema == nil || schema.Value == nil {
		return false
	}

	for _, t := range schema.Value.Type {
		if t == "object" {
			return true
		}
	}

	return schema.Value.Properties != nil ||
		len(schema.Value.Required) > 0 ||
		len(schema.AllOf) > 0 ||
		len(schema.AnyOf) > 0 ||
		len(schema.OneOf) > 0
}

func lookupRequiredPropertyInSchema(schema *drV3.Schema, required string, visited map[*yaml.Node]struct{}) RequiredPropertyLookup {
	if schema == nil || schema.Value == nil {
		return RequiredPropertyLookup{}
	}

	if lowSchema := schema.Value.GoLow(); lowSchema != nil && lowSchema.RootNode != nil {
		if _, seen := visited[lowSchema.RootNode]; seen {
			return RequiredPropertyLookup{}
		}
		visited[lowSchema.RootNode] = struct{}{}
	}

	lookup := RequiredPropertyLookup{}
	if schema.Value.Properties != nil {
		lookup.PropertiesFound = true
		if schema.Value.Properties.GetOrZero(required) != nil {
			lookup.PropertyDefined = true
		}
	}

	lookup = lookup.merge(lookupRequiredPropertyInProxies(schema.AnyOf, required, visited))
	lookup = lookup.merge(lookupRequiredPropertyInProxies(schema.OneOf, required, visited))
	lookup = lookup.merge(lookupRequiredPropertyInProxies(schema.AllOf, required, visited))

	return lookup
}

func lookupRequiredPropertyInProxies(proxies []*drV3.SchemaProxy, required string,
	visited map[*yaml.Node]struct{},
) RequiredPropertyLookup {
	lookup := RequiredPropertyLookup{}
	for _, proxy := range proxies {
		if proxy == nil || proxy.Schema == nil {
			continue
		}
		lookup = lookup.merge(lookupRequiredPropertyInSchema(proxy.Schema, required, visited))
		if lookup.PropertyDefined {
			return lookup
		}
	}
	return lookup
}

func requiredKeyNode(schema *drV3.Schema) *yaml.Node {
	if schema == nil || schema.Value == nil || schema.Value.GoLow() == nil {
		return schemaNode(schema)
	}
	if schema.Value.GoLow().Required.KeyNode != nil {
		return schema.Value.GoLow().Required.KeyNode
	}
	return schemaNode(schema)
}

func requiredItemNode(schema *drV3.Schema, index int, fallback *yaml.Node) *yaml.Node {
	if schema == nil || schema.Value == nil || schema.Value.GoLow() == nil {
		return fallback
	}
	required := schema.Value.GoLow().Required.Value
	if index >= 0 && index < len(required) && required[index].ValueNode != nil {
		return required[index].ValueNode
	}
	if valueNode := schema.Value.GoLow().Required.ValueNode; valueNode != nil && valueNode.Kind == yaml.SequenceNode &&
		index >= 0 && index < len(valueNode.Content) {
		return valueNode.Content[index]
	}
	return fallback
}

func checkPolymorphicProperty(schema *drV3.Schema, propertyName string) bool {
	if schema == nil || schema.Value == nil {
		return false
	}

	visited := make(map[*lowBase.Schema]struct{})
	return checkSchemaPropertyRecursive(schema.Value, propertyName, visited)
}
