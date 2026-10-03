// Copyright 2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"go.yaml.in/yaml/v4"
)

// isNotValidationError checks whether a validation failure is caused by a failed 'not' keyword.
func isNotValidationError(err *errors.SchemaValidationFailure) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Reason, "'not' failed") {
		return true
	}
	if err.OriginalJsonSchemaError != nil {
		return hasNotErrorKind(err.OriginalJsonSchemaError)
	}
	return false
}

func hasNotErrorKind(e *jsonschema.ValidationError) bool {
	if e == nil {
		return false
	}
	if _, ok := e.ErrorKind.(*kind.Not); ok {
		return true
	}
	for _, c := range e.Causes {
		if hasNotErrorKind(c) {
			return true
		}
	}
	return false
}

// formatExampleValidationReason formats a schema validation failure reason for examples,
// enriching failed 'not' validations with actionable context about which constraint matched.
func formatExampleValidationReason(err *errors.SchemaValidationFailure) string {
	if err == nil {
		return ""
	}
	if !isNotValidationError(err) {
		return err.Reason
	}
	return formatSchemaValidationReason(err, "example violates `not`")
}

// formatSchemaValidationReason enriches a 'not' validation failure with details about
// the constraints defined inside the 'not' subschema.
func formatSchemaValidationReason(err *errors.SchemaValidationFailure, prefix string) string {
	if err == nil {
		return ""
	}
	if !isNotValidationError(err) {
		return err.Reason
	}

	notNode := findNotNode(err.ReferenceSchema, err.KeywordLocation, err.OriginalJsonSchemaError)
	if notNode == nil {
		return err.Reason
	}

	details := describeNotConstraints(notNode)
	if len(details) > 0 {
		return fmt.Sprintf("%s: %s", prefix, strings.Join(details, "; "))
	}

	if err.KeywordLocation != "" {
		return fmt.Sprintf("%s constraint at `%s`", prefix, err.KeywordLocation)
	}
	return fmt.Sprintf("%s constraint", prefix)
}

// findNotNode attempts to locate the YAML node for the 'not' subschema using the
// keyword location, error schema URL, and the rendered reference schema.
func findNotNode(refSchema string, keywordLocation string, origErr *jsonschema.ValidationError) *yaml.Node {
	if refSchema == "" {
		return nil
	}
	var docNode yaml.Node
	if err := yaml.Unmarshal([]byte(refSchema), &docNode); err != nil {
		return nil
	}
	root := documentContentNode(&docNode)
	if root == nil {
		return nil
	}

	ptr := keywordLocation
	if ptr == "" && origErr != nil {
		ptr = extractPointerFromSchemaURL(origErr)
	}

	if node := locateNotNodeByPointer(root, ptr); node != nil {
		return node
	}

	return findFirstNotNode(root)
}

func locateNotNodeByPointer(root *yaml.Node, ptr string) *yaml.Node {
	if ptr == "" || ptr == "#" || ptr == "/" || ptr == "#/" {
		return findMappingValue(root, "not")
	}

	cleaned := ptr
	if strings.HasPrefix(cleaned, "#") {
		cleaned = cleaned[1:]
	}
	cleaned = strings.TrimPrefix(cleaned, "/")

	segments := strings.Split(cleaned, "/")
	current := root
	for _, rawSeg := range segments {
		seg := strings.ReplaceAll(strings.ReplaceAll(rawSeg, "~1", "/"), "~0", "~")
		if seg == "" {
			continue
		}
		if current.Kind == yaml.MappingNode {
			val := findMappingValue(current, seg)
			if val == nil {
				return nil
			}
			current = val
		} else if current.Kind == yaml.SequenceNode {
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(current.Content) {
				return nil
			}
			current = current.Content[idx]
		} else {
			return nil
		}
	}

	if current.Kind == yaml.MappingNode {
		if notNode := findMappingValue(current, "not"); notNode != nil {
			return notNode
		}
	}
	if len(segments) > 0 && segments[len(segments)-1] == "not" {
		return current
	}
	return nil
}

