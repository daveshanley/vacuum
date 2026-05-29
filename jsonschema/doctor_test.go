// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestNewDoctorDocumentFromRolodexIndexBuildsSchemas(t *testing.T) {
	var root yaml.Node
	err := yaml.Unmarshal([]byte(`
$schema: https://json-schema.org/draft/2020-12/schema
title: Widget
type: object
properties:
  id:
    type: integer
  name:
    type: string
`), &root)
	require.NoError(t, err)

	rolodex := index.NewRolodex(index.CreateClosedAPIIndexConfig())
	rolodex.SetRootNode(&root)
	require.NoError(t, rolodex.IndexTheRolodex(context.Background()))
	idx := rolodex.GetRootIndex()
	require.NotNil(t, idx)
	require.NotNil(t, idx.GetRolodex())

	drDoc, err := NewDoctorDocumentFromRolodexIndex(idx, RolodexDoctorBuildConfig{
		DeterministicPaths: true,
		UseSchemaCache:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, drDoc)
	require.NotEmpty(t, drDoc.Schemas)

	paths := make([]string, 0, len(drDoc.Schemas))
	for _, schema := range drDoc.Schemas {
		paths = append(paths, schema.GenerateJSONPath())
	}
	assert.Contains(t, paths, "$")
	assert.Contains(t, paths, "$.properties['id']")
	assert.Contains(t, paths, "$.properties['name']")
	assert.Equal(t, 2, drDoc.Schemas[0].GetKeyNode().Line)
}

func TestNewDoctorDocumentFromRolodexIndexRejectsDetachedIndex(t *testing.T) {
	var root yaml.Node
	err := yaml.Unmarshal([]byte(`type: object`), &root)
	require.NoError(t, err)

	idx := index.NewSpecIndex(&root)
	require.Nil(t, idx.GetRolodex())

	drDoc, err := NewDoctorDocumentFromRolodexIndex(idx, RolodexDoctorBuildConfig{})
	require.Nil(t, drDoc)
	require.ErrorContains(t, err, "rolodex")
}

func TestNewDoctorDocumentFromRolodexIndexCollectsReferencedSchemasFromWalk(t *testing.T) {
	var root yaml.Node
	err := yaml.Unmarshal([]byte(`
$schema: https://json-schema.org/draft/2020-12/schema
type: object
properties:
  boat:
    $ref: "#/$defs/Boat"
$defs:
  Boat:
    type: object
    properties:
      hullId:
        type: string
`), &root)
	require.NoError(t, err)

	rolodex := index.NewRolodex(index.CreateClosedAPIIndexConfig())
	rolodex.SetRootNode(&root)
	require.NoError(t, rolodex.IndexTheRolodex(context.Background()))

	drDoc, err := NewDoctorDocumentFromRolodexIndex(rolodex.GetRootIndex(), RolodexDoctorBuildConfig{
		DeterministicPaths: true,
		UseSchemaCache:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, drDoc)

	paths := make([]string, 0, len(drDoc.Schemas))
	for _, schema := range drDoc.Schemas {
		paths = append(paths, schema.GenerateJSONPath())
	}
	assert.Contains(t, paths, "$")
	assert.Contains(t, paths, "$.properties['boat']")
	assert.Contains(t, paths, "$.properties['boat'].properties['hullId']")
}
