package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleArrayLimit_Success(t *testing.T) {

	yml1 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: array
      maxItems: 99
`

	yml2 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    type:
      type: string
      maxLength: 99
    User:
      type: object
      properties:
        type:
          enum: ['user', 'admin']
`

	for _, yml := range []string{yml1, yml2} {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleArrayLimit() // TODO

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 0)
	}
}

func TestRuleSet_GetOWASPRuleArrayLimit_Error(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: array
`
	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleArrayLimit() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &motor.RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := motor.ApplyRulesToRuleSet(rse)
	assert.NotEqualValues(t, len(results.Results), 0) // Should output an error and not three
}
