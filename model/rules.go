package model

import (
	_ "embed" // embedding is not supported by golint,
	"encoding/json"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"log/slog"
	"regexp"
	"time"
)

const (
	SeverityError        = "error"
	SeverityWarn         = "warn"
	SeverityInfo         = "info"
	SeverityHint         = "hint"
	CategoryExamples     = "examples"
	CategoryOperations   = "operations"
	CategoryInfo         = "information"
	CategoryDescriptions = "descriptions"
	CategorySchemas      = "schemas"
	CategorySecurity     = "security"
	CategoryTags         = "tags"
	CategoryValidation   = "validation"
	CategoryOWASP        = "OWASP"
	CategoryAll          = "all"
)

type RuleCategory struct {
	Id          string `json:"id" yaml:"id"`                             // The category ID
	Name        string `json:"name,omitempty" yaml:"name"`               // The name of the category
	Description string `json:"description,omitempty" yaml:"description"` // What is the category all about?
}

// RuleFunctionContext defines a RuleAction, Rule and Options for a RuleFunction being run.
type RuleFunctionContext struct {
	RuleAction *RuleAction         `json:"ruleAction,omitempty" yaml:"ruleAction,omitempty"` // A reference to the action defined configured by the rule
	Rule       *Rule               `json:"rule,omitempty" yaml:"rule,omitempty"`             // A reference to the Rule being used for the function
	Given      interface{}         `json:"given,omitempty" yaml:"given,omitempty"`           // Path/s being used by rule, multiple paths can be used
	Options    interface{}         `json:"options,omitempty" yaml:"options,omitempty"`       // Function options
	SpecInfo   *datamodel.SpecInfo `json:"specInfo,omitempty" yaml:"specInfo,omitempty"`     // A reference to all specification information for the spec being parsed.
	Index      *index.SpecIndex    `json:"-" yaml:"-"`                                       // A reference to the index created for the spec being parsed
	Document   libopenapi.Document `json:"-" yaml:"-"`                                       // A reference to the document being parsed
	DrDocument *model.DrDocument   `json:"-" yaml:"-"`                                       // A high level, more powerful representation of the document being parsed. Powered by the doctor.
	Logger     *slog.Logger        `json:"-" yaml:"-"`                                       // Custom logger
}

// RuleFunctionResult describes a failure with linting after being run through a rule
type RuleFunctionResult struct {
	Message      string            `json:"message" yaml:"message"`                   // What failed and why?
	Range        reports.Range     `json:"range" yaml:"range"`                       // Where did it happen?
	Path         string            `json:"path" yaml:"path"`                         // the JSONPath to where it can be found, the first is extracted if there are multiple.
	Paths        []string          `json:"paths,omitempty" yaml:"paths,omitempty"`   // the JSONPath(s) to where it can be found, if there are multiple.
	RuleId       string            `json:"ruleId" yaml:"ruleId"`                     // The ID of the rule
	RuleSeverity string            `json:"ruleSeverity" yaml:"ruleSeverity"`         // the severity of the rule used
	Origin       *index.NodeOrigin `json:"origin,omitempty" yaml:"origin,omitempty"` // Where did the result come from (source)?
	Rule         *Rule             `json:"-" yaml:"-"`                               // The rule used
	StartNode    *yaml.Node        `json:"-" yaml:"-"`                               // Start of the violation
	EndNode      *yaml.Node        `json:"-" yaml:"-"`                               // end of the violation
	Timestamp    *time.Time        `json:"-" yaml:"-"`                               // When the result was created.

	// ModelContext may or may nor be populated, depending on the rule used and the context of the rule. If it is
	// populated, then this is a reference to the model that fired the rule. (not currently used yet)
	ModelContext any `json:"-" yaml:"-"`
}

// IgnoredItems is a map of the rule ID to an array of violation paths
type IgnoredItems map[string][]string

