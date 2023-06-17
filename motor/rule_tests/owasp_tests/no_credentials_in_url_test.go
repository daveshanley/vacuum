package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPNoCredentialsInURL_Success(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: id
          in: path
          required: true
        - name: filter
          in: query
          required: true`

	t.Run("invalid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-no-credentials-in-url"] = rulesets.GetOWASPNoCredentialsInURLRule() // TODO

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

func TestRuleSet_OWASPNoCredentialsInURL_Error(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: client_secret
          in: query
          required: true
        - name: token
          in: query
          required: true
        - name: refresh_token
          in: query
          required: true
        - name: id_token
          in: query
          required: true
        - name: password
          in: query
          required: true
        - name: secret
          in: query
          required: true
        - name: apikey
          in: query
          required: true
        - name: apikey
          in: path
          required: true
        - name: API-KEY
          in: query
          required: true`

	t.Run("valid case", func(t *testing.T) {
		rules := make(map[string]*model.Rule)
		rules["owasp-no-credentials-in-url"] = rulesets.GetOWASPNoCredentialsInURLRule() // TODO

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 9)
	})
}
