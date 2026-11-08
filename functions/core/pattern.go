// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/dlclark/regexp2/v2"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/utils"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

// Pattern is a rule that will match or not match (or both) a regular expression.
// This struct is stateless - all state is passed through function parameters.
type Pattern struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Pattern rule.
func (p Pattern) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "pattern",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "match",
				Description: "'pattern' requires a match",
			},
			{
				Name:        "notMatch",
				Description: "'pattern' needs something to not match against",
			},
		},
		MinProperties: 1,
		MaxProperties: 2,
		ErrorMessage:  "'pattern' needs 'match' or 'notMatch' function options being set to operate",
	}
}

// GetCategory returns the category of the Pattern rule.
func (p Pattern) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Pattern rule, based on supplied context and a supplied []*yaml.Node slice.
func (p Pattern) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	// extract match/notMatch from props - these are local variables, not struct state
	props := context.GetOptionsStringMap()
	match, hasMatch := props["match"]
	notMatch, hasNotMatch := props["notMatch"]
	if context.Contract != model.OpenLintV1Draft {
		hasMatch = match != ""
		hasNotMatch = notMatch != ""
	}

	// make sure we have something to look at
	if (!hasMatch && !hasNotMatch) || (context.Contract != model.OpenLintV1Draft && match == "" && notMatch == "") {
		return nil
	}

	// compile regexes once at the start - no caching needed since we compile per-invocation
	var matchRx, notMatchRx patternMatcher
	var matchErr, notMatchErr error

	if hasMatch {
		if context.Rule.PrecompiledPattern != nil && context.Contract != model.OpenLintV1Draft {
			matchRx.native = context.Rule.PrecompiledPattern
		} else {
			matchRx, matchErr = compilePattern(match, context)
		}
	}

	if hasNotMatch {
		// PrecompiledPattern is typically for 'match', compile notMatch separately
		notMatchRx, notMatchErr = compilePattern(notMatch, context)
	}

	var results []model.RuleFunctionResult

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	message := context.Rule.Message
	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	if context.RuleAction.Field != "" {
		for _, node := range nodes {
			if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
				node = node.Content[0]
			}
			if !utils.IsNodeMap(node) {
				continue
			}

			fieldResult := vacuumUtils.FindFieldPath(context.RuleAction.Field, node.Content, fieldLookupOptions(context, false))
			if fieldResult.Found && fieldResult.ValueNode != nil {
				locatedPath, allPaths, locatedObjects := locateExistingFieldPaths(&context, node, context.RuleAction.Field, fieldResult)
				results = append(results, p.validatePatternOnNode(
					fieldResult.ValueNode, locatedPath, allPaths, locatedObjects, message, ruleMessage,
					match, notMatch, matchRx, notMatchRx, matchErr, notMatchErr, context)...)
			}
		}
		return results
	}

	// if multiple patterns are being pulled in, unpack them
	if context.Contract != model.OpenLintV1Draft && len(nodes) == 1 && len(nodes[0].Content) > 0 {
		nodes = nodes[0].Content
	}

	// iterate through key-value pairs
	var currentField string
	for x, node := range nodes {
		if context.Contract == model.OpenLintV1Draft && (node == nil || node.Tag != "!!str") {
			continue
		}
		if utils.IsNodeMap(node) {
			continue
		}
		if x%2 == 0 {
			currentField = node.Value
			if context.RuleAction.Field != "" {
				continue
			}
		}
		if context.RuleAction.Field != "" && currentField != context.RuleAction.Field {
			continue
		}

		results = append(results, p.validatePatternOnNode(
			node, pathValue, nil, nil, message, ruleMessage,
			match, notMatch, matchRx, notMatchRx, matchErr, notMatchErr, context)...)
	}

	return results
}

