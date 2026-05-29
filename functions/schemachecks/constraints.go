// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"go.yaml.in/yaml/v4"
)

type constraintDef struct {
	category string
	name     string
	validFor string
}

var allConstraints = []constraintDef{
	{name: "pattern", category: "string", validFor: "string"},
	{name: "minLength", category: "string", validFor: "string"},
	{name: "maxLength", category: "string", validFor: "string"},
	{name: "contentEncoding", category: "string", validFor: "string"},
	{name: "contentMediaType", category: "string", validFor: "string"},
	{name: "contentSchema", category: "string", validFor: "string"},
	{name: "minimum", category: "number", validFor: "number/integer"},
	{name: "maximum", category: "number", validFor: "number/integer"},
	{name: "multipleOf", category: "number", validFor: "number/integer"},
	{name: "exclusiveMinimum", category: "number", validFor: "number/integer"},
	{name: "exclusiveMaximum", category: "number", validFor: "number/integer"},
	{name: "minItems", category: "array", validFor: "array"},
	{name: "maxItems", category: "array", validFor: "array"},
	{name: "uniqueItems", category: "array", validFor: "array"},
	{name: "minContains", category: "array", validFor: "array"},
	{name: "maxContains", category: "array", validFor: "array"},
	{name: "contains", category: "array", validFor: "array"},
	{name: "prefixItems", category: "array", validFor: "array"},
	{name: "unevaluatedItems", category: "array", validFor: "array"},
	{name: "minProperties", category: "object", validFor: "object"},
	{name: "maxProperties", category: "object", validFor: "object"},
	{name: "patternProperties", category: "object", validFor: "object"},
	{name: "propertyNames", category: "object", validFor: "object"},
	{name: "dependentSchemas", category: "object", validFor: "object"},
	{name: "unevaluatedProperties", category: "object", validFor: "object"},
}

func checkConstraint(c *constraintDef, high *highBase.Schema, low *lowBase.Schema) *yaml.Node {
	if high == nil || low == nil {
		return nil
	}
	switch c.name {
	case "pattern":
		if high.Pattern != "" {
			return low.Pattern.KeyNode
		}
	case "minLength":
		if high.MinLength != nil {
			return low.MinLength.KeyNode
		}
	case "maxLength":
		if high.MaxLength != nil {
			return low.MaxLength.KeyNode
		}
	case "minimum":
		if high.Minimum != nil {
			return low.Minimum.KeyNode
		}
	case "maximum":
		if high.Maximum != nil {
			return low.Maximum.KeyNode
		}
	case "multipleOf":
		if high.MultipleOf != nil {
			return low.MultipleOf.KeyNode
		}
	case "exclusiveMinimum":
		if high.ExclusiveMinimum != nil {
			return low.ExclusiveMinimum.KeyNode
		}
	case "exclusiveMaximum":
		if high.ExclusiveMaximum != nil {
			return low.ExclusiveMaximum.KeyNode
		}
	case "minItems":
		if high.MinItems != nil {
			return low.MinItems.KeyNode
		}
	case "maxItems":
		if high.MaxItems != nil {
			return low.MaxItems.KeyNode
		}
	case "uniqueItems":
		if high.UniqueItems != nil {
			return low.UniqueItems.KeyNode
		}
	case "minContains":
		if high.MinContains != nil {
			return low.MinContains.KeyNode
		}
	case "maxContains":
		if high.MaxContains != nil {
			return low.MaxContains.KeyNode
		}
	case "minProperties":
		if high.MinProperties != nil {
			return low.MinProperties.KeyNode
		}
	case "maxProperties":
		if high.MaxProperties != nil {
			return low.MaxProperties.KeyNode
		}
	case "contentEncoding":
		if high.ContentEncoding != "" {
			return low.ContentEncoding.KeyNode
		}
	case "contentMediaType":
		if high.ContentMediaType != "" {
			return low.ContentMediaType.KeyNode
		}
	case "contentSchema":
		if high.ContentSchema != nil {
			return low.ContentSchema.KeyNode
		}
	case "contains":
		if high.Contains != nil {
			return low.Contains.KeyNode
		}
	case "prefixItems":
		if len(high.PrefixItems) > 0 {
			return low.PrefixItems.KeyNode
		}
	case "unevaluatedItems":
		if high.UnevaluatedItems != nil {
			return low.UnevaluatedItems.KeyNode
		}
	case "patternProperties":
		if high.PatternProperties != nil && high.PatternProperties.Len() > 0 {
			return low.PatternProperties.KeyNode
		}
	case "propertyNames":
		if high.PropertyNames != nil {
			return low.PropertyNames.KeyNode
		}
	case "dependentSchemas":
		if high.DependentSchemas != nil && high.DependentSchemas.Len() > 0 {
			return low.DependentSchemas.KeyNode
		}
	case "unevaluatedProperties":
		if high.UnevaluatedProperties != nil {
			return low.UnevaluatedProperties.KeyNode
		}
	}
	return nil
}

// CheckTypeMismatchedConstraints validates that a schema only uses constraints appropriate for its declared types.
func CheckTypeMismatchedConstraints(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	if schema == nil || schema.Value == nil || schema.Value.GoLow() == nil {
		return nil
	}

	var results []model.RuleFunctionResult
	lowSchema := schema.Value.GoLow()
	highSchema := schema.Value
	schemaTypes := schema.Value.Type

	for i := range allConstraints {
		c := &allConstraints[i]
		if constraintAppliesToSchemaTypes(c, schemaTypes) {
			continue
		}
		if node := checkConstraint(c, highSchema, lowSchema); node != nil {
			message := fmt.Sprintf("`%s` constraint is only applicable to %s types, not `%s`",
				c.name, c.validFor, formatSchemaTypesForMessage(schemaTypes))
			result := BuildResult(message, schema.GenerateJSONPath(), c.name, -1, schema, node, context)
			results = append(results, result)
		}
	}

	return results
}

func formatSchemaTypesForMessage(schemaTypes []string) string {
	if len(schemaTypes) == 1 {
		return schemaTypes[0]
	}
	return "[" + strings.Join(schemaTypes, ", ") + "]"
}

func constraintAppliesToSchemaTypes(c *constraintDef, schemaTypes []string) bool {
	for _, schemaType := range schemaTypes {
		if c.category == schemaType || (c.category == "number" && schemaType == "integer") {
			return true
		}
	}
	return false
}
