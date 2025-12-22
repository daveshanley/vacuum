package core

import (
	"github.com/daveshanley/vacuum/model"
	gen_utils "github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCasing_RunRule_CamelSuccess(t *testing.T) {

	sampleYaml := `beer: "isYummy"`

	path := "$.beer"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "camel"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_CamelFail(t *testing.T) {

	sampleYaml := `beer: "ISGREAT"`

	path := "$.beer"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "camel"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule
	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_PascalSuccess(t *testing.T) {

	sampleYaml := `spaghetti: "IsMyFav"`

	path := "$.spaghetti"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "pascal"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule
	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_PascalFail(t *testing.T) {

	sampleYaml := `spaghetti: "is-the-best"`

	path := "$.spaghetti"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "pascal"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_KebabSuccess(t *testing.T) {

	sampleYaml := `melody: "is-what-makes-life-worth-living"`

	path := "$.melody"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "kebab"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_KebabFail(t *testing.T) {

	sampleYaml := `melody: "is_what-Makes-life_worth-living"`

	path := "$.melody"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "kebab"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_PascalKebabSuccess(t *testing.T) {

	testCases := []string{
		`melody: "Is-What-Makes-Life-Worth-Living"`,
		`melody: "Is-Living"`,
		`melody: "Living"`,
	}

	for _, sampleYaml := range testCases {

		path := "$.melody"

		nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
		assert.Len(t, nodes, 1)

		opts := make(map[string]any)
		opts["type"] = "pascal-kebab"

		rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
		ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
		ctx.Given = path
		ctx.Rule = &rule

		def := &Casing{}
		res := def.RunRule(nodes, ctx)

		assert.Len(t, res, 0)
	}
}

func TestCasing_RunRule_PascalKebabFail(t *testing.T) {

	testCases := []string{
		`melody: "Is-What-Makes-Life-Worth-living"`,
		`melody: "Is-"`,
		`melody: "Is-what"`,
		`melody: "IS-WHAT"`,
		`melody: "Is_What-Makes-life_worth-living"`,
	}

	for _, sampleYaml := range testCases {
		path := "$.melody"

		nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
		assert.Len(t, nodes, 1)

		opts := make(map[string]any)
		opts["type"] = "pascal-kebab"

		rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
		ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
		ctx.Given = path
		ctx.Rule = &rule

		def := &Casing{}
		res := def.RunRule(nodes, ctx)

		assert.Len(t, res, 1, "test case: '%s'", sampleYaml)
	}
}

func TestCasing_RunRule_CobolSuccess(t *testing.T) {

	sampleYaml := `maddy: "THE-LITTLE-CHAMPION"`

	path := "$.maddy"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "cobol"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_CobolFail(t *testing.T) {

	sampleYaml := `maddy: "THE-little-CHAMPION"`

	path := "$.maddy"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "cobol"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_SnakeSuccess(t *testing.T) {

	sampleYaml := `ember: "naughty_puppy_get_off_the_couch"`

	path := "$.ember"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "snake"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_SnakeFail(t *testing.T) {

	sampleYaml := `ember: "Naughty_ember-get-off-THAT_COUCH"`

	path := "$.ember"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "snake"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_MacroSuccess(t *testing.T) {

	sampleYaml := `chicken: "THE_NANNY_DOG"`

	path := "$.chicken"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "macro"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_MacroFail(t *testing.T) {

	sampleYaml := `chicken: "THE-Nanny_dog"`

	path := "$.chicken"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "macro"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_CamelNoDigits_Success(t *testing.T) {

	sampleYaml := `alchomohol: "afterHoursNoDigits"`

	path := "$.alchomohol"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "camel"
	opts["disallowDigits"] = "true"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_CamelNoDigits_Fail(t *testing.T) {

	sampleYaml := `alchomohol: "aft3rHoursN0Dig1ts"`

	path := "$.alchomohol"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "camel"
	opts["disallowDigits"] = "true"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_Snake_SeparatingChar_Success(t *testing.T) {

	sampleYaml := `alchomohol: "after_hours,want_a_drink"`

	path := "$.alchomohol"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "snake"
	opts["separator.char"] = ","

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_Snake_SeparatingChar_Fail(t *testing.T) {

	sampleYaml := `alchomohol: "after_hours|want_a_drink"`

	path := "$.alchomohol"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "snake"
	opts["separator.char"] = ","

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_RunRule_Snake_AllowLeading_Success(t *testing.T) {

	sampleYaml := `mo_money: ",mo_problems,rub_a,dub_dub"`

	path := "$.mo_money"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "snake"
	opts["separator.char"] = ","
	opts["separator.allowLeading"] = "true"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_Snake_AllowLeading_Fail(t *testing.T) {

	sampleYaml := `mo_money: ",mo_problems,rub_a,dub_dub"`

	path := "$.mo_money"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	opts := make(map[string]any)
	opts["type"] = "snake"
	opts["separator.char"] = ","
	opts["separator.allowLeading"] = "false"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestCasing_GetSchema_Valid(t *testing.T) {

	opts := make(map[string]any)
	opts["type"] = "snake"

	rf := &Casing{}

	res, errs := model.ValidateRuleFunctionContextAgainstSchema(rf, model.RuleFunctionContext{Options: opts})
	assert.Len(t, errs, 0)
	assert.True(t, res)

}

// This tests a recursive match on a field ('properties') and ensures that the objects that match have their fields
// (for objects) or elements (for arrays) all satisfy the casing.
func TestCasing_RunRule_MatchMapAndArray_Recursive_Success(t *testing.T) {

	sampleYaml :=
		`
chicken:
  properties:
   - NAME
   - AGE
pork:
  properties:
    SIZE: large
    GENDER: male
    FROM:
      properties:
        COUNTRY: Canada
        CITY: Guelph
`

	// Recursively match any object that has a 'properties' object
	path := "$..properties"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 3, "expected 3 'properties' objects")

	opts := make(map[string]any)
	opts["type"] = "macro"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestCasing_RunRule_MatchMapAndArray_Recursive_Fail(t *testing.T) {

	sampleYaml :=
		`
chicken:
  properties:
   - NAME
   - Age
pork:
  properties:
    SIZE: large
    Gender: male
    FROM:
      properties:
        COUNTRY: Canada
        city: Guelph
`

	path := "$..properties"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 3, "expected 3 'properties' objects")

	opts := make(map[string]any)
	opts["type"] = "macro"

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 3, "expected all fields of 'properties' objects to be MACRO case")
}

// Tests for Spectral-compatible nested separator options (Issue #651)
// Spectral uses nested YAML format: separator: { char: '-', allowLeading: true }
// These tests verify that vacuum accepts both formats.

func TestCasing_RunRule_SpectralNestedSeparator_Success(t *testing.T) {
	// Test case from GitHub issue #651 - Pascal case with hyphen separator
	sampleYaml := `header: "X-Custom-Header"`

	path := "$.header"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	// Spectral format: nested YAML structure instead of dot-notation
	opts := map[string]interface{}{
		"type": "pascal",
		"separator": map[string]interface{}{
			"char": "-",
		},
	}

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0, "Pascal case with hyphen separator should pass")
}

func TestCasing_RunRule_SpectralNestedSeparator_Fail(t *testing.T) {
	// Invalid pascal-case with hyphen separator
	sampleYaml := `header: "x-custom-header"`

	path := "$.header"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	// Spectral format: nested YAML structure
	opts := map[string]interface{}{
		"type": "pascal",
		"separator": map[string]interface{}{
			"char": "-",
		},
	}

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1, "lowercase with hyphen separator should fail pascal case")
}

func TestCasing_RunRule_SpectralNestedSeparatorWithAllowLeading_Success(t *testing.T) {
	// Test with allowLeading option in nested format
	sampleYaml := `path: "-my-path-value"`

	path := "$.path"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	// Spectral format: nested YAML structure with allowLeading
	opts := map[string]interface{}{
		"type": "kebab",
		"separator": map[string]interface{}{
			"char":         "-",
			"allowLeading": true,
		},
	}

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0, "kebab case with leading separator should pass when allowLeading is true")
}

func TestCasing_RunRule_SpectralNestedSeparatorWithAllowLeading_Fail(t *testing.T) {
	// Test with allowLeading=false in nested format
	// Uses comma separator with snake case (comma is not part of snake pattern)
	sampleYaml := `value: ",mo_problems,rub_a,dub_dub"`

	path := "$.value"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	// Spectral format: nested YAML structure with allowLeading=false
	opts := map[string]interface{}{
		"type": "snake",
		"separator": map[string]interface{}{
			"char":         ",",
			"allowLeading": false,
		},
	}

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1, "snake case with leading separator should fail when allowLeading is false")
}

func TestCasing_RunRule_SpectralNestedSeparatorSnake_Success(t *testing.T) {
	// Test snake case with comma separator using Spectral nested format
	sampleYaml := `value: "after_hours,want_a_drink"`

	path := "$.value"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	// Spectral format: nested YAML structure
	opts := map[string]interface{}{
		"type": "snake",
		"separator": map[string]interface{}{
			"char": ",",
		},
	}

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0, "snake case with comma separator should pass")
}

// Test that map[interface{}]interface{} format also works (common in YAML parsing)
func TestCasing_RunRule_SpectralNestedSeparator_MapInterfaceInterface(t *testing.T) {
	sampleYaml := `header: "X-Custom-Header"`

	path := "$.header"

	nodes, _ := gen_utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	// Simulate how YAML parsing sometimes creates map[interface{}]interface{}
	opts := map[string]interface{}{
		"type": "pascal",
		"separator": map[interface{}]interface{}{
			"char": "-",
		},
	}

	rule := buildCoreTestRule(path, model.SeverityError, "casing", "", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Given = path
	ctx.Rule = &rule

	def := &Casing{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0, "nested separator with map[interface{}]interface{} should work")
}
