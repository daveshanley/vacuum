package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLength_RunRule_Pass(t *testing.T) {

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
	ops["min"] = "3"

	rule := buildCoreTestRule(path, severityError, "length", "paths", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

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

	rule := buildCoreTestRule(path, severityError, "length", "paths", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

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

	rule := buildCoreTestRule(path, severityError, "length", "paths", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

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

	rule := buildCoreTestRule(path, severityError, "length", "tags", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

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

	rule := buildCoreTestRule(path, severityError, "length", "tags", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, res[0].Message, "'tags' must not be longer/greater than '2'")
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

	rule := buildCoreTestRule(path, severityError, "length", "tags", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, res[0].Message, "'tags' must not be longer/greater than '4'")
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

	rule := buildCoreTestRule(path, severityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

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

	rule := buildCoreTestRule(path, severityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

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

	rule := buildCoreTestRule(path, severityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

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

	rule := buildCoreTestRule(path, severityError, "length", "description", ops)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), ops)

	le := Length{}
	res := le.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}
