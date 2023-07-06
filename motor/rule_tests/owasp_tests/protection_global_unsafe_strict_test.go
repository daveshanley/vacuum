package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPProtectionGlobalUnsafeStrict_Success(t *testing.T) {

	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
security:
  - BasicAuth: []
paths:
  /security-gloabl-ok-put:
    put:
      responses: {}
  /security-ok-put:
    put:
      security:
        -  BasicAuth: []
      responses: {}
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-protection-global-unsafe-strict"] = rulesets.GetOWASPProtectionGlobalUnsafeStrictRule()

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

func TestRuleSet_OWASPProtectionGlobalUnsafeStrict_Error(t *testing.T) {

	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
security:
  - BasicAuth: []
paths:
  /security-ko-patch-noauth:
    patch:
      security:
        - {}
      responses: {}
  /security-ko-post-noauth:
    patch:
      security:
        - BasicAuth: []
        - {}
      responses: {}
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-protection-global-unsafe-strict"] = rulesets.GetOWASPProtectionGlobalUnsafeStrictRule()

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
