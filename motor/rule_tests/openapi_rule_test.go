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
		motor.ApplyRules(rules, []byte(badDoc))
	}

}

func Test_Default_OpenAPIRuleSet_oasOpSuccessResponse(t *testing.T) {

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
	results, err := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), []byte(badDoc))
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 13)

	for n := 0; n < len(results); n++ {
		assert.NotNil(t, results[n].Path)
		assert.NotNil(t, results[n].StartNode)
		assert.NotNil(t, results[n].EndNode)
	}

}
