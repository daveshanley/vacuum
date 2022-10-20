package core

import (
	"github.com/daveshanley/vacuum/model"
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

	def := Defined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestDefined_RunRule_Fail(t *testing.T) {

	sampleYaml := `pizza:
  noCake: "noFun"`

	path := "$.pizza"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "defined", "cake", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path

	def := Defined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
