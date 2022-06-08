package model

import (
	_ "embed" // embedding is not supported by golint,
	"encoding/json"
	"github.com/daveshanley/vacuum/model/reports"
	"gopkg.in/yaml.v3"
	"regexp"
)

const (
	severityError        = "error"
	severityWarn         = "warn"
	severityInfo         = "info"
	severityHint         = "hint"
	CategoryExamples     = "examples"
	CategoryOperations   = "operations"
	CategoryInfo         = "information"
	CategoryDescriptions = "descriptions"
	CategorySchemas      = "schemas"
	CategorySecurity     = "security"
	CategoryTags         = "tags"
	CategoryValidation   = "validation"
	CategoryAll          = "all"
)

type RuleCategory struct {
	Id          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

// RuleFunctionContext defines a RuleAction, Rule and Options for a RuleFunction being run.
type RuleFunctionContext struct {
	RuleAction *RuleAction
	Rule       *Rule
	Given      interface{} // path/s being used by rule.
	Options    interface{}
	Index      *SpecIndex
	SpecInfo   *SpecInfo
}

// RuleFunctionResult describes a failure with linting after being run through a rule
type RuleFunctionResult struct {
	Message      string        `json:"message" yaml:"message"`
	Range        reports.Range `json:"range" yaml:"range"`
	Path         string        `json:"path" yaml:"path"`
	RuleId       string        `json:"ruleId" yaml:"ruleId"`
	RuleSeverity string        `json:"ruleSeverity" yaml:"ruleSeverity"`
	Rule         *Rule         `json:"-" yaml:"-"`
	StartNode    *yaml.Node    `json:"-" yaml:"-"`
	EndNode      *yaml.Node    `json:"-" yaml:"-"`
}

// RuleResultSet contains all the results found during a linting run, and all the methods required to
// filter, sort and calculate counts.
type RuleResultSet struct {
	Results     []*RuleFunctionResult                   `json:"results" yaml:"results"`
	warnCount   int                                     `json:"warningCount" yaml:"warningCount"`
	errorCount  int                                     `json:"errorCount" yaml:"errorCount"`
	infoCount   int                                     `json:"infoCount" yaml:"infoCount"`
	categoryMap map[*RuleCategory][]*RuleFunctionResult `json:"-" yaml:"-"`
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
	Id                 string         `json:"id,omitempty" yaml:"id,omitempty"`
	Description        string         `json:"description,omitempty" yaml:"description,omitempty"`
	Given              interface{}    `json:"given,omitempty" yaml:"given,omitempty"`
	Formats            []string       `json:"formats,omitempty" yaml:"formats,omitempty"`
	Resolved           bool           `json:"resolved,omitempty" yaml:"resolved,omitempty"`
	Recommended        bool           `json:"recommended,omitempty" yaml:"recommended,omitempty"`
	Type               string         `json:"type,omitempty" yaml:"type,omitempty"`
	Severity           string         `json:"severity,omitempty" yaml:"severity,omitempty"`
	Then               interface{}    `json:"then,omitempty" yaml:"then,omitempty"`
	PrecompiledPattern *regexp.Regexp `json:"-"` // regex is slow.
	RuleCategory       *RuleCategory  `json:"-"`
	Name               string         `json:"-"`
	HowToFix           string         `json:"-"`
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

// GetSeverityAsIntValue will return the severity state of the rule as an integer. If the severity is not known
// then -1 is returned.
func (r *Rule) GetSeverityAsIntValue() int {
	switch r.Severity {
	case severityError:
		return 0
	case severityWarn:
		return 1
	case severityInfo:
		return 2
	case severityHint:
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
