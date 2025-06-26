package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
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

func TestEnumeration_RunRule_Array_Success(t *testing.T) {
	sampleYaml := `christmas: "ham"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["values"] = []string{"turkey", "sprouts", "presents", "ham", ",,,,,,,"}

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
	opts["values"] = []string{"turkey", "sprouts", "presents", "ham", ",,,,,,,"}

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
