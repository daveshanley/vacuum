package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPAuthInsecureSchemes_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bearer is ok":
      type: "http"
      scheme: "bearer"`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-auth-insecure-schemes"] = rulesets.GetOWASPAuthInsecureSchemesRule()

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

func TestRuleSet_OWASPAuthInsecureSchemes_Error(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad negotiate":
      type: "http"
      scheme: "negotiate"
    "bad negotiate 2":
      type: "http"
      scheme: "oauth"`

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-auth-insecure-schemes"] = rulesets.GetOWASPAuthInsecureSchemesRule()

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 2)
	})
}
