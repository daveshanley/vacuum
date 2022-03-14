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

const (
	severityError        = "error"
	severityWarn         = "warn"
	severityInfo         = "info"
	CategoryExamples     = "examples"
	CategoryOperations   = "operations"
	CategoryInfo         = "into"
	CategoryDescriptions = "descriptions"
	CategorySchemas      = "schemas"
	CategorySecurity     = "security"
	CategoryTags         = "tags"
	CategoryValidation   = "validation"
)

//go:embed schemas/ruleset.schema.json
var rulesetSchema string

var RuleCategories = make(map[string]*RuleCategory)

func init() {
	RuleCategories[CategoryExamples] = &RuleCategory{
		Id:   CategoryExamples,
		Name: "Examples",
		Description: "Examples help consumers understand how API calls should look. They are really important for" +
			"automated tooling for mocking and testing. These rules check examples have been added to component schemas, " +
			"parameters and operations. These rules also check that examples match the schema and types provided.",
	}
	RuleCategories[CategoryOperations] = &RuleCategory{
		Id:   CategoryOperations,
		Name: "Operations",
		Description: "Operations are the core of the contract, they define paths and HTTP methods. These rules check" +
			" operations have been well constructed, looks for operationId, parameter, schema and return types in depth.",
	}
	RuleCategories[CategoryInfo] = &RuleCategory{
		Id:   CategoryInfo,
		Name: "Contract Information",
		Description: "The info object contains licencing, contact, authorship details and more. Checks to confirm " +
			"required details have been completed.",
	}
	RuleCategories[CategoryDescriptions] = &RuleCategory{
		Id:   CategoryDescriptions,
		Name: "Descriptions",
		Description: "Documentation is really important, in OpenAPI, just about everything can and should have a " +
			"description. This set of rules checks for absent descriptions, poor quality descriptions (copy/paste)," +
			" or short descriptions.",
	}
	RuleCategories[CategorySchemas] = &RuleCategory{
		Id:   CategorySchemas,
		Name: "Schemas",
		Description: "Schemas are how request bodies and response payloads are defined. They define the data going in " +
			"and the data flowing out of an operation. These rules check for structural validity, checking types, checking" +
			"required fields and validating correct use of structures.",
	}
	RuleCategories[CategorySecurity] = &RuleCategory{
		Id:   CategorySecurity,
		Name: "Security",
		Description: "Security plays a central role in RESTful APIs. These rules make sure that the correct definitions" +
			"have been used and put in the right places.",
	}
	RuleCategories[CategoryTags] = &RuleCategory{
		Id:   CategoryTags,
		Name: "Tags",
		Description: "Tags are used as meta-data for operations. They are mainly used by tooling as a taxonomy mechanism" +
			" to build navigation, search and more. Tags are important as they help consumers navigate the contract when " +
			"using documentation, testing, code generation or analysis tools.",
	}
	RuleCategories[CategoryValidation] = &RuleCategory{
		Id:   CategoryValidation,
		Name: "Validation",
		Description: "Validation rules make sure that certain characters or patterns have not been used that may cause" +
			"issues when rendering in different types of applications.",
	}
}

type RuleCategory struct {
	Id          string
	Name        string
	Description string
}

// RuleFunctionContext defines a RuleAction, Rule and Options for a RuleFunction being run.
type RuleFunctionContext struct {
	RuleAction *RuleAction
	Rule       *Rule
	Given      interface{} // path/s being used by rule.
	Options    interface{}
}

// RuleFunctionResult describes a failure with linting after being run through a rule
type RuleFunctionResult struct {
	Message   string
	StartNode *yaml.Node
	EndNode   *yaml.Node
	Path      string
	Rule      *Rule
}

// TODO: Start here in the morning, we're going to want to be able to sort, calculate severity and categories.
// TODO: think about a super structure that contains all the sorting and filtering mechanisms.

// RuleResultSet contains all the results found during a linting run, and all the methods required to
// filter, sort and calculate counts.
type RuleResultSet struct {
	Results     []*RuleFunctionResult
	warnCount   int
	errorCount  int
	infoCount   int
	categoryMap map[*RuleCategory][]*RuleFunctionResult
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
	Description  string        `json:"description"`
	Given        interface{}   `json:"given"`
	Formats      []string      `json:"formats"`
	Resolved     bool          `json:"resolved"`
	Recommended  bool          `json:"recommended"`
	Type         string        `json:"type"`
	Severity     string        `json:"severity"`
	Then         interface{}   `json:"then"`
	RuleCategory *RuleCategory `json:"-"`
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

// NewRuleResultSet will encapsulate a set of results into a set, that can then be queried.
// the function will create pointers to results, instead of copying them again.
func NewRuleResultSet(results []RuleFunctionResult) *RuleResultSet {
	// use pointers for speed down the road, we don't need to keep copying this data.
	var pointerResults []*RuleFunctionResult
	for _, res := range results {
		n := res
		pointerResults = append(pointerResults, &n)

	}
	return &RuleResultSet{
		Results:     pointerResults,
		categoryMap: make(map[*RuleCategory][]*RuleFunctionResult),
	}
}

// GetErrorCount will return the number of errors returned by the rule results.
func (rr *RuleResultSet) GetErrorCount() int {
	if rr.errorCount > 0 {
		return rr.errorCount
	} else {
		rr.errorCount = getCount(rr, severityError)
		return rr.errorCount
	}
}

// GetWarnCount will return the number of warnings returned by the rule results.
func (rr *RuleResultSet) GetWarnCount() int {
	if rr.warnCount > 0 {
		return rr.warnCount
	} else {
		rr.warnCount = getCount(rr, severityWarn)
		return rr.warnCount
	}
}

// GetInfoCount will return the number of warnings returned by the rule results.
func (rr *RuleResultSet) GetInfoCount() int {
	if rr.infoCount > 0 {
		return rr.infoCount
	} else {
		rr.infoCount = getCount(rr, severityInfo)
		return rr.infoCount
	}
}

// GetResultsByRuleCategory will return results filtered by the supplied category
func (rr *RuleResultSet) GetResultsByRuleCategory(category string) []*RuleFunctionResult {

	// check for seen state.
	if RuleCategories[category] != nil && rr.categoryMap[RuleCategories[category]] != nil {
		return rr.categoryMap[RuleCategories[category]]
	}

	var results []*RuleFunctionResult
	for _, result := range rr.Results {
		if result.Rule.RuleCategory.Id == category {
			results = append(results, result)
		}
	}
	if RuleCategories[category] != nil && len(results) > 0 {
		rr.categoryMap[RuleCategories[category]] = results
	}
	return results
}

func getCount(rr *RuleResultSet, severity string) int {
	c := 0
	for _, res := range rr.Results {
		if res.Rule.Severity == severity {
			c++
		}
	}
	return c
}
