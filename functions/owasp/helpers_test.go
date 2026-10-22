package owasp

import (
	"sync"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestExplicitInsecureScheme(t *testing.T) {
	for _, tc := range []struct {
		url      string
		insecure bool
	}{
		{"https://example.com", false},
		{"HTTPS://example.com", false},
		{"/oauth/token", false},
		{"//example.com/token", false},
		{"http://example.com", true},
		{"https-insecure://example.com", true},
		{":invalid", false},
	} {
		t.Run(tc.url, func(t *testing.T) {
			assert.Equal(t, tc.insecure, explicitInsecureScheme(tc.url))
		})
	}
}

func TestComposedOnlySchemas_CacheIsolationAndConcurrentReads(t *testing.T) {
	cache := &sync.Map{}
	for _, direct := range []bool{false, true} {
		spec := `openapi: 3.1.0
info: {title: Test, version: '1'}
paths: {}
components:
  schemas:
    Input:
      allOf:
        - $ref: '#/components/schemas/Base'
      unevaluatedProperties: false
    Base:
      type: object
      properties:
        name: {type: string}
`
		if direct {
			spec += "    Direct: {$ref: '#/components/schemas/Base'}\n"
		}
		document, err := libopenapi.NewDocument([]byte(spec))
		require.NoError(t, err)
		m, err := document.BuildV3Model()
		require.NoError(t, err)
		ctx := model.RuleFunctionContext{
			Document:        document,
			DrDocument:      drModel.NewDrDocument(m),
			Index:           m.Index,
			SchemaPathCache: cache,
		}
		target := m.Model.Components.Schemas.GetOrZero("Base").Schema().GoLow().RootNode
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				assert.Equal(t, !direct, composedOnlySchemas(ctx)[target])
			})
		}
		wg.Wait()
		assert.Equal(t, !direct, composedOnlySchemas(ctx)[target])
		ctx.SchemaPathCache = nil
		assert.Equal(t, !direct, composedOnlySchemas(ctx)[target])
	}
	assert.Nil(t, composedOnlySchemas(model.RuleFunctionContext{}))
}
