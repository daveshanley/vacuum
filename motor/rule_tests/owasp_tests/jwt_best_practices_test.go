package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPJWTBestPractices_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad oauth2":
      type: "http"
      description: "These JWTs use RFC8725."
    "bad bearer jwt":
      type: "http"
      bearerFormat: "jwt"
      description: "These JWTs use RFC8725."`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-jwt-best-practices"] = rulesets.GetOWASPJWTBestPracticesRule()

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

func TestRuleSet_OWASPJWTBestPractices_Error(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad oauth2":
      type: "oauth2"
      description: "No way of knowing if these JWTs are following best practices."
    "bad bearer jwt":
      type: "http"
      bearerFormat: "jwt"
      description: "No way of knowing if these JWTs are following best practices."`

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-jwt-best-practices"] = rulesets.GetOWASPJWTBestPracticesRule()

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
