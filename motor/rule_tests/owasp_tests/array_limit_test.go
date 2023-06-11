package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleArrayLimit_Success(t *testing.T) {

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
    type: array
    maxItems: 99
`,
		},
		{
			name: "valid case: oas3",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: array
      maxItems: 99
`,
		},
		{
			name: "valid case: oas3.1",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    type:
      type: string
      maxLength: 99
    User:
      type: object
      properties:
        type:
          enum: ['user', 'admin']
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["here"] = rulesets.GetOWASPRuleArrayLimit() // TODO

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

func TestRuleSet_GetOWASPRuleArrayLimit_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
		n    int
	}{
		{
			name: "invalid case: oas2 missing maxItems",
			n:    3, // TODO: Should be one (problem: if and else branching cause)
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: array
`,
		},
		{
			name: "invalid case: oas3 missing maxItems",
			n:    3, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: array
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["here"] = rulesets.GetOWASPRuleArrayLimit() // TODO

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
