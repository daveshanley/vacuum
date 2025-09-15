package rulesets

import (
	"bytes"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

var totalRules = 64
var totalOwaspRules = 23
var totalRecommendedRules = 52

func TestBuildDefaultRuleSets(t *testing.T) {

	rs := BuildDefaultRuleSets()
	assert.NotNil(t, rs.GenerateOpenAPIDefaultRuleSet())
	assert.Len(t, rs.GenerateOpenAPIDefaultRuleSet().Rules, totalRules)

}

func TestCreateRuleSetUsingJSON_Fail(t *testing.T) {

	// this is not going to work.
	json := `{ "pizza" : "cake" }`

	_, err := CreateRuleSetUsingJSON([]byte(json))
	assert.Error(t, err)

}

func TestCreateRuleSetUsingJSON_Success(t *testing.T) {

	// this should work.
	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "fish-cakes": {
      "description": "yummy sea food",
      "recommended": true,
      "type": "style",
      "given": "$.some.JSON.PATH",
      "then": {
        "field": "nextSteps",
        "function": "cookForTenMins"
      }
    }
  }
}
`
	rs, err := CreateRuleSetUsingJSON([]byte(json))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)

}

func TestRuleSet_GetExtendsValue_Single(t *testing.T) {

	yaml := `extends: vacuum:oas
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMins`

	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)
	assert.NotNil(t, rs.GetExtendsValue())
	assert.Equal(t, "vacuum:oas", rs.GetExtendsValue()["vacuum:oas"])

}

func TestRuleSet_GetExtendsValue_Multi(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - recommended
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMins`

	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)
	assert.NotNil(t, rs.GetExtendsValue())
	assert.Equal(t, "recommended", rs.GetExtendsValue()["vacuum:oas"])

}

func TestRuleSet_GetExtendsValue_Multi_Noflag(t *testing.T) {

	yaml := `extends:
  - vacuum:oas
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMins`

	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)
	assert.NotNil(t, rs.GetExtendsValue())
	assert.Equal(t, "vacuum:oas", rs.GetExtendsValue()["vacuum:oas"])
	assert.Equal(t, "vacuum:oas", rs.GetExtendsValue()["vacuum:oas"]) // idempotence state check.

}

func TestRuleSet_GetConfiguredRules_All(t *testing.T) {

	// read spec and parse to dashboard.
	rs := BuildDefaultRuleSets()
	ruleSet := rs.GenerateOpenAPIDefaultRuleSet()
	assert.Len(t, ruleSet.Rules, totalRules)

	ruleSet = rs.GenerateOpenAPIRecommendedRuleSet()
	assert.Len(t, ruleSet.Rules, totalRecommendedRules)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Rec_OverrideNotFound(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - recommended
rules:
 soda-pop: "off"`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, totalRecommendedRules)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_OverrideNotFound(t *testing.T) {

	yaml := fmt.Sprintf(`extends:
  -
    - vacuum:oas
    - off
rules:
 soda-pop: "%s"`, model.SeverityWarn)

	def := BuildDefaultRuleSets()
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 0)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_All_OverrideNotFound(t *testing.T) {

	yaml := fmt.Sprintf(`extends:
  -
    - vacuum:oas
    - all
rules:
 soda-pop: "%s"`, model.SeverityWarn)

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, totalRules)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Rec_RemoveRule(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - recommended
rules:
 operation-success-response: "off"`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, totalRecommendedRules-1)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Rec_SeverityInfo(t *testing.T) {

	yaml := fmt.Sprintf(`extends:
  -
    - vacuum:oas
    - recommended
rules:
 operation-success-response: "%s"`, model.SeverityHint)

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, totalRecommendedRules)
	assert.Equal(t, model.SeverityHint, override.Rules["operation-success-response"].Severity)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_EnableRules(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - off
rules:
 operation-success-response: true
 info-contact: true
 `

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 2)
	assert.Equal(t, model.SeverityWarn, override.Rules["operation-success-response"].Severity)
	assert.Equal(t, model.SeverityWarn, override.Rules["info-contact"].Severity)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_EnableRulesNotFound(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - off
rules:
 chewy-dinner: true
 burpy-baby: true
 `

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 0)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_All_NewRule(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - all
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMin`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	newrs := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, newrs.Rules, totalRules+1)
	assert.Equal(t, true, newrs.Rules["fish-cakes"].Recommended)
	assert.Equal(t, "yummy sea food", newrs.Rules["fish-cakes"].Description)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_All_NewRuleReplace(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - all
rules:
 info-contact:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMin`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, totalRules)
	assert.Equal(t, true, repl.Rules["info-contact"].Recommended)
	assert.Equal(t, "yummy sea food", repl.Rules["info-contact"].Description)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_CustomRule(t *testing.T) {

	yaml := `extends:
  -
    - vacuum:oas
    - all
rules:
 info-contact:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMin`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, totalRules)
	assert.Equal(t, true, repl.Rules["info-contact"].Recommended)
	assert.Equal(t, "yummy sea food", repl.Rules["info-contact"].Description)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_RuleCategory(t *testing.T) {

	yaml := `extends: [[vacuum:oas, off]]
rules:
  check-title-is-exactly-this:
    description: Check the title of the spec is exactly, 'this specific thing'
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern
      functionOptions:
        match: 'this specific thing'
    howToFix: Make sure the title matches 'this specific thing'
    category:
      id: schemas`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, 1)
	assert.Equal(t, "schemas", repl.Rules["check-title-is-exactly-this"].RuleCategory.Id)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Oas_SpectralOwasp(t *testing.T) {

	yaml := `extends: [[vacuum:oas, all], [spectral:owasp, all]]`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, totalOwaspRules+totalRules)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Oas_VacuumOwasp(t *testing.T) {

	yaml := `extends: [[vacuum:oas, all], [vacuum:owasp, all]]`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, totalOwaspRules+totalRules)

}

