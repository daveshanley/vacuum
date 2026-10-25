// Copyright 2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonschema/v6"
	"github.com/pb33f/jsonschema/v6/kind"
	"github.com/pb33f/libopenapi-validator/errors"
)

// isNotValidationError checks whether a validation failure is caused by a failed 'not' keyword.
func isNotValidationError(err *errors.SchemaValidationFailure) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Reason, "'not' failed") {
		return true
	}
	if strings.HasSuffix(err.KeywordLocation, "/not") || err.KeywordLocation == "not" || err.KeywordLocation == "/not" {
		return true
	}
	if err.OriginalJsonSchemaError != nil {
		specificErr := findSpecificJsonSchemaError(err.OriginalJsonSchemaError, err)
		if specificErr != nil {
			return hasNotErrorKind(specificErr)
		}
	}
	return false
}

func findSpecificJsonSchemaError(root *jsonschema.ValidationError, err *errors.SchemaValidationFailure) *jsonschema.ValidationError {
	if root == nil {
		return nil
	}

	var allErrors []*jsonschema.ValidationError
	var collect func(e *jsonschema.ValidationError)
	collect = func(e *jsonschema.ValidationError) {
		if e == nil {
			return
		}
		allErrors = append(allErrors, e)
		for _, c := range e.Causes {
			collect(c)
		}
	}
	collect(root)

	hasInstPath := len(err.InstancePath) > 0
	hasKW := err.KeywordLocation != ""

	// 1. Match both InstanceLocation and KeywordLocation
	if hasInstPath && hasKW {
		for _, e := range allErrors {
			if pathMatches(e.InstanceLocation, err.InstancePath) && keywordMatches(e.SchemaURL, err.KeywordLocation) {
				return e
			}
		}
	}

	// 2. Match by InstanceLocation alone
	if hasInstPath {
		for _, e := range allErrors {
			if pathMatches(e.InstanceLocation, err.InstancePath) {
				return e
			}
		}
	}

	// 3. Match by KeywordLocation alone
	if hasKW {
		for _, e := range allErrors {
			if keywordMatches(e.SchemaURL, err.KeywordLocation) {
				return e
			}
		}
	}

	// 4. Fallback for single error without location paths (e.g. synthetic test cases)
	if !hasInstPath && !hasKW {
		if len(root.Causes) == 1 {
			return root.Causes[0]
		}
		if len(root.Causes) == 0 {
			return root
		}
	}

	return nil
}

