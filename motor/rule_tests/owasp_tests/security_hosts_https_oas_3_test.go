package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPSecurityHostsHttpsOAS3_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
servers:
  - url: https://api.example.com/
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-security-hosts-https-oas3"] = rulesets.GetOWASPSecurityHostsHttpsOAS3Rule() // TODO

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

func TestRuleSet_OWASPSecurityHostsHttpsOAS3_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "an invalid server.url using http",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
servers:
  - url: http://api.example.com/
`,
		},
		{
			name: "an invalid server using ftp",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
servers:
  - url: ftp://api.example.com/
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-security-hosts-https-oas3"] = rulesets.GetOWASPSecurityHostsHttpsOAS3Rule() // TODO

			rs := &rulesets.RuleSet{
				Rules: rules,
			}

			rse := &motor.RuleSetExecution{
				RuleSet: rs,
				Spec:    []byte(tt.yml),
			}
			results := motor.ApplyRulesToRuleSet(rse)
			assert.Len(t, results.Results, 1)
		})
	}
}
