package rule_tests

import (
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/motor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Default_OpenAPIRuleSet_oasOpSuccessResponse(t *testing.T) {

	badDoc := `paths:
  /curry:
    get:
      responses:
      "500":
        description: no curry!`

	rs := functions.BuildDefaultRuleSets()
	results, err := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), []byte(badDoc))
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 1)

}
