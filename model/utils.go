package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

// ExtractSpecInfo will look at a supplied OpenAPI specification, and return a *SpecInfo pointer, or an error
// if the spec cannot be parsed correctly.\
func ExtractSpecInfo(spec []byte) (*SpecInfo, error) {
	var parsedSpec map[string]interface{}
	specVersion := &SpecInfo{}
	runes := []rune(strings.TrimSpace(string(spec)))
	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		// try JSON
		err := json.Unmarshal(spec, &parsedSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
		}
		specVersion.Version = "json"
	} else {
		// try YAML
		err := yaml.Unmarshal(spec, &parsedSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
		}
		specVersion.Version = "yaml"
	}

	// check for specific keys
	if parsedSpec[utils.OpenApi3] != nil {
		specVersion.SpecType = utils.OpenApi3
		version, majorVersion := parseVersionTypeData(parsedSpec[utils.OpenApi3])

		// double check for the right version, people mix this up.
		if majorVersion < 3 {
			return nil, errors.New("spec is defined as an openapi spec, but is using a swagger (2.0), or unknown version")
		}
		specVersion.Version = version
	}
	if parsedSpec[utils.OpenApi2] != nil {
		specVersion.SpecType = utils.OpenApi2
		version, majorVersion := parseVersionTypeData(parsedSpec[utils.OpenApi2])

		// I am not certain this edge-case is very frequent, but let's make sure we handle it anyway.
		if majorVersion > 2 {
			return nil, errors.New("spec is defined as a swagger (openapi 2.0) spec, but is an openapi 3 or unknown version")
		}
		specVersion.Version = version
	}
	if parsedSpec[utils.AsyncApi] != nil {
		specVersion.SpecType = utils.AsyncApi
		version, majorVersion := parseVersionTypeData(parsedSpec[utils.AsyncApi])

		// so far there is only 2 as a major release of AsyncAPI
		if majorVersion > 2 {
			return nil, errors.New("spec is defined as asyncapi, but has a major version that is invalid")
		}
		specVersion.Version = version

	}

	if specVersion.SpecType == "" {
		return nil, errors.New("spec type not supported by vacuum, sorry")
	}
	return specVersion, nil
}

func parseVersionTypeData(d interface{}) (string, int) {
	switch dat := d.(type) {
	case int:
		return strconv.Itoa(dat), dat
	case float64:
		return strconv.FormatFloat(dat, 'f', 2, 32), int(dat)
	case bool:
		if dat {
			return "true", 0
		}
		return "false", 0
	case []string:
		return "multiple versions detected", 0
	}
	r := []rune(strings.TrimSpace(fmt.Sprintf("%v", d)))
	return string(r), int(r[0]) - '0'
}

// BuildFunctionResult will create a RuleFunctionResult from a key, message and value.
func BuildFunctionResult(key, message string, value interface{}) RuleFunctionResult {
	return RuleFunctionResult{
		Message: fmt.Sprintf("'%s' %s '%v'", key, message, value),
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

// AreValuesCorrectlyTyped will look through an array of unknown values and check they match
// against the supplied type as a string. The return value is empty if everything is OK, or it
// contains failures in the form of a value as a key and a message as to why it's not valid
func AreValuesCorrectlyTyped(valType string, values interface{}) map[string]string {
	var arr []interface{}
	if _, ok := values.([]interface{}); !ok {
		return nil
	} else {
		arr = values.([]interface{})
	}
	results := make(map[string]string)
	for _, v := range arr {
		switch v.(type) {
		case string:
			if valType != "string" {
				results[v.(string)] = fmt.Sprintf("enum value '%v' is a "+
					"string, but it's defined as a '%v'", v, valType)
			}
		case int64:
			if valType != "integer" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"integer, but it's defined as a '%v'", v, valType)
			}
		case float64:
			if valType != "number" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"number, but it's defined as a '%v'", v, valType)
			}
		case bool:
			if valType != "boolean" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"boolean, but it's defined as a '%v'", v, valType)
			}
		}
	}
	return results
}
