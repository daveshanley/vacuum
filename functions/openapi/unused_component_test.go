package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnusedComponent_GetSchema(t *testing.T) {
	def := UnusedComponent{}
	assert.Equal(t, "unused_component", def.GetSchema().Name)
}

func TestUnusedComponent_RunRule(t *testing.T) {
	def := UnusedComponent{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestUnusedComponent_RunRule_Success(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - $ref: '#/components/parameters/Chewy'
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Puppy'
components:
  schemas:
    Puppy:
      type: string
      description: pup
  parameters:
    Chewy:
      description: chewy
      in: query
      name: chewy`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unused_component", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := UnusedComponent{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestUnusedComponent_RunRule_Success_Fail_TwoMissing_Two_Undefined(t *testing.T) {

	yml := `parameters:
  Chewy:
    description: chewy
    in: query
    name: chewy
paths:
  /naughty/{puppy}:
    parameters:
      - $ref: '#/parameters/Nothing'
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Cupcakes_And_Sugar'
components:
  schemas:
    Puppy:
      type: string
      description: pup
    Kitty:
      $ref: '#/components/schemas/Puppy' `

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unused_component", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := UnusedComponent{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)
}

func TestUnusedComponent_RunRule_Success_Fail_Four_Undefined(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - $ref: '#/components/parameters/Chewy'
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Puppy'
components:
  responses:
    Chappy:
      type: string
      description: Chappy
  schemas:  
    Chippy:
      type: string
      description: chippy
    Puppy:
      type: string
      description: pup
    Kitty:
      $ref: '#/components/schemas/Puppy'
  parameters:
    Minty:
      description: minty
      in: header
      name: minty
    Chewy:
      description: chewy
      in: query
      name: chewy`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unused_component", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := UnusedComponent{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)
}
