// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"os"
	"runtime"
	"sync"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocateSchemaPropertyPaths_DoesNotCacheIncompleteFallback(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: test
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
  /pets2:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
`
	document, err := libopenapi.NewDocument([]byte(yml))
	require.NoError(t, err)

	v3Model, err := document.BuildV3Model()
	require.NoError(t, err)

	drDocument := drModel.NewDrDocument(v3Model)

	var petSchema *v3.Schema
	for _, schema := range drDocument.Schemas {
		if schema.GenerateJSONPath() == "$.components.schemas['Pet']" {
			petSchema = schema
			break
		}
	}
	require.NotNil(t, petSchema)

	ctx := model.RuleFunctionContext{
		DrDocument:      drDocument,
		SchemaPathCache: &sync.Map{},
	}

	// First call is intentionally incomplete: no key/value nodes.
	primaryPath, allPaths := LocateSchemaPropertyPaths(ctx, petSchema, nil, nil)
	assert.Equal(t, "$.components.schemas['Pet']", primaryPath)
	assert.Equal(t, []string{"$.components.schemas['Pet']"}, allPaths)

	_, cachedAfterIncompleteLookup := ctx.SchemaPathCache.Load(petSchema)
	assert.False(t, cachedAfterIncompleteLookup, "incomplete fallback results must not be cached")

	keyNode := petSchema.Value.GoLow().Type.KeyNode
	valueNode := petSchema.Value.GoLow().Type.ValueNode
	require.NotNil(t, keyNode)
	require.NotNil(t, valueNode)

	// Second call is complete and should resolve all schema locations.
	primaryPath, allPaths = LocateSchemaPropertyPaths(ctx, petSchema, keyNode, valueNode)
	assert.Equal(t, "$.components.schemas['Pet']", primaryPath)
	assert.Greater(t, len(allPaths), 1)
	assert.Contains(t, allPaths, "$.paths['/pets'].get.responses['200'].content['application/json'].schema")
	assert.Contains(t, allPaths, "$.paths['/pets2'].get.responses['200'].content['application/json'].schema")

	cached, ok := ctx.SchemaPathCache.Load(petSchema)
	require.True(t, ok)
	cachedResult, ok := cached.(*schemaPathResult)
	require.True(t, ok)
	assert.Greater(t, len(cachedResult.allPaths), 1)
}

func TestLocateSchemaPropertyPaths_Issue768_CanonicalPrimaryPath(t *testing.T) {
	specBytes, err := os.ReadFile("../model/test_files/issue_768_test.yaml")
	require.NoError(t, err, "issue_768_test.yaml must exist in model/test_files")

	origProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(origProcs)

	const expectedPath = "$.components.schemas['ServiceLevelObjectiveAlertSeverityV1'].properties['labels'].properties['hedwig_scope']"
	const iterations = 50

	for i := 0; i < iterations; i++ {
		document, err := libopenapi.NewDocument(specBytes)
		require.NoError(t, err)

		v3Model, err := document.BuildV3Model()
		require.NoError(t, err)

		drDocument := drModel.NewDrDocumentWithConfig(v3Model, &drModel.DrConfig{
			UseSchemaCache:     true,
			DeterministicPaths: true,
		})

		resource := drDocument.V3Document.Components.Schemas.GetOrZero("ServiceLevelObjectiveAlertSeverityV1")
		require.NotNil(t, resource)
		require.NotNil(t, resource.Schema)

		labels := resource.Schema.Properties.GetOrZero("labels")
		require.NotNil(t, labels)
		require.NotNil(t, labels.Schema)

		hedwig := labels.Schema.Properties.GetOrZero("hedwig_scope")
		require.NotNil(t, hedwig)
		require.NotNil(t, hedwig.Schema)

		keyNode := hedwig.Schema.Value.GoLow().Type.KeyNode
		valueNode := hedwig.Schema.Value.GoLow().Type.ValueNode
		require.NotNil(t, keyNode)
		require.NotNil(t, valueNode)

		ctx := model.RuleFunctionContext{
			DrDocument:      drDocument,
			SchemaPathCache: &sync.Map{},
		}

		primaryPath, allPaths := LocateSchemaPropertyPaths(ctx, hedwig.Schema, keyNode, valueNode)
		assert.Equalf(t, expectedPath, primaryPath, "iteration %d should keep the canonical path", i)
		require.NotEmpty(t, allPaths)
		assert.Equalf(t, expectedPath, allPaths[0], "iteration %d should keep canonical path first", i)
		assert.Contains(t, allPaths, expectedPath)
	}
}
