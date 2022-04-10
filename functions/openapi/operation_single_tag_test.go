package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestOperationSingleTag_GetSchema(t *testing.T) {
	def := OperationSingleTag{}
	assert.Equal(t, "operation_single_tag", def.GetSchema().Name)
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
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "operation_single_tag", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = model.NewSpecIndex(&rootNode)

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
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "operation_single_tag", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := OperationSingleTag{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}
