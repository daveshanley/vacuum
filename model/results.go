package model

import (
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/libopenapi/datamodel"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// RuleResultsForCategory boils down result statistics for a linting category
type RuleResultsForCategory struct {
	RuleResults []*RuleCategoryResult
	Category    *RuleCategory
}

// RuleCategoryResult contains metrics for a rule scored as part of a category.
type RuleCategoryResult struct {
	Rule      *Rule
	Results   []*RuleFunctionResult
	Seen      int
	Health    int
	Errors    int
	Warnings  int
	Info      int
	Hints     int
	Truncated bool
}

// Len returns the length of the results
func (rr *RuleResultsForCategory) Len() int { return len(rr.RuleResults) }

// Less determines which result has the lower severity (errors bubble to top)
func (rr *RuleResultsForCategory) Less(i, j int) bool {
	return rr.RuleResults[i].Rule.GetSeverityAsIntValue() < rr.RuleResults[j].Rule.GetSeverityAsIntValue()
}

// Swap will re-sort a result if it's in the wrong order.
func (rr *RuleResultsForCategory) Swap(i, j int) {
	rr.RuleResults[i], rr.RuleResults[j] = rr.RuleResults[j], rr.RuleResults[i]
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
	rrs := &RuleResultSet{
		Results:     pointerResults,
		categoryMap: make(map[*RuleCategory][]*RuleFunctionResult),
	}
	rrs.GetErrorCount()
	rrs.GetInfoCount()
	rrs.GetWarnCount()
	return rrs
}

// NewRuleResultSetPointer will encapsulate a set of results into a set, that can then be queried.
// the function will create pointers to results, instead of copying them again.
func NewRuleResultSetPointer(results []*RuleFunctionResult) *RuleResultSet {
	// use pointers for speed down the road, we don't need to keep copying this data.
	var pointerResults []*RuleFunctionResult
	for _, res := range results {
		n := res
		pointerResults = append(pointerResults, n)

	}
	return &RuleResultSet{
		Results:     pointerResults,
		categoryMap: make(map[*RuleCategory][]*RuleFunctionResult),
	}
}

var paramRegex = regexp.MustCompile(`(\w+)\['([\w{}/:_-]+)'`)
var indexRegex = regexp.MustCompile(`(\w+)\[(\d+)]`)

// GenerateSpectralReport will return a Spectral compatible report structure, easily serializable
func (rr *RuleResultSet) GenerateSpectralReport(source string) []reports.SpectralReport {

	var report []reports.SpectralReport
	for _, result := range rr.Results {

		sev := 1
		switch result.Rule.Severity {
		case SeverityError:
			sev = 0
		case SeverityInfo:
			sev = 2
		case SeverityHint:
			sev = 3
		}

		resultRange := reports.Range{
			Start: reports.RangeItem{
				Line: result.StartNode.Line,
				Char: result.StartNode.Column,
			},
			End: reports.RangeItem{
				Line: result.EndNode.Line,
				Char: result.EndNode.Column,
			},
		}
		var path []string
		pathArr := strings.Split(result.Path, ".")
		for _, pItem := range pathArr {
			if pItem == "" {
				path = append(path, "..") // https://github.com/daveshanley/vacuum/issues/583
			}
			if pItem != "$" {

				p := paramRegex.FindStringSubmatch(pItem)
				i := indexRegex.FindStringSubmatch(pItem)
				if len(p) == 3 {
					path = append(path, p[1], p[2])
					continue
				}
				if len(i) == 3 {
					path = append(path, i[1], i[2])
					continue
				}

				path = append(path, pItem)
			}
		}

		report = append(report, reports.SpectralReport{
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
	if rr.ErrorCount > 0 {
		return rr.ErrorCount
	} else {
		rr.ErrorCount = getCount(rr, SeverityError)
		return rr.ErrorCount
	}
}

// GetWarnCount will return the number of warnings returned by the rule results.
func (rr *RuleResultSet) GetWarnCount() int {
	if rr.WarnCount > 0 {
		return rr.WarnCount
	} else {
		rr.WarnCount = getCount(rr, SeverityWarn)
		return rr.WarnCount
	}
}

// GetInfoCount will return the number of warnings returned by the rule results.
func (rr *RuleResultSet) GetInfoCount() int {
	if rr.InfoCount > 0 {
		return rr.InfoCount
	} else {
		rr.InfoCount = getCount(rr, SeverityInfo)
		return rr.InfoCount
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
		case SeverityError:
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
		case SeverityWarn:
			filtered = append(filtered, cat)
		}
		// by default rules with no severity, are warnings.
		if cat.Rule.Severity == "" {
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
		case SeverityInfo:
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
		case SeverityHint:
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
			rrfc.RuleResults = append(rrfc.RuleResults, rcr)
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
	for x, catResult := range rrfc.RuleResults {
		if len(catResult.Results) > limit {
			rrfc.RuleResults[x].Results = rrfc.RuleResults[x].Results[:limit]
			rrfc.RuleResults[x].Truncated = true
		}
	}
	return rrfc
}

func getCount(rr *RuleResultSet, severity string) int {
	c := 0
	for _, res := range rr.Results {
		if res.Rule != nil {
			if res.Rule.Severity == severity {
				c++
			}
			// if there is no severity, mark it as a warning by default.
			if res.Rule.Severity == "" && severity == SeverityWarn {
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

// PrepareForSerialization will create a new Range for start and end nodes as well as pre-render code.
// When saving a vacuum report, this will be required, so the report can be re-constructed later without
// the original spec being required.
func (rr *RuleResultSet) PrepareForSerialization(info *datamodel.SpecInfo) {

	var wg sync.WaitGroup
	if rr == nil || info == nil {
		return
	}
	wg.Add(len(rr.Results))

	data := strings.Split(string(*info.SpecBytes), "\n")

	var prep = func(result *RuleFunctionResult, wg *sync.WaitGroup, data []string) {

		var start, end reports.RangeItem

		if result.StartNode != nil {
			start = reports.RangeItem{
				Line: result.StartNode.Line,
				Char: result.StartNode.Column,
			}
		}
		if result.EndNode != nil {
			end = reports.RangeItem{
				Line: result.EndNode.Line,
				Char: result.EndNode.Column,
			}
		}

		result.Range = reports.Range{
			Start: start,
			End:   end,
		}
		if result.Rule != nil {
			result.RuleId = result.Rule.Id
			result.RuleSeverity = result.Rule.Severity
		}
		wg.Done()
	}

	for _, res := range rr.Results {
		go prep(res, &wg, data)
	}

	wg.Wait()
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
	if rr.Results[i].StartNode != nil && rr.Results[j].StartNode != nil {
		if rr.Results[i].StartNode.Line < rr.Results[j].StartNode.Line {
			return true
		}
		if rr.Results[i].StartNode.Line > rr.Results[j].StartNode.Line {
			return false
		}
		return rr.Results[i].RuleId < rr.Results[j].RuleId
	}
	return false
}

// Swap will re-sort a result if it's in the wrong order.
func (rr *RuleResultSet) Swap(i, j int) { rr.Results[i], rr.Results[j] = rr.Results[j], rr.Results[i] }
