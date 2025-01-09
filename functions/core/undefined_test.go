package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUndefined_GetSchema(t *testing.T) {
	def := Undefined{}
	assert.Equal(t, "undefined", def.GetSchema().Name)
}

func TestUndefined_RunRule(t *testing.T) {
	def := Undefined{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestUndefined_RunRule_Success(t *testing.T) {

	sampleYaml :=
		`openapi: 3.0.0
pizza:
  cake: "fridge"`

	path := "$.pizza"

	document, err := libopenapi.NewDocument([]byte(sampleYaml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "undefined", "cake", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule
	ctx.Document = document
	ctx.DrDocument = drDocument

	def := Undefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestUndefined_RunRule_Fail(t *testing.T) {

	sampleYaml :=
		`openapi: 3.0.0
pizza:
  noCake: "noFun"`

	path := "$.pizza"

	document, err := libopenapi.NewDocument([]byte(sampleYaml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "undefined", "cake", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule
	ctx.Document = document
	ctx.DrDocument = drDocument

	def := Undefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}
