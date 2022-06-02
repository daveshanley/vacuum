package model

import (
	_ "embed" // embedding is not supported by golint,
	"encoding/json"
	"gopkg.in/yaml.v3"
	"math"
	"regexp"
	"sort"
	"strings"
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
	Message   string
	StartNode *yaml.Node
	EndNode   *yaml.Node
	Path      string
	Rule      *Rule
}

// SpectralReport represents a model that can be deserialized into a spectral compatible output.
type SpectralReport struct {
	Code     string        `json:"code" yaml:"code"`         // the rule that was run
	Path     []string      `json:"path" yaml:"path"`         // the path to the item, broken down into a slice
	Message  string        `json:"message" yaml:"message"`   // the result message
	Severity int           `json:"severity" yaml:"severity"` // the severity reported
	Range    SpectralRange `json:"range" yaml:"range"`       // the location of the issue in the spec.
	Source   string        `json:"source" yaml:"source"`     // the source of the report.
}

// SpectralRange indicates the start and end of a report item
type SpectralRange struct {
	Start SpectralRangeItem `json:"start" yaml:"start"`
	End   SpectralRangeItem `json:"end" yaml:"end"`
}

// SpectralRangeItem indicates the line and character of a range.
type SpectralRangeItem struct {
	Line int `json:"line" yaml:"line"`
	Char int `json:"character" yaml:"character"`
}

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
	Id                string         `json:"-"`
	Description       string         `json:"description"`
	Given             interface{}    `json:"given"`
	Formats           []string       `json:"formats"`
	Resolved          bool           `json:"resolved"`
	Recommended       bool           `json:"recommended"`
	Type              string         `json:"type"`
	Severity          string         `json:"severity"`
	Then              interface{}    `json:"then"`
	PrecomiledPattern *regexp.Regexp `json:"-"` // regex is slow.
	RuleCategory      *RuleCategory  `json:"category"`
	Name              string         `json:"name"`
	HowToFix          string         `json:"howToFix"`
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

// RuleResultsForCategory boils down result statistics for a linting category
type RuleResultsForCategory struct {
	Rules     []*RuleCategoryResult
	Category  *RuleCategory
	Truncated bool
}

// RuleCategoryResult contains metrics for a rule scored as part of a category.
type RuleCategoryResult struct {
	Rule     *Rule
	Results  []*RuleFunctionResult
	Seen     int
	Health   int
	Errors   int
	Warnings int
	Info     int
	Hints    int
}

// Len returns the length of the results
func (rr *RuleResultsForCategory) Len() int { return len(rr.Rules) }

// Less determines which result has the lower severity (errors bubble to top)
func (rr *RuleResultsForCategory) Less(i, j int) bool {
	return rr.Rules[i].Rule.GetSeverityAsIntValue() < rr.Rules[j].Rule.GetSeverityAsIntValue()
}

// Swap will re-sort a result if it's in the wrong order.
func (rr *RuleResultsForCategory) Swap(i, j int) { rr.Rules[i], rr.Rules[j] = rr.Rules[j], rr.Rules[i] }

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

// GenerateSpectralReport will return a Spectral compatible report structure, easily serializable
func (rr *RuleResultSet) GenerateSpectralReport(source string) []SpectralReport {

	var report []SpectralReport
	for _, result := range rr.Results {

		sev := 1
		switch result.Rule.Severity {
		case "error":
			sev = 0
		case "info":
			sev = 2
		case "hint":
			sev = 3
		}

		resultRange := SpectralRange{
			Start: SpectralRangeItem{
				Line: result.StartNode.Line,
				Char: result.StartNode.Column,
			},
			End: SpectralRangeItem{
				Line: result.EndNode.Line,
				Char: result.EndNode.Column,
			},
		}
		var path []string
		pathArr := strings.Split(result.Path, ".")
		for _, pItem := range pathArr {
			if pItem != "$" {
				path = append(path, pItem)
			}
		}

		report = append(report, SpectralReport{
			Code:     result.Rule.Id,
			Path:     path,
			Message:  result.Message,
			Severity: sev,
			Range:    resultRange,
			Source:   source,
		})
	}
	return report
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
		if result.Rule != nil && result.Rule.RuleCategory != nil {

			// if the category is 'all' then, dump in the lot, regardless.
			if category == CategoryAll {
				results = append(results, result)
				continue
			}

			if result.Rule.RuleCategory.Id == category {
				results = append(results, result)
			}
		}
	}
	if RuleCategories[category] != nil && len(results) > 0 {
		rr.categoryMap[RuleCategories[category]] = results
	}
	return results
}

// GetErrorsByRuleCategory will return all results with an error level severity from rule category.
func (rr *RuleResultSet) GetErrorsByRuleCategory(category string) []*RuleFunctionResult {
	var filtered []*RuleFunctionResult
	allCats := rr.GetResultsByRuleCategory(category)
	for _, cat := range allCats {
		switch cat.Rule.Severity {
		case severityError:
			filtered = append(filtered, cat)
		}
	}
	return filtered
}

