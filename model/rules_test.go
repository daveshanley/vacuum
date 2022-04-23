package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
)

func TestRuleSchema(t *testing.T) {

	schemaMain, err := ioutil.ReadFile("schemas/ruleset.schema.json")
	assert.NoError(t, err)

	goodRules, err := ioutil.ReadFile("test_files/rules.json")
	assert.NoError(t, err)

	schemaLoader := gojsonschema.NewStringLoader(string(schemaMain))
	ruleLoader := gojsonschema.NewStringLoader(string(goodRules))

	result, err := gojsonschema.Validate(schemaLoader, ruleLoader)
	assert.NoError(t, err)
	assert.True(t, result.Valid())
	assert.Len(t, result.Errors(), 0)

}

func TestCreateRuleSetUsingJSON_Fail(t *testing.T) {

	// this is not going to work.
	json := `{ "pizza" : "cake" }`

	_, err := CreateRuleSetUsingJSON([]byte(json))
	assert.Error(t, err)

}

func TestRuleFunctionSchema_GetPropertyDescription(t *testing.T) {
	df := dummyFunc{}
	assert.Equal(t, "a type", df.GetSchema().GetPropertyDescription("type"))
}

func TestRuleFunctionSchema_GetPropertyDescription_Fail(t *testing.T) {
	df := dummyFunc{}
	assert.Empty(t, df.GetSchema().GetPropertyDescription("pizza"))
}

func TestRule_ToJSON(t *testing.T) {
	r := Rule{}
	assert.NotEmpty(t, r.ToJSON())

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

func TestNewRuleResultSet(t *testing.T) {

	r1 := RuleFunctionResult{
		Message: "pip",
		Rule: &Rule{
			Severity: severityError,
		},
	}
	results := NewRuleResultSet([]RuleFunctionResult{r1})

	assert.Equal(t, r1, *results.Results[0])

}

func TestRuleResults_GetErrorCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityError,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityError,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 2, results.GetErrorCount())
	assert.Equal(t, 2, results.GetErrorCount())

}

func TestRuleResults_GetWarnCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityInfo,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityError,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 1, results.GetWarnCount())
	assert.Equal(t, 1, results.GetWarnCount())

}

func TestRuleResults_GetInfoCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityInfo,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityInfo,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 2, results.GetInfoCount())
	assert.Equal(t, 2, results.GetInfoCount())

}

func TestRuleResultSet_GetResultsByRuleCategory(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Severity:     severityWarn,
		RuleCategory: RuleCategories[CategoryOperations],
	}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3})

	assert.Len(t, results.GetResultsByRuleCategory(CategoryInfo), 2)
	assert.Len(t, results.GetResultsByRuleCategory(CategoryOperations), 1)
	assert.Len(t, results.GetResultsByRuleCategory(CategoryInfo), 2)

}

func TestRuleResultSet_SortResultsByLineNumber(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "ten",
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "twenty",
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 20}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityWarn,
		RuleCategory: RuleCategories[CategoryOperations],
	}, StartNode: &yaml.Node{Line: 3}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3})
	sorted := results.SortResultsByLineNumber()

	assert.Equal(t, "three", sorted[0].Rule.Description)
	assert.Equal(t, "ten", sorted[1].Rule.Description)
	assert.Equal(t, "twenty", sorted[2].Rule.Description)

}

func TestRuleResultSet_CheckCategoryCounts(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     severityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 20}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 3}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3})

	assert.Len(t, results.GetErrorsByRuleCategory(CategoryInfo), 1)
	assert.Len(t, results.GetWarningsByRuleCategory(CategoryInfo), 1)
	assert.Len(t, results.GetInfoByRuleCategory(CategoryInfo), 1)
}

func TestRuleResultSet_GenerateSpectralReport(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     severityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})
	assert.Len(t, results.GenerateSpectralReport("test"), 4)
}

func TestRuleResultSet_CalculateCategoryHealth_Errors(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     severityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})
	assert.Equal(t, 89, results.CalculateCategoryHealth(CategoryInfo))

}

func TestRuleResultSet_CalculateCategoryHealth_Warnings(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     severityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     severityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     severityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})
	assert.Equal(t, 99, results.CalculateCategoryHealth(CategoryInfo))

}

func TestRuleResultSet_CalculateCategoryHealth_Warnings_Lots(t *testing.T) {
	var r []RuleFunctionResult
	for i := 0; i < 100; i++ {
		r = append(r, RuleFunctionResult{Rule: &Rule{
			Description:  "one",
			Severity:     severityWarn,
			RuleCategory: RuleCategories[CategoryInfo],
		}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}})
	}

	results := NewRuleResultSet(r)
	assert.Equal(t, 50, results.CalculateCategoryHealth(CategoryInfo))

}
