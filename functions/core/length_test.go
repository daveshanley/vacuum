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

func TestLength_GetSchema(t *testing.T) {
	def := Length{}
	assert.Equal(t, "length", def.GetSchema().Name)
}

func TestLength_RunRule(t *testing.T) {
	def := Length{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestLength_RunRule_Pass(t *testing.T) {

	sampleYaml := `openapi: 3.1.0
paths:
    /something:
        get:
          description: "this is a description"
    /nothing:
        post:
          description: "this is a description"
    /free:
        patch:
          description: "this is a description"`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(sampleYaml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["min"] = "3"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "paths", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Rule = &rule
	ctx.Given = path
	ctx.Document = document
	ctx.DrDocument = drDocument

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestLength_RunRule_Fail(t *testing.T) {

	sampleYaml := `
paths:
    /something:
        get:
    /nothing:
        post:
    /free:
        patch:`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["min"] = "4"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "paths", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestLength_RunRule_Fail_BadJSONPath(t *testing.T) {

	sampleYaml := `
paths:
    /something:
        get:
    /nothing:
        post:
    /free:
        patch:`

	path := "$.paths[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["min"] = "4"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "paths", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // this fails because we're looking for 'paths' in a node array already made of paths.
}

func TestLength_RunRule_CheckArray(t *testing.T) {

	sampleYaml := `
tags:
  - name: "bad tag 1"
  - name: "bad tag 2"
  - name: "bad tag 3"
  - name: "bad tag 4"`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["min"] = "6"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "tags", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestLength_RunRule_CheckArrayMaxTooBig(t *testing.T) {

	sampleYaml := `
tags:
  - name: "bad tag 1"
  - name: "bad tag 2"
  - name: "bad tag 3"
  - name: "bad tag 4"`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "2"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "tags", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "test rule: 'tags' must not be longer/greater than '2'", res[0].Message)
}

func TestLength_RunRule_CheckArrayOutOfBounds(t *testing.T) {

	sampleYaml := `
tags:
  - name: "bad tag 1"
  - name: "bad tag 2"
  - name: "bad tag 3"
  - name: "bad tag 4"
  - name: "bad tag 5"`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "4"
	ops["min"] = "2"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "tags", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "test rule: 'tags' must not be longer/greater than '4'", res[0].Message)
}

func TestLength_RunRule_CheckLengthOfStringValue(t *testing.T) {

	sampleYaml := `
tags:
  - name: "taggy"
    description: "five"
  - name: "tiggy"
    description: "o"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "3"
	ops["min"] = "2"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestLength_RunRule_CheckLengthOfNumberValue(t *testing.T) {

	sampleYaml := `
tags:
  - name: "taggy"
    description: 10
  - name: "tiggy"
    description: 1`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "9"
	ops["min"] = "2"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestLength_RunRule_CheckLengthOfFloatValue(t *testing.T) {

	// should have the same effect as an int.

	sampleYaml := `
tags:
  - name: "taggy"
    description: 10.12
  - name: "tiggy"
    description: 1.22`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "9"
	ops["min"] = "2"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestLength_RunRule_EmptyRuleActionField_Min(t *testing.T) {

	sampleYaml := `
tags:
  - name: "taggy"
    description: 10.12
  - name: "tiggy"
    description: 1.22`

	path := "$.tags"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "9"
	ops["min"] = "3"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestLength_RunRule_EmptyRuleActionField_Max(t *testing.T) {

	sampleYaml := `
tags:
  - name: "taggy"
    description: 10.12
  - name: "tiggy"
    description: 1.22`

	path := "$.tags"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "1"
	ops["min"] = "0"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestLength_RunRule_NoNodes(t *testing.T) {

	// should have the same effect as an int.

	sampleYaml := `
tags:`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "9"
	ops["min"] = "2"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestLength_RunRule_NoOptions(t *testing.T) {

	sampleYaml := `
tags:
  - name: "taggy"
    description: 10.12
  - name: "tiggy"
    description: 1.22`

	path := "$.tags"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	rule := buildCoreTestRule(path, model.SeverityError, "length", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Options = nil
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // no opts/mix/max returns nothing.

}

func TestLength_RunRule_InvalidOptions(t *testing.T) {

	sampleYaml := `
tags:
  - name: "taggy"
    description: 10.12
  - name: "tiggy"
    description: 1.22`

	path := "$.tags"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	rule := buildCoreTestRule(path, model.SeverityError, "length", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Options = "not options at all"
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // should just do nothing.

}

func TestLength_RunRule_TestPathAllTags(t *testing.T) {

	sampleYaml := `
tags:
  - name: "taggy"
    description: 10.12
  - name: "tiggy"
    description: 1.22`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	ops := make(map[string]string)
	ops["max"] = "1"
	ops["min"] = "0"

	rule := buildCoreTestRule(path, model.SeverityError, "length", "name", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Given = path
	ctx.Rule = &rule

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}
