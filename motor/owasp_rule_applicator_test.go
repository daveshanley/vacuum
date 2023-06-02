package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
)

func TestRuleSet_TestGetOwaspAPIRuleNoNumericIDsSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: id
          in: path
          schema:
            type: string
            format: uuid`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOwaspAPIRuleNoNumericIDs()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_TestGetOwaspAPIRuleNoNumericIDsError(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: id
          in: path
          schema:
            type: integer
        - name: notanid
          in: path
          schema:
            type: integer
        - name: underscore_id
          in: path
          schema:
            type: integer
        - name: hyphen-id
          in: path
          schema:
            type: integer
            format: int32
        - name: camelId
          in: path
          schema:
            type: integer`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOwaspAPIRuleNoNumericIDs()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 5) // in spectral, this outputs 4 errors
}

func TestRuleSet_GetOWASPRuleSecuritySchemeUseHTTPBasicSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "anything-else":
      type: "http"
      scheme: "bearer"`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleSecuritySchemeUseHTTPBasic()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_GetOWASPRuleSecuritySchemeUseHTTPBasicError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad negotiate":
      type: "http"
      scheme: "negotiate"
    "please-hack-me":
      type: "http"
      scheme: basic`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleSecuritySchemeUseHTTPBasic()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 1)
}

func TestRuleSet_GetOWASPRuleNoAPIKeysInURLSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "API Key in URL":
      type: "APIKey"
      in: "header"`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleNoAPIKeysInURL()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_GetOWASPRuleNoAPIKeysInURLError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "API Key in Query":
      type: apiKey
      in: query
    "API Key in Path":
      type: apiKey
      in: path`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleNoAPIKeysInURL() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 2)
}

func TestRuleSet_GetOWASPRuleSecurityCredentialsDetectedSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: id
          in: path
          required: true
        - name: filter
          in: query
          required: true`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleSecurityCredentialsDetected() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_GetOWASPRuleSecurityCredentialsDetectedError(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: client_secret
          in: query
          required: true
        - name: token
          in: query
          required: true
        - name: refresh_token
          in: query
          required: true
        - name: id_token
          in: query
          required: true
        - name: password
          in: query
          required: true
        - name: secret
          in: query
          required: true
        - name: apikey
          in: query
          required: true
        - name: apikey
          in: path
          required: true
        - name: API-KEY
          in: query
          required: true`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleSecurityCredentialsDetected() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 9)
}

func TestRuleSet_GetOWASPRuleAuthInsecureSchemesSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bearer is ok":
      type: "http"
      scheme: "bearer"`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleAuthInsecureSchemes() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_GetOWASPRuleAuthInsecureSchemesError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad negotiate":
      type: "http"
      scheme: "negotiate"
    "bad negotiate":
      type: "http"
      scheme: "oauth"`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleAuthInsecureSchemes() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 2)
}

func TestRuleSet_GetOWASPRuleJWTBestPracticesSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad oauth2":
      type: "http"
      description: "These JWTs use RFC8725."
    "bad bearer jwt":
      type: "http"
      bearerFormat: "jwt"
      description: "These JWTs use RFC8725."`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleJWTBestPractices() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_GetOWASPRuleJWTBestPracticesError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad oauth2":
      type: "oauth2"
      description: "No way of knowing if these JWTs are following best practices."
    "bad bearer jwt":
      type: "http"
      bearerFormat: "jwt"
      description: "No way of knowing if these JWTs are following best practices."`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleJWTBestPractices() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 2)
}

// TODO: Not working as expected
func TestRuleSet_GetOWASPRuleDefineErrorValidationSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "400":
          description: "classic validation fail"`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorValidation() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_GetOWASPRuleDefineErrorValidationError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        200:
          description: "ok"
          content:
            "application/json":
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorValidation() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 1)
}
