package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPProtectionGlobalSafe_Success(t *testing.T) {

	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
paths:
  /security-ko-missing:
    put:
      responses: {}
    post:
      security: []
  /security-ok-put:
    put:
      security:
        -  BasicAuth: []
      responses: {}
  /security-ok-get:
    get:
      security:
        - {}
      responses: {}
    head:
      security:
        - {}
        - BasicAuth: []
  /security-ko-info:
    post:
      security:
        - {}
        - BasicAuth: []
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-protection-global-safe"] = rulesets.GetOWASPProtectionGlobalSafeRule()

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

func TestRuleSet_OWASPProtectionGlobalSafe_Error(t *testing.T) {

	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
paths:
  /security-ko-get:
    get:
      responses: {}
    head:
      security: []
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-protection-global-safe"] = rulesets.GetOWASPProtectionGlobalSafeRule()

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
