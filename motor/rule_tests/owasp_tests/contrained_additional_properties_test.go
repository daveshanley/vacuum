package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPConstrainedAdditionalProperties_Success(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: indeterminate
	  maxProperties: 1
`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-constrained-additionalProperties"] = rulesets.GetOWASPConstrainedAdditionalPropertiesRule()

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 0)
	})
}

func TestRuleSet_OWASPConstrainedAdditionalProperties_Error(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: indeterminate
`

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-constrained-additionalProperties"] = rulesets.GetOWASPConstrainedAdditionalPropertiesRule()

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 1)
		assert.Equal(t, "build schema failed: unexpected data type: 'string', line 8, col 29", results.Results[0].Message)
		assert.Equal(t, "$.components.schemas['Foo']", results.Results[0].Path)

	})
}

func TestRuleSet_OWASPConstrainedAdditionalProperties_Error_NoBuildFail(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: true
`

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-constrained-additionalProperties"] = rulesets.GetOWASPConstrainedAdditionalPropertiesRule()

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 1)
		assert.Equal(t, "schema should also define `maxProperties` when `additionalProperties` is an object", results.Results[0].Message)
		assert.Equal(t, "$.components.schemas['Foo']", results.Results[0].Path)

	})
}
