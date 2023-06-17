package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_OWASPStringRestricted_Success(t *testing.T) {

	tc := []struct {
		name string
		yml  string
	}{
		{
			name: "valid case: format (oas2)",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: string
    format: email`,
		},
		{
			name: "valid case: format (oas2)",
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: string
    pattern: "/^foo/"`,
		},
		{
			name: "valid case: format (oas3)",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string
      format: email`,
		},
		{
			name: "valid case: pattern (oas3)",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string
      pattern: "/^foo/"`,
		},
		{
			name: "valid case: format (oas3.1)",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
      format: email`,
		},
		{
			name: "valid case: pattern (oas3.1)",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
      pattern: "/^foo/"`,
		},
		{
			name: "valid case: enum (oas3)",
			yml: `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string
      enum: ["a", "b", "c"]`,
		},
		{
			name: "valid case: format + pattern (oas3.1)",
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string
      format: hex
      pattern: "^[0-9a-fA-F]+$"
      maxLength: 16`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-string-restricted"] = rulesets.GetOWASPStringRestrictedRule()

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

func TestRuleSet_OWASPStringRestricted_Error(t *testing.T) {

	tc := []struct {
		name string
		n    int
		yml  string
	}{
		{
			name: "invalid case: neither format or pattern (oas2)",
			n:    6, // TODO: Should be one (problem: if and else branching cause)
			yml: `swagger: "2.0"
info:
  version: "1.0"
definitions:
  Foo:
    type: string
`,
		},
		{
			name: "invalid case: neither format or pattern (oas3)",
			n:    14, // TODO: Should be one (problem: if and else branching cause)
			yml: `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: [null, string]
    Bar:
      type: string
`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rules := make(map[string]*model.Rule)
			rules["owasp-string-restricted"] = rulesets.GetOWASPStringRestrictedRule()

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
