package core

import (
    "github.com/daveshanley/vacuum/model"
    highBase "github.com/pb33f/libopenapi/datamodel/high/base"
    "github.com/pb33f/libopenapi/datamodel/low"
    lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
    "github.com/pb33f/libopenapi/utils"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

func TestOpenAPISchema_GetSchema(t *testing.T) {
    def := Schema{}
    assert.Equal(t, "oas_schema", def.GetSchema().Name)
}

func TestOpenAPISchema_RunRule(t *testing.T) {
    def := Schema{}
    res := def.RunRule(nil, model.RuleFunctionContext{})
    assert.Len(t, res, 0)
}

func TestOpenAPISchema_DuplicateEntryInEnum(t *testing.T) {

    yml := `components:
  schemas:
    Color:
      type: string
      enum:
        - black
        - white
        - black`

    path := "$..[?(@.enum)]"

    nodes, _ := utils.FindNodes([]byte(yml), path)
    opts := make(map[string]interface{})

    validate := `type: array
items:
  type: string
uniqueItems: true`

    var n yaml.Node
    _ = yaml.Unmarshal([]byte(validate), &n)

    schema := testGenerateJSONSchema(n.Content[0])

    opts["schema"] = schema

    rule := model.Rule{
        Given: path,
        Then: &model.RuleAction{
            Field:           "enum",
            Function:        "enum",
            FunctionOptions: opts,
        },
        Description: "Enum values must not have duplicate entry",
    }

    ctx := model.RuleFunctionContext{
        RuleAction: model.CastToRuleAction(rule.Then),
        Rule:       &rule,
        Options:    opts,
        Given:      rule.Given,
    }

    def := Schema{}
    res := def.RunRule(nodes, ctx)

    assert.Len(t, res, 1)
    assert.Equal(t, "Enum values must not have duplicate entry: items at index 0 and 2 are equal", res[0].Message)

}

func TestOpenAPISchema_InvalidSchemaInteger(t *testing.T) {

    yml := `smell: not a number`

    path := "$"

    nodes, _ := utils.FindNodes([]byte(yml), path)

    validate := `type: integer`

    var n yaml.Node
    _ = yaml.Unmarshal([]byte(validate), &n)

    schema := testGenerateJSONSchema(n.Content[0])

    opts := make(map[string]interface{})
    opts["schema"] = schema

    rule := model.Rule{
        Given: path,
        Then: &model.RuleAction{
            Field:           "smell",
            Function:        "schema",
            FunctionOptions: opts,
        },
        Description: "schema must be valid",
    }

    ctx := model.RuleFunctionContext{
        RuleAction: model.CastToRuleAction(rule.Then),
        Rule:       &rule,
        Options:    opts,
        Given:      rule.Given,
    }

    def := Schema{}
    res := def.RunRule(nodes, ctx)

    assert.Len(t, res, 1)
    assert.Equal(t, "schema must be valid: expected integer, but got string", res[0].Message)

}

func testGenerateJSONSchema(node *yaml.Node) *highBase.Schema {
    sch := lowBase.Schema{}
    _ = low.BuildModel(node, &sch)
    _ = sch.Build(node, nil)
    highSch := highBase.NewSchema(&sch)
    return highSch
}

func TestOpenAPISchema_InvalidSchemaBoolean(t *testing.T) {

    yml := `smell:
  stank: not a bool`

    path := "$"

    nodes, _ := utils.FindNodes([]byte(yml), path)

    props := make(map[string]*highBase.Schema)
    props["stank"] = &highBase.Schema{
        Type: []string{utils.BooleanLabel},
    }

    opts := make(map[string]interface{})

    validate := `type: object
properties:
  stank:
    type: boolean`

    var n yaml.Node
    _ = yaml.Unmarshal([]byte(validate), &n)

    schema := testGenerateJSONSchema(n.Content[0])

    opts["schema"] = schema

    rule := model.Rule{
        Given: path,
        Then: &model.RuleAction{
            Field:           "smell",
            Function:        "schema",
            FunctionOptions: opts,
        },
        Description: "schema must be valid",
    }

    ctx := model.RuleFunctionContext{
        RuleAction: model.CastToRuleAction(rule.Then),
        Rule:       &rule,
        Options:    opts,
        Given:      rule.Given,
    }

    def := Schema{}
    res := def.RunRule(nodes, ctx)

    assert.Len(t, res, 1)
    assert.Equal(t, "schema must be valid: expected boolean, but got string", res[0].Message)

}

func TestOpenAPISchema_MissingFieldForceValidation(t *testing.T) {

    yml := `eliminate:
  cyberhacks: not a bool`

    path := "$"

    nodes, _ := utils.FindNodes([]byte(yml), path)

    props := make(map[string]*highBase.Schema)
    props["stank"] = &highBase.Schema{
        Type: []string{utils.BooleanLabel},
    }

    opts := make(map[string]interface{})

    validate := `type: object
properties:
  stank:
    type: boolean`

    var n yaml.Node
    _ = yaml.Unmarshal([]byte(validate), &n)

    schema := testGenerateJSONSchema(n.Content[0])

    opts["schema"] = schema
    opts["forceValidation"] = true

    rule := model.Rule{
        Given: path,
        Then: &model.RuleAction{
            Field:           "lolly",
            Function:        "schema",
            FunctionOptions: opts,
        },
        Description: "schema must be valid",
    }

    ctx := model.RuleFunctionContext{
        RuleAction: model.CastToRuleAction(rule.Then),
        Rule:       &rule,
        Options:    opts,
        Given:      rule.Given,
    }

    def := Schema{}
    res := def.RunRule(nodes, ctx)

    assert.Len(t, res, 1)
    assert.Equal(t, "schema must be valid: `lolly`, is missing and is required", res[0].Message)

}
