package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPIntegerLimit_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case: minimum and maximum",
			yml: `openapi: "3.1.0"
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
		{
			name: "valid case: exclusiveMinimum and exclusiveMaximum",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      exclusiveMinimum: 1
      exclusiveMaximum: 99
`,
		},
		{
			name: "valid case: minimum and exclusiveMaximum",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      minimum: 1
      exclusiveMaximum: 99
`,
		},
		{
			name: "valid case: exclusiveMinimum and maximum",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      exclusiveMinimum: 1
      maximum: 99
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-integer-limit"] = rulesets.GetOWASPIntegerLimitRule()

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

func TestRuleSet_OWASPIntegerLimit_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
		n    int
	}{
		{
			name: "invalid case: only maximum",
			n:    7, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      exclusiveMinimum: 99
      minimum: 99
`,
		},
		{
			name: "invalid case: only exclusiveMaximum",
			n:    6, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      exclusiveMaximum: 99
`,
		},
		{
			name: "invalid case: only maximum",
			n:    6, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      maximum: 99
`,
		},
		{
			name: "invalid case: only exclusiveMinimum",
			n:    6, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      exclusiveMinimum: 1
`,
		},
		{
			name: "invalid case: both minimums and an exclusiveMaximum",
			n:    3, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      minimum: 1
      exclusiveMinimum: 1
      exclusiveMaximum: 4
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-integer-limit"] = rulesets.GetOWASPIntegerLimitRule()

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
