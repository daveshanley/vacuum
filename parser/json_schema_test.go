package parser

import (
	"github.com/daveshanley/vacuum/model"
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
	yaml.Unmarshal([]byte(yml), &node)

	resolved, _ := model.ResolveOpenAPIDocument(&node)

	p, _ := yamlpath.NewPath("$.components.schemas.Citrus")
	r, _ := p.Find(resolved)

	schema, err := ConvertNodeDefinitionIntoSchema(r[0])
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Len(t, schema.Properties, 3)

	// now check the schema is valid
	res, e := ValidateNodeAgainstSchema(schema, r[0])
	assert.NoError(t, e)
	assert.Equal(t, true, res.Valid())
}