// GetWarningsByRuleCategory will return all results with a warning level severity from rule category.
func (rr *RuleResultSet) GetWarningsByRuleCategory(category string) []*RuleFunctionResult {
	var filtered []*RuleFunctionResult
	allCats := rr.GetResultsByRuleCategory(category)
	for _, cat := range allCats {
		switch cat.Rule.Severity {
		case severityWarn:
			filtered = append(filtered, cat)
		}
	}
	return filtered
}

// GetInfoByRuleCategory will return all results with an info level severity from rule category.
func (rr *RuleResultSet) GetInfoByRuleCategory(category string) []*RuleFunctionResult {
	var filtered []*RuleFunctionResult
	allCats := rr.GetResultsByRuleCategory(category)
	for _, cat := range allCats {
		switch cat.Rule.Severity {
		case severityInfo:
			filtered = append(filtered, cat)
		}
	}
	return filtered
}

// GetHintByRuleCategory will return all results with hint level severity from rule category.
func (rr *RuleResultSet) GetHintByRuleCategory(category string) []*RuleFunctionResult {
	var filtered []*RuleFunctionResult
	allCats := rr.GetResultsByRuleCategory(category)
	for _, cat := range allCats {
		switch cat.Rule.Severity {
		case severityHint:
			filtered = append(filtered, cat)
		}
	}
	return filtered
}

// GetRuleResultsForCategory will return all rules that returned results during linting, complete with pre
// compiled statistics for easy indexing.
func (rr *RuleResultSet) GetRuleResultsForCategory(category string) *RuleResultsForCategory {
	cat := RuleCategories[category]
	if cat == nil {
		return nil
	}

	rrfc := RuleResultsForCategory{}
	catResults := rr.GetResultsByRuleCategory(category)
	rrfc.Category = cat

	seenRules := make(map[*Rule]bool)
	seenRuleMap := make(map[string]*RuleCategoryResult)

	for _, res := range catResults {
		var rcr *RuleCategoryResult
		if !seenRules[res.Rule] {
			rcr = &RuleCategoryResult{
				Rule: res.Rule,
			}
			rrfc.Rules = append(rrfc.Rules, rcr)
			seenRuleMap[res.Rule.Id] = rcr
			seenRules[res.Rule] = true
		} else {
			rcr = seenRuleMap[res.Rule.Id]
		}
		rcr.Results = append(rcr.Results, res)
		rcr.Seen = rcr.Seen + 1
	}
	return &rrfc
}

// GetResultsForCategoryWithLimit is identical to GetRuleResultsForCategory, except for the fact that there
// will be a limit on the number of results returned, defined by the limit arg. This is used by the HTML report
// to stop gigantic files from being created, iterating through all the results.
func (rr *RuleResultSet) GetResultsForCategoryWithLimit(category string, limit int) *RuleResultsForCategory {
	rrfc := rr.GetRuleResultsForCategory(category)
	for x, catResult := range rrfc.Rules {
		if len(catResult.Results) > limit {
			rrfc.Rules[x].Results = rrfc.Rules[x].Results[:limit]
			rrfc.Truncated = true
		}
	}
	return rrfc
}

func getCount(rr *RuleResultSet, severity string) int {
	c := 0
	for _, res := range rr.Results {
		if res.Rule != nil && res.Rule.Severity != "" {
			if res.Rule.Severity == severity {
				c++
			}
		}
	}
	return c
}

// CalculateCategoryHealth checks how many errors and warnings a category has generated and determine
// a value between 0 and 100, 0 being errors fired, 100 being no warnings and no errors.
func (rr *RuleResultSet) CalculateCategoryHealth(category string) int {

	errs := rr.GetErrorsByRuleCategory(category)
	warnings := rr.GetWarningsByRuleCategory(category)
	info := rr.GetInfoByRuleCategory(category)

	errorCount := len(errs)
	warningCount := len(warnings)
	infoCount := len(info)

	totalScore := 0.0
	totalScore += float64(errorCount) * 10.0
	totalScore += float64(warningCount) * 0.5
	totalScore += float64(infoCount) * 0.01

	health := 100.0
	if totalScore >= 100 {
		health = 0
	} else {
		health = health - math.RoundToEven(totalScore)
	}

	return int(health)
}

// SortResultsByLineNumber will re-order the results by line number. This is a destructive sort,
// Once the results are sorted, they are permanently sorted.
func (rr *RuleResultSet) SortResultsByLineNumber() []*RuleFunctionResult {
	sort.Sort(rr)
	return rr.Results
}

// Len returns the length of the results
func (rr *RuleResultSet) Len() int { return len(rr.Results) }

// Less determines which result has the lower line number
func (rr *RuleResultSet) Less(i, j int) bool {
	return rr.Results[i].StartNode.Line < rr.Results[j].StartNode.Line
}

// Swap will re-sort a result if it's in the wrong order.
func (rr *RuleResultSet) Swap(i, j int) { rr.Results[i], rr.Results[j] = rr.Results[j], rr.Results[i] }
