package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPattern_GetSchema(t *testing.T) {
	def := Pattern{}
	assert.Equal(t, "pattern", def.GetSchema().Name)
}

func TestPattern_RunRule(t *testing.T) {
	def := Pattern{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPattern_RunRule_PatternMatchSuccess(t *testing.T) {

	sampleYaml := `carpet: "abc"`
	path := "$.carpet"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "[abc]+"

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPattern_RunRule_PatternNothingSupplied(t *testing.T) {

	sampleYaml := `carpet: "abc"`
	path := "$.carpet"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "", nil)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPattern_RunRule_PatternNotMatchError(t *testing.T) {

	sampleYaml := `carpet: "nice-rice"`
	path := "$"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["notMatch"] = "[[abc)"

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "carpet", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestPattern_RunRule_PatternMatchFail(t *testing.T) {

	sampleYaml := `carpet: "def"`
	path := "$"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "[abc]+"

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "carpet", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestPattern_RunRule_PatternMatchError(t *testing.T) {

	sampleYaml := `carpet: "abc"`
	path := "$"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "([abc]"

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "carpet", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestPattern_RunRule_PatternNotMatchFail(t *testing.T) {

	sampleYaml := `pizza: "cat1"`
	path := "$"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["notMatch"] = `\w{3}\d`

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "pizza", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestPattern_RunRule_UseFieldName(t *testing.T) {

	sampleYaml := `no: 
  sleep: until`
	path := "$.no"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "cake"

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "sleep", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestPattern_RunRule_ContainMap(t *testing.T) {

	sampleYaml := `no: 
  sleep: until`
	path := "$"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "until"

	rule := buildCoreTestRule(path, model.SeverityError, "pattern", "sleep", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}
