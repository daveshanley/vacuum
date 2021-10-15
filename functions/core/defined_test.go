package core

import (
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefined_RunRule_Success(t *testing.T) {

	sampleYaml := `pizza:
  cake: "fridge"`

	path := "$.pizza"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, severityError, "defined", "cake", nil)
	ctx := buildCoreTestContext(rule.Then, nil)

	def := Defined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestDefined_RunRule_Fail(t *testing.T) {

	sampleYaml := `pizza:
  noCake: "noFun"`

	path := "$.pizza"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, severityError, "defined", "cake", nil)
	ctx := buildCoreTestContext(rule.Then, nil)

	def := Defined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
