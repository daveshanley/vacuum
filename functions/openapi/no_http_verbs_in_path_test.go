package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestVerbsInPaths_GetSchema(t *testing.T) {
	def := VerbsInPaths{}
	assert.Equal(t, "noVerbsInPath", def.GetSchema().Name)
}

func TestVerbsInPaths_RunRule(t *testing.T) {
	def := VerbsInPaths{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestVerbsInPaths_Success(t *testing.T) {

	yml := `openapi: 3.0.0
paths:
  '/oh/no/post/man':
    get:
      summary: bad path
  '/this/one/is/OK':
    get:
      summary: good path
  '/i/am/going/to/get/failed':
    get:
      summary: not a good one.
  '/will/you/patch/my/code':
    get:
      summary: this is also doomed
  '/put/my/cake/away':
    get:
      summary: another bad one.
  '/this/is/fine':
    get:
      summary: all done squire`
	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "verbsInPath", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := VerbsInPaths{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)

}