func pathMatches(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func keywordMatches(schemaURL, keywordLocation string) bool {
	if keywordLocation == "" || schemaURL == "" {
		return false
	}
	cleanKW := strings.TrimPrefix(keywordLocation, "#")
	cleanKW = strings.TrimPrefix(cleanKW, "/")
	if idx := strings.Index(schemaURL, "#"); idx != -1 {
		schemaPtr := strings.TrimPrefix(schemaURL[idx+1:], "/")
		return schemaPtr == cleanKW || strings.HasSuffix(schemaPtr, "/"+cleanKW) || strings.HasPrefix(schemaPtr, cleanKW)
	}
	return strings.HasSuffix(schemaURL, cleanKW)
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
func formatExampleValidationReason(err *errors.SchemaValidationFailure, instanceNode ...*yaml.Node) string {
	if err == nil {
		return ""
	}
	if !isNotValidationError(err) {
		return err.Reason
	}
	return formatSchemaValidationReason(err, "example violates `not`", instanceNode...)
}

// formatSchemaValidationReason enriches a 'not' validation failure with details about
// the constraints defined inside the 'not' subschema.
func formatSchemaValidationReason(err *errors.SchemaValidationFailure, prefix string, instanceNode ...*yaml.Node) string {
	if err == nil {
		return ""
	}
	if !isNotValidationError(err) {
		return err.Reason
	}

	var origErr *jsonschema.ValidationError
	if err.OriginalJsonSchemaError != nil {
		origErr = findSpecificJsonSchemaError(err.OriginalJsonSchemaError, err)
		if origErr == nil {
			origErr = err.OriginalJsonSchemaError
		}
	}

	notNode := findNotNode(err.ReferenceSchema, err.KeywordLocation, origErr)
	if notNode == nil {
		return err.Reason
	}

	var rootInst *yaml.Node
	if len(instanceNode) > 0 && instanceNode[0] != nil {
		rootInst = documentContentNode(instanceNode[0])
	} else if err.ReferenceObject != "" {
		var doc yaml.Node
		if uErr := yaml.Unmarshal([]byte(err.ReferenceObject), &doc); uErr == nil {
			rootInst = documentContentNode(&doc)
		}
	}

	var failingInstance *yaml.Node
	if rootInst != nil {
		failingInstance = resolveInstanceNode(rootInst, err.InstancePath)
		if failingInstance == nil && len(err.InstancePath) == 0 {
			failingInstance = rootInst
		}
	}

	details := describeNotConstraints(notNode, failingInstance)
	if len(details) > 0 {
		return fmt.Sprintf("%s: %s", prefix, strings.Join(details, "; "))
	}

	if err.KeywordLocation != "" {
		return fmt.Sprintf("%s constraint at `%s`", prefix, err.KeywordLocation)
	}
	return fmt.Sprintf("%s constraint", prefix)
}

func resolveInstanceNode(root *yaml.Node, path []string) *yaml.Node {
	if root == nil {
		return nil
	}
	current := documentContentNode(root)
	for _, seg := range path {
		if current == nil {
			return nil
		}
		switch current.Kind {
		case yaml.MappingNode:
			val := findMappingValue(current, seg)
			if val == nil {
				return nil
			}
			current = val
		case yaml.SequenceNode:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(current.Content) {
				return nil
			}
			current = current.Content[idx]
		default:
			return nil
		}
	}
	return current
}

func isNumericNode(node *yaml.Node) bool {
	if node == nil || node.Kind != yaml.ScalarNode {
		return false
	}
	if node.Tag == "!!int" || node.Tag == "!!float" {
		return true
	}
	if node.Tag == "!!str" {
		return false
	}
	_, err := strconv.ParseFloat(node.Value, 64)
	return err == nil
}

func isStringNode(node *yaml.Node) bool {
	if node == nil || node.Kind != yaml.ScalarNode {
		return false
	}
	if node.Tag == "!!int" || node.Tag == "!!float" || node.Tag == "!!bool" || node.Tag == "!!null" {
		return false
	}
	return true
}

func isArrayNode(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.SequenceNode
}

func isObjectNode(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.MappingNode
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

	if ptr == "" || ptr == "#" || ptr == "/" || ptr == "#/" {
		return findFirstNotNode(root)
	}

	return nil
}

func locateNotNodeByPointer(root *yaml.Node, ptr string) *yaml.Node {
	if ptr == "" || ptr == "#" || ptr == "/" || ptr == "#/" {
		return findMappingValue(root, "not")
	}

	cleaned := strings.TrimPrefix(strings.TrimPrefix(ptr, "#"), "/")

	segments := strings.Split(cleaned, "/")
	current := root
	for _, rawSeg := range segments {
		seg := strings.ReplaceAll(strings.ReplaceAll(rawSeg, "~1", "/"), "~0", "~")
		if seg == "" {
			continue
		}
		switch current.Kind {
		case yaml.MappingNode:
			val := findMappingValue(current, seg)
			if val == nil {
				return nil
			}
			current = val
		case yaml.SequenceNode:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(current.Content) {
				return nil
			}
			current = current.Content[idx]
		default:
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
	switch node.Kind {
	case yaml.MappingNode:
		if notVal := findMappingValue(node, "not"); notVal != nil {
			return notVal
		}
		for i := 1; i < len(node.Content); i += 2 {
			if found := findFirstNotNode(node.Content[i]); found != nil {
				return found
			}
		}
	case yaml.SequenceNode:
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

const combinedNotConstraintMessage = "matches the schema excluded by not"

func describeNotConstraints(notNode *yaml.Node, instanceNode ...*yaml.Node) []string {
	if notNode == nil {
		return nil
	}

	if notNode.Kind == yaml.ScalarNode && notNode.Value == "true" {
		return []string{"schema allows no instances (`not: true`)"}
	}

	if notNode.Kind != yaml.MappingNode {
		return nil
	}

	var inst *yaml.Node
	if len(instanceNode) > 0 {
		inst = instanceNode[0]
	}

	var singleConstraints []string
	isCombined := false
	constraintCount := 0

	// 1. required
	if reqNode := findMappingValue(notNode, "required"); reqNode != nil && reqNode.Kind == yaml.SequenceNode {
		constraintCount++
		var props []string
		for _, item := range reqNode.Content {
			if item.Value != "" {
				props = append(props, item.Value)
			}
		}
		if len(props) == 1 {
			if inst != nil && !isObjectNode(inst) {
				isCombined = true
			} else {
				singleConstraints = append(singleConstraints, fmt.Sprintf("property `%s` must not be present", props[0]))
			}
		} else if len(props) > 1 {
			isCombined = true
		}
	}

	// 2. type
	if typeNode := findMappingValue(notNode, "type"); typeNode != nil {
		constraintCount++
		if typeNode.Kind == yaml.ScalarNode && typeNode.Value != "" {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be of type `%s`", typeNode.Value))
		} else if typeNode.Kind == yaml.SequenceNode && len(typeNode.Content) > 0 {
			var types []string
			for _, item := range typeNode.Content {
				if item.Value != "" {
					types = append(types, fmt.Sprintf("`%s`", item.Value))
				}
			}
			if len(types) > 0 {
				singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be of type %s", strings.Join(types, " or ")))
			}
		}
	}

	// 3. const (counted independently of node value, falls back to general exclusion if not a non-empty scalar)
	if constNode := findMappingValue(notNode, "const"); constNode != nil {
		constraintCount++
		if constNode.Kind == yaml.ScalarNode && constNode.Value != "" {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be `%s`", constNode.Value))
		}
	}

	// 4. enum
	if enumNode := findMappingValue(notNode, "enum"); enumNode != nil && enumNode.Kind == yaml.SequenceNode {
		constraintCount++
		var vals []string
		for _, item := range enumNode.Content {
			vals = append(vals, fmt.Sprintf("`%s`", item.Value))
		}
		if len(vals) > 0 {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be one of: %s", strings.Join(vals, ", ")))
		}
	}

	// 5. pattern
	if patNode := findMappingValue(notNode, "pattern"); patNode != nil && patNode.Value != "" {
		constraintCount++
		if inst != nil && !isStringNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not match pattern `%s`", patNode.Value))
		}
	}

	// 6. minLength / maxLength
	if minLen := findMappingValue(notNode, "minLength"); minLen != nil && minLen.Value != "" {
		constraintCount++
		if inst != nil && !isStringNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("length must not be >= `%s`", minLen.Value))
		}
	}
	if maxLen := findMappingValue(notNode, "maxLength"); maxLen != nil && maxLen.Value != "" {
		constraintCount++
		if inst != nil && !isStringNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("length must not be <= `%s`", maxLen.Value))
		}
	}

	// 7. minimum / maximum / exclusiveMinimum / exclusiveMaximum
	if minNode := findMappingValue(notNode, "minimum"); minNode != nil && minNode.Value != "" {
		constraintCount++
		if inst != nil && !isNumericNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be >= `%s`", minNode.Value))
		}
	}
	if maxNode := findMappingValue(notNode, "maximum"); maxNode != nil && maxNode.Value != "" {
		constraintCount++
		if inst != nil && !isNumericNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be <= `%s`", maxNode.Value))
		}
	}
	if exMin := findMappingValue(notNode, "exclusiveMinimum"); exMin != nil && exMin.Value != "" {
		constraintCount++
		if inst != nil && !isNumericNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be > `%s`", exMin.Value))
		}
	}
	if exMax := findMappingValue(notNode, "exclusiveMaximum"); exMax != nil && exMax.Value != "" {
		constraintCount++
		if inst != nil && !isNumericNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("value must not be < `%s`", exMax.Value))
		}
	}

	// 8. minItems / maxItems
	if minItems := findMappingValue(notNode, "minItems"); minItems != nil && minItems.Value != "" {
		constraintCount++
		if inst != nil && !isArrayNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("must not have >= `%s` items", minItems.Value))
		}
	}
	if maxItems := findMappingValue(notNode, "maxItems"); maxItems != nil && maxItems.Value != "" {
		constraintCount++
		if inst != nil && !isArrayNode(inst) {
			isCombined = true
		} else {
			singleConstraints = append(singleConstraints, fmt.Sprintf("must not have <= `%s` items", maxItems.Value))
		}
	}

	// 9. properties
	if propsNode := findMappingValue(notNode, "properties"); propsNode != nil && propsNode.Kind == yaml.MappingNode {
		constraintCount++
		numProps := len(propsNode.Content) / 2
		if numProps > 1 {
			isCombined = true
		} else if numProps == 1 {
			propName := propsNode.Content[0].Value
			propSchema := propsNode.Content[1]
			propDesc := describePropertySchema(propName, propSchema, inst)
			if propDesc != "" {
				singleConstraints = append(singleConstraints, propDesc)
			} else {
				isCombined = true
			}
		}
	}

	// 10. $ref
	if refNode := findMappingValue(notNode, "$ref"); refNode != nil && refNode.Value != "" {
		constraintCount++
		singleConstraints = append(singleConstraints, fmt.Sprintf("matches schema at `%s`", refNode.Value))
	}

	// 11. Check for any other constraints that are not metadata and not handled above
	for i := 0; i+1 < len(notNode.Content); i += 2 {
		k := notNode.Content[i].Value
		if isSchemaMetadataKey(k) || isHandledNotConstraintKey(k) {
			continue
		}
		constraintCount++
	}

	if isCombined || constraintCount > 1 {
		return []string{combinedNotConstraintMessage}
	}

	if constraintCount == 1 {
		if len(singleConstraints) == 1 {
			return singleConstraints
		}
		return []string{combinedNotConstraintMessage}
	}

	return nil
}

