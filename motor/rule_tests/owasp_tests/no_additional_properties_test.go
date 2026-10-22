package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
)

func TestRuleSet_OWASPNoAdditionalProperties_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case: oas3",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Foo'
      responses: {'200': {description: OK}}
components:
  schemas:
    Foo:
      type: object
      additionalProperties: false
`,
		},
		{
			name: "valid case: bounded map",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Foo'
      responses: {'200': {description: OK}}
components:
  schemas:
    Foo:
      type: object
      additionalProperties: {type: string}
      maxProperties: 10
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-no-additionalProperties"] = rulesets.GetOWASPNoAdditionalPropertiesRule()

			rs := &rulesets.RuleSet{
				Rules: rules,
			}

			rse := &motor.RuleSetExecution{
				RuleSet: rs,
				Spec:    []byte(tt.yml),
			}
			results := motor.ApplyRulesToRuleSet(rse)
			assert.Len(t, results.Results, 0)
		})
	}
}

func TestRuleSet_OWASPNoAdditionalProperties_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "invalid case: additionalProperties set to true (oas3)",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Foo'
      responses: {'200': {description: OK}}
components:
  schemas:
    Foo:
      type: object
      additionalProperties: true
`,
		},
		{
			name: "invalid case: additionalProperties true with a count limit (oas3)",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Foo'
      responses: {'200': {description: OK}}
components:
  schemas:
    Foo:
      type: object
      additionalProperties: true
      maxProperties: 10
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-no-additionalProperties"] = rulesets.GetOWASPNoAdditionalPropertiesRule()

			rs := &rulesets.RuleSet{
				Rules: rules,
			}

			rse := &motor.RuleSetExecution{
				RuleSet: rs,
				Spec:    []byte(tt.yml),
			}
			results := motor.ApplyRulesToRuleSet(rse)
			assert.Len(t, results.Results, 1)
		})
	}
}
