package utils

import (
	"encoding/json"
	"fmt"
	"github.com/quobix/vaccum/model"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

const (
	OpenApi3 = "openapi"
	OpenApi2 = "swagger"
	AsyncApi = "asyncapi"
)

func ExtractSpecInfo(spec []byte) (*model.SpecInfo, error) {
	var parsedSpec map[string]interface{}
	specVersion := &model.SpecInfo{}
	err := yaml.Unmarshal(spec, &parsedSpec)
	if err != nil {
		// yaml failed, try JSON
		err = json.Unmarshal(spec, &parsedSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
		}
		specVersion.Version = "json"
	} else {
		specVersion.SpecFormat = "yaml"
	}

	// check for specific keys
	if parsedSpec[OpenApi3] != nil {
		specVersion.SpecType = OpenApi3
		specVersion.Version = parseVersionTypeData(parsedSpec[OpenApi3])
	}
	if parsedSpec[OpenApi2] != nil {
		specVersion.SpecType = OpenApi2
		specVersion.Version = parseVersionTypeData(parsedSpec[OpenApi2])
	}
	if parsedSpec[AsyncApi] != nil {
		specVersion.SpecType = AsyncApi
		specVersion.Version = parseVersionTypeData(parsedSpec[AsyncApi])
	}

	if specVersion.SpecType == "" {
		specVersion.SpecType = "unknown specification type, unsupported schema"
		specVersion.Version = "-1"
	}
	return specVersion, nil
}

// FindNodes will find a node based on JSONPath.
func FindNodes(yamlData []byte, jsonPath string) ([]*yaml.Node, error) {
	jsonPath = FixContext(jsonPath)

	var node yaml.Node
	yaml.Unmarshal(yamlData, &node)

	path, err := yamlpath.NewPath(jsonPath)
	if err != nil {
		return nil, err
	} else {
		results, err := path.Find(&node)
		if err != nil {
			return nil, err
		}
		return results, nil
	}
}

// FixContext will clean up a JSONpath string to be correctly traversable.
func FixContext(context string) string {

	tokens := strings.Split(context, ".")
	var cleaned = []string{}
	for i, t := range tokens {

		if v, err := strconv.Atoi(t); err == nil {

			if v < 200 { // codes start here
				if cleaned[i-1] != "" {
					cleaned[i-1] += fmt.Sprintf("[%v]", t)
				}
			} else {
				cleaned = append(cleaned, t)
			}
			continue
		}
		cleaned = append(cleaned, strings.ReplaceAll(t, "(root)", "$"))

	}
	return strings.Join(cleaned, ".")
}

// IsJSON will tell you if a string is JSON or not.
func IsJSON(testString string) bool {
	if testString == "" {
		return false
	}
	var n interface{}
	err := json.Unmarshal([]byte(testString), &n)
	if err != nil {
		return false
	}
	return true
}

// IsYAML will tell you if a string is YAML or not.
func IsYAML(testString string) bool {
	if testString == "" {
		return false
	}
	var n interface{}
	err := yaml.Unmarshal([]byte(testString), &n)
	if err != nil {
		return false
	}
	return true
}

func parseVersionTypeData(d interface{}) string {
	switch d.(type) {
	case int:
		return strconv.Itoa(d.(int))
	case float64:
		return strconv.FormatFloat(d.(float64), 'f', 2, 32)
	case bool:
		if d.(bool) {
			return "true"
		}
		return "false"
	case []string:
		return "multiple versions detected"
	}
	return fmt.Sprintf("%v", d)
}
