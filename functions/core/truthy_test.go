package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"testing"
)

func TestTruthy_GetSchema(t *testing.T) {
	def := Truthy{}
	assert.Equal(t, "truthy", def.GetSchema().Name)
}

func TestTruthy_RunRule(t *testing.T) {
	def := Truthy{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestTruthy_RunRule_Fail(t *testing.T) {

	sampleYaml := `
tags:
  - name: "bad tag 1"
    description: false
  - name: "bad tag 2"
    description: 0
  - name: "bad tag 3"
    description: ""
  - name: "bad tag 4"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 4)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 4)
}

func TestTruthy_RunRule_Fail_NoNodes(t *testing.T) {

	sampleYaml := `
notTags:
  - name: "bad tag 1"
    description: false
  - name: "bad tag 2"
    description: 0
  - name: "bad tag 3"
    description: ""
  - name: "bad tag 4"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 0)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestTruthy_RunRule_Pass(t *testing.T) {

	sampleYaml := `
tags:
  - name: "good tag 1"
    description: "yeah"
  - name: "bad tag 2"
    description: 0
  - name: "bad tag 3"
    description: ""
  - name: "good Tag 2"
    description: "a nice description"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 4)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "description", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

func TestTruthy_RunRule_NoContent(t *testing.T) {

	sampleYaml :=
		`openapi: 3.0.0
paths:
  /v1/cake:
    get:
      parameters:
        - in: query
          name: type
          required: true
        - in: query
          name: flavor
          required: false
        - in: query
          name: weight
`

	path := "$.paths.*.*.parameters[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 3)

	document, err := libopenapi.NewDocument([]byte(sampleYaml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "required", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule
	ctx.Document = document
	ctx.DrDocument = drDocument

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	// Two of the three nodes should match because one has a truthy value
	assert.Len(t, res, 2)
	assert.Equal(t, res[0].Path, "$.paths['/v1/cake'].get.parameters[1].required")
	assert.Equal(t, res[1].Path, "$.paths['/v1/cake'].get.parameters[2].required")
}

func TestTruthy_RunRule_ArrayTest(t *testing.T) {

	sampleYaml := `- lemons:
  rags:
    - name: fish
    - name: cake
    - name: pizza
- limes:
  tags:
    - name: fish
    - name: cake`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "rags", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestTruthy_RunRule_CheckSecurity(t *testing.T) {

	sampleYaml := `notSecurity: none`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "security", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

// Issue #1032: like Spectral, 'field' does not apply to a matched scalar, so the scalar itself is checked.
func TestTruthy_RunRule_Field_ScalarTarget_Issue1032(t *testing.T) {
	sampleYaml := `schema:
  country: {type: string, example: DE}
example:
  country: DE
empty:
  country: ""`

	path := "$..country"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 3)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "example", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	var res []model.RuleFunctionResult
	for _, node := range nodes {
		res = append(res, tru.RunRule([]*yaml.Node{node}, ctx)...)
	}

	assert.Len(t, res, 1, "only the empty scalar is falsy")
	assert.Equal(t, "test rule: `value` must be set", res[0].Message)
	assert.Equal(t, "$..country", res[0].Path, "no field is appended to a scalar path")
}

// Elements of a matched array still have the field looked up.
func TestTruthy_RunRule_Field_ArrayOfScalars(t *testing.T) {
	sampleYaml := `tags: [a, b]`

	path := "$.tags"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 1)

	rule := buildCoreTestRule(path, model.SeverityError, "truthy", "name", nil)
	ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Given = path
	ctx.Rule = &rule

	tru := Truthy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

func TestTruthyRepairOwnerUsesParsedFieldPath(t *testing.T) {
	source := []byte("servers:\n  - {}\nproperties:\n  x.y: {}\n  other: {}\na.b: {}\nwrapped:\n  allOf:\n    - properties:\n        x.y: {}\n")
	var root yaml.Node
	assert.NoError(t, yaml.Unmarshal(source, &root))
	node := root.Content[0]
	for _, tc := range []struct {
		field     string
		ownerLine int
	}{
		{"servers[0].url", 2}, {"properties['x.y'].description", 4}, {"properties.other.description", 5},
		{`a\.b.description`, 6}, {"wrapped.properties['x.y'].description", 10},
	} {
		field := tc.field
		t.Run(field, func(t *testing.T) {
			rule := buildCoreTestRule("$", model.SeverityError, "truthy", field, nil)
			rule.Resolved = true
			ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
			ctx.Rule, ctx.Given = &rule, "$"
			results := (&Truthy{}).RunRule([]*yaml.Node{node}, ctx)
			if !assert.Len(t, results, 1) {
				return
			}
			assert.NotNil(t, results[0].RepairNode)
			assert.NotSame(t, node, results[0].RepairNode)
			assert.Equal(t, tc.ownerLine, results[0].RepairNode.Line)
			assert.Equal(t, yaml.MappingNode, results[0].RepairNode.Kind)
			assert.Empty(t, results[0].RepairNode.Content)
			assert.Equal(t, field+"-missing", results[0].DiagnosticCode)
		})
	}
}

func TestTruthyRepairOwnerLiteralBracketsAndBuiltinCode(t *testing.T) {
	source := []byte("properties:\n  x[y]: {}\ninfo: {}\n")
	var root yaml.Node
	assert.NoError(t, yaml.Unmarshal(source, &root))
	for _, tc := range []struct {
		field, code string
		line        int
	}{
		{"properties['x[y]'].description", "properties['x[y]'].description-missing", 2},
		{"info.contact", "contact-missing", 3},
		{"info.description", "description-missing", 3},
	} {
		rule := buildCoreTestRule("$", model.SeverityError, "truthy", tc.field, nil)
		ctx := buildCoreTestContext(model.CastToRuleAction(rule.Then), nil)
		ctx.Rule, ctx.Given = &rule, "$"
		results := (&Truthy{}).RunRule([]*yaml.Node{root.Content[0]}, ctx)
		if !assert.Len(t, results, 1) {
			continue
		}
		if assert.NotNil(t, results[0].RepairNode) {
			assert.Equal(t, tc.line, results[0].RepairNode.Line)
		}
		assert.Equal(t, tc.code, results[0].DiagnosticCode)
	}
}
