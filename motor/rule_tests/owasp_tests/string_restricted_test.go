package tests

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_GetOWASPRuleStringRestricted_Success(t *testing.T) {

	yml1 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
	  const: bar
`
	yml2 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
	  format: uuid
`
	yml3 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
	  pattern: [0-9]+
`
	yml4 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
	  enum: [bar, pit]
`
	yml5 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: integer
`

	for _, yml := range []string{yml1, yml2, yml3, yml4, yml5} {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleStringRestricted() // TODO

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

func TestRuleSet_GetOWASPRuleStringRestricted_Error(t *testing.T) {

	yml1 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: string
`
	yml2 := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: ["null", "string"]
`

	for _, yml := range []string{yml1, yml2} {
		rules := make(map[string]*model.Rule)
		rules["here"] = rulesets.GetOWASPRuleStringRestricted() // TODO

		rs := &rulesets.RuleSet{
			Rules: rules,
		}

		rse := &motor.RuleSetExecution{
			RuleSet: rs,
			Spec:    []byte(yml),
		}
		results := motor.ApplyRulesToRuleSet(rse)
		assert.NotEqualValues(t, len(results.Results), 0) // Should output an error and not 8
	}
}