// validatePatternOnNode checks both match and notMatch patterns on a node value.
// All parameters are passed explicitly - no struct state is used.
func (p Pattern) validatePatternOnNode(
	node *yaml.Node,
	locatedPath string,
	allPaths []string,
	locatedObjects []v3.Foundational,
	message, ruleMessage string,
	match, notMatch string,
	matchRx, notMatchRx patternMatcher,
	matchErr, notMatchErr error,
	context model.RuleFunctionContext,
) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if context.Contract == model.OpenLintV1Draft && node.Tag != "!!str" {
		return nil
	}

	if context.Arazzo != nil {
		locatedPath, allPaths, locatedObjects = locateNodePaths(&context, node)
	} else if locatedPath == "" {
		locatedPath, allPaths, locatedObjects = p.locateNode(node, context)
	}

	// check match pattern
	if matchRx.configured() || matchErr != nil {
		if matchErr != nil {
			results = append(results, p.buildRegexErrorResult(node, locatedPath, allPaths, locatedObjects, message, ruleMessage, match, matchErr, context))
		} else if matched, err := matchRx.matchString(node.Value); err != nil {
			if context.ReportError != nil {
				context.ReportError(err)
			}
			results = append(results, p.buildRegexErrorResult(node, locatedPath, allPaths, locatedObjects, message, ruleMessage, match, err, context))
		} else if !matched {
			results = append(results, p.buildPatternMismatchResult(node, locatedPath, allPaths, locatedObjects, message, ruleMessage, match, node.Value, context))
		}
	}

	// check notMatch pattern
	if notMatchRx.configured() || notMatchErr != nil {
		if notMatchErr != nil {
			results = append(results, p.buildRegexErrorResult(node, locatedPath, allPaths, locatedObjects, message, ruleMessage, notMatch, notMatchErr, context))
		} else if matched, err := notMatchRx.matchString(node.Value); err != nil {
			if context.ReportError != nil {
				context.ReportError(err)
			}
			results = append(results, p.buildRegexErrorResult(node, locatedPath, allPaths, locatedObjects, message, ruleMessage, notMatch, err, context))
		} else if matched {
			results = append(results, p.buildPatternMatchedResult(node, locatedPath, allPaths, locatedObjects, message, ruleMessage, notMatch, context))
		}
	}

	return results
}

