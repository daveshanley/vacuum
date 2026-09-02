package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"testing"
)

func TestOr_RunRule(t *testing.T) {
	def := Or{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOr_RunRule_SuccessPropsStringArray(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string][]string)
	opts["properties"] = []string{"sparkles", "rainbows"}

	rule := buildCoreTestRule(path, model.SeverityError, "or", "", nil)
	ctx := model.RuleFunctionContext{RuleAction: model.CastToRuleAction(rule.Then), Rule: &rule, Options: opts}
	ctx.Given = path
	ctx.Rule = &rule

	def := Or{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOr_RunRule_Success(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["properties"] = "sparkles, rainbows"

	rule := buildCoreTestRule(path, model.SeverityError, "or", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := Or{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOr_RunRule_NoProps(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)

	rule := buildCoreTestRule(path, model.SeverityError, "or", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := Or{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // no props? the rule is useless, validation should catch this however.
}

func TestOr_RunRule_All_Present(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["properties"] = "sparkles, shiny"

	rule := buildCoreTestRule(path, model.SeverityError, "or", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := Or{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOr_RunRule_Fail_AllUndefined(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["properties"] = "clouds, rain"

	rule := buildCoreTestRule(path, model.SeverityError, "or", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := Or{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestOr_GetSchema_Invalid_Min(t *testing.T) {

	opts := make(map[string]any)
	opts["properties"] = ""

	rf := &Or{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}

func TestOr_GetSchema_Invalid_Min_NotEnough(t *testing.T) {

	opts := make(map[string]any)
	opts["properties"] = "notenough"

	rf := &Or{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}

func TestOr_GetSchema_Does_Not_Have_Max(t *testing.T) {

	opts := make(map[string]any)
	opts["properties"] = "chip, chop, chap"

	rf := &Or{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 0)
	assert.True(t, res)

}
