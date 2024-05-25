package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestDescriptionDuplication_GetSchema(t *testing.T) {
	def := DescriptionDuplication{}
	assert.Equal(t, "oasDescriptionDuplication", def.GetSchema().Name)
}

func TestDescriptionDuplication_RunRule(t *testing.T) {
	def := DescriptionDuplication{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestDescriptionDuplication_DescriptionDuplication(t *testing.T) {

	yml := `paths:
  /fish/paste:
    get:
      description: a nice cup of tea
    put:
      description: a nice cup of coffee
    post:
      description: a nice cup of coca
components:
  schemas:
    Tea:
      description: a nice cup of tea `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := DescriptionDuplication{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestDescriptionDuplication_SummaryDuplication(t *testing.T) {

	yml := `paths:
  /fish/paste:
    get:
      summary: a nice cup of tea
    put:
      summary: a nice cup of coffee
    post:
      summary: a nice cup of coca
components:
  schemas:
    Tea:
      summary: a nice cup of tea `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := DescriptionDuplication{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestDescriptionDuplication_SummaryDescriptionDuplication(t *testing.T) {

	yml := `paths:
  /fish/paste:
    get:
      description: a nice cup of tea
    put:
      summary: a nice cup of coffee
    post:
      summary: a nice cup of tea
components:
  schemas:
    Tea:
      summary: a nice cup of tea `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)
	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := DescriptionDuplication{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 3)

}
