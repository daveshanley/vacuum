package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleNoAdditionalProperties_Success(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleNoAdditionalProperties() // TODO

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

func TestRuleSet_GetOWASPRuleNoAdditionalProperties_Error(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: true
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleNoAdditionalProperties() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &motor.RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := motor.ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 1)
}