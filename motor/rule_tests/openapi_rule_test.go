package rule_tests

import (
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Benchmark_DefaultOpenAPI(b *testing.B) {
	badDoc := `paths:
  /curry/{hurry}/{salsa}:
    get:
      tags:
        - rice
        - ice
        - fresh
      parameters:
      - in: path
        name: hurry
      - in: query
        name: hurry  
      responses:
      "500":
        description: no curry!
    post:
      description: can I get a curry?    
  /curry/{chips}/{cheese}:
    get:
      parameters:
      - in: path
        name: hurry`

	rs := rulesets.BuildDefaultRuleSets()
	rules := rs.GenerateOpenAPIDefaultRuleSet()
	for n := 0; n < b.N; n++ {
		er := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{RuleSet: rules, Spec: []byte(badDoc)})
		if er.Errors != nil {
			continue // we don't care but the linter does.
		}
	}

}

func Test_Default_OpenAPIRuleSet_FireABunchOfIssues(t *testing.T) {

	badDoc := `openapi: 3.0.3
paths:
  /curry/{hurry}/{salsa}:
    get:
      tags:
        - rice
        - ice
        - fresh
      parameters:
      - in: path
        name: hurry
      - in: path
        name: hurry  
      responses:
        "500":
          description:can I get a curry?
    post:
      description: can I get a curry?    
  /curry/{chips}/{cheese}:
    get:
      parameters:
      - in: path
        name: hurry`

	rs := rulesets.BuildDefaultRuleSets()
	rules := rs.GenerateOpenAPIDefaultRuleSet()
	lintExecution := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{RuleSet: rules, Spec: []byte(badDoc)})
	assert.Len(t, lintExecution.Errors, 0)
	assert.GreaterOrEqual(t, len(lintExecution.Results), 41)

	for n := 0; n < len(lintExecution.Results); n++ {
		assert.NotNil(t, lintExecution.Results[n].Path)
		assert.NotNil(t, lintExecution.Results[n].StartNode)
		assert.NotNil(t, lintExecution.Results[n].EndNode)
	}
}
