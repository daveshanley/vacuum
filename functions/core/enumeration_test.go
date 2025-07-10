package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestEnumeration_GetSchema(t *testing.T) {
	def := &Enumeration{}
	assert.Equal(t, "enumeration", def.GetSchema().Name)
	assert.Equal(t, 1, def.GetSchema().MinProperties)
}

func TestEnumeration_RunRule(t *testing.T) {
	def := &Enumeration{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Success(t *testing.T) {
	sampleYaml := `christmas: "ham"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = "turkey, sprouts, presents, ham"

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Array_bool_Fail(t *testing.T) {
	sampleYaml := `christmas: true`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{false}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	require.Len(t, res, 1)
	assert.Equal(t, "test rule: `true` must equal to one of: [false]", res[0].Message)
}

func TestEnumeration_RunRule_Array_bool_Success(t *testing.T) {
	sampleYaml := `christmas: true`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{true, false}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Array_float_Fail(t *testing.T) {
	sampleYaml := `christmas: 16.2`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{1, 2, 3, 4, 5}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	require.Len(t, res, 1)
	assert.Equal(t, "test rule: `16.2` must equal to one of: [1 2 3 4 5]", res[0].Message)
}

func TestEnumeration_RunRule_Array_float_Success(t *testing.T) {
	sampleYaml := `christmas: 2.2`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{1, 2.2, 3, 4, 5}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Array_int_Fail(t *testing.T) {
	sampleYaml := `christmas: 16`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{1, 2, 3, 4, 5}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	require.Len(t, res, 1)
	assert.Equal(t, "test rule: `16` must equal to one of: [1 2 3 4 5]", res[0].Message)
}

func TestEnumeration_RunRule_Array_int_Success(t *testing.T) {
	sampleYaml := `christmas: 2`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{1, 2, 3, 4, 5}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Array_Success(t *testing.T) {
	sampleYaml := `christmas: "ham"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{"turkey", "sprouts", "presents", "ham", ",,,,,,,"}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Array_Fail(t *testing.T) {
	sampleYaml := `christmas: "arguments"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{"turkey", "sprouts", "presents", "ham", ",,,,,,,"}

	rule := buildCoreTestRule(path, model.SeverityError, "enumeration", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestEnumeration_RunRule_Fail(t *testing.T) {
	sampleYaml := `christmas: "arguments"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = "turkey, sprouts, presents, ham"

	rule := buildCoreTestRule(path, model.SeverityError, "enumeration", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestEnumeration_RunRule_FalseFail(t *testing.T) {
	sampleYaml := `christmas: "arguments"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any) // don't add opts.

	rule := buildCoreTestRule(path, model.SeverityError, "enumeration", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // should fail, but no opts are passed. will be checked by validation.
}

func TestEnumeration_RunRule_Array_int64_Fail(t *testing.T) {
	sampleYaml := `christmas: 9223372036854775807`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{int64(1), int64(2), int64(3), int64(4), int64(5)}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	require.Len(t, res, 1)
	assert.Equal(t, "test rule: `9223372036854775807` must equal to one of: [1 2 3 4 5]", res[0].Message)
}

func TestEnumeration_RunRule_Array_int64_Success(t *testing.T) {
	sampleYaml := `christmas: 2`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []any{int64(1), int64(2), int64(3), int64(4), int64(5)}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Array_default_Fail(t *testing.T) {
	sampleYaml := `christmas: "complex"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	// Using a map as a complex type that will hit the default case
	complexValue := map[string]string{"key": "value"}
	opts["values"] = []any{complexValue}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	require.Len(t, res, 1)
	assert.Equal(t, "test rule: `complex` must equal to one of: [map[key:value]]", res[0].Message)
}

func TestEnumeration_RunRule_Array_default_Success(t *testing.T) {
	// Create a YAML with a string that matches the string representation of our complex type
	sampleYaml := `christmas: "map[key:value]"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	// Using a map as a complex type that will hit the default case
	complexValue := map[string]string{"key": "value"}
	opts["values"] = []any{complexValue}

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}
