// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"fmt"

	"github.com/daveshanley/vacuum/functions/schemachecks"
	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"go.yaml.in/yaml/v4"
)

// Valid validates a JSON Schema document against its declared metaschema.
type Valid struct{}

func (v Valid) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "jsonSchemaValid"}
}

func (v Valid) GetCategory() string {
	return model.FunctionCategoryJSONSchema
}

func (v Valid) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) == 0 {
		return nil
	}
	issues, err := schemautil.ValidateAgainstMetaschema(nodes[0])
	if err != nil {
		root := schemautil.RootNode(nodes[0])
		return []model.RuleFunctionResult{{
			Message:   err.Error(),
			StartNode: root,
			EndNode:   vacuumUtils.BuildEndNode(root),
			Path:      "$",
			Rule:      context.Rule,
		}}
	}
	results := make([]model.RuleFunctionResult, 0, len(issues))
	for _, issue := range issues {
		message := issue.Message
		if issue.Pointer != "" && issue.Pointer != "#" {
			message = fmt.Sprintf("%s at %s", message, issue.Pointer)
		}
		results = append(results, model.RuleFunctionResult{
			Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, message),
			StartNode: issue.Node,
			EndNode:   issue.EndNode,
			Path:      issue.Path,
			Rule:      context.Rule,
		})
	}
	return results
}

// Sanity runs lightweight JSON Schema consistency checks.
type Sanity struct{}

func (s Sanity) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "jsonSchemaSanity",
		Properties: []model.RuleFunctionProperty{{
			Name:        "check",
			Description: "The JSON Schema sanity check to run.",
		}},
	}
}

func (s Sanity) GetCategory() string {
	return model.FunctionCategoryJSONSchema
}

func (s Sanity) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) == 0 {
		return nil
	}
	props := context.GetOptionsStringMap()
	check := props["check"]
	var results []model.RuleFunctionResult
	root := schemautil.RootNode(nodes[0])
	if context.DrDocument != nil && len(context.DrDocument.Schemas) > 0 {
		for _, schema := range context.DrDocument.Schemas {
			results = append(results, schemachecks.RunSchemaSanityCheck(schema, root, &context, check)...)
		}
	}
	return results
}

// RefValid declares the JSON Schema reference validation hook.
//
// libopenapi resolver errors are collected by the motor while building the rolodex and emitted as
// json-schema-ref-valid results before rule functions run.
type RefValid struct{}

func (r RefValid) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "jsonSchemaRefValid"}
}

func (r RefValid) GetCategory() string {
	return model.FunctionCategoryJSONSchema
}

func (r RefValid) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	return nil
}