func TestGetAllBuiltInRules(t *testing.T) {
	assert.Len(t, GetAllBuiltInRules(), totalRules)
}

func TestCreateRuleSetFromRuleMap(t *testing.T) {
	rules := GetAllBuiltInRules()
	rs := CreateRuleSetFromRuleMap(rules)
	assert.Len(t, rs.Rules, totalRules)
}

func TestRuleSet_GetExtendsRemoteSpec_Single(t *testing.T) {

	mockRemote := func() *httptest.Server {
		bs, _ := os.ReadFile("examples/custom-ruleset.yaml")
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write(bs)
		}))
	}

	server := mockRemote()
	defer server.Close()

	yaml := `extends: {{URL}}`

	yaml = strings.ReplaceAll(yaml, "{{URL}}", server.URL)

	def := BuildDefaultRuleSets()
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 1)
	assert.Len(t, override.RuleDefinitions, 1)

}

func TestRuleSet_GetExtendsRemoteSpec_Single_HttpError(t *testing.T) {

	yaml := `extends: http://kajshdkjahsdkajshdouaysoewuqyrkajshd.com`
	var logBuf []byte
	logBuffer := bytes.NewBuffer(logBuf)
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	def := BuildDefaultRuleSetsWithLogger(logger)
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 0)
	assert.Len(t, override.RuleDefinitions, 0)
	assert.Contains(t, logBuffer.String(), "cannot open external ruleset")

}

func TestRuleSet_GetExtendsLocalSpec_Single_HttpError(t *testing.T) {

	yaml := `extends: ./doesnotexist.yaml`

	var logBuf []byte
	logBuffer := bytes.NewBuffer(logBuf)
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	def := BuildDefaultRuleSetsWithLogger(logger)
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 0)
	assert.Len(t, override.RuleDefinitions, 0)
	assert.Contains(t, logBuffer.String(), "cannot open external ruleset")

}

func TestRuleSet_GetExtendsRemoteSpec_Multi(t *testing.T) {

	mockRemoteA := func() *httptest.Server {
		bs, _ := os.ReadFile("examples/custom-ruleset.yaml")
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write(bs)
		}))
	}

	mockRemoteB := func() *httptest.Server {
		bs, _ := os.ReadFile("examples/specific-ruleset.yaml")
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write(bs)
		}))
	}

	serverA := mockRemoteA()
	defer serverA.Close()

	serverB := mockRemoteB()
	defer serverB.Close()

	yaml := `extends: [{{URLA}}, {{URLB}}]`

	yaml = strings.ReplaceAll(yaml, "{{URLA}}", serverA.URL)
	yaml = strings.ReplaceAll(yaml, "{{URLB}}", serverB.URL)

	def := BuildDefaultRuleSets()
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 4)
	assert.Len(t, override.RuleDefinitions, 4)

}

