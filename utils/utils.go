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
	// OpenApi3 is used by all OpenAPI 3+ docs
	OpenApi3 = "openapi"

	// OpenApi2 is used by all OpenAPI 2 docs, formerly known as swagger.
	OpenApi2 = "swagger"

	// AsyncApi is used by akk AsyncAPI docs, all versions.
	AsyncApi = "asyncapi"
)

// FindNodes will find a node based on JSONPath, it accepts raw yaml/json as input.
func FindNodes(yamlData []byte, jsonPath string) ([]*yaml.Node, error) {
	jsonPath = FixContext(jsonPath)

	var node yaml.Node
	yaml.Unmarshal(yamlData, &node)

	path, err := yamlpath.NewPath(jsonPath)
	if err != nil {
		return nil, err
	}
	results, err := path.Find(&node)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func FindLastChildNode(node *yaml.Node) *yaml.Node {
	s := len(node.Content) - 1
	if s < 0 {
		s = 0
	}
	if len(node.Content) > 0 && len(node.Content[s].Content) > 0 {
		return FindLastChildNode(node.Content[s])
	} else {
		if len(node.Content) > 0 {
			return node.Content[s]
		}
		return node
	}
}

// BuildPath will construct a JSONPath from a base and an array of strings.
func BuildPath(basePath string, segs []string) string {

	path := strings.Join(segs, ".")

	// trim that last period.
	if len(path) > 0 && path[len(path)-1] == '.' {
		path = path[:len(path)-1]
	}
	return fmt.Sprintf("%s.%s", basePath, path)
}

// FindNodesWithoutDeserializing will find a node based on JSONPath, without deserializing from yaml/json
func FindNodesWithoutDeserializing(node *yaml.Node, jsonPath string) ([]*yaml.Node, error) {
	jsonPath = FixContext(jsonPath)

	path, err := yamlpath.NewPath(jsonPath)
	if err != nil {
		return nil, err
	}
	results, err := path.Find(node)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ConvertInterfaceIntoIntMap will convert an unknown input into a integer map.
//func ConvertInterfaceIntoIntMap(context interface{}) map[string]int {
//	converted := make(map[string]int)
//	if context != nil {
//		if v, ok := context.(map[string]interface{}); ok {
//			for k, n := range v {
//				if s, okB := n.(int); okB {
//					converted[k] = s
//				}
//			}
//		}
//		if v, ok := context.(map[string]int); ok {
//			for k, n := range v {
//				converted[k] = n
//			}
//		}
//	}
//	return converted
//}

// ConvertInterfaceIntoStringMap will convert an unknown input into a string map.
func ConvertInterfaceIntoStringMap(context interface{}) map[string]string {
	converted := make(map[string]string)
	if context != nil {
		if v, ok := context.(map[string]interface{}); ok {
			for k, n := range v {
				if s, okB := n.(string); okB {
					converted[k] = s
				}
			}
		}
		if v, ok := context.(map[string]string); ok {
			for k, n := range v {
				converted[k] = n
			}
		}
		//if v, ok := context.(map[string]string); ok {
		//	for k, n := range v {
		//		converted[k] = n
		//	}
		//}
	}
	return converted
}

// ConvertInterfaceToStringArray will convert an unknown input map type into a string array/slice
func ConvertInterfaceToStringArray(raw interface{}) []string {
	if vals, ok := raw.(map[string]interface{}); ok {
		var s []string
		for _, v := range vals {
			if v, ok := v.([]interface{}); ok {
				for _, q := range v {
					s = append(s, fmt.Sprint(q))
				}
			}
		}
		return s
	}
	if vals, ok := raw.(map[string][]string); ok {
		var s []string
		for _, v := range vals {
			s = append(s, v...)
		}
		return s
	}
	return nil
}

// ConvertInterfaceArrayToStringArray will convert an unknown interface array type, into a string slice
func ConvertInterfaceArrayToStringArray(raw interface{}) []string {
	if vals, ok := raw.([]interface{}); ok {
		s := make([]string, len(vals))
		for i, v := range vals {
			s[i] = fmt.Sprint(v)
		}
		return s
	}
	if vals, ok := raw.([]string); ok {
		return vals
	}
	return nil
}

// ExtractValueFromInterfaceMap pulls out an unknown value from a map using a string key
func ExtractValueFromInterfaceMap(name string, raw interface{}) interface{} {

	if propMap, ok := raw.(map[string]interface{}); ok {
		if props, okn := propMap[name].([]interface{}); okn {
			return props
		} else {
			return propMap[name]
		}
	}
	if propMap, ok := raw.(map[string][]string); ok {
		return propMap[name]
	}

	return nil
}

// FindFirstKeyNode will locate the first key and value yaml.Node based on a key.
func FindFirstKeyNode(key string, nodes []*yaml.Node, depth int) (keyNode *yaml.Node, valueNode *yaml.Node) {
	if depth > 100 {
		return nil, nil
	}
	for i, v := range nodes {
		if key != "" && key == v.Value {
			if i+1 >= len(nodes) {
				return v, nodes[i] // next node is what we need.
			}
			return v, nodes[i+1] // next node is what we need.
		}
		if len(v.Content) > 0 {
			depth++
			x, y := FindFirstKeyNode(key, v.Content, depth)
			if x != nil && y != nil {
				return x, y
			}
		}
	}
	return nil, nil
}

// KeyNodeResult is a result from a KeyNodeSearch performed by the FindAllKeyNodesWithPath
type KeyNodeResult struct {
	KeyNode   *yaml.Node
	ValueNode *yaml.Node
	Parent    *yaml.Node
	Path      []yaml.Node
}

// KeyNodeSearch keeps a track of everything we have found on our adventure down the trees.
type KeyNodeSearch struct {
	Key             string
	Ignore          []string
	Results         []*KeyNodeResult
	AllowExtensions bool
}

// FindAllKeyNodesWithPath This function will search for a key node recursively. Once it finds the node, it will
// then update the KeyNodeSearch struct
func FindAllKeyNodesWithPath(search *KeyNodeSearch, parent *yaml.Node, searchNodes []*yaml.Node, foundPath []yaml.Node, depth int) {
	if depth > 100 {
		return
	}
	for i, v := range searchNodes {

		if v.Kind == yaml.MappingNode || v.Kind == yaml.SequenceNode {
			depth++
			FindAllKeyNodesWithPath(search, v, v.Content, foundPath, depth)

		}

		if v.Kind == yaml.ScalarNode {
			readMe := false

			if parent.Kind == yaml.MappingNode && i%2 == 0 {
				readMe = true
			}
			if parent.Kind == yaml.SequenceNode {
				readMe = true
			}

			if readMe && search.Key != "" && search.Key == v.Value {

				// we need to copy found path, it keeps messing up our results
				fp := make([]yaml.Node, len(foundPath))
				for x, foundPathNode := range foundPath {
					fp[x] = foundPathNode
				}

				for _, ignore := range search.Ignore {
					if len(foundPath) > 0 && foundPath[len(foundPath)-1].Value == ignore {
						continue
					}
				}
				res := KeyNodeResult{
					KeyNode:   searchNodes[i],
					ValueNode: searchNodes[i+1],
					Parent:    parent,
					Path:      fp,
				}
				search.Results = append(search.Results, &res)
				continue
			}

			if readMe && search.Key != "" && search.Key != v.Value {
				foundPath = append(foundPath, *v)
				continue

			}
		}
		if len(foundPath) > 0 {
			foundPath = foundPath[:len(foundPath)-1]
		}
	}
	if len(foundPath) > 0 {
		foundPath = foundPath[:len(foundPath)-1]
	}
}

// FindKeyNodeTop is a non-recursive search of top level nodes for a key, will not look at content.
// Returns the key and value
func FindKeyNodeTop(key string, nodes []*yaml.Node) (keyNode *yaml.Node, valueNode *yaml.Node) {

	for i, v := range nodes {
		if key == v.Value {
			return v, nodes[i+1] // next node is what we need.
		}
	}
	return nil, nil
}

// FindKeyNode is a non-recursive search of a *yaml.Node Content for a child node with a key.
// Returns the key and value
func FindKeyNode(key string, nodes []*yaml.Node) (keyNode *yaml.Node, valueNode *yaml.Node) {

	//numNodes := len(nodes)
	for i, v := range nodes {
		if i%2 == 0 && key == v.Value {
			return v, nodes[i+1] // next node is what we need.
		}
		for x, j := range v.Content {
			if key == j.Value {
				return v, v.Content[x+1] // next node is what we need.
			}
		}
	}
	return nil, nil
}

var ObjectLabel = "object"
var IntegerLabel = "integer"
var NumberLabel = "number"
var StringLabel = "string"
var BinaryLabel = "binary"
var ArrayLabel = "array"
var BooleanLabel = "boolean"
var SchemaSource = "https://json-schema.org/draft/2020-12/schema"
var SchemaId = "https://quobix.com/api/vacuum"

func MakeTagReadable(node *yaml.Node) string {
	switch node.Tag {
	case "!!map":
		return ObjectLabel
	case "!!seq":
		return ArrayLabel
	case "!!str":
		return StringLabel
	case "!!int":
		return IntegerLabel
	case "!!float":
		return NumberLabel
	case "!!bool":
		return BooleanLabel
	}
	return "unknown"
}

// IsNodeMap checks if the node is a map type
func IsNodeMap(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Tag == "!!map"
}

// IsNodeArray checks if a node is an array type
func IsNodeArray(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Tag == "!!seq"
}

// IsNodeStringValue checks if a node is a string value
func IsNodeStringValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Tag == "!!str"
}

// IsNodeIntValue will check if a node is an int value
func IsNodeIntValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Tag == "!!int"
}

// IsNodeFloatValue will check is a node is a float value.
func IsNodeFloatValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Tag == "!!float"
}

// IsNodeBoolValue will check is a node is a bool
func IsNodeBoolValue(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Tag == "!!bool"
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
	return err == nil
}

//TODO: Deprecate this and use imported library, this is more complex than it seems.
// ConvertYAMLtoJSON will do exactly what you think it will. It will deserialize YAML into serialized JSON.
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

// IsHttpVerb will check if an operation is valid or not.
func IsHttpVerb(verb string) bool {
	verbs := []string{"get", "post", "put", "patch", "delete", "options", "trace", "head"}
	for _, v := range verbs {
		if verb == v {
			return true
		}
	}
	return false
}

//func parseVersionTypeData(d interface{}) string {
//	switch d.(type) {
//	case int:
//		return strconv.Itoa(d.(int))
//	case float64:
//		return strconv.FormatFloat(d.(float64), 'f', 2, 32)
//	case bool:
//		if d.(bool) {
//			return "true"
//		}
//		return "false"
//	case []string:
//		return "multiple versions detected"
//	}
//	return fmt.Sprintf("%v", d)
//}
