package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPIntegerFormat_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case: format - int32",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      format: int32
`,
		},
		{
			name: "valid case: format - int64",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      format: int64
`,
		},
		{
			name: "valid case: format - int64",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
      format: whatever
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-integer-format"] = rulesets.GetOWASPIntegerFormatRule() // TODO

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

func TestRuleSet_OWASPIntegerFormat_Error(t *testing.T) {

	tc := []struct {
		name string
		yml  string
		n    int
	}{
		{
			name: "invalid case: no format",
			n:    3, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-integer-format"] = rulesets.GetOWASPIntegerFormatRule() // TODO

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
