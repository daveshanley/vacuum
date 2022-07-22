package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestFormDataConsumeCheck_GetSchema(t *testing.T) {
	def := FormDataConsumeCheck{}
	assert.Equal(t, "formData_consume_check", def.GetSchema().Name)
}

func TestFormDataConsumeCheck_RunRule(t *testing.T) {
	def := FormDataConsumeCheck{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestFormDataConsumeCheck_RunRule_SuccessCheck(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /survey:
    post:
      consumes:
        - application/x-www-form-urlencoded
      parameters:
        - in: formData
          name: name
          type: string
          description: A person's name.
        - in: formData
          name: fav_number
          type: number
          description: A person's favorite number.`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "formData_consume_check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := FormDataConsumeCheck{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)

}

func TestFormDataConsumeCheck_RunRule_SuccessCheck_MultipleConsumes(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /survey:
    post:
      consumes:
        - application/x-www-form-urlencoded
        - multipart/form-data
      parameters:
        - in: formData
          name: name
          type: string
          description: A person's name.
        - in: formData
          name: fav_number
          type: number
          description: A person's favorite number.`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "formData_consume_check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := FormDataConsumeCheck{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)

}

func TestFormDataConsumeCheck_RunRule_TopSuccessCheck(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /survey:
    parameters:
      - in: formData
        name: name
        type: string
        description: A person's name.
      - in: formData
        name: fav_number
        type: number
        description: A person's favorite number
    post:
      consumes:
        - application/x-www-form-urlencoded`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "formData_consume_check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := FormDataConsumeCheck{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)

}

func TestFormDataConsumeCheck_RunRule_TopFailCheck(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /survey:
    parameters:
      - in: formData
        name: name
        type: string
        description: A person's name.
      - in: formData
        name: fav_number
        type: number
        description: A person's favorite number
    post:
      consumes:
        - chicken-soup/and-cake`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "formData_consume_check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := FormDataConsumeCheck{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 2)

}

func TestFormDataConsumeCheck_RunRule_FailCheck(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /survey:
    post:
      consumes:
        - pizza/fish
      parameters:
        - in: formData
          name: name
          type: string
          description: A person's name.
        - in: formData
          name: fav_number
          type: number
          description: A person's favorite number.`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "formData_consume_check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := FormDataConsumeCheck{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 2)

}

func TestFormDataConsumeCheck_RunRule_FailNoConsumesCheck(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /survey:
    post:
      parameters:
        - in: formData
          name: name
          type: string
          description: A person's name.
        - in: formData
          name: fav_number
          type: number
          description: A person's favorite number.`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "formData_consume_check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := FormDataConsumeCheck{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 4)

}
