package motor

import (
	"fmt"
	"sync"
	"testing"

	"github.com/daveshanley/vacuum/rulesets"
)

func TestConcurrentRuleExecution(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
  /another:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TestSchema'
      responses:
        '201':
          description: Created
components:
  schemas:
    TestSchema:
      type: object
      properties:
        invalid_snake_case:
          type: string
        anotherInvalid_field:
          type: integer
    invalidCamelCase:
      type: object
      properties:
        bad_naming:
          type: string`

	// Share the same ruleset across goroutines to trigger race condition
	rs := rulesets.BuildDefaultRuleSets()
	sharedRuleSet := rs.GenerateOpenAPIDefaultRuleSet()

	const numGoroutines = 1000
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			// Add some variation to increase race potential
			specVariation := spec + fmt.Sprintf(`
    Schema%d:
      type: object
      properties:
        field_%d:
          type: string`, id, id)

			ApplyRulesToRuleSet(&RuleSetExecution{
				RuleSet:      sharedRuleSet, // Same ruleset shared across goroutines
				Spec:         []byte(specVariation),
				SpecFileName: fmt.Sprintf("test%d.yaml", id),
			})
		}(i)
	}

	wg.Wait()
}
