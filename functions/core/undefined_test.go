package core

import (
	"github.com/daveshanley/vacuum/model"
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

	sampleYaml := `pizza:
  cake: "fridge"`

	path := "$.pizza"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "undefined", "cake", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path

	def := Undefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestUndefined_RunRule_Fail(t *testing.T) {

	sampleYaml := `pizza:
  noCake: "noFun"`

	path := "$.pizza"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "undefined", "cake", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path

	def := Undefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}
