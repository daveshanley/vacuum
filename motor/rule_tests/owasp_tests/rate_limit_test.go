package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleRateLimit_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid use of IETF Draft HTTP RateLimit Headers",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "201":
          description: "ok"
          headers:
            "X-RateLimit-Limit":
              schema:
                type: string
            "X-RateLimit-Reset":
              schema:
                type: string`,
		},
		{
			name: "valid use of Twitter-style Rate Limit Headers",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "201":
          description: "ok"
          headers:
            "X-Rate-Limit-Limit":
              schema:
                type: string`,
		},
		{
			name: "valid use of GitHub-style Rate Limit Headers",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "201":
          description: "ok"
          headers:
            "X-RateLimit-Limit":
              schema:
                type: string`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["here"] = rulesets.GetOWASPRuleRateLimit() // TODO

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

func TestRuleSet_GetOWASPRuleRateLimit_Error(t *testing.T) {

	tc := []struct {
		name string
		n    int
		yml  string
	}{
		{
			name: "invalid case: no limit headers set",
			n:    1,
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      description: "get"
      responses:
        "201":
          description: "ok"
          headers:
            "SomethingElse":
              schema:
                type: string
`,
		},
		{
			name: "invalid case: no rate limit headers set",
			n:    1,
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "201":
          description: "ok"
          headers:
            "Wrong-RateLimit-Limit":
              schema:
                type: string
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["here"] = rulesets.GetOWASPRuleRateLimit() // TODO

			rs := &rulesets.RuleSet{
				Rules: rules,
			}

			rse := &motor.RuleSetExecution{
				RuleSet: rs,
				Spec:    []byte(tt.yml),
			}
			results := motor.ApplyRulesToRuleSet(rse)
			assert.Len(t, results.Results, tt.n)
		})
	}
}
