// Copyright 2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"reflect"
)

// MarshalingIssue represents a location where JSON marshaling will fail
type MarshalingIssue struct {
	Line        int
	Column      int
	Path        string
	Reason      string
	KeyValue    string // The actual key that's problematic
}

// CheckJSONMarshaling attempts to marshal the data and returns any marshaling issues found.
// It only performs deep checking if the initial marshal fails.
func CheckJSONMarshaling(data interface{}, rootNode *yaml.Node) []MarshalingIssue {
	// First, try to marshal the entire document
	_, err := json.Marshal(data)
	if err == nil {
		// No issues, return empty slice
		return nil
	}

	// Marshaling failed, now find all the locations where it fails
	var issues []MarshalingIssue
	
	// The error usually contains "json: unsupported type: map[interface {}]interface {}"
	// This means we have maps with non-string keys
	if rootNode != nil {
		findMarshalingIssues(rootNode, "", &issues)
	}
	
	return issues
}

// FindMarshalingIssuesInYAML directly checks the YAML AST for marshaling issues
// without needing the unmarshaled data. This is useful when SpecJSON is nil.
func FindMarshalingIssuesInYAML(rootNode *yaml.Node) []MarshalingIssue {
	var issues []MarshalingIssue
	if rootNode != nil {
		findMarshalingIssues(rootNode, "", &issues)
	}
	return issues
}

// findMarshalingIssues recursively walks the YAML AST to find marshaling problems
func findMarshalingIssues(node *yaml.Node, currentPath string, issues *[]MarshalingIssue) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			findMarshalingIssues(node.Content[0], currentPath, issues)
		}

	case yaml.MappingNode:
		// Pre-allocate capacity to avoid repeated allocations
		mapLen := len(node.Content) / 2
		
		// Process all keys first, then recurse
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			
			// Build path efficiently
			var keyPath string
			if currentPath == "" {
				keyPath = keyNode.Value
			} else {
				// Use more efficient string building for paths
				keyPath = currentPath + "." + keyNode.Value
			}
			
			// Check if the key is not a string (this is a marshaling problem)
			if keyNode.Tag != "!!str" && keyNode.Tag != "!!null" {
				var keyType string
				switch keyNode.Tag {
				case "!!int":
					keyType = "integer"
				case "!!float":
					keyType = "float"  
				case "!!bool":
					keyType = "boolean"
				case "!!map":
					keyType = "map"
				case "!!seq":
					keyType = "array"
				default:
					// Just use the tag if we don't recognize it
					if len(keyNode.Tag) > 2 {
						keyType = keyNode.Tag[2:] // Remove !! prefix
					} else {
						keyType = "unknown"
					}
				}
				
				issue := MarshalingIssue{
					Line:     keyNode.Line,
					Column:   keyNode.Column,
					Path:     keyPath,
					Reason:   fmt.Sprintf("map has %s key '%s'", keyType, keyNode.Value),
					KeyValue: keyNode.Value,
				}
				*issues = append(*issues, issue)
			}
			
			// Recurse into the value
			findMarshalingIssues(valueNode, keyPath, issues)
		}
		_ = mapLen // avoid unused variable warning

	case yaml.SequenceNode:
		// Arrays - recurse into each element
		for i, child := range node.Content {
			arrayPath := fmt.Sprintf("%s[%d]", currentPath, i)
			findMarshalingIssues(child, arrayPath, issues)
		}
		
	case yaml.ScalarNode:
		// Check for other problematic scalar types if needed
		// Currently, scalars are generally fine for JSON marshaling
		
	case yaml.AliasNode:
		// Aliases (references) might cause circular reference issues
		// For now, we'll skip these as they're handled differently
	}
}

// QuickCheckMarshalable does a quick check to see if data can be marshaled to JSON
// without doing a deep analysis. Returns true if marshalable, false otherwise.
func QuickCheckMarshalable(data interface{}) bool {
	_, err := json.Marshal(data)
	return err == nil
}

// ConvertToMarshalable attempts to convert problematic structures to JSON-marshalable ones.
// This is a helper that can be used to fix issues, but for validation we just report them.
func ConvertToMarshalable(data interface{}) interface{} {
	return convertValue(reflect.ValueOf(data))
}

func convertValue(v reflect.Value) interface{} {
	switch v.Kind() {
	case reflect.Map:
		// Check if this is a map[interface{}]interface{} or similar
		if v.Type().Key().Kind() == reflect.Interface {
			// Convert to map[string]interface{}
			result := make(map[string]interface{})
			for _, key := range v.MapKeys() {
				// Convert key to string
				keyStr := fmt.Sprintf("%v", key.Interface())
				value := v.MapIndex(key)
				// Recursively convert the value
				result[keyStr] = convertValue(value)
			}
			return result
		}
		// For other map types, create a new map with converted values
		result := reflect.MakeMap(reflect.MapOf(v.Type().Key(), reflect.TypeOf((*interface{})(nil)).Elem()))
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			result.SetMapIndex(key, reflect.ValueOf(convertValue(value)))
		}
		return result.Interface()
		
	case reflect.Slice, reflect.Array:
		// Convert slice/array elements
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = convertValue(v.Index(i))
		}
		return result
		
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return convertValue(v.Elem())
		
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return convertValue(v.Elem())
		
	default:
		// For other types, return as is
		return v.Interface()
	}
}