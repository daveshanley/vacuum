package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleRateLimit_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "401":
          description: "ok"
          headers:
            "RateLimit-Limit":
              schema:
                type: string
            "RateLimit-Reset":
              schema:
                type: string
        "201":
          description: "ok"
          headers:
            "X-RateLimit-Limit":
              schema:
                type: string
        "203":
          description: "ok"
          headers:
            "X-Rate-Limit-Limit":
              schema:
                type: string
        "301":
          description: "ok"
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleRateLimit() // TODO

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

func TestRuleSet_GetOWASPRuleRateLimit_Error(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "401":
          description: "ok"
          headers:
            "RateLimit-Limit":
              schema:
                type: string
        "201":
          description: "ok"
          headers:
            "Wrong-RateLimit-Limit":
              schema:
                type: string
        "303":
          description: "ok"
          headers:
            "Wrong-Rate-Limit-Limit":
              schema:
                type: string
        "203":
          description: "ok"
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleRateLimit() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &motor.RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := motor.ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 3)
}