func TestRuleSet_GetExtendsRemoteSpec_Chain(t *testing.T) {

	yamlA := `extends: [{{URLA}}]`
	yamlB := `extends: [{{URLB}}]
rules:
  ding:
    description: ding
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern
`
	yamlC := `extends: [[vacuum:oas, recommended]]
rules:
  dong:
    description: dong
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern`

	mockRemoteA := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(yamlB))
		}))
	}

	mockRemoteB := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(yamlC))
		}))
	}

	serverA := mockRemoteA()
	defer serverA.Close()

	serverB := mockRemoteB()
	defer serverB.Close()

	yamlA = strings.ReplaceAll(yamlA, "{{URLA}}", serverA.URL)
	yamlB = strings.ReplaceAll(yamlB, "{{URLB}}", serverB.URL)

	def := BuildDefaultRuleSets()
	rs, err := CreateRuleSetFromData([]byte(yamlA))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 54)
	assert.Len(t, override.RuleDefinitions, 2)
	assert.NotNil(t, rs.Rules["ding"])
	assert.NotNil(t, rs.Rules["dong"])
	assert.NotNil(t, rs.Rules["oas3-schema"])
}

func TestRuleSet_GetExtendsRemoteSpec_Chain_Loop(t *testing.T) {

	yamlA := `extends: [{{URLA}}]`
	yamlB := `extends: [{{URLB}}]
rules:
  ding:
    description: ding
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern
`
	yamlC := `extends: [{{URLA}}]
rules:
  dong:
    description: dong
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern`

	mockRemoteA := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(yamlB))
		}))
	}

	mockRemoteB := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(yamlC))
		}))
	}

	serverA := mockRemoteA()
	defer serverA.Close()

	serverB := mockRemoteB()
	defer serverB.Close()

	// loopy loo!
	yamlA = strings.ReplaceAll(yamlA, "{{URLA}}", serverA.URL)
	yamlB = strings.ReplaceAll(yamlB, "{{URLB}}", serverB.URL)
	yamlC = strings.ReplaceAll(yamlC, "{{URLA}}", serverA.URL)

	var logBuf []byte
	logBuffer := bytes.NewBuffer(logBuf)
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	def := BuildDefaultRuleSetsWithLogger(logger)
	rs, err := CreateRuleSetFromData([]byte(yamlA))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 2)
	assert.Len(t, override.RuleDefinitions, 2)
	assert.NotNil(t, rs.Rules["ding"])
	assert.NotNil(t, rs.Rules["dong"])
	assert.Contains(t, logBuffer.String(), "ruleset links to its self, circular rulesets are not permitted")
}

func TestRuleSet_GetExtendsRemoteSpec_Chain_Timeout(t *testing.T) {

	yamlA := `extends: [{{URLA}}]`
	yamlB := `extends: [{{URLB}}]
rules:
  ding:
    description: ding
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern
`
	yamlC := `extends: [[vacuum:oas, recommended]]
rules:
  dong:
    description: dong
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern`

	mockRemoteA := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(yamlB))
		}))
	}

	mockRemoteB := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			time.Sleep(time.Second * 6)
			_, _ = rw.Write([]byte(yamlC))
		}))
	}

	serverA := mockRemoteA()
	defer serverA.Close()

	serverB := mockRemoteB()
	defer serverB.Close()

	yamlA = strings.ReplaceAll(yamlA, "{{URLA}}", serverA.URL)
	yamlB = strings.ReplaceAll(yamlB, "{{URLB}}", serverB.URL)

	var logBuf []byte
	logBuffer := bytes.NewBuffer(logBuf)
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	def := BuildDefaultRuleSetsWithLogger(logger)

	rs, err := CreateRuleSetFromData([]byte(yamlA))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 1)
	assert.Len(t, override.RuleDefinitions, 1)
	assert.NotNil(t, rs.Rules["ding"])
	assert.Nil(t, rs.Rules["dong"])
	assert.Contains(t, logBuffer.String(), "external ruleset fetch timed out after 5 seconds")
}

