// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type schemaDoctorRecorder struct {
	paths []string
}

func (r *schemaDoctorRecorder) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil {
		return nil
	}
	for _, schema := range context.DrDocument.Schemas {
		r.paths = append(r.paths, schema.GenerateJSONPath())
	}
	return nil
}

func (r *schemaDoctorRecorder) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "schemaDoctorRecorder"}
}

func (r *schemaDoctorRecorder) GetCategory() string {
	return model.FunctionCategoryJSONSchema
}

func TestJSONSchemaExecutionBuildsDoctorDocument(t *testing.T) {
	recorder := &schemaDoctorRecorder{}
	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"schema-doctor-recorder": {
				Id:           "schema-doctor-recorder",
				Given:        "$",
				Formats:      []string{model.JSONSchemaDraft2020},
				Severity:     model.SeverityInfo,
				Then:         model.RuleAction{Function: "schemaDoctorRecorder"},
				RuleCategory: model.RuleCategories[model.CategorySchemas],
			},
		},
	}

	execution := &RuleSetExecution{
		RuleSet: rs,
		Spec: []byte(`$schema: https://json-schema.org/draft/2020-12/schema
type: object
properties:
  id:
    type: integer
`),
		SkipDocumentCheck: true,
		SpecFormat:        model.JSONSchemaDraft2020,
		CustomFunctions: map[string]model.RuleFunction{
			"schemaDoctorRecorder": recorder,
		},
	}

	result := ApplyRulesToRuleSet(execution)
	require.NotNil(t, result)
	require.NotNil(t, execution.DrDocument)
	require.NotEmpty(t, execution.DrDocument.Schemas)
	assert.Empty(t, result.Errors)
	assert.Contains(t, recorder.paths, "$")
	assert.Contains(t, recorder.paths, "$.properties['id']")
}

func TestJSONSchemaExecutionDoesNotEnterOpenAPIModelBranch(t *testing.T) {
	recorder := &schemaDoctorRecorder{}
	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"schema-doctor-recorder": {
				Id:           "schema-doctor-recorder",
				Given:        "$",
				Formats:      []string{model.JSONSchemaDraft2020},
				Severity:     model.SeverityInfo,
				Then:         model.RuleAction{Function: "schemaDoctorRecorder"},
				RuleCategory: model.RuleCategories[model.CategorySchemas],
			},
		},
	}

	execution := &RuleSetExecution{
		RuleSet: rs,
		Spec: []byte(`$schema: https://json-schema.org/draft/2020-12/schema
openapi: 3.1.0
type: object
properties:
  vesselId:
    type: string
`),
		SkipDocumentCheck: true,
		SpecFormat:        model.JSONSchemaDraft2020,
		CustomFunctions: map[string]model.RuleFunction{
			"schemaDoctorRecorder": recorder,
		},
	}

	var result *RuleSetExecutionResult
	require.NotPanics(t, func() {
		result = ApplyRulesToRuleSet(execution)
	})
	require.NotNil(t, result)
}
