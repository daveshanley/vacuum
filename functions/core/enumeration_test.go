package core

import (
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnumeration_RunRule_Success(t *testing.T) {
	sampleYaml := `christmas: "ham"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["values"] = "turkey, sprouts, presents, ham"

	rule := buildCoreTestRule(path, severityError, "pattern", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestEnumeration_RunRule_Fail(t *testing.T) {
	sampleYaml := `christmas: "arguments"`
	path := "$.christmas"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]string)
	opts["values"] = "turkey, sprouts, presents, ham"

	rule := buildCoreTestRule(path, severityError, "enumeration", "", opts)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)

	def := &Enumeration{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
