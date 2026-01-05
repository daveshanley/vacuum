package model

import (
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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
	// Expected errors:
	// 1. Max properties exceeded (3 > 2)
	// 2. Missing required property: type
	// 3-5. Invalid properties: beer, lime, carrot (not defined in schema)
	assert.Len(t, errs, 5)
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
	err := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, err)
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

func TestCompileRegex(t *testing.T) {

	ctx := RuleFunctionContext{}
	var res []RuleFunctionResult
	regex := CompileRegex(ctx, "type", &res)
	assert.True(t, regex.Match([]byte("type")))
	assert.Len(t, res, 0)
}

func TestCompileRegex_Fail(t *testing.T) {

	ctx := RuleFunctionContext{}
	var res []RuleFunctionResult
	_ = CompileRegex(ctx, `^\/(?!\/)(.*?)`, &res)
	assert.Len(t, res, 1)
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

func (df dummyFunc) GetCategory() string {
	return "dummy"
}

func (df dummyFunc) RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult {
	return nil
}

type dummyFuncMinMax struct {
}

func (df dummyFuncMinMax) GetCategory() string {
	return "dummy"
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

// Tests for extractOptionKeys helper function
func TestExtractOptionKeys_MapStringInterface(t *testing.T) {
	opts := map[string]interface{}{
		"type":   "pascal",
		"schema": map[string]interface{}{"type": "array"},
	}
	keys := extractOptionKeys(opts)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "type")
	assert.Contains(t, keys, "schema")
}

func TestExtractOptionKeys_MapStringString(t *testing.T) {
	opts := map[string]string{
		"type":           "pascal",
		"separator.char": "-",
	}
	keys := extractOptionKeys(opts)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "type")
	assert.Contains(t, keys, "separator.char")
}

func TestExtractOptionKeys_SliceInterface(t *testing.T) {
	opts := []interface{}{
		map[string]interface{}{"name": "value1"},
		map[string]interface{}{"other": "value2"},
	}
	keys := extractOptionKeys(opts)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "name")
	assert.Contains(t, keys, "other")
}

func TestExtractOptionKeys_Nil(t *testing.T) {
	keys := extractOptionKeys(nil)
	assert.Nil(t, keys)
}

func TestExtractOptionKeys_UnsupportedType(t *testing.T) {
	keys := extractOptionKeys("just a string")
	assert.Nil(t, keys)
}

// Tests for optionKeyMatchesProperty helper function
func TestOptionKeyMatchesProperty_ExactMatch(t *testing.T) {
	assert.True(t, optionKeyMatchesProperty("schema", "schema"))
	assert.True(t, optionKeyMatchesProperty("type", "type"))
}

func TestOptionKeyMatchesProperty_PrefixMatch(t *testing.T) {
	// "separator" should match "separator.char"
	assert.True(t, optionKeyMatchesProperty("separator", "separator.char"))
	assert.True(t, optionKeyMatchesProperty("separator", "separator.allowLeading"))
}

func TestOptionKeyMatchesProperty_NoMatch(t *testing.T) {
	assert.False(t, optionKeyMatchesProperty("sep", "separator.char"))
	assert.False(t, optionKeyMatchesProperty("invalid", "schema"))
	assert.False(t, optionKeyMatchesProperty("schema", "type"))
}

func TestOptionKeyMatchesProperty_PartialNoMatch(t *testing.T) {
	// "separatorX" should NOT match "separator.char" (not a proper prefix)
	assert.False(t, optionKeyMatchesProperty("separatorX", "separator.char"))
}

// Tests for findInvalidOptionKeys helper function
func TestFindInvalidOptionKeys_AllValid(t *testing.T) {
	keys := []string{"type", "schema"}
	props := []RuleFunctionProperty{
		{Name: "type"},
		{Name: "schema"},
	}
	invalid := findInvalidOptionKeys(keys, props)
	assert.Len(t, invalid, 0)
}

func TestFindInvalidOptionKeys_SomeInvalid(t *testing.T) {
	keys := []string{"type", "invalid", "schema"}
	props := []RuleFunctionProperty{
		{Name: "type"},
		{Name: "schema"},
	}
	invalid := findInvalidOptionKeys(keys, props)
	assert.Len(t, invalid, 1)
	assert.Contains(t, invalid, "invalid")
}

