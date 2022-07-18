package model

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestValidateRuleFunctionContextAgainstSchema_Success(t *testing.T) {

	opts := make(map[string]string)
	opts["type"] = "snake"
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_SuccessMultiple(t *testing.T) {

	opts := make(map[string]string)
	opts["type"] = "snake,camel"
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Success_SimulateYAML(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = "snake"
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Success_SimulateYAML_Multiple(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = "snake,camel,pascal"
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Success_SimulateYAML_IntType(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = 123
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Success_SimulateYAML_BoolType(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = true
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Success_SimulateYAML_Float(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = 123.456
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Success_SimulateYAML_InterfaceArray(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = []interface{}{123, "oh hai!"}
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Fail_SimulateYAML_NoField(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = "woah"
	rf := dummyFuncMinMax{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 1)
}

func TestValidateRuleFunctionContextAgainstSchema_Fail(t *testing.T) {

	opts := make(map[string]string)
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.False(t, res)
	assert.Len(t, errs, 1)
}

func TestValidateRuleFunctionContextAgainstSchema_MinMax_FailMin(t *testing.T) {

	opts := make(map[string]string)
	rf := dummyFuncMinMax{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.False(t, res)
	assert.Len(t, errs, 2)
}

func TestValidateRuleFunctionContextAgainstSchema_MinMax_FailMax(t *testing.T) {

	opts := make(map[string]string)
	opts["beer"] = "shoes"
	opts["lime"] = "kitty"
	opts["carrot"] = "cake"
	rf := dummyFuncMinMax{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.False(t, res)
	assert.Len(t, errs, 2)
}

func TestBuildFunctionResult(t *testing.T) {
	fr := BuildFunctionResult("pizza", "party", "tonight")
	assert.Equal(t, "'pizza' party 'tonight'", fr.Message)
}

func TestCastToRuleAction(t *testing.T) {
	ra := &RuleAction{
		Field: "choco",
	}
	assert.Equal(t, "choco", CastToRuleAction(ra).Field)
}

func TestCastToRuleAction_Fail_WrongType(t *testing.T) {
	ra := "not a rule action"
	assert.Nil(t, CastToRuleAction(ra))
}

func TestCastToRuleAction_Fail_Nil(t *testing.T) {
	var ra interface{}
	assert.Nil(t, CastToRuleAction(ra))
}

func TestMapPathAndNodesToResults(t *testing.T) {

	results := []RuleFunctionResult{
		{Path: "$.pie.and.mash"},
		{Path: "$.splish.and.splash"},
	}

	path := "$.fish.and.chips"
	yml := "cake: bake"

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	mapped := MapPathAndNodesToResults(path, &rootNode, &rootNode, results)

	for _, mappedValue := range mapped {
		assert.Equal(t, path, mappedValue.Path)
		assert.Equal(t, &rootNode, mappedValue.StartNode)
		assert.Equal(t, &rootNode, mappedValue.EndNode)

	}
}

func TestBuildFunctionResultString(t *testing.T) {
	assert.Equal(t, "wow, a cheese ball",
		BuildFunctionResultString("wow, a cheese ball").Message)
}

type dummyFunc struct {
}

func (df dummyFunc) GetSchema() RuleFunctionSchema {
	return RuleFunctionSchema{
		Required: []string{"type"},
		Properties: []RuleFunctionProperty{
			{
				Name:        "type",
				Description: "a type",
			},
		},
		ErrorMessage: "missing the type my friend.",
	}
}

func (df dummyFunc) RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult {
	return nil
}

type dummyFuncMinMax struct {
}

func (df dummyFuncMinMax) GetSchema() RuleFunctionSchema {
	return RuleFunctionSchema{
		Required:      []string{"type"},
		RequiresField: true,
		Properties: []RuleFunctionProperty{
			{
				Name:        "type",
				Description: "a type",
			},
		},
		MinProperties: 1,
		MaxProperties: 2,
		ErrorMessage:  "missing the type my friend.",
	}
}

func (df dummyFuncMinMax) RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult {
	return nil
}
