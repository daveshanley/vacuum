package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestXorChecksEachObjectIndependently(t *testing.T) {
	nodes, err := utils.FindNodes([]byte(`items:
  - {a: null}
  - {b: 1}
  - {a: null, b: 1}
  - {}
  - text
  - null
  - []
`), "$.items[*]")
	require.NoError(t, err)
	opts := map[string]any{"properties": []any{"a", "b"}}
	rule := buildCoreTestRule("$.items[*]", model.SeverityError, "xor", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = "$.items[*]"
	results := (Xor{}).RunRule(nodes, ctx)
	require.Len(t, results, 5)
	for i, result := range results {
		require.Same(t, nodes[i+2], result.StartNode)
	}
}

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

	rule := buildCoreTestRule(path, model.SeverityError, "xor", "", nil)
	ctx := model.RuleFunctionContext{RuleAction: model.CastToRuleAction(rule.Then), Rule: &rule, Options: opts}
	ctx.Given = path
	ctx.Rule = &rule

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

	opts := make(map[string]any)
	opts["properties"] = "sparkles, rainbows"

	rule := buildCoreTestRule(path, model.SeverityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

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

	opts := make(map[string]any)

	rule := buildCoreTestRule(path, model.SeverityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

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

	opts := make(map[string]any)
	opts["properties"] = "sparkles, shiny"

	rule := buildCoreTestRule(path, model.SeverityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

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

	opts := make(map[string]any)
	opts["properties"] = "clouds, rain"

	rule := buildCoreTestRule(path, model.SeverityError, "xor", "", opts)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := Xor{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestXor_GetSchema_Invalid_Min(t *testing.T) {

	opts := make(map[string]any)
	opts["properties"] = ""

	rf := &Xor{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}

func TestXor_GetSchema_Invalid_Min_NotEnough(t *testing.T) {

	opts := make(map[string]any)
	opts["properties"] = "notenough"

	rf := &Xor{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}

func TestXor_GetSchema_Invalid_Max(t *testing.T) {

	opts := make(map[string]any)
	opts["properties"] = "chip, chop, chap"

	rf := &Xor{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 1)
	assert.False(t, res)

}
