package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleDefineErrorValidation_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "4XX":
          description: "classic validation fail"`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorValidation() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorValidation_Error(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        200:
          description: "ok"
          content:
            "application/json":
        401:
          description: "ok"
          content:
            "application/json":
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorValidation() // TODO

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