// buildRegexErrorResult creates a result for regex compilation errors.
func (p Pattern) buildRegexErrorResult(
	node *yaml.Node,
	locatedPath string,
	allPaths []string,
	locatedObjects []v3.Foundational,
	message, ruleMessage, pattern string,
	err error,
	context model.RuleFunctionContext,
) model.RuleFunctionResult {
	result := model.RuleFunctionResult{
		Message: vacuumUtils.SuppliedOrDefault(message,
			model.GetStringTemplates().BuildRegexCompileErrorMessage(ruleMessage, pattern, err.Error())),
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	p.attachResultToModel(locatedObjects, &result)
	return result
}

// buildPatternMismatchResult creates a result when a value doesn't match the expected pattern.
func (p Pattern) buildPatternMismatchResult(
	node *yaml.Node,
	locatedPath string,
	allPaths []string,
	locatedObjects []v3.Foundational,
	message, ruleMessage, pattern, value string,
	context model.RuleFunctionContext,
) model.RuleFunctionResult {
	result := model.RuleFunctionResult{
		Message: vacuumUtils.SuppliedOrDefault(message,
			model.GetStringTemplates().BuildPatternMessage(ruleMessage, value, pattern)),
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	p.attachResultToModel(locatedObjects, &result)
	return result
}

// buildPatternMatchedResult creates a result when a value matches a notMatch pattern.
func (p Pattern) buildPatternMatchedResult(
	node *yaml.Node,
	locatedPath string,
	allPaths []string,
	locatedObjects []v3.Foundational,
	message, ruleMessage, pattern string,
	context model.RuleFunctionContext,
) model.RuleFunctionResult {
	result := model.RuleFunctionResult{
		Message: vacuumUtils.SuppliedOrDefault(message,
			model.GetStringTemplates().BuildPatternMatchMessage(ruleMessage, pattern)),
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	p.attachResultToModel(locatedObjects, &result)
	return result
}

// locateNode finds the location information for a node using DrDocument.
func (p Pattern) locateNode(node *yaml.Node, context model.RuleFunctionContext) (string, []string, []v3.Foundational) {
	locatedPath := givenPathValue(context.Given)
	var allPaths []string
	var locatedObjects []v3.Foundational

	if context.DrDocument != nil {
		var err error
		locatedObjects, err = context.DrDocument.LocateModel(node)
		if err == nil && locatedObjects != nil {
			for i, obj := range locatedObjects {
				if i == 0 {
					locatedPath = obj.GenerateJSONPath()
				}
				allPaths = append(allPaths, obj.GenerateJSONPath())
			}
		}
	}

	return locatedPath, allPaths, locatedObjects
}

// attachResultToModel attaches the result to the first located model if it accepts results.
func (p Pattern) attachResultToModel(locatedObjects []v3.Foundational, result *model.RuleFunctionResult) {
	if len(locatedObjects) > 0 {
		if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
			arr.AddRuleFunctionResult(v3.ConvertRuleResult(result))
		}
	}
}

type patternMatcher struct {
	native  *regexp.Regexp
	ecma    *regexp2.Regexp
	context context.Context
	unicode bool
	sticky  bool
}

func (m patternMatcher) configured() bool { return m.native != nil || m.ecma != nil }

func (m patternMatcher) matchString(value string) (bool, error) {
	if m.native != nil {
		return m.native.MatchString(value), nil
	}
	if m.context != nil {
		if err := m.context.Err(); err != nil {
			return false, err
		}
	}
	if m.unicode {
		if m.sticky {
			match, err := m.ecma.FindStringMatch(value)
			return match != nil && match.RuneIndex == 0, err
		}
		return m.ecma.MatchString(value)
	}
	units := make([]rune, 0, len(value))
	for _, r := range value {
		if r > 0xffff {
			high, low := utf16.EncodeRune(r)
			units = append(units, high, low)
		} else {
			units = append(units, r)
		}
	}
	if m.sticky {
		match, err := m.ecma.FindRunesMatch(units)
		return match != nil && match.RuneIndex == 0, err
	}
	return m.ecma.MatchRunes(units)
}

func compilePattern(pattern string, ctx model.RuleFunctionContext) (patternMatcher, error) {
	if ctx.Contract != model.OpenLintV1Draft {
		rx, err := regexp.Compile(pattern)
		if err != nil {
			return patternMatcher{}, err
		}
		return patternMatcher{native: rx}, nil
	}
	expression, options, sticky, err := openLintPattern(pattern)
	if err != nil {
		return patternMatcher{}, err
	}
	if err := validateOpenLintPatternGroups(expression); err != nil {
		return patternMatcher{}, err
	}
	unicode := options&regexp2.Unicode != 0
	if !unicode {
		var encoded strings.Builder
		for _, r := range expression {
			if r > 0xffff {
				high, low := utf16.EncodeRune(r)
				fmt.Fprintf(&encoded, `\u%04x\u%04x`, high, low)
			} else {
				encoded.WriteRune(r)
			}
		}
		expression = encoded.String()
	}
	rx, err := regexp2.Compile(expression, options)
	if err != nil {
		return patternMatcher{}, err
	}
	rx.MatchTimeout = time.Second
	if ctx.RuleTimeout > 0 && ctx.RuleTimeout < rx.MatchTimeout {
		rx.MatchTimeout = ctx.RuleTimeout
	}
	return patternMatcher{ecma: rx, context: ctx.ExecutionContext, unicode: unicode, sticky: sticky}, nil
}

func openLintPattern(pattern string) (string, regexp2.RegexOptions, bool, error) {
	options := regexp2.RegexOptions(regexp2.ECMAScript)
	if !strings.HasPrefix(pattern, "/") {
		return pattern, options, false, nil
	}
	end := strings.LastIndex(pattern[1:], "/") + 1
	if end == 0 {
		return pattern, options, false, nil
	}
	expression, flags := pattern[1:end], pattern[end+1:]
	seen := map[rune]bool{}
	sticky := false
	for _, flag := range flags {
		if seen[flag] {
			return "", 0, false, fmt.Errorf("duplicate regular expression flag %q", flag)
		}
		seen[flag] = true
		switch flag {
		case 'i':
			options |= regexp2.IgnoreCase
		case 'm':
			options |= regexp2.Multiline
		case 's':
			options |= regexp2.Singleline
		case 'u':
			options |= regexp2.Unicode
		case 'd', 'g': // Index captures and persistent lastIndex do not affect a fresh boolean test.
		case 'y':
			sticky = true
		default:
			return "", 0, false, fmt.Errorf("unsupported regular expression flag %q", flag)
		}
	}
	return expression, options, sticky, nil
}

// regexp2 also accepts .NET groups in ECMAScript mode. Reject those extensions
// before compilation, including groups after lookarounds and backreferences.
func validateOpenLintPatternGroups(expression string) error {
	inClass := false
	for i := 0; i < len(expression); i++ {
		switch expression[i] {
		case '\\':
			i++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '(':
			if inClass || i+1 >= len(expression) || expression[i+1] != '?' {
				continue
			}
			invalid := func() error {
				return fmt.Errorf("invalid JavaScript regular expression group at byte %d", i)
			}
			if i+2 >= len(expression) {
				return invalid()
			}
			switch expression[i+2] {
			case ':', '=', '!':
				continue
			case '<':
				if i+3 < len(expression) && (expression[i+3] == '=' || expression[i+3] == '!') {
					continue
				}
				end := strings.IndexByte(expression[i+3:], '>')
				if end < 0 || strings.Contains(expression[i+3:i+3+end], "-") {
					return invalid()
				}
				// The compiler validates the capture name itself.
				i += 3 + end
				continue
			}
			// ECMAScript permits scoped i, m and s modifiers, but no global
			// inline flags. A flag cannot appear twice or on both sides of '-'.
			var seen uint8
			minus, scoped := false, false
			for j := i + 2; j < len(expression); j++ {
				var bit uint8
				switch expression[j] {
				case 'i':
					bit = 1
				case 'm':
					bit = 2
				case 's':
					bit = 4
				case '-':
					if minus {
						return invalid()
					}
					minus = true
					continue
				case ':':
					scoped = seen != 0
				default:
					return invalid()
				}
				if scoped {
					break
				}
				if bit == 0 || seen&bit != 0 {
					return invalid()
				}
				seen |= bit
			}
			if !scoped {
				return invalid()
			}
		}
	}
	return nil
}
