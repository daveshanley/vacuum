package model

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/utils"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

const (
	OAS2  = "oas2"
	OAS3  = "oas3"
	OAS31 = "oas3_1"
)

//go:embed schemas/oas3-schema.yaml
var OpenAPI3SchemaData string

//go:embed schemas/swagger2-schema.yaml
var OpenAPI2SchemaData string

var OAS3_1Format = []string{OAS31}
var OAS3Format = []string{OAS3}
var OAS3AllFormat = []string{OAS3, OAS31}
var OAS2Format = []string{OAS2}
var AllFormats = []string{OAS3, OAS31, OAS2}

// ExtractSpecInfo will look at a supplied OpenAPI specification, and return a *SpecInfo pointer, or an error
// if the spec cannot be parsed correctly.
func ExtractSpecInfo(spec []byte) (*SpecInfo, error) {

	var parsedSpec yaml.Node

	specVersion := &SpecInfo{}
	specVersion.jsonParsingChannel = make(chan bool)

	// set original bytes
	specVersion.SpecBytes = &spec

	runes := []rune(strings.TrimSpace(string(spec)))
	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		specVersion.SpecFileType = "json"
	} else {
		specVersion.SpecFileType = "yaml"
	}

	err := yaml.Unmarshal(spec, &parsedSpec)
	if err != nil {
		return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
	}

	specVersion.RootNode = &parsedSpec

	_, openAPI3 := utils.FindKeyNode(utils.OpenApi3, parsedSpec.Content)
	_, openAPI2 := utils.FindKeyNode(utils.OpenApi2, parsedSpec.Content)
	_, asyncAPI := utils.FindKeyNode(utils.AsyncApi, parsedSpec.Content)

	parseJSON := func(bytes []byte, spec *SpecInfo) {
		var jsonSpec map[string]interface{}

		// no point in worrying about errors here, extract JSON friendly format.
		// run in a separate thread, don't block.

		if spec.SpecType == utils.OpenApi3 {
			openAPI3JSON, _ := utils.ConvertYAMLtoJSON([]byte(OpenAPI3SchemaData))
			spec.APISchema = gojsonschema.NewStringLoader(string(openAPI3JSON))
		}
		if spec.SpecType == utils.OpenApi2 {
			openAPI2JSON, _ := utils.ConvertYAMLtoJSON([]byte(OpenAPI2SchemaData))
			spec.APISchema = gojsonschema.NewStringLoader(string(openAPI2JSON))
		}

		if utils.IsYAML(string(bytes)) {
			yaml.Unmarshal(bytes, &jsonSpec)
			jsonData, _ := json.Marshal(jsonSpec)
			spec.SpecJSONBytes = &jsonData
			spec.SpecJSON = &jsonSpec
		} else {
			json.Unmarshal(bytes, &jsonSpec)
			spec.SpecJSONBytes = &bytes
			spec.SpecJSON = &jsonSpec
		}
		spec.jsonParsingChannel <- true
		close(spec.jsonParsingChannel)
	}
	// check for specific keys
	if openAPI3 != nil {
		specVersion.SpecType = utils.OpenApi3
		version, majorVersion := parseVersionTypeData(openAPI3.Value)

		// parse JSON
		go parseJSON(spec, specVersion)

		// double check for the right version, people mix this up.
		if majorVersion < 3 {
			specVersion.Error = errors.New("spec is defined as an openapi spec, but is using a swagger (2.0), or unknown version")
			return specVersion, specVersion.Error
		}
		specVersion.Version = version
		specVersion.SpecFormat = OAS3
	}
	if openAPI2 != nil {
		specVersion.SpecType = utils.OpenApi2
		version, majorVersion := parseVersionTypeData(openAPI2.Value)

		// parse JSON
		go parseJSON(spec, specVersion)

		// I am not certain this edge-case is very frequent, but let's make sure we handle it anyway.
		if majorVersion > 2 {
			specVersion.Error = errors.New("spec is defined as a swagger (openapi 2.0) spec, but is an openapi 3 or unknown version")
			return specVersion, specVersion.Error
		}
		specVersion.Version = version
		specVersion.SpecFormat = OAS2
	}
	if asyncAPI != nil {
		specVersion.SpecType = utils.AsyncApi
		version, majorVersion := parseVersionTypeData(asyncAPI.Value)

		// parse JSON
		go parseJSON(spec, specVersion)

		// so far there is only 2 as a major release of AsyncAPI
		if majorVersion > 2 {
			specVersion.Error = errors.New("spec is defined as asyncapi, but has a major version that is invalid")
			return specVersion, specVersion.Error
		}
		specVersion.Version = version
		// TODO: format for AsyncAPI.

	}

	if specVersion.SpecType == "" {

		// parse JSON
		go parseJSON(spec, specVersion)

		specVersion.Error = errors.New("spec type not supported by vacuum, sorry")
		return specVersion, specVersion.Error
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
	}
	arr = values.([]interface{})

	results := make(map[string]string)
	for _, v := range arr {
		switch v.(type) {
		case string:
			if valType != "string" {
				results[v.(string)] = fmt.Sprintf("enum value '%v' is a "+
					"string, but it's defined as a '%v'", v, valType)
			}
		case int64:
			if valType != "integer" && valType != "number" {
				results[fmt.Sprintf("%v", v)] = fmt.Sprintf("enum value '%v' is a "+
					"integer, but it's defined as a '%v'", v, valType)
			}
		case int:
			if valType != "integer" && valType != "number" {
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

// CheckEnumForDuplicates will check an array of nodes to check if there are any duplicates.
func CheckEnumForDuplicates(seq []*yaml.Node) []*yaml.Node {
	var res []*yaml.Node
	seen := make(map[string]*yaml.Node)

	for _, enum := range seq {
		if seen[enum.Value] != nil {
			res = append(res, enum)
			continue
		}
		seen[enum.Value] = enum
	}
	return res
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
