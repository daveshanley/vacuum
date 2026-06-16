package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
)

func TestRequiredFieldsDefined_GetSchema(t *testing.T) {
	def := RequiredFieldsDefined{}
	assert.Equal(t, "requiredFieldsDefined", def.GetSchema().Name)
}

func TestRequiredFieldsDefined_ReportsMissingPropertyDefinitions(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Gum:
      type: object
      required:
        - hello
      properties:
        goodbye:
          type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "requiredFieldsDefined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := RequiredFieldsDefined{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`required` field `hello` is not defined in `properties`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].required[0]", res[0].Path)
}

func TestRequiredFieldsDefined_NoResultsWithoutDoctorDocument(t *testing.T) {
	def := RequiredFieldsDefined{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}
