// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// NullableEnum checks that nullable enums explicitly contain a null value in their enum array.
// This validates both OpenAPI 3.0 (nullable: true) and OpenAPI 3.1 (type: [string, "null"]) patterns.
type NullableEnum struct{}

func (ne NullableEnum) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "nullableEnum",
	}
}

func (ne NullableEnum) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

func (ne NullableEnum) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	enums := context.Index.GetAllEnums()

	for _, enum := range enums {
		// check if this enum is nullable
		isNullable := ne.isSchemaNullable(enum)

		if !isNullable {
			// not nullable, skip this enum
			continue
		}

		// check if enum array contains an actual null value
		hasNullValue := ne.enumContainsNull(enum.Node)

		if !hasNullValue {
			// nullable schema with enum, but no null in the enum array
			results = append(results, model.RuleFunctionResult{
				Message: "enum is defined as nullable but does not contain a `null` value. " +
					"Nullable enums must explicitly include `null` in the enum array (not the string \"null\")",
				StartNode: enum.SchemaNode,
				EndNode:   vacuumUtils.BuildEndNode(enum.SchemaNode),
				Path:      enum.Path,
				Rule:      context.Rule,
			})
		}
	}

	return results
}

// isSchemaNullable checks if a schema is nullable using either OpenAPI 3.0 or 3.1 patterns
func (ne NullableEnum) isSchemaNullable(enum *index.EnumReference) bool {
	if enum.SchemaNode == nil {
		return false
	}

	// openAPI 3.0: check for nullable: true
	_, nullableValue := utils.FindKeyNode("nullable", enum.SchemaNode.Content)
	if nullableValue != nil && nullableValue.Value == "true" {
		return true
	}

	// openAPI 3.1: check if type is an array containing "null"
	// enum.Type could be a single node or have Content if it's an array
	if enum.Type != nil {
		if len(enum.Type.Content) > 0 {
			// type is an array, check if it contains "null"
			for _, typeNode := range enum.Type.Content {
				if typeNode.Value == "null" {
					return true
				}
			}
		}
	}

	return false
}

// enumContainsNull checks if an enum array contains an actual null value (not the string "null")
func (ne NullableEnum) enumContainsNull(enumNode *yaml.Node) bool {
	if enumNode == nil || len(enumNode.Content) == 0 {
		return false
	}

	for _, item := range enumNode.Content {
		if utils.IsNodeNull(item) {
			return true
		}
	}

	return false
}
