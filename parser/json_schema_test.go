package parser

import (
    "github.com/pb33f/libopenapi/index"
    "github.com/pb33f/libopenapi/resolver"
    "github.com/stretchr/testify/assert"
    "github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
    "gopkg.in/yaml.v3"
    "testing"
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

    resolver := resolver.NewResolver(idx)
    resolver.Resolve()

    p, _ := yamlpath.NewPath("$.components.schemas.Citrus")
    r, _ := p.Find(&node)

    schema, err := ConvertNodeIntoJSONSchema(r[0], idx)
    assert.NoError(t, err)
    assert.NotNil(t, schema)
    assert.Len(t, schema.Properties, 3)

    // now check the schema is valid
    res, e := ValidateNodeAgainstSchema(schema, r[0], false)
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
          example: 1234
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

    rslvr := resolver.NewResolver(idx)
    rslvr.Resolve()

    p, _ := yamlpath.NewPath("$.components.schemas.Savory")
    r, _ := p.Find(&node)

    schema, _ := ConvertNodeIntoJSONSchema(r[0], idx)

    results := ValidateExample(schema)
    assert.Len(t, results, 6)

}
