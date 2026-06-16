// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"context"
	"testing"

	"github.com/daveshanley/vacuum/functions/schemachecks"
	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/model"
	doctorModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestSanityUsesECMA262Patterns(t *testing.T) {
	root, drDoc := buildJSONSchemaDoctor(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: string
pattern: "^(?=.*[A-Z]).+$"
`)

	results := runJSONSchemaSanity(root, drDoc, schemachecks.SanityCheckPatterns)
	require.Empty(t, results)
}

func TestSanityTreatsWholeFloatAsInteger(t *testing.T) {
	root, drDoc := buildJSONSchemaDoctor(t, `
$schema: https://json-schema.org/draft/2020-12/schema
type: integer
const: 42.0
enum:
  - 42.0
`)

	results := runJSONSchemaSanity(root, drDoc, schemachecks.SanityCheckEnumConst)
	require.Empty(t, results)
}

func buildJSONSchemaDoctor(t *testing.T, input string) (*yaml.Node, *doctorModel.DrDocument) {
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

func runJSONSchemaSanity(root *yaml.Node, drDoc *doctorModel.DrDocument, check string) []model.RuleFunctionResult {
	rule := &model.Rule{Id: "test-json-schema-sanity"}
	return Sanity{}.RunRule([]*yaml.Node{root}, model.RuleFunctionContext{
		DrDocument: drDoc,
		Options:    map[string]string{"check": check},
		Rule:       rule,
	})
}
