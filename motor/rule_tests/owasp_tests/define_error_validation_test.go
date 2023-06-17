package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPDefineErrorValidation_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case: 400",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "400":
          description: "classic validation fail"`,
		},
		{
			name: "valid case: 422",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "422":
          description: "classic validation fail"`,
		},
		{
			name: "valid case: 4XX",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "4XX":
          description: "classic validation fail"`,
		},
	}

	for _, tt := range tc {
		t.Run("valid case: 400", func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-define-error-validation"] = rulesets.GetOWASPDefineErrorValidationRule() // TODO

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

func TestRuleSet_OWASPDefineErrorValidation_Error(t *testing.T) {

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

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-define-error-validation"] = rulesets.GetOWASPDefineErrorValidationRule() // TODO

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
