package rule_tests

import (
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/motor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Benchmark_DefaultOpenAPI(b *testing.B) {
	badDoc := ` paths:
  /curry/{hurry}/{salsa}:
    get:
      tags:
        - rice
      parameters:
      - in: path
        name: hurry
      - in: query
        name: hurry  
      responses:
      "500":
        description: no curry!
  /curry/{chips}/{cheese}:
    get:
      parameters:
      - in: path
        name: hurry`

	rs := functions.BuildDefaultRuleSets()
	rules := rs.GenerateOpenAPIDefaultRuleSet()
	for n := 0; n < b.N; n++ {
		motor.ApplyRules(rules, []byte(badDoc))
	}

}

func Test_Default_OpenAPIRuleSet_oasOpSuccessResponse(t *testing.T) {

	badDoc := ` paths:
  /curry/{hurry}/{salsa}:
    get:
      tags:
        - rice
      parameters:
      - in: path
        name: hurry
      - in: query
        name: hurry  
      responses:
      "500":
        description: no curry!
  /curry/{chips}/{cheese}:
    get:
      parameters:
      - in: path
        name: hurry`

	rs := functions.BuildDefaultRuleSets()
	results, err := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), []byte(badDoc))
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 10)

}
