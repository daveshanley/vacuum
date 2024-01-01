package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
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

	yml := `openapi: 3.1
components:
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

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "component-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ComponentDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 4)
	assert.Equal(t, "`schemas` component `Minty` is missing a description", res[0].Message)
	assert.Equal(t, "`schemas` component `Mouse` description must be at least `10` words long", res[1].Message)

}
