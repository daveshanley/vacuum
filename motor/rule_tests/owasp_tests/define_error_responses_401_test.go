package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleDefineErrorResponses401_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        401:
          description: "ok"
          content:
            "application/json":
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses401() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorResponses401_Error(t *testing.T) {

	tc := []struct {
		yml string
		n   int
	}{
		{
			n: 2,
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
			n: 1,
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        401:
          description: "ok"
          invalid-content:
            "application/problem+json"
`,
		},
	}

	for _, tt := range tc {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses401() // TODO

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(tt.yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, tt.n)
	}
}
