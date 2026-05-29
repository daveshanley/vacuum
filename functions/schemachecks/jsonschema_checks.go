// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	"fmt"
	"strings"

	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

func validatePatternKeywords(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if schema.Value.Pattern != "" && !ecma262PatternValid(schema.Value.Pattern) {
		result := BuildResult("schema `pattern` should be a ECMA-262 regular expression dialect",
			schema.GenerateJSONPath(), "pattern", -1, schema, schema.Value.GoLow().Pattern.KeyNode, context)
		results = append(results, result)
	}

	patternProperties := schemautil.MappingValueNode(schema.Value.GoLow().RootNode, "patternProperties")
	if patternProperties == nil || patternProperties.Kind != yaml.MappingNode {
		return results
	}
	for i := 0; i+1 < len(patternProperties.Content); i += 2 {
		keyNode := patternProperties.Content[i]
		if !ecma262PatternValid(keyNode.Value) {
			result := BuildResult(fmt.Sprintf("patternProperties key `%s` should be a ECMA-262 regular expression dialect",
				keyNode.Value), schema.GenerateJSONPath(), "patternProperties", -1, schema, keyNode, context)
			results = append(results, result)
		}
	}
	return results
}

func checkShallowComposition(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	var seenType string
	for i, proxy := range schema.AllOfForRead() {
		if proxy == nil || proxy.Schema == nil || len(proxy.Schema.Value.Type) != 1 {
			continue
		}
		if seenType != "" && seenType != proxy.Schema.Value.Type[0] {
			result := BuildResult(fmt.Sprintf("allOf contains directly conflicting types `%s` and `%s`",
				seenType, proxy.Schema.Value.Type[0]), proxy.Schema.GenerateJSONPath(), "", i, proxy.Schema,
				schemaNode(proxy.Schema), context)
			results = append(results, result)
		}
		seenType = proxy.Schema.Value.Type[0]
	}

	seenBranches := make(map[string]int)
	for i, proxy := range schema.OneOfForRead() {
		if proxy == nil || proxy.Schema == nil || proxy.Schema.Value == nil || proxy.Schema.Value.GoLow() == nil {
			continue
		}
		key := stableNodeValue(proxy.Schema.Value.GoLow().RootNode)
		if first, ok := seenBranches[key]; ok {
			result := BuildResult(fmt.Sprintf("oneOf branch duplicates oneOf[%d]", first),
				proxy.Schema.GenerateJSONPath(), "", i, proxy.Schema, schemaNode(proxy.Schema), context)
			results = append(results, result)
			continue
		}
		seenBranches[key] = i
	}

	return results
}

func checkQualityRules(root *yaml.Node, schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	node := schema.Value.GoLow().RootNode
	if node == nil {
		return nil
	}
	if node == schemautil.RootNode(root) && (schemautil.IsFragmentRoot(root) || schemautil.IsDelegatingRefRoot(root)) {
		return nil
	}
	if schemautil.MappingValueNode(node, "$ref") != nil ||
		schemautil.MappingValueNode(node, "$dynamicRef") != nil ||
		schemautil.MappingValueNode(node, "$recursiveRef") != nil {
		return nil
	}

	var results []model.RuleFunctionResult
	for _, key := range []string{"title", "description"} {
		value := schemautil.MappingValueNode(node, key)
		if value == nil || strings.TrimSpace(value.Value) == "" {
			result := BuildResult(fmt.Sprintf("schema should define `%s`", key),
				schema.GenerateJSONPath(), "", -1, schema, node, context)
			results = append(results, result)
		}
	}

	types := schema.Value.Type
	if len(types) == 0 {
		result := BuildResult("schema should define an explicit type", schema.GenerateJSONPath(), "", -1,
			schema, node, context)
		results = append(results, result)
		return results
	}
	if containsType(types, "object") && schema.Value.Properties == nil &&
		schema.Value.AdditionalProperties == nil && schema.Value.UnevaluatedProperties == nil {
		result := BuildResult("object schema should constrain properties", schema.GenerateJSONPath(), "", -1,
			schema, node, context)
		results = append(results, result)
	}
	if containsType(types, "array") && schema.Value.Items == nil &&
		len(schema.Value.PrefixItems) == 0 && schema.Value.Contains == nil {
		result := BuildResult("array schema should constrain items", schema.GenerateJSONPath(), "", -1,
			schema, node, context)
		results = append(results, result)
	}

	return results
}

func checkExamplesAndDefault(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	node := schema.Value.GoLow().RootNode
	defaultNode := schemautil.MappingValueNode(node, "default")
	examplesNode := schemautil.MappingValueNode(node, "examples")
	if defaultNode == nil && examplesNode == nil {
		return nil
	}

	compiled, err := schemautil.CompileSchema(node)
	if err != nil || compiled == nil {
		return nil
	}

	var results []model.RuleFunctionResult
	if defaultNode != nil {
		if value, err := schemautil.NodeToInterface(defaultNode); err == nil {
			if validateErr := compiled.Validate(value); validateErr != nil {
				result := BuildResult(fmt.Sprintf("default value does not validate against schema: %s", validateErr.Error()),
					schema.GenerateJSONPath(), "default", -1, schema, defaultNode, context)
				results = append(results, result)
			}
		}
	}

	if examplesNode != nil && examplesNode.Kind == yaml.SequenceNode {
		for i, example := range examplesNode.Content {
			if value, err := schemautil.NodeToInterface(example); err == nil {
				if validateErr := compiled.Validate(value); validateErr != nil {
					path := model.GetStringTemplates().BuildPropertyArrayPath(schema.GenerateJSONPath(), "examples", i)
					result := BuildResult(fmt.Sprintf("example does not validate against schema: %s", validateErr.Error()),
						path, "examples", i, schema, example, context)
					results = append(results, result)
				}
			}
		}
	}

	return results
}

func validateDiscriminator(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Discriminator == nil {
		return results
	}

	discriminator := schema.Discriminator
	propertyName := discriminator.Value.PropertyName
	if propertyName == "" {
		result := BuildResult("discriminator object is missing required `propertyName` field",
			schema.GenerateJSONPath(), "discriminator", -1, schema, discriminator.KeyNode, context)
		results = append(results, result)
		return results
	}

	propertyExists := false
	if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(propertyName) != nil {
		propertyExists = true
	}
	if !propertyExists {
		propertyExists = checkPolymorphicProperty(schema, propertyName)
	}
	if !propertyExists {
		result := BuildResult(fmt.Sprintf("discriminator property `%s` is not defined in schema properties", propertyName),
			schema.GenerateJSONPath(), "discriminator", -1, schema, discriminator.KeyNode, context)
		results = append(results, result)
	}

	return results
}
