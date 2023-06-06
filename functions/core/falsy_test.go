package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFalsy_RunRule_Fail(t *testing.T) {

	sampleYaml := `
tags:
  - name: "non-falsy tag 1"
    description: true
  - name: "non-falsy tag 2"
    description: 1
  - name: "non-falsy tag 3"
    description: "hello"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 3)

	rule := buildCoreTestRule(path, model.SeverityError, "falsy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Falsy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 3)
}

func TestFalsy_RunRule_Fail_NoNodes(t *testing.T) {

	sampleYaml := `
notTags:
 - name: "falsy tag 1"
   description: false
 - name: "non-falsy tag 1"
   description: 1
 - name: "non-falsy tag 2"
   description: "2"
 - name: "falsy tag 2"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 0)

	rule := buildCoreTestRule(path, model.SeverityError, "falsy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Falsy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestFalsy_RunRule_Pass(t *testing.T) {

	sampleYaml := `
tags:
 - name: "falsy tag 1"
 - name: "falsy tag 2"
   description: "false"
 - name: "falsy tag 3"
   description: ""
 - name: "falsy Tag 4"
   description: "0"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 4)

	rule := buildCoreTestRule(path, model.SeverityError, "Falsy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Falsy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}
