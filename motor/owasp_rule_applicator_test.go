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

func TestRuleSet_GetOWASPRuleDefineErrorValidationSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "4XX":
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

func TestRuleSet_GetOWASPRuleDefineErrorResponses401Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        401:
          description: "ok"
          content:
            "application/json":
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses401() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorResponses401Error(t *testing.T) {

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
            "application/problem+json":
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses401() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorResponses401ErrorMissing(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        401:
          description: "ok"
          invalid-content:
            "application/problem+json"
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses401() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorResponses500Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        500:
          description: "ok"
          content:
            "application/json":
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses500() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorResponses500Error(t *testing.T) {

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
            "application/problem+json":
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses500() // TODO

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

func TestRuleSet_GetOWASPRuleDefineErrorResponses500ErrorMissing(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        500:
          description: "ok"
          invalid-content:
            "application/problem+json"
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleDefineErrorResponses500() // TODO

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

func TestRuleSet_GetOWASPRuleRateLimitSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "401":
          description: "ok"
          headers:
            "RateLimit-Limit":
              schema:
                type: string
            "RateLimit-Reset":
              schema:
                type: string
        "201":
          description: "ok"
          headers:
            "X-RateLimit-Limit":
              schema:
                type: string
        "203":
          description: "ok"
          headers:
            "X-Rate-Limit-Limit":
              schema:
                type: string
        "301":
          description: "ok"
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleRateLimit() // TODO

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

func TestRuleSet_GetOWASPRuleRateLimitError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "401":
          description: "ok"
          headers:
            "RateLimit-Limit":
              schema:
                type: string
        "201":
          description: "ok"
          headers:
            "Wrong-RateLimit-Limit":
              schema:
                type: string
        "303":
          description: "ok"
          headers:
            "Wrong-Rate-Limit-Limit":
              schema:
                type: string
        "203":
          description: "ok"
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleRateLimit() // TODO

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 3)
}

func TestRuleSet_GetOWASPRuleRateLimitRetryAfterSuccess(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "429":
          description: "ok"
          headers:
            "Retry-After":
              description: "standard retry header"
              schema:
                type: string
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleRateLimitRetryAfter() // TODO

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

func TestRuleSet_GetOWASPRuleRateLimitRetryAfterError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        429:
          description: "ok"
          headers:
        200:
          description: "ok"
          headers:
            "Retry-After":
              description: "standard retry header"
              schema:
                type: string
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleRateLimitRetryAfter() // TODO

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

func TestRuleSet_GetOWASPRuleArrayLimitError(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        429:
          description: "ok"
          headers:
        200:
          description: "ok"
          headers:
            "Retry-After":
              description: "standard retry header"
              schema:
                type: string
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleArrayLimit() // TODO

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

func TestRuleSet_GetOWASPRuleNoAdditionalPropertiesValidOAS3Success(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: false
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleNoAdditionalProperties() // TODO

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

func TestRuleSet_GetOWASPRuleNoAdditionalPropertiesNoAdditionalPropertiesDefinedSuccess(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleNoAdditionalProperties() // TODO

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

func TestRuleSet_GetOWASPRuleNoAdditionalPropertiesAdditionalPropertiesDefinedError(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  version: "1.0"
components:
  schemas:
    Foo:
      type: object
      additionalProperties: true
`

	rules := make(map[string]*model.Rule)
	rules["here"] = rulesets.GetOWASPRuleNoAdditionalProperties() // TODO

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