func extractPointerFromSchemaURL(origErr *jsonschema.ValidationError) string {
	if origErr == nil {
		return ""
	}
	var notErr *jsonschema.ValidationError
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if e == nil || notErr != nil {
			return
		}
		if _, ok := e.ErrorKind.(*kind.Not); ok {
			notErr = e
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(origErr)
	if notErr != nil && notErr.SchemaURL != "" {
		if idx := strings.Index(notErr.SchemaURL, "#"); idx != -1 {
			return notErr.SchemaURL[idx:]
		}
	}
	if origErr.SchemaURL != "" {
		if idx := strings.Index(origErr.SchemaURL, "#"); idx != -1 {
			return origErr.SchemaURL[idx:]
		}
	}
	return ""
}

func findFirstNotNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.MappingNode {
		if notVal := findMappingValue(node, "not"); notVal != nil {
			return notVal
		}
		for i := 1; i < len(node.Content); i += 2 {
			if found := findFirstNotNode(node.Content[i]); found != nil {
				return found
			}
		}
	} else if node.Kind == yaml.SequenceNode {
		for _, child := range node.Content {
			if found := findFirstNotNode(child); found != nil {
				return found
			}
		}
	}
	return nil
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func describeNotConstraints(notNode *yaml.Node) []string {
	if notNode == nil {
		return nil
	}

	if notNode.Kind == yaml.ScalarNode && notNode.Value == "true" {
		return []string{"schema allows no instances (`not: true`)"}
	}

	if notNode.Kind != yaml.MappingNode {
		return nil
	}

	var details []string

	// required
	if reqNode := findMappingValue(notNode, "required"); reqNode != nil && reqNode.Kind == yaml.SequenceNode {
		var props []string
		for _, item := range reqNode.Content {
			if item.Value != "" {
				props = append(props, item.Value)
			}
		}
		if len(props) == 1 {
			details = append(details, fmt.Sprintf("property `%s` must not be present", props[0]))
		} else if len(props) > 1 {
			formatted := make([]string, len(props))
			for i, p := range props {
				formatted[i] = fmt.Sprintf("`%s`", p)
			}
			details = append(details, fmt.Sprintf("properties %s must not be present", strings.Join(formatted, ", ")))
		}
	}

	// type
	if typeNode := findMappingValue(notNode, "type"); typeNode != nil {
		if typeNode.Kind == yaml.ScalarNode && typeNode.Value != "" {
			details = append(details, fmt.Sprintf("value must not be of type `%s`", typeNode.Value))
		} else if typeNode.Kind == yaml.SequenceNode && len(typeNode.Content) > 0 {
			var types []string
			for _, item := range typeNode.Content {
				if item.Value != "" {
					types = append(types, fmt.Sprintf("`%s`", item.Value))
				}
			}
			if len(types) > 0 {
				details = append(details, fmt.Sprintf("value must not be of type %s", strings.Join(types, " or ")))
			}
		}
	}

	// const
	if constNode := findMappingValue(notNode, "const"); constNode != nil && constNode.Value != "" {
		details = append(details, fmt.Sprintf("value must not be `%s`", constNode.Value))
	}

	// enum
	if enumNode := findMappingValue(notNode, "enum"); enumNode != nil && enumNode.Kind == yaml.SequenceNode {
		var vals []string
		for _, item := range enumNode.Content {
			vals = append(vals, fmt.Sprintf("`%s`", item.Value))
		}
		if len(vals) > 0 {
			details = append(details, fmt.Sprintf("value must not be one of: %s", strings.Join(vals, ", ")))
		}
	}

	// pattern
	if patNode := findMappingValue(notNode, "pattern"); patNode != nil && patNode.Value != "" {
		details = append(details, fmt.Sprintf("value must not match pattern `%s`", patNode.Value))
	}

	// minLength / maxLength
	if minLen := findMappingValue(notNode, "minLength"); minLen != nil && minLen.Value != "" {
		details = append(details, fmt.Sprintf("length must not be >= `%s`", minLen.Value))
	}
	if maxLen := findMappingValue(notNode, "maxLength"); maxLen != nil && maxLen.Value != "" {
		details = append(details, fmt.Sprintf("length must not be <= `%s`", maxLen.Value))
	}

	// minimum / maximum
	if minNode := findMappingValue(notNode, "minimum"); minNode != nil && minNode.Value != "" {
		details = append(details, fmt.Sprintf("value must not be >= `%s`", minNode.Value))
	}
	if maxNode := findMappingValue(notNode, "maximum"); maxNode != nil && maxNode.Value != "" {
		details = append(details, fmt.Sprintf("value must not be <= `%s`", maxNode.Value))
	}
	if exMin := findMappingValue(notNode, "exclusiveMinimum"); exMin != nil && exMin.Value != "" {
		details = append(details, fmt.Sprintf("value must not be > `%s`", exMin.Value))
	}
	if exMax := findMappingValue(notNode, "exclusiveMaximum"); exMax != nil && exMax.Value != "" {
		details = append(details, fmt.Sprintf("value must not be < `%s`", exMax.Value))
	}

	// minItems / maxItems
	if minItems := findMappingValue(notNode, "minItems"); minItems != nil && minItems.Value != "" {
		details = append(details, fmt.Sprintf("must not have >= `%s` items", minItems.Value))
	}
	if maxItems := findMappingValue(notNode, "maxItems"); maxItems != nil && maxItems.Value != "" {
		details = append(details, fmt.Sprintf("must not have <= `%s` items", maxItems.Value))
	}

	// properties
	if propsNode := findMappingValue(notNode, "properties"); propsNode != nil && propsNode.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(propsNode.Content); i += 2 {
			propName := propsNode.Content[i].Value
			propSchema := propsNode.Content[i+1]
			if propSchema.Kind == yaml.MappingNode {
				if constN := findMappingValue(propSchema, "const"); constN != nil && constN.Value != "" {
					details = append(details, fmt.Sprintf("property `%s` must not be `%s`", propName, constN.Value))
				} else if typeN := findMappingValue(propSchema, "type"); typeN != nil && typeN.Value != "" {
					details = append(details, fmt.Sprintf("property `%s` must not be of type `%s`", propName, typeN.Value))
				} else if patN := findMappingValue(propSchema, "pattern"); patN != nil && patN.Value != "" {
					details = append(details, fmt.Sprintf("property `%s` must not match pattern `%s`", propName, patN.Value))
				} else {
					details = append(details, fmt.Sprintf("property `%s` must not match schema", propName))
				}
			}
		}
	}

	// $ref
	if refNode := findMappingValue(notNode, "$ref"); refNode != nil && refNode.Value != "" {
		details = append(details, fmt.Sprintf("matches schema at `%s`", refNode.Value))
	}

	return details
}
