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

func TestTruthy_GetSchema(t *testing.T) {
	def := Truthy{}
	assert.Equal(t, "truthy", def.GetSchema().Name)
}

func TestTruthy_RunRule(t *testing.T) {
	def := Truthy{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestTruthy_RunRule_Fail(t *testing.T) {

	sampleYaml := `
tags:
  - name: "bad tag 1"
    description: false
  - name: "bad tag 2"
    description: 0
  - name: "bad tag 3"
    description: ""
  - name: "bad tag 4"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 4)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 4)
}

func TestTruthy_RunRule_Fail_NoNodes(t *testing.T) {

	sampleYaml := `
notTags:
  - name: "bad tag 1"
    description: false
  - name: "bad tag 2"
    description: 0
  - name: "bad tag 3"
    description: ""
  - name: "bad tag 4"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 0)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestTruthy_RunRule_Pass(t *testing.T) {

	sampleYaml := `
tags:
  - name: "good tag 1"
    description: "yeah"
  - name: "bad tag 2"
    description: 0
  - name: "bad tag 3"
    description: ""
  - name: "good Tag 2"
    description: "a nice description"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 4)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

func TestTruthy_RunRule_NoContent(t *testing.T) {

	sampleYaml :=
		`openapi: 3.0.0
paths:
  /v1/cake:
    get:
      parameters:
        - in: query
          name: type
          required: true
        - in: query
          name: flavor
          required: false
        - in: query
          name: weight
`

	path := "$.paths.*.*.parameters[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 3)

	document, err := libopenapi.NewDocument([]byte(sampleYaml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "required", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule
	ctx.Document = document
	ctx.DrDocument = drDocument

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	// Two of the three nodes should match because one has a truthy value
	assert.Len(t, res, 2)
	assert.Equal(t, res[0].Path, "$.paths['/v1/cake'].get.parameters[1].required")
	assert.Equal(t, res[1].Path, "$.paths['/v1/cake'].get.parameters[2].required")
}

func TestTruthy_RunRule_ArrayTest(t *testing.T) {

	sampleYaml := `- lemons:
  rags:
    - name: fish
    - name: cake
    - name: pizza
- limes:
  tags:
    - name: fish
    - name: cake`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "rags", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestTruthy_RunRule_CheckSecurity(t *testing.T) {

	sampleYaml := `notSecurity: none`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "security", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
