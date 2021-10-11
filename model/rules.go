package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
	"strings"
)

type RuleFunctionContext struct {
}

type RuleFunctionResult struct {
	Message string
	Path    string
}

type RuleFunction interface {
	RunRule(input string, options map[string]interface{}, context RuleFunctionContext)
}

type RuleAction struct {
	Field           string                 `json:"field"`
	FunctionName    string                 `json:"function"`
	FunctionOptions map[string]interface{} `json:"functionOptions"`
}

type Rule struct {
	Description string     `json:"description"`
	Given       string     `json:"given"`
	Formats     []string   `json:"formats"`
	Resolved    bool       `json:"resolved"`
	Recommended bool       `json:"recommended"`
	Severity    int        `json:"severity"`
	Then        RuleAction `json:"then"`
}

type RuleSet struct {
	DocumentationURI string          `json:"documentationUrl"`
	Formats          []string        `json:"formats"`
	Rules            map[string]Rule `json:"rules"`
	schemaLoader     gojsonschema.JSONLoader
}

func CreateRuleSetUsingJSON(jsonData []byte) (*RuleSet, error) {
	jsonString := string(jsonData)
	if !IsJSON(jsonString) {
		return nil, errors.New("data is not JSON")
	}

	jsonLoader := gojsonschema.NewStringLoader(jsonString)
	schemaLoader, err := LoadRulesetSchema()
	if err != nil {
		return nil, err
	}

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

		return nil, errors.New(fmt.Sprintf("rules not valid: %s", buf.String()))
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

func LoadRulesetSchema() (gojsonschema.JSONLoader, error) {

	schemaMain, err := ioutil.ReadFile("schemas/ruleset.schema.json")
	if err != nil {
		return nil, err
	}
	return gojsonschema.NewStringLoader(string(schemaMain)), nil
}
