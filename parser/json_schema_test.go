package parser

import (
	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// test we can generate a schema from a simple object
func TestConvertNode_Simple(t *testing.T) {
	yml := `components:
  schemas:
    Citrus:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        savory:
          $ref: '#/components/schemas/Savory'  
    Savory:
      type: object
      properties:
        tasteIndex:
          type: integer
        butter:
          type: boolean`

	var node yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, config)

	resolver := index.NewResolver(idx)
	resolver.Resolve()

	p, _ := jsonpath.NewPath("$.components.schemas.Citrus")
	r := p.Query(&node)

	schema, err := ConvertNodeIntoJSONSchema(r[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, 3, orderedmap.Len(schema.Properties))

	// now check the schema is valid
	res, e := ValidateNodeAgainstSchema(nil, schema, r[0], false)
	assert.Nil(t, e)
	assert.True(t, res)
}

func TestValidateExample_AllInvalid(t *testing.T) {
	yml := `components:
  schemas:
    Citrus:
      type: object
      properties:
        id:
          type: integer
          example: 1234.5
        name:
          type: string
          example: false
    Savory:
      type: object
      properties:
        tasteIndex:
          type: integer
          example: hello
        butter:
          type: boolean
          example: 123.224
        fridge:
          type: number
          example: false
        cake:
          type: string
          example: 1233
        pizza:
          $ref: '#/components/schemas/Citrus'`

	var node yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, config)

	rslvr := index.NewResolver(idx)
	rslvr.Resolve()

	p, _ := jsonpath.NewPath("$.components.schemas.Savory")
	r := p.Query(&node)

	schema, _ := ConvertNodeIntoJSONSchema(r[0], idx)

	results := ValidateExample(schema)
	assert.Len(t, results, 6)
}
