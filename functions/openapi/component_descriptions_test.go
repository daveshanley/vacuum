package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComponentDescription_GetSchema(t *testing.T) {
	def := ComponentDescription{}
	assert.Equal(t, "component_description", def.GetSchema().Name)
}

func TestComponentDescription_RunRule(t *testing.T) {
	def := ComponentDescription{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestComponentDescription_CheckDescriptionMissing(t *testing.T) {

	yml := `components:
  schemas:
    Minty:
      type: object
    Mouse:
      type: object
      description: There is a description here, not long enough.
  responses:
    Chippy:
      content:
        boo: yeah      
    Choppy:
      description: this should be long enough to pass the test I think
  requestBodies:
    Puppy:
      description: this is not long enough
      content:
    Kitty:
      description: this is long enough, so we should not see any errors here in  `

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "component-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := ComponentDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)

}
