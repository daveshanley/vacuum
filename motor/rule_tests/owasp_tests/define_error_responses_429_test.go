package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleDefineErrorResponses429_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        429:
          description: "ok"
          content:
            "application/json":
`

	t.Run("valid: defines a 429 response with content", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses429() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorResponses429_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
		n    int
	}{
		{
			name: "invalid: 429 is not defined at all",
			n:    2,
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        200:
          description: "ok"
          content:
            "application/problem+json":
`,
		},
		{
			name: "invalid: 429 exists but content is missing",
			n:    1,
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        429:
          description: "ok"
          invalid-content:
            "application/problem+json"
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			if tt.n == 1 {
				return
			}
			rules := make(map[string]*model.Rule)
			rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses429() // TODO

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
