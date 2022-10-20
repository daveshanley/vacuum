package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAlphabetical_RunRule_FailStringArray(t *testing.T) {

	sampleYaml := `mega:
 - apple
 - bee
 - andrew
 - cakes`

	path := "$.mega"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestAlphabetical_RunRule_PassStringArray(t *testing.T) {

	sampleYaml := `dinner:
 - chicken
 - nuggets
 - yummy`

	path := "$.dinner"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestAlphabetical_RunRule_FailIntegerArray(t *testing.T) {

	sampleYaml := "whippy:\n - 1\n - 2\n - 7\n - 3"

	path := "$.whippy"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestAlphabetical_RunRule_FailFloatArray(t *testing.T) {

	sampleYaml := "herbs:\n - 1.782\n - 2.9981\n - 2.8812\n - 3.98166239"

	path := "$.herbs"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestAlphabetical_RunRule_SuccessIntegerArray(t *testing.T) {

	sampleYaml := "puppy:\n - 8\n - 10\n - 120\n - 3000"

	path := "$.puppy"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestAlphabetical_RunRule_SuccessFloatArray(t *testing.T) {

	sampleYaml := "lemons:\n - 9.12345\n - 9.123456\n - 9.234567\n - 9.3456789"

	path := "$.lemons"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestAlphabetical_RunRule_IgnoreBooleanArray(t *testing.T) {

	sampleYaml := "grim:\n - 1.782\n - 2.9981\n - true\n - 1.1"

	path := "$.grim"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestAlphabetical_RunRule_ObjectFail(t *testing.T) {

	sampleYaml := `dinner:
 chicken:
  nuggets: nice
 chops:
  nuggets: breaded 
 pizza:
  nuggets: yuck`

	path := "$.dinner"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["keyedBy"] = "nuggets"

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestAlphabetical_RunRule_ObjectSuccess(t *testing.T) {

	sampleYaml := `places:
 mountains:
  heat: cold
 beach:
  heat: hot 
 desert:
  heat: very hot`

	path := "$.places"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["keyedBy"] = "heat"

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestAlphabetical_RunRule_ObjectIntegersSuccess(t *testing.T) {

	sampleYaml := `places:
 mountains:
  heat: 1
 beach:
  heat: 2
 desert:
  heat: 3`

	path := "$.places"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["keyedBy"] = "heat"

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestAlphabetical_RunRule_ObjectIntegersFail(t *testing.T) {

	sampleYaml := `places:
 mountains:
  heat: 1
 beach:
  heat: 7
 desert:
  heat: 2`

	path := "$.places"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["keyedBy"] = "heat"

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestAlphabetical_RunRule_ObjectFailNoKeyedBy(t *testing.T) {

	sampleYaml := `places:
 mountains:
  heat: cold
 beach:
  heat: hot 
 desert:
  heat: very hot`

	path := "$.places"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)

	rule := buildCoreTestRule(path, model.SeverityError, "alphabetical", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Alphabetical{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
