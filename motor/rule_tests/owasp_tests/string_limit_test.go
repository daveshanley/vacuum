package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPStringLimit_Success(t *testing.T) {

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
    type: string
    maxLength: 99`,
		},
		{
			name: "valid case: oas3.0",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string
      maxLength: 99`,
		},
		{
			name: "valid case: oas3.1",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
      maxLength: 99`,
		},
		{
			name: "valid case: oas3.0",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: "string"
      enum: [a, b, c]`,
		},
		{
			name: "valid case: oas3.1",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: "string"
      const: "constant"`,
		},
		{
			name: "valid case: pattern and maxLength, oas3.1",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: "string"
      format: "hex"
      pattern: "^[0-9a-fA-F]+$"
      maxLength: 10`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-string-limit"] = rulesets.GetOWASPStringLimitRule()

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

func TestRuleSet_OWASPStringLimit_Error(t *testing.T) {

	tc := []struct {
		name string
		n    int
		yml  string
	}{
		{
			name: "invalid case: oas2 missing maxLength",
			n:    5, // TODO: Should be one (problem: if and else branching cause)
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: string`,
		},
		{
			name: "invalid case: oas3.0 missing maxLength",
			n:    5, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string`,
		},
		{
			name: "invalid case: oas3.1 missing maxLength",
			n:    7, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: [null, string]`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-string-limit"] = rulesets.GetOWASPStringLimitRule()

			rs := &rulesets.RuleSet{
				Rules: rules,
			}

			rse := &motor.RuleSetExecution{
				RuleSet: rs,
				Spec:    []byte(tt.yml),
			}
			results := motor.ApplyRulesToRuleSet(rse)
			assert.Len(t, results.Results, tt.n) // Should output an error and not five
		})
	}
}
