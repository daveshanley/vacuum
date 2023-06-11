package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleIntegerLimit_Success(t *testing.T) {

	yml1 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      minimum: 1
      maximum: 99
`

	for _, yml := range []string{yml1} {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleIntegerLimit() // TODO

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

func TestRuleSet_GetOWASPRuleIntegerLimit_Error(t *testing.T) {

	yml1 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
`

	for _, yml := range []string{yml1} {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleIntegerLimit() // TODO

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