func TestFindInvalidOptionKeys_PrefixMatch(t *testing.T) {
	// "separator" should be valid when "separator.char" is a property
	keys := []string{"type", "separator"}
	props := []RuleFunctionProperty{
		{Name: "type"},
		{Name: "separator.char"},
		{Name: "separator.allowLeading"},
	}
	invalid := findInvalidOptionKeys(keys, props)
	assert.Len(t, invalid, 0)
}

func TestFindInvalidOptionKeys_EmptyKeys(t *testing.T) {
	props := []RuleFunctionProperty{{Name: "type"}}
	invalid := findInvalidOptionKeys(nil, props)
	assert.Len(t, invalid, 0)
}

// Test for issue #790 - schema function with nested JSON schema object
type schemaFunc struct{}

func (sf schemaFunc) GetSchema() RuleFunctionSchema {
	return RuleFunctionSchema{
		Required: []string{"schema"},
		Properties: []RuleFunctionProperty{
			{Name: "schema", Description: "A valid JSON Schema object"},
			{Name: "unpack", Description: "Unpack the node"},
			{Name: "forceValidation", Description: "Force validation"},
		},
		ErrorMessage: "'schema' function needs a 'schema' property",
	}
}

func (sf schemaFunc) GetCategory() string { return "core" }
func (sf schemaFunc) RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult {
	return nil
}

func TestValidateRuleFunctionContextAgainstSchema_NestedSchemaObject(t *testing.T) {
	// This is the issue #790 scenario - schema function with nested JSON schema
	opts := map[string]interface{}{
		"schema": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
			},
			"minItems": 1,
		},
	}
	rf := schemaFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "tags",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

// Test for issue #651 - casing function with Spectral nested separator format
type casingFunc struct{}

func (cf casingFunc) GetSchema() RuleFunctionSchema {
	return RuleFunctionSchema{
		Required: []string{"type"},
		Properties: []RuleFunctionProperty{
			{Name: "type", Description: "The casing type"},
			{Name: "separator.char", Description: "Separator character"},
			{Name: "separator.allowLeading", Description: "Allow leading separator"},
		},
		ErrorMessage: "'casing' function has invalid options",
	}
}

func (cf casingFunc) GetCategory() string { return "core" }
func (cf casingFunc) RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult {
	return nil
}

func TestValidateRuleFunctionContextAgainstSchema_SpectralNestedSeparator(t *testing.T) {
	// This is the issue #651 scenario - Spectral nested format for separator
	opts := map[string]interface{}{
		"type": "pascal",
		"separator": map[string]interface{}{
			"char": "-",
		},
	}
	rf := casingFunc{}
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

func TestFormatMatches(t *testing.T) {
	tests := []struct {
		name       string
		ruleFormat string
		specFormat string
		expected   bool
	}{
		// oas3 family matching - oas3 should match all 3.x versions
		{"oas3 matches oas3", OAS3, OAS3, true},
		{"oas3 matches oas3_1", OAS3, OAS31, true},
		{"oas3 matches oas3_2", OAS3, OAS32, true},
		{"oas3 does not match oas2", OAS3, OAS2, false},

		// Specific version matching - oas3_1 only matches oas3_1
		{"oas3_1 matches oas3_1", OAS31, OAS31, true},
		{"oas3_1 does not match oas3", OAS31, OAS3, false},
		{"oas3_1 does not match oas3_2", OAS31, OAS32, false},
		{"oas3_1 does not match oas2", OAS31, OAS2, false},

		// Specific version matching - oas3_2 only matches oas3_2
		{"oas3_2 matches oas3_2", OAS32, OAS32, true},
		{"oas3_2 does not match oas3", OAS32, OAS3, false},
		{"oas3_2 does not match oas3_1", OAS32, OAS31, false},
		{"oas3_2 does not match oas2", OAS32, OAS2, false},

		// oas2 matching - oas2 only matches oas2
		{"oas2 matches oas2", OAS2, OAS2, true},
		{"oas2 does not match oas3", OAS2, OAS3, false},
		{"oas2 does not match oas3_1", OAS2, OAS31, false},
		{"oas2 does not match oas3_2", OAS2, OAS32, false},

		// Edge cases with empty strings
		{"empty rule format does not match oas3", "", OAS3, false},
		{"oas3 does not match empty spec format", OAS3, "", false},
		{"both empty matches", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMatches(tt.ruleFormat, tt.specFormat)
			assert.Equal(t, tt.expected, result)
		})
	}
}
