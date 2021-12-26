package utils

import (
	"encoding/json"
	"fmt"
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

func ConvertInterfaceIntoStringMap(context interface{}) map[string]string {
	if context != nil {
		if v, ok := context.(map[string]string); ok {
			return v
		}
	}
	return nil
}

func ConvertInterfaceIntoStringMapStringSlice(context interface{}) map[string][]string {
	if context != nil {
		if v, ok := context.(map[string][]string); ok {
			return v
		}
	}
	return nil
}

func ConvertInterfaceIntoStringArrayMap(context interface{}) map[string][]string {
	if context != nil {
		if v, ok := context.(map[string][]string); ok {
			return v
		}
	}
	return nil
}

func ConvertInterfaceIntoIntMap(context interface{}) map[string]int {
	if context != nil {
		if v, ok := context.(map[string]int); ok {
			return v
		}
	}
	return nil
}

func ConvertInterfaceArrayToStringArray(raw interface{}) []string {
	if vals, ok := raw.([]interface{}); ok {
		s := make([]string, len(vals))
		for i, v := range vals {
			s[i] = fmt.Sprint(v)
		}
		return s
	} else {
		return nil
	}
}

func ExtractValueFromInterfaceMap(name string, raw interface{}) interface{} {

	if propMap, ok := raw.(map[string]interface{}); ok {
		if props, ok := propMap[name].([]interface{}); ok {
			return props
		}
	}
	return nil
}

func FindFirstKeyNode(key string, nodes []*yaml.Node) (*yaml.Node, *yaml.Node) {

	for i, v := range nodes {
		if key == v.Value {
			return v, nodes[i+1] // next node is what we need.
		}
		if len(v.Content) > 0 {
			return FindFirstKeyNode(key, v.Content)
		}
	}
	return nil, nil
}

func FindKeyNode(key string, nodes []*yaml.Node) (*yaml.Node, *yaml.Node) {

	for i, v := range nodes {
		if key == v.Value {
			return v, nodes[i+1] // next node is what we need.
		}
	}
	return nil, nil
}

func IsNodeMap(node *yaml.Node) bool {
	if node.Tag == "!!map" {
		return true
	}
	return false
}

func IsNodeArray(node *yaml.Node) bool {
	if node.Tag == "!!seq" {
		return true
	}
	return false
}

func IsNodeStringValue(node *yaml.Node) bool {
	if node.Tag == "!!str" {
		return true
	}
	return false
}

func IsNodeIntValue(node *yaml.Node) bool {
	if node.Tag == "!!int" {
		return true
	}
	return false
}

func IsNodeFloatValue(node *yaml.Node) bool {
	if node.Tag == "!!float" {
		return true
	}
	return false
}

func FindAllKeyNodes(key string, nodes []*yaml.Node, foundNodes []*yaml.Node) []*yaml.Node {

	for i, v := range nodes {
		if key == v.Value {
			foundNodes = append(foundNodes, nodes[i+1])
			return foundNodes
		}
		if len(v.Content) > 0 {
			return FindAllKeyNodes(key, v.Content, foundNodes)
		}
	}
	return nil
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
	runes := []rune(strings.TrimSpace(testString))
	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		return true
	}
	return false
}

// IsYAML will tell you if a string is YAML or not.
func IsYAML(testString string) bool {
	if testString == "" {
		return false
	}
	if IsJSON(testString) {
		return false
	}
	var n interface{}
	err := yaml.Unmarshal([]byte(testString), &n)
	if err != nil {
		return false
	}
	_, err = yaml.Marshal(n)
	if err != nil {
		return false
	}
	return true
}

func ConvertYAMLtoJSON(yamlData []byte) ([]byte, error) {
	var decodedYaml map[string]interface{}
	err := yaml.Unmarshal(yamlData, &decodedYaml)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(decodedYaml)
	if err != nil {
		return nil, err
	}
	return jsonData, nil

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
