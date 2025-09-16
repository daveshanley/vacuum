package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestOperd_GetSchema(t *testing.T) {
	def := TagDefined{}
	assert.Equal(t, "oasTagDefined", def.GetSchema().Name)
}

func TestTagDefined_RunRule(t *testing.T) {
	def := TagDefined{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestTagDefined_RunRule_Success(t *testing.T) {

	yml := `tags:
  - name: "princess"
  - name: "prince"
  - name: "hope"
  - name: "naughty_dog"
paths:
  /melody:
    post:
      tags:
       - "princess"
       - "hope"
  /maddox:
    get:
      tags:
       - "prince"
       - "hope"
  /ember:
    get:
      tags:
       - "naughty_dog"`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "tag_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := TagDefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
	//assert.Equal(t, "the 'get' operation at path '/ember' contains a duplicate operationId 'littleSong'", res[0].Message)
}

func TestTagDefined_RunRule_Fail(t *testing.T) {

	yml := `openapi: 3.0.1
tags:
  - name: "princess"
  - name: "prince"
  - name: "hope"
  - name: "naughty_dog"
paths:
  /melody:
    post:
      tags:
       - "princess"
       - "hope"
  /maddox:
    get:
      tags:
       - "prince"
       - "hope"
  /ember:
    get:
      tags:
       - "such_a_naughty_dog"`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := TagDefined{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "tag `such_a_naughty_dog` for `GET` operation is not defined as a global tag", res[0].Message)
}
