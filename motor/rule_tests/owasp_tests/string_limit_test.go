package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleStringLimit_Success(t *testing.T) {

	yml1 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
      maxLength: 16
`
	yml2 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: "string"
      enum:
        - 1
        - 2
        - 3
`
	yml3 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: "string"
      const: 1
`
	yml4 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: "string"
      maxLength: 5
`
	yml5 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: "stringer"
`
	yml6 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["stringer", "onestringto", "somestring", "String"]
    Bar:
      example: okay
      type: stringer
  type: integer
`

	for _, yml := range []string{yml1, yml2, yml3, yml4, yml5, yml6} {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleStringLimit() // TODO

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.Len(t, results.Results, 0)
	}
}

func TestRuleSet_GetOWASPRuleStringLimit_Error(t *testing.T) {

	yml1 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: [integer, string, boolean]
`
	yml2 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string
    Bar:
      example: "bar"
      type: string
`

	for _, yml := range []string{yml1, yml2} {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleStringLimit() // TODO

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.NotEqualValues(t, len(results.Results), 0) // Should output an error and not five
	}
}