func describePropertySchema(propName string, propSchema *yaml.Node, inst *yaml.Node) string {
	if propSchema == nil || propSchema.Kind != yaml.MappingNode {
		return ""
	}

	// Check failing instance: if instance is provided, verify propName is present
	if inst != nil {
		if inst.Kind != yaml.MappingNode {
			return ""
		}
		propVal := findMappingValue(inst, propName)
		if propVal == nil {
			// Property is absent in the failing instance, so the inner property keyword does not constrain it
			return ""
		}
	}

	var subConstraints []string
	otherCount := 0
	for i := 0; i+1 < len(propSchema.Content); i += 2 {
		k := propSchema.Content[i].Value
		if isSchemaMetadataKey(k) {
			continue
		}
		switch k {
		case "const":
			if val := propSchema.Content[i+1].Value; val != "" {
				subConstraints = append(subConstraints, fmt.Sprintf("property `%s` must not be `%s`", propName, val))
			} else {
				otherCount++
			}
		case "type":
			if val := propSchema.Content[i+1].Value; val != "" {
				subConstraints = append(subConstraints, fmt.Sprintf("property `%s` must not be of type `%s`", propName, val))
			} else {
				otherCount++
			}
		case "pattern":
			if val := propSchema.Content[i+1].Value; val != "" {
				subConstraints = append(subConstraints, fmt.Sprintf("property `%s` must not match pattern `%s`", propName, val))
			} else {
				otherCount++
			}
		default:
			otherCount++
		}
	}
	if len(subConstraints) == 1 && otherCount == 0 {
		return subConstraints[0]
	}
	return ""
}

func isSchemaMetadataKey(k string) bool {
	switch k {
	case "description", "title", "$id", "$schema", "$comment", "examples", "example",
		"deprecated", "default", "readOnly", "writeOnly":
		return true
	default:
		return false
	}
}

func isHandledNotConstraintKey(k string) bool {
	switch k {
	case "required", "type", "const", "enum", "pattern",
		"minLength", "maxLength", "minimum", "maximum",
		"exclusiveMinimum", "exclusiveMaximum", "minItems", "maxItems",
		"properties", "$ref":
		return true
	default:
		return false
	}
}
