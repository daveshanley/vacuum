package core

import (
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCasing_RunRule_PatternMatchSuccess(t *testing.T) {

	sampleYaml := `carpet: "abc"`
	path := "$.carpet"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "[abc]+"

	rule := buildCoreTestRule(path, severityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(rule.Then, rule)

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_PatternNothingSupplied(t *testing.T) {

	sampleYaml := `carpet: "abc"`
	path := "$.carpet"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, severityError, "pattern", "", nil)
	ctx := buildCoreTestContextFromRule(rule.Then, rule)

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_PatternNotMatchError(t *testing.T) {

	sampleYaml := `carpet: "nice-rice"`
	path := "$.carpet"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["notMatch"] = "[[abc)"

	rule := buildCoreTestRule(path, severityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(rule.Then, rule)

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_PatternMatchFail(t *testing.T) {

	sampleYaml := `carpet: "def"`
	path := "$.carpet"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "[abc]+"

	rule := buildCoreTestRule(path, severityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(rule.Then, rule)

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_PatternMatchError(t *testing.T) {

	sampleYaml := `carpet: "abc"`
	path := "$.carpet"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["match"] = "([abc]"

	rule := buildCoreTestRule(path, severityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(rule.Then, rule)

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_PatternNotMatchFail(t *testing.T) {

	sampleYaml := `pizza: "cat1"`
	path := "$.pizza"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["notMatch"] = `\w{3}\d`

	rule := buildCoreTestRule(path, severityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(rule.Then, rule)

	def := &Pattern{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
