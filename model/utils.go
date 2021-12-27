package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

const (
	OpenApi3 = "openapi"
	OpenApi2 = "swagger"
	AsyncApi = "asyncapi"
)

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
	if parsedSpec[OpenApi3] != nil {
		specVersion.SpecType = OpenApi3
		version, majorVersion := parseVersionTypeData(parsedSpec[OpenApi3])

		// double check for the right version, people mix this up.
		if majorVersion < 3 {
			return nil, errors.New("spec is defined as an openapi spec, but is using a swagger (2.0), or unknown version")
		}
		specVersion.Version = version
	}
	if parsedSpec[OpenApi2] != nil {
		specVersion.SpecType = OpenApi2
		version, majorVersion := parseVersionTypeData(parsedSpec[OpenApi2])

		// I am not certain this edge-case is very frequent, but let's make sure we handle it anyway.
		if majorVersion > 2 {
			return nil, errors.New("spec is defined as a swagger (openapi 2.0) spec, but is an openapi 3 or unknown version")
		}
		specVersion.Version = version
	}
	if parsedSpec[AsyncApi] != nil {
		specVersion.SpecType = AsyncApi
		version, majorVersion := parseVersionTypeData(parsedSpec[AsyncApi])

		// so far there is only 2 as a major release of AsyncAPI
		if majorVersion > 2 {
			return nil, errors.New("spec is defined as asyncapi, but has a major version that is invalid")
		}
		specVersion.Version = version

	}

	if specVersion.SpecType == "" {
		return nil, errors.New("spec type not supported by vaccum, sorry")
	}
	return specVersion, nil
}

func parseVersionTypeData(d interface{}) (string, int) {
	switch d.(type) {
	case int:
		return strconv.Itoa(d.(int)), d.(int)
	case float64:
		return strconv.FormatFloat(d.(float64), 'f', 2, 32), int(d.(float64))
	case bool:
		if d.(bool) {
			return "true", 0
		}
		return "false", 0
	case []string:
		return "multiple versions detected", 0
	}
	r := []rune(strings.TrimSpace(fmt.Sprintf("%v", d)))
	return string(r), int(r[0]) - '0'
}

func BuildFunctionResult(key, message string, value interface{}) RuleFunctionResult {
	return RuleFunctionResult{
		Message: fmt.Sprintf("'%s' %s '%v'", key, message, value),
	}
}

func ValidateRuleFunctionContextAgainstSchema(ruleFunction RuleFunction, ctx RuleFunctionContext) (bool, []string) {
	valid := true
	var errs []string
	schema := ruleFunction.GetSchema()
	numProps := 0

	if options, ok := ctx.Options.(map[string]interface{}); ok {
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
		}
	}

	// check if this schema has required properties, then check them out.
	if len(schema.Required) > 0 {
		var missingProps []string
		for _, req := range schema.Required {
			found := false
			for _, prop := range schema.Properties {
				if prop.Name == req {
					found = true
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
	return valid, errs
}
