// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

// MessageExamples validates basic message example shape.
type MessageExamples struct{}

// GetSchema returns the AsyncAPI message-examples function schema.
func (m MessageExamples) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiMessageExamples"}
}

// GetCategory returns the AsyncAPI function category.
func (m MessageExamples) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks that message examples declare payload or headers. Full schema
// instance validation is handled by a later schema-format strategy.
func (m MessageExamples) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		_, examples := mappingValue(node, "examples")
		if examples == nil || examples.Kind != yaml.SequenceNode {
			continue
		}
		for _, example := range examples.Content {
			_, payload := mappingValue(example, "payload")
			_, headers := mappingValue(example, "headers")
			if payload == nil && headers == nil {
				results = append(results, result(context, example, nodePath(context, example, ""), "Message example must define `payload` or `headers`."))
			}
		}
	}
	return results
}

// ContentType validates AsyncAPI default and message-level content type usage.
type ContentType struct{}

// GetSchema returns the AsyncAPI content-type function schema.
func (c ContentType) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiContentType"}
}

// GetCategory returns the AsyncAPI function category.
func (c ContentType) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule requires either a root defaultContentType or explicit contentType on
// each message that carries a payload.
func (c ContentType) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	root := rootNode(context)
	_, defaultContentType := mappingValue(root, "defaultContentType")
	hasDefault := defaultContentType != nil && defaultContentType.Value != ""

	var results []model.RuleFunctionResult
	for _, node := range nodes {
		_, payload := mappingValue(node, "payload")
		if payload == nil {
			continue
		}
		_, contentType := mappingValue(node, "contentType")
		if !hasDefault && (contentType == nil || contentType.Value == "") {
			results = append(results, result(context, node, nodePath(context, node, ""), "Message payloads must define `contentType` or the document must define `defaultContentType`."))
		}
	}
	return results
}

// TagsUnique validates tag name uniqueness inside tag arrays.
type TagsUnique struct{}

// GetSchema returns the AsyncAPI tags uniqueness function schema.
func (t TagsUnique) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiTagsUnique"}
}

// GetCategory returns the AsyncAPI function category.
func (t TagsUnique) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule reports duplicate tag names in every matched tag array.
func (t TagsUnique) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		if node == nil || node.Kind != yaml.SequenceNode {
			continue
		}
		seen := make(map[string]*yaml.Node)
		for _, tag := range node.Content {
			_, name := mappingValue(tag, "name")
			if name == nil || name.Value == "" {
				continue
			}
			if seen[name.Value] != nil {
				results = append(results, result(context, name, nodePath(context, name, ""), fmt.Sprintf("Tag `%s` must be unique.", name.Value)))
				continue
			}
			seen[name.Value] = name
		}
	}
	return results
}
