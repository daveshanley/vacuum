package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleSecurityHostsHttpsOAS2_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: integer
paths:
  "/"
host:
  - example.com
schemes:
  - https
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["here"] = rulesets.GetOWASPRuleSecurityHostsHttpsOAS2() // TODO

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

func TestRuleSet_GetOWASPRuleSecurityHostsHttpsOAS2_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "an invalid server.url using http",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: integer
paths:
  "/"
host:
  - example.com
schemes:
  - http
`,
		},
		{
			name: "an invalid server.url using http and https",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: integer
paths:
  "/"
host:
  - example.com
schemes: [https, http]
`,
		},
		{
			name: "an invalid server using ftp",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: integer
paths:
  "/"
host:
  - example.com
schemes: [ftp]
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["here"] = rulesets.GetOWASPRuleSecurityHostsHttpsOAS2() // TODO

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
