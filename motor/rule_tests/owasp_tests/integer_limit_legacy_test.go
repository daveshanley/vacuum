package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPIntegerLimitLegacy_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case: oas2",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: integer
    minimum: 1
    maximum: 99
`,
		},
		{
			name: "valid case: oas3.0",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      minimum: 1
      maximum: 99
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-integer-limit-legacy"] = rulesets.GetOWASPIntegerLimitLegacyRule() // TODO

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

func TestRuleSet_OWASPIntegerLimitLegacy_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
		n    int
	}{
		{
			name: "invalid case: oas2 missing maximum",
			n:    5, // TODO: Should be one (problem: if and else branching cause)
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: integer
`,
		},
		{
			name: "invalid case: oas3.0 missing maximum",
			n:    5, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
`,
		},
		{
			name: "invalid case: oas2 has maximum but missing minimum",
			n:    3, // TODO: Should be one (problem: if and else branching cause)
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: integer
    maximum: 99
`,
		},
		{
			name: "invalid case: oas3.0 has maximum but missing minimum",
			n:    3, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      maximum: 99
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-integer-limit-legacy"] = rulesets.GetOWASPIntegerLimitLegacyRule() // TODO

			rs := &rulesets.RuleSet{
				Rules: rules,
			}

			rse := &motor.RuleSetExecution{
				RuleSet: rs,
				Spec:    []byte(tt.yml),
			}
			results := motor.ApplyRulesToRuleSet(rse)
			assert.Len(t, results.Results, tt.n)
		})
	}
}
