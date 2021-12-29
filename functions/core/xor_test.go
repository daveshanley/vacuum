package core

import (
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestXor_RunRule(t *testing.T) {
	def := Xor{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestXor_RunRule_SuccessPropsStringArray(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string][]string)
	opts["properties"] = []string{"sparkles", "rainbows"}

	rule := buildCoreTestRule(path, severityError, "xor", "", nil)
	ctx := model.RuleFunctionContext{RuleAction: model.CastToRuleAction(rule.Then), Rule: &rule, Options: opts}

	def := Xor{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestXor_RunRule_Success(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["properties"] = "sparkles, rainbows"

	rule := buildCoreTestRule(path, severityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)

	def := Xor{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestXor_RunRule_NoProps(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, severityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)

	def := Xor{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // no props? the rule is useless, validation should catch this however.
}

func TestXor_RunRule_Fail(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["properties"] = "sparkles, shiny"

	rule := buildCoreTestRule(path, severityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)

	def := Xor{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestXor_RunRule_Fail_AllUndefined(t *testing.T) {

	sampleYaml := `glitter:
  sparkles: "lots"
  shiny: 1000`

	path := "$.glitter"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["properties"] = "clouds, rain"

	rule := buildCoreTestRule(path, severityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)

	def := Xor{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestXor_GetSchema_Invalid_Min(t *testing.T) {

	opts := make(map[string]string)
	opts["properties"] = ""

	rf := &Xor{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}

func TestXor_GetSchema_Invalid_Min_NotEnough(t *testing.T) {

	opts := make(map[string]string)
	opts["properties"] = "notenough"

	rf := &Xor{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}

func TestXor_GetSchema_Invalid_Max(t *testing.T) {

	opts := make(map[string]string)
	opts["properties"] = "chip, chop, chap"

	rf := &Xor{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}
