package model

import (
	_ "embed" // embedding is not supported by golint,
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/utils"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	"strings"
)

//go:embed schemas/ruleset.schema.json
var rulesetSchema string

// RuleFunctionContext defines a RuleAction, Rule and Options for a RuleFunction being run.
type RuleFunctionContext struct {
	RuleAction *RuleAction
	Rule       *Rule
	Options    interface{}
}

// RuleFunctionResult describes a failure with linting after being run through a rule
type RuleFunctionResult struct {
	Message string
	Path    string
}

// RuleFunction is any compatible structure that can be used to run vacuum rules.
type RuleFunction interface {
	RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult
	GetSchema() RuleFunctionSchema
}

// RuleAction is what to do, on what field, and what options are to be used.
type RuleAction struct {
	Field           string      `json:"field"`
	Function        string      `json:"function"`
	FunctionOptions interface{} `json:"functionOptions"`
}

// Rule is a structure that represents a rule as part of a ruleset.
type Rule struct {
	Description string      `json:"description"`
	Given       interface{} `json:"given"`
	Formats     []string    `json:"formats"`
	Resolved    bool        `json:"resolved"`
	Recommended bool        `json:"recommended"`
	Severity    string      `json:"severity"`
	Then        interface{} `json:"then"`
}

// RuleFunctionProperty is used by RuleFunctionSchema to describe the functionOptions a Rule accepts
type RuleFunctionProperty struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RuleFunctionSchema describes the name, required properties and a slice of RuleFunctionProperty properties.
type RuleFunctionSchema struct {
	Name          string                 `json:"name,omitempty"`
	Required      []string               `json:"required,omitempty"`
	RequiresField bool                   `json:"requiresField,omitempty"`
	Properties    []RuleFunctionProperty `json:"properties"`
	MinProperties int                    `json:"minProperties,omitempty"`
	MaxProperties int                    `json:"maxProperties,omitempty"`
	ErrorMessage  string                 `json:"errorMessage,omitempty"`
}

// GetPropertyDescription is a shortcut method for extracting the description of a property by its name.
func (rfs RuleFunctionSchema) GetPropertyDescription(name string) string {
	for _, prop := range rfs.Properties {
		if prop.Name == name {
			return prop.Description
		}
	}
	return ""
}

// ToJSON render out a rule to JSON.
func (r Rule) ToJSON() string {
	d, _ := json.Marshal(r)
	return string(d)
}

// RuleSet represents a collection of Rule definitions.
type RuleSet struct {
	DocumentationURI string           `json:"documentationUrl"`
	Formats          []string         `json:"formats"`
	Rules            map[string]*Rule `json:"rules"`
	schemaLoader     gojsonschema.JSONLoader
}

// CreateRuleSetUsingJSON will create a new RuleSet instance from a JSON byte array
func CreateRuleSetUsingJSON(jsonData []byte) (*RuleSet, error) {
	jsonString := string(jsonData)
	if !utils.IsJSON(jsonString) {
		return nil, errors.New("data is not JSON")
	}

	jsonLoader := gojsonschema.NewStringLoader(jsonString)
	schemaLoader := LoadRulesetSchema()

	// check blob is a valid contract, before creating ruleset.
	res, err := gojsonschema.Validate(schemaLoader, jsonLoader)
	if err != nil {
		return nil, err
	}

	if !res.Valid() {
		var buf strings.Builder
		for _, e := range res.Errors() {
			buf.WriteString(fmt.Sprintf("%s (line),", e.Description()))
		}

		return nil, fmt.Errorf("rules not valid: %s", buf.String())
	}

	// unmarshal JSON into new RuleSet
	rs := &RuleSet{}
	err = json.Unmarshal(jsonData, rs)
	if err != nil {
		return nil, err
	}

	// save our loaded schema for later.
	rs.schemaLoader = schemaLoader
	return rs, nil
}

// LoadRulesetSchema creates a new JSON Schema loader for the RuleSet schema.
func LoadRulesetSchema() gojsonschema.JSONLoader {
	return gojsonschema.NewStringLoader(rulesetSchema)
}
