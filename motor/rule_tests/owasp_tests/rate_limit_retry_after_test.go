package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPRateLimitRetryAfter_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "429":
          description: "ok"
          headers:
            "Retry-After":
              description: "standard retry header"
              schema:
                type: string
`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-rate-limit-retry-after"] = rulesets.GetOWASPRateLimitRetryAfterRule() // TODO

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

func TestRuleSet_OWASPRateLimitRetryAfter_Error(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        429:
          description: "ok"
          headers:
        200:
          description: "ok"
          headers:
            "Retry-After":
              description: "standard retry header"
              schema:
                type: string
`

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-rate-limit-retry-after"] = rulesets.GetOWASPRateLimitRetryAfterRule() // TODO

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
