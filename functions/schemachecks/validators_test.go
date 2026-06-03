// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	"context"
	"testing"

	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/model"
	doctorModel "github.com/pb33f/doctor/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestRunSchemaSanityCheckTypeConstraints(t *testing.T) {
	root, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: string
minimum: 10
`)

	results := runSanity(t, root, drDoc, SanityCheckType)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "`minimum` constraint")
}

func TestRunSchemaSanityCheckEnumConstUsesSharedIntegerLogic(t *testing.T) {
	root, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: integer
const: 42.0
enum:
  - 42.0
`)

	results := runSanity(t, root, drDoc, SanityCheckEnumConst)
	assert.Empty(t, results)
}

func TestRunSchemaSanityCheckPatternUsesECMA262(t *testing.T) {
	ecma262PatternCache.Delete("^(?=.*[A-Z]).+$")
	root, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: string
pattern: "^(?=.*[A-Z]).+$"
`)

	results := runSanity(t, root, drDoc, SanityCheckPatterns)
	assert.Empty(t, results)
	cached, ok := ecma262PatternCache.Load("^(?=.*[A-Z]).+$")
	require.True(t, ok)
	assert.Equal(t, true, cached)
}

func TestRunSchemaSanityCheckRequired(t *testing.T) {
	root, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: object
required:
  - hullId
  - hullId
  - captain
properties:
  hullId:
    type: string
`)

	results := runSanity(t, root, drDoc, SanityCheckRequired)
	require.Len(t, results, 2)
	assert.Contains(t, results[0].Message, "duplicates")
	assert.Contains(t, results[1].Message, "captain")
}

func TestRunSchemaSanityCheckDependentRequired(t *testing.T) {
	root, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: object
dependentRequired:
  marinaId:
    - dockId
properties:
  vesselId:
    type: string
`)

	results := runSanity(t, root, drDoc, SanityCheckDependent)
	require.NotEmpty(t, results)
	assert.Contains(t, results[0].Message, "marinaId")
}

func TestRunSchemaSanityCheckComposition(t *testing.T) {
	root, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
allOf:
  - type: string
  - type: object
`)

	results := runSanity(t, root, drDoc, SanityCheckComposition)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "directly conflicting types")
}

func buildDoctorSchemas(t *testing.T, input string) (*yaml.Node, *doctorModel.DrDocument) {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(input), &root))

	rolodex := index.NewRolodex(index.CreateClosedAPIIndexConfig())
	rolodex.SetRootNode(&root)
	require.NoError(t, rolodex.IndexTheRolodex(context.Background()))

	drDoc, err := schemautil.NewDoctorDocumentFromRolodexIndex(rolodex.GetRootIndex(), schemautil.RolodexDoctorBuildConfig{
		DeterministicPaths: true,
		UseSchemaCache:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, drDoc)

	return &root, drDoc
}

func runSanity(t *testing.T, root *yaml.Node, drDoc *doctorModel.DrDocument, check string) []model.RuleFunctionResult {
	t.Helper()
	context := model.RuleFunctionContext{
		DrDocument: drDoc,
		Rule:       &model.Rule{Id: "test-schema-check"},
	}
	var results []model.RuleFunctionResult
	for _, schema := range drDoc.Schemas {
		results = append(results, RunSchemaSanityCheck(schema, schemautil.RootNode(root), &context, check)...)
	}
	return results
}

func TestRunTypeChecksUsesOpenAPIOptions(t *testing.T) {
	_, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: string
pattern: "["
`)

	context := model.RuleFunctionContext{
		DrDocument: drDoc,
		Rule:       &model.Rule{Id: "test-schema-type-check"},
	}
	results := RunTypeChecks(drDoc.Schemas, context, TypeCheckOptions{
		ValidatePatterns:           true,
		ValidateValueCompatibility: true,
	})
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "ECMA-262")
}

func TestRunTypeChecksSkipsConstraintMismatchForUntypedSchemas(t *testing.T) {
	_, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
minProperties: 1
`)

	context := model.RuleFunctionContext{
		DrDocument: drDoc,
		Rule:       &model.Rule{Id: "test-schema-type-check"},
	}
	results := RunTypeChecks(drDoc.Schemas, context, TypeCheckOptions{})
	assert.Empty(t, results)
}

func TestSchemaUsesObjectKeywords(t *testing.T) {
	_, drDoc := buildDoctorSchemas(t, `
$schema: https://json-schema.org/draft/2020-12/schema
required:
  - vesselId
`)

	var rootSchema *drV3.Schema
	for _, schema := range drDoc.Schemas {
		if schema.GenerateJSONPath() == "$" {
			rootSchema = schema
		}
	}
	require.NotNil(t, rootSchema)
	assert.True(t, SchemaUsesObjectKeywords(rootSchema))
}
