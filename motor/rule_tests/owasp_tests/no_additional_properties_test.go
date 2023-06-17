package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPNoAdditionalProperties_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case: oas2 does not allow additionalProperties by default so dont worry about it",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: object
    additionalProperties: false
`,
		},
		{
			name: "valid case: oas3",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: false
`,
		},
		{
			name: "valid case: no additionalProperties defined",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
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

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: true
`

	t.Run("invalid case: additionalProperties set to true (oas3)", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-no-additionalProperties"] = rulesets.GetOWASPNoAdditionalPropertiesRule()

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 1)
	})
}
