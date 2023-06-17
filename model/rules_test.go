package model

import (
	"encoding/json"
	"fmt"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestRuleSchema(t *testing.T) {

	schemaMain, err := os.ReadFile("../rulesets/schemas/ruleset.schema.json")
	assert.NoError(t, err)

	goodRules, err := os.ReadFile("test_files/rules.json")
	assert.NoError(t, err)

	var rules map[string]interface{}
	_ = json.Unmarshal(goodRules, &rules)

	compiler := jsonschema.NewCompiler()
	_ = compiler.AddResource("schema.json", strings.NewReader(string(schemaMain)))
	jsch, _ := compiler.Compile("schema.json")

	err = jsch.Validate(rules)
	assert.NoError(t, err)

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

func TestNewRuleResultSet(t *testing.T) {

	r1 := RuleFunctionResult{
		Message: "pip",
		Rule: &Rule{
			Severity: SeverityError,
		},
	}
	results := NewRuleResultSet([]RuleFunctionResult{r1})

	assert.Equal(t, r1, *results.Results[0])

}

func TestRuleResults_GetErrorCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityError,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityError,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 2, results.GetErrorCount())
	assert.Equal(t, 2, results.GetErrorCount())

}

func TestRuleResults_GetWarnCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityInfo,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityError,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 1, results.GetWarnCount())
	assert.Equal(t, 1, results.GetWarnCount())

}

func TestRuleResults_GetInfoCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityInfo,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityInfo,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: SeverityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 2, results.GetInfoCount())
	assert.Equal(t, 2, results.GetInfoCount())

}

func TestRuleResultSet_GetResultsByRuleCategory(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryOperations],
	}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3})

	assert.Len(t, results.GetResultsByRuleCategory(CategoryInfo), 2)
	assert.Len(t, results.GetResultsByRuleCategory(CategoryOperations), 1)
	assert.Len(t, results.GetResultsByRuleCategory(CategoryInfo), 2)

}

func TestRuleResultSet_GetResultsByRuleCategoryWithLimit(t *testing.T) {

	rule1 := &Rule{
		Id:           "one",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}

	rule2 := &Rule{
		Id:           "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}

	var r []RuleFunctionResult
	for i := 0; i < 5000; i++ {
		var rule *Rule
		if i%2 == 0 {
			rule = rule1
		} else {
			rule = rule2
		}

		r = append(r, RuleFunctionResult{Rule: rule,
			StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}})
	}
	results := NewRuleResultSet(r)

	limitedResults := results.GetResultsForCategoryWithLimit(CategoryInfo, 50)

	assert.Len(t, limitedResults.RuleResults, 2)

	for _, rule := range limitedResults.RuleResults {
		assert.Len(t, rule.Results, 50)
	}

}

func TestRuleResultSet_SortResultsByLineNumber(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "ten",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "twenty",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 20}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
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
		Severity:     SeverityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 20}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 3}}

	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 3}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})

	assert.Len(t, results.GetErrorsByRuleCategory(CategoryInfo), 1)
	assert.Len(t, results.GetWarningsByRuleCategory(CategoryInfo), 1)
	assert.Len(t, results.GetInfoByRuleCategory(CategoryInfo), 1)
	assert.Len(t, results.GetHintByRuleCategory(CategoryInfo), 1)
}

func TestRuleResultSet_GenerateSpectralReport(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     SeverityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})
	assert.Len(t, results.GenerateSpectralReport("test"), 4)
}

func TestRuleResultSet_CalculateCategoryHealth_Errors(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     SeverityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})
	assert.Equal(t, 89, results.CalculateCategoryHealth(CategoryInfo))

}

func TestRuleResultSet_CalculateCategoryHealth_Warnings(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
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
			Severity:     SeverityWarn,
			RuleCategory: RuleCategories[CategoryInfo],
		}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}})
	}

	results := NewRuleResultSet(r)
	assert.Equal(t, 50, results.CalculateCategoryHealth(CategoryInfo))

}

func TestRuleResultSet_CalculateCategoryHealth_Errors_Lots(t *testing.T) {
	var r []RuleFunctionResult
	for i := 0; i < 900; i++ {
		r = append(r, RuleFunctionResult{Rule: &Rule{
			Description:  fmt.Sprintf("%d", i),
			Severity:     SeverityError,
			RuleCategory: RuleCategories[CategoryInfo],
		}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}})
	}

	results := NewRuleResultSet(r)
	score := results.CalculateCategoryHealth(CategoryInfo)
	assert.Equal(t, 0, score)

}

func TestRuleResultSet_GetRuleResultsForCategory(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategorySchemas],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategorySecurity],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategorySchemas],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})
	assert.Len(t, results.GetRuleResultsForCategory(CategorySchemas).RuleResults, 2)

}

func TestRuleResultSet_GetRuleResultsForCategoryAll(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategorySchemas],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategorySecurity],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategorySchemas],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})
	assert.Len(t, results.GetRuleResultsForCategory(CategoryAll).RuleResults, 4)

}

func TestRule_GetSeverityAsIntValue(t *testing.T) {

	r1 := &Rule{
		Description:  "one",
		Severity:     SeverityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}
	assert.Equal(t, 0, r1.GetSeverityAsIntValue())

	r2 := &Rule{
		Description:  "two",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}
	assert.Equal(t, 1, r2.GetSeverityAsIntValue())

	r3 := &Rule{
		Description:  "three",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}
	assert.Equal(t, 2, r3.GetSeverityAsIntValue())

	r4 := &Rule{
		Description:  "four",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}
	assert.Equal(t, 3, r4.GetSeverityAsIntValue())

	r5 := &Rule{
		Description:  "five",
		Severity:     "unknown",
		RuleCategory: RuleCategories[CategoryInfo],
	}
	assert.Equal(t, -1, r5.GetSeverityAsIntValue())

}

func TestRuleResultsForCategory_Sort(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategorySchemas],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategorySecurity],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategorySchemas],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}
	r5 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityError,
		RuleCategory: RuleCategories[CategorySchemas],
	}, StartNode: &yaml.Node{Line: 10, Column: 10}, EndNode: &yaml.Node{Line: 10, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4, r5})
	catResults := results.GetRuleResultsForCategory(CategorySchemas)
	sort.Sort(catResults)

	assert.Equal(t, "three", catResults.RuleResults[0].Rule.Description) // first result should be lowest sev.

}