func TestRuleSet_GetExtendsLocalSpec_Single(t *testing.T) {

	yaml := `extends: {{FILE}}`
	yaml = strings.ReplaceAll(yaml, "{{FILE}}", "examples/custom-ruleset.yaml")

	def := BuildDefaultRuleSets()
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 1)
	assert.Len(t, override.RuleDefinitions, 1)

}

func TestRuleSet_GetExtendsLocalSpec_Multi_Chain(t *testing.T) {

	yaml3 := `rules:
  dong:
    description: dong
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern`

	tmpFile3, _ := os.Create("spec3.yaml")
	defer os.Remove(tmpFile3.Name())
	_, _ = tmpFile3.Write([]byte(yaml3))

	yaml2 := `extends: {{FILE}}`
	yaml2 = strings.ReplaceAll(yaml2, "{{FILE}}", tmpFile3.Name())

	tmpFile2, _ := os.Create("spec2.yaml")
	_, _ = tmpFile2.Write([]byte(yaml2))
	defer os.Remove(tmpFile2.Name())

	yaml := `extends: [{{FILE}}, examples/all-ruleset.yaml]`
	tmpFile, _ := os.Create("spec.yaml")
	yaml = strings.ReplaceAll(yaml, "{{FILE}}", tmpFile2.Name())
	_, _ = tmpFile.Write([]byte(yaml))
	defer os.Remove(tmpFile.Name())

	def := BuildDefaultRuleSets()
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 65)
	assert.Len(t, override.RuleDefinitions, 1)

}

func BenchmarkTestRuleSet_GetExtendsLocalSpec_Multi(b *testing.B) {

	yaml3 := `rules:
  dong:
    description: dong
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern`

	tmpFile3, _ := os.Create("spec3.yaml")
	defer os.Remove(tmpFile3.Name())
	_, _ = tmpFile3.Write([]byte(yaml3))

	yaml2 := `extends: {{FILE}}`
	yaml2 = strings.ReplaceAll(yaml2, "{{FILE}}", tmpFile3.Name())

	tmpFile2, _ := os.Create("spec2.yaml")
	_, _ = tmpFile2.Write([]byte(yaml2))
	defer os.Remove(tmpFile2.Name())

	yaml := `extends: [{{FILE}}, examples/all-ruleset.yaml]`
	tmpFile, _ := os.Create("spec.yaml")
	yaml = strings.ReplaceAll(yaml, "{{FILE}}", tmpFile2.Name())
	_, _ = tmpFile.Write([]byte(yaml))
	defer os.Remove(tmpFile.Name())

	for i := 0; i < b.N; i++ {

		def := BuildDefaultRuleSets()
		rs, err := CreateRuleSetFromData([]byte(yaml))
		assert.NoError(b, err)
		override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
		assert.Len(b, override.Rules, 56)
		assert.Len(b, override.RuleDefinitions, 1)

	}
}

func TestRuleSet_GetExtendsLocalSpec_Multi_Chain_Loop(t *testing.T) {

	yaml3 := `rules:
  dong:
    description: dong
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern`

	tmpFile3, _ := os.Create("spec3.yaml")
	defer os.Remove(tmpFile3.Name())
	_, _ = tmpFile3.Write([]byte(yaml3))

	yaml2 := `extends: {{FILE}}`
	yaml2 = strings.ReplaceAll(yaml2, "{{FILE}}", "spec2.yaml")

	tmpFile2, _ := os.Create("spec2.yaml")
	_, _ = tmpFile2.Write([]byte(yaml2))
	defer os.Remove(tmpFile2.Name())

	yaml := `extends: [{{FILE}}, examples/all-ruleset.yaml]`
	tmpFile, _ := os.Create("spec.yaml")
	yaml = strings.ReplaceAll(yaml, "{{FILE}}", tmpFile2.Name())
	_, _ = tmpFile.Write([]byte(yaml))
	defer os.Remove(tmpFile.Name())

	var logBuf []byte
	logBuffer := bytes.NewBuffer(logBuf)
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	def := BuildDefaultRuleSetsWithLogger(logger)

	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	_ = def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Contains(t, logBuffer.String(), "ruleset links to its self, circular rulesets are not permitted")

}
