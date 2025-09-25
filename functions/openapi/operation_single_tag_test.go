package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestOperationSingleTag_GetSchema(t *testing.T) {
	def := OperationSingleTag{}
	assert.Equal(t, "oasOpSingleTag", def.GetSchema().Name)
}

func TestOperationSingleTag_RunRule(t *testing.T) {
	def := OperationSingleTag{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationSingleTag_RunRule_Fail(t *testing.T) {

	yml := `paths:
  /melody:
    post:
      tags: 
        - little
        - song
  /maddox:
    get:
      tags:
        - beautiful
        - boy
  /ember:
    get:
      tags:
        - naughty
        - dog`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "operation_single_tag", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationSingleTag{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 3)
}

func TestOperationSingleTag_RunRule_Success(t *testing.T) {

	yml := `paths:
  /melody:
    post:
      tags:
        - song
  /maddox:
    get:
      tags:
        - beautiful
  /ember:
    get:
      tags:
        - naughty`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "operation_single_tag", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationSingleTag{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}
