package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_TestGetOWASPRuleProtectionGlobalUnsafe_Success(t *testing.T) {

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
        - BasicAuth: []
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
  /security-ko-get:
    get:
      responses: {}
    head:
      security: []
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
		rules["here"] = rulesets.GetOWASPRuleProtectionGlobalUnsafe()

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

func TestRuleSet_TestGetOWASPRuleProtectionGlobalUnsafe_Error(t *testing.T) {

	// TODO here

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: id
          in: path
          schema:
            type: integer
        - name: notanid
          in: path
          schema:
            type: integer
        - name: underscore_id
          in: path
          schema:
            type: integer
        - name: hyphen-id
          in: path
          schema:
            type: integer
            format: int32
        - name: camelId
          in: path
          schema:
            type: integer`

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleProtectionGlobalUnsafe()

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 5)
	})
}
