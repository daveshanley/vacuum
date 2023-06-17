package model

import (
	_ "embed"
	"fmt"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

const (
	OAS2  = "oas2"
	OAS3  = "oas3"
	OAS31 = "oas3_1"
)

var OAS3_1Format = []string{OAS31}
var OAS3Format = []string{OAS3}
var OAS3AllFormat = []string{OAS3, OAS31}
var OAS2Format = []string{OAS2}
var AllFormats = []string{OAS3, OAS31, OAS2}

// BuildFunctionResult will create a RuleFunctionResult from a key, message and value.
// Deprecated: use BuildFunctionResultWithDescription instead.
func BuildFunctionResult(key, message string, value interface{}) RuleFunctionResult {
	return RuleFunctionResult{
		Message: fmt.Sprintf("'%s' %s '%v'", key, message, value),
	}
}

// BuildFunctionResultWithDescription will create a RuleFunctionResult from a description, key, message and value.
func BuildFunctionResultWithDescription(desc, key, message string, value interface{}) RuleFunctionResult {
	return RuleFunctionResult{
		Message: fmt.Sprintf("%s: '%s' %s '%v'", desc, key, message, value),
	}
}

// BuildFunctionResultString will create a RuleFunctionResult from a string already parsed into a message.
func BuildFunctionResultString(message string) RuleFunctionResult {
	return RuleFunctionResult{
		Message: message,
	}
}

// ValidateRuleFunctionContextAgainstSchema will perform run-time validation against a rule to ensure that
// options being passed in are acceptable and meet the needs of the Rule schema
func ValidateRuleFunctionContextAgainstSchema(ruleFunction RuleFunction, ctx RuleFunctionContext) (bool, []string) {
	valid := true
	var errs []string
	schema := ruleFunction.GetSchema()
	numProps := 0

	if options, ok := ctx.Options.(map[string]interface{}); ok {
		numProps = countPropsInterface(options, numProps)
	}

	if options, ok := ctx.Options.(map[string]string); ok {
		numProps = countPropsString(options, numProps)
	}

	if schema.MinProperties > 0 && numProps < schema.MinProperties {
		valid = false
		errs = append(errs, fmt.Sprintf("%s: minimum property number not met (%v)",
			schema.ErrorMessage, schema.MinProperties))
	}

	if schema.MaxProperties > 0 && numProps > schema.MaxProperties {
		valid = false
		errs = append(errs, fmt.Sprintf("%s: maximum number (%v) of properties exceeded. '%v' provided.",
			schema.ErrorMessage, schema.MaxProperties, numProps))
	}

	// check if the function requires a field or not, and check if it's been set
	if schema.RequiresField && ctx.RuleAction.Field == "" {
		errs = append(errs, fmt.Sprintf("'%s' requires a 'field' value to be set", schema.Name))
	}

	// check if this schema has required properties, then check them out.
	if len(schema.Required) > 0 {
		var missingProps []string
		for _, req := range schema.Required {
			found := false

			if options, ok := ctx.Options.(map[string]interface{}); ok {
				for k := range options {
					if k == req {
						found = true
					}
				}
			}
			if options, ok := ctx.Options.(map[string]string); ok {
				for k := range options {
					if k == req {
						found = true
					}
				}
			}

			if !found {
				missingProps = append(missingProps, req)
			}
		}
		if len(missingProps) > 0 {
			valid = false
			for _, mProp := range missingProps {
				errs = append(errs, fmt.Sprintf("%s: missing required property: %s (%s)",
					schema.ErrorMessage, mProp, schema.GetPropertyDescription(mProp)))
			}
		}
	}

	return valid, errs
}

func countPropsInterface(options map[string]interface{}, numProps int) int {
	for _, v := range options {
		if stringVal, ok := v.(string); ok {
			if strings.Contains(stringVal, ",") {
				split := strings.Split(stringVal, ",")
				numProps += len(split)
			} else {
				numProps++
			}
		}
		if _, ok := v.(int); ok {
			numProps++
		}
		if _, ok := v.(bool); ok {
			numProps++
		}
		if arr, ok := v.([]interface{}); ok {
			numProps += len(arr)
		}
	}
	return numProps
}

func countPropsString(options map[string]string, numProps int) int {
	for _, v := range options {
		if strings.Contains(v, ",") {
			split := strings.Split(v, ",")
			numProps += len(split)
		} else {
			numProps++
		}
	}
	return numProps
}

// CastToRuleAction is a utility function to cast an unknown structure into a RuleAction.
// useful for when building rules or testing out concepts.
func CastToRuleAction(action interface{}) *RuleAction {
	if action == nil {
		return nil
	}
	if ra, ok := action.(*RuleAction); ok {
		return ra
	}
	return nil
}

// MapPathAndNodesToResults will map the same start/end nodes with the same path.
func MapPathAndNodesToResults(path string, startNode, endNode *yaml.Node, results []RuleFunctionResult) []RuleFunctionResult {
	var mapped []RuleFunctionResult
	for _, result := range results {
		result.Path = path
		result.StartNode = startNode
		result.EndNode = endNode
		mapped = append(mapped, result)
	}

	return mapped
}

// CompileRegex attempts to compile the provided `Pattern` from the ruleset. If it fails, returns nil
// and adds an error to the result set. Any rule using this should then return the results if there is no *Regexp
// returned.
func CompileRegex(context RuleFunctionContext, pattern string, results *[]RuleFunctionResult) *regexp.Regexp {
	compiledRegex, err := regexp.Compile(pattern)
	if err != nil {
		*results = append(*results, RuleFunctionResult{
			Message: fmt.Sprintf("Error: cannot run rule, pattern `%s` cannot be compiled", pattern),
			Rule:    context.Rule,
		})
		return nil
	}
	return compiledRegex
}
