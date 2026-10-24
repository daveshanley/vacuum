package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"testing"
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

func TestEnumeration_RejectsUnsupportedOptionMap(t *testing.T) {
	valid, errs := model.ValidateRuleFunctionContextAgainstSchema(Enumeration{}, model.RuleFunctionContext{
		Options: map[string][]string{"values": {"allowed"}},
	})
	require.False(t, valid)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "missing required property: values")
}

func TestEnumerationScalarTypes(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		allowed     []any
		findings    int
	}{
		{"string is not number", `"1"`, []any{1}, 1},
		{"number is not string", `1`, []any{"1"}, 1},
		{"decimal equals integer", `1.0`, []any{1}, 0},
		{"integer equals decimal", `1`, []any{float64(1)}, 0},
		{"large integer stays exact", `9007199254740993`, []any{float64(9007199254740992)}, 1},
		{"string is not boolean", `"true"`, []any{true}, 1},
		{"boolean is not string", `true`, []any{"true"}, 1},
		{"null matches null", `null`, []any{nil}, 0},
		{"string is not null", `"null"`, []any{nil}, 1},
		{"array strings are exact", `" a "`, []any{"a"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := utils.FindNodes([]byte("value: "+tc.input), "$.value")
			require.NoError(t, err)
			opts := map[string]any{"values": tc.allowed}
			rule := buildCoreTestRule("$.value", model.SeverityError, "enumeration", "", opts)
			ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
			ctx.Given = "$.value"
			results := (Enumeration{}).RunRule(nodes, ctx)
			require.Len(t, results, tc.findings)
		})
	}
}
