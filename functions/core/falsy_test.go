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
  - name: "good tag 1"
    description: true
  - name: "good tag 2"
    description: 1
  - name: "good tag 3"
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
 - name: "bad tag 1"
   description: false
 - name: "bad tag 2"
   description: 0
 - name: "bad tag 3"
   description: "32"
 - name: "bad tag 4"`

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
 - name: "bad tag 1"
 - name: "bad tag 2"
   description: "false"
 - name: "bad tag 3"
   description: ""
 - name: "bad Tag 4"
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