// RuleResultSet contains all the results found during a linting run, and all the methods required to
// filter, sort and calculate counts.
type RuleResultSet struct {
	Results     []*RuleFunctionResult                   `json:"results,omitempty" yaml:"results,omitempty"` // All the results!
	WarnCount   int                                     `json:"warningCount" yaml:"warningCount"`           // Total warnings
	ErrorCount  int                                     `json:"errorCount" yaml:"errorCount"`               // Total errors
	InfoCount   int                                     `json:"infoCount" yaml:"infoCount"`                 // Total info
	categoryMap map[*RuleCategory][]*RuleFunctionResult `json:"-" yaml:"-"`
}

// RuleFunction is any compatible structure that can be used to run vacuum rules.
type RuleFunction interface {
	RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult // The place where logic is run
	GetSchema() RuleFunctionSchema                                                // How to use the function and its details.
	GetCategory() string                                                          // Returns the category the function is a part of.
}

// RuleAction is what to do, on what field, and what options are to be used.
type RuleAction struct {
	Field           string      `json:"field,omitempty" yaml:"field,omitempty"`
	Function        string      `json:"function,omitempty" yaml:"function,omitempty"`
	FunctionOptions interface{} `json:"functionOptions,omitempty" yaml:"functionOptions,omitempty"`
}

// Rule is a structure that represents a rule as part of a ruleset.
type Rule struct {
	Id                 string         `json:"id,omitempty" yaml:"id,omitempty"`
	Description        string         `json:"description,omitempty" yaml:"description,omitempty"`
	Message            string         `json:"message,omitempty" yaml:"message,omitempty"`
	Given              interface{}    `json:"given,omitempty" yaml:"given,omitempty"`
	Formats            []string       `json:"formats,omitempty" yaml:"formats,omitempty"`
	Resolved           bool           `json:"resolved,omitempty" yaml:"resolved,omitempty"`
	Recommended        bool           `json:"recommended,omitempty" yaml:"recommended,omitempty"`
	Type               string         `json:"type,omitempty" yaml:"type,omitempty"`
	Severity           string         `json:"severity,omitempty" yaml:"severity,omitempty"`
	Then               interface{}    `json:"then,omitempty" yaml:"then,omitempty"`
	PrecompiledPattern *regexp.Regexp `json:"-" yaml:"-"` // regex is slow.
	RuleCategory       *RuleCategory  `json:"category,omitempty" yaml:"category,omitempty"`
	Name               string         `json:"-" yaml:"-"`
	HowToFix           string         `json:"howToFix,omitempty" yaml:"howToFix,omitempty"`
}

// RuleFunctionProperty is used by RuleFunctionSchema to describe the functionOptions a Rule accepts
type RuleFunctionProperty struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

// RuleFunctionSchema describes the name, required properties and a slice of RuleFunctionProperty properties.
type RuleFunctionSchema struct {
	Name          string                 `json:"name,omitempty" yaml:"name,omitempty"`                   // The name of this function **important**
	Required      []string               `json:"required,omitempty" yaml:"required,omitempty"`           // List of all required properties to be set
	RequiresField bool                   `json:"requiresField,omitempty" yaml:"requiresField,omitempty"` // 'field' must be used with this function
	Properties    []RuleFunctionProperty `json:"properties,omitempty" yaml:"properties,omitempty"`       // all properties to be passed to the function
	MinProperties int                    `json:"minProperties,omitempty" yaml:"minProperties,omitempty"` // Minimum number of properties
	MaxProperties int                    `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"` // Maximum number of properties
	ErrorMessage  string                 `json:"errorMessage,omitempty" yaml:"errorMessage,omitempty"`   // Error message to be used in case of failed validartion.
}

// GetSeverityAsIntValue will return the severity state of the rule as an integer. If the severity is not known
// then -1 is returned.
func (r *Rule) GetSeverityAsIntValue() int {
	switch r.Severity {
	case SeverityError:
		return 0
	case SeverityWarn:
		return 1
	case SeverityInfo:
		return 2
	case SeverityHint:
		return 3
	}
	return -1
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
