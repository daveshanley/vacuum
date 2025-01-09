package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefined_GetSchema(t *testing.T) {
	def := Defined{}
	assert.Equal(t, "defined", def.GetSchema().Name)
}

func TestDefined_RunRule(t *testing.T) {
	def := Defined{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestDefined_RunRule_Success(t *testing.T) {

	sampleYaml := `pizza:
  cake: "fridge"`

	path := "$.pizza"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "defined", "cake", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	def := Defined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestDefined_RunRule_Fail(t *testing.T) {

	sampleYaml :=
		`openapi: 3.0.0
paths:
  /v1/cake:
    get:
      responses:
        '200':
           content:
             application/xml:
               schema:
                 type: object
    post:
      responses:
        '200':
           content:
             application/json:
               schema:
                 type: object
`

	path := "$.paths.*.*.responses[*].content"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 2)

	document, err := libopenapi.NewDocument([]byte(sampleYaml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildCoreTestRule(path, model.SeverityError, "defined", "application/json", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule
	ctx.Document = document
	ctx.DrDocument = drDocument

	def := Defined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, res[0].Path, "$.paths['/v1/cake'].get.responses['200'].content['application/xml']")
}

func TestDefined_RunRule_DrNodeLookup(t *testing.T) {

	sampleYaml := `openapi: 3.0.0
tags:
  - name: "good"
  - name: "noFun"
  - name: "fridge"`

	path := "$.tags[*]"

	document, err := libopenapi.NewDocument([]byte(sampleYaml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 3)

	rule := buildCoreTestRule(path, model.SeverityError, "defined", "cake", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule
	ctx.Document = document
	ctx.DrDocument = drDocument

	def := Defined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 3)
	n, e := drDocument.LocateModelByLine(3)
	assert.NoError(t, e)
	assert.NotNil(t, n)
	assert.Equal(t, "good", n[0].(*base.Tag).Value.Name)

	n, e = drDocument.LocateModelByLine(4)
	assert.NoError(t, e)
	assert.NotNil(t, n)
	assert.Equal(t, "noFun", n[0].(*base.Tag).Value.Name)

	n, e = drDocument.LocateModelByLine(5)
	assert.NoError(t, e)
	assert.NotNil(t, n)
	assert.Equal(t, "fridge", n[0].(*base.Tag).Value.Name)

}
