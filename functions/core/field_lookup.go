package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	openapiUtils "github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
	"strconv"
)

func fieldLookupOptions(context model.RuleFunctionContext, recursiveFirstSegment bool) vacuumUtils.FieldPathOptions {
	return vacuumUtils.FieldPathOptions{
		RecursiveFirstSegment:        recursiveFirstSegment,
		ResolveSingleItemCombinators: context.Rule != nil && context.Rule.Resolved,
	}
}

func givenPathValue(given interface{}) string {
	switch v := given.(type) {
	case string:
		if v != "" {
			return v
		}
	case []string:
		if len(v) > 0 && v[0] != "" {
			return v[0]
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				return s
			}
		}
	}
	return "unknown"
}

func locateNodePaths(context model.RuleFunctionContext, node *yaml.Node) (string, []string, []v3.Foundational) {
	fallbackPath := givenPathValue(context.Given)
	if context.DrDocument == nil || node == nil {
		return fallbackPath, nil, nil
	}

	if path, allPaths, locatedObjects, found := locateNodePathsDirect(context, node, fallbackPath); found {
		return path, allPaths, locatedObjects
	}

	if node.Line > 0 {
		if path, allPaths, locatedObjects, found := locateNodePathsByLine(context, node.Line, fallbackPath); found {
			return path, allPaths, locatedObjects
		}
	}

	if path, allPaths, found := locateReferencedParameterPaths(context, node); found {
		return path, allPaths, nil
	}

	if context.Index != nil {
		origin := context.Index.FindNodeOrigin(node)
		if origin != nil {
			if origin.Node != nil {
				if path, allPaths, locatedObjects, found := locateNodePathsDirect(context, origin.Node, fallbackPath); found {
					return path, allPaths, locatedObjects
				}
			}
			if origin.Line > 0 {
				if path, allPaths, locatedObjects, found := locateNodePathsByLine(context, origin.Line, fallbackPath); found {
					return path, allPaths, locatedObjects
				}
			}
		}
	}

	if path, found := locateNodePathInDocument(context, node); found {
		return path, nil, nil
	}

	if context.Index != nil {
		if origin := context.Index.FindNodeOrigin(node); origin != nil {
			if path, found := locateNodePathInDocument(context, origin.Node); found {
				return path, nil, nil
			}
			if path, found := locateNodePathInDocument(context, origin.ValueNode); found {
				return path, nil, nil
			}
		}
	}

	return fallbackPath, nil, nil
}

func locateReferencedParameterPaths(context model.RuleFunctionContext, node *yaml.Node) (string, []string, bool) {
	refValue := extractReferenceValue(node)
	if refValue == "" && context.Index != nil {
		if origin := context.Index.FindNodeOrigin(node); origin != nil {
			refValue = extractReferenceValue(origin.Node)
			if refValue == "" {
				refValue = extractReferenceValue(origin.ValueNode)
			}
		}
	}
	if refValue == "" || context.Document == nil || context.Document.GetRolodex() == nil {
		return "", nil, false
	}

	rootIndex := context.Document.GetRolodex().GetRootIndex()
	if rootIndex == nil || rootIndex.GetRootNode() == nil {
		return "", nil, false
	}

	paths := findReferencedParameterPaths(rootIndex.GetRootNode(), refValue)
	if len(paths) == 0 {
		return "", nil, false
	}
	return paths[0], paths, true
}

func locateNodePathsDirect(context model.RuleFunctionContext, node *yaml.Node, fallbackPath string) (string, []string, []v3.Foundational, bool) {
	locatedObjects, err := context.DrDocument.LocateModelsByKeyAndValue(node, node)
	if err == nil && len(locatedObjects) > 0 {
		locatedPath, allPaths := buildLocatedPaths(locatedObjects, fallbackPath)
		return locatedPath, allPaths, locatedObjects, true
	}

	locatedObjects, err = context.DrDocument.LocateModel(node)
	if err != nil || len(locatedObjects) == 0 {
		return fallbackPath, nil, nil, false
	}
	locatedPath, allPaths := buildLocatedPaths(locatedObjects, fallbackPath)
	return locatedPath, allPaths, locatedObjects, true
}

func locateNodePathsByLine(context model.RuleFunctionContext, line int, fallbackPath string) (string, []string, []v3.Foundational, bool) {
	locatedByLine, err := context.DrDocument.LocateModelByLine(line)
	if err != nil || len(locatedByLine) == 0 {
		return fallbackPath, nil, nil, false
	}

	locatedObjects := make([]v3.Foundational, 0, len(locatedByLine))
	for _, obj := range locatedByLine {
		if foundational, ok := obj.(v3.Foundational); ok {
			locatedObjects = append(locatedObjects, foundational)
		}
	}
	if len(locatedObjects) == 0 {
		return fallbackPath, nil, nil, false
	}

	locatedPath, allPaths := buildLocatedPaths(locatedObjects, fallbackPath)
	return locatedPath, allPaths, locatedObjects, true
}

func buildLocatedPaths(locatedObjects []v3.Foundational, fallbackPath string) (string, []string) {
	locatedPath := fallbackPath
	allPaths := make([]string, 0, len(locatedObjects))
	for i, obj := range locatedObjects {
		path := obj.GenerateJSONPath()
		if path == "" {
			continue
		}
		if i == 0 {
			locatedPath = path
		}
		allPaths = append(allPaths, path)
	}
	if len(allPaths) == 0 {
		return fallbackPath, nil
	}
	return locatedPath, allPaths
}

func locateExistingFieldPaths(
	context model.RuleFunctionContext,
	containerNode *yaml.Node,
	fieldPath string,
	fieldResult vacuumUtils.FieldPathResult,
) (string, []string, []v3.Foundational) {
	fallbackPath := givenPathValue(context.Given)
	if context.DrDocument != nil &&
		fieldResult.KeyNode != nil &&
		fieldResult.ValueNode != nil &&
		!fieldResult.UsedSingleItemCombinator {
		locatedObjects, err := context.DrDocument.LocateModelsByKeyAndValue(fieldResult.KeyNode, fieldResult.ValueNode)
		if err == nil && len(locatedObjects) > 0 {
			locatedPath, allPaths := buildLocatedPaths(locatedObjects, fallbackPath)
			return appendTerminalFieldPathToLocatedPaths(locatedPath, allPaths, locatedObjects, fieldPath)
		}
	}

	basePath, basePaths, locatedObjects := locateNodePaths(context, containerNode)
	return appendFieldPathToLocatedPaths(basePath, basePaths, locatedObjects, fieldPath)
}

func appendTerminalFieldPathToLocatedPaths(
	basePath string,
	basePaths []string,
	locatedObjects []v3.Foundational,
	fieldPath string,
) (string, []string, []v3.Foundational) {
	return appendFieldPathToLocatedPaths(basePath, basePaths, locatedObjects, terminalFieldPathSegment(fieldPath))
}

func appendFieldPathToLocatedPaths(
	basePath string,
	basePaths []string,
	locatedObjects []v3.Foundational,
	fieldPath string,
) (string, []string, []v3.Foundational) {
	primaryPath := joinJSONPath(basePath, fieldPath)
	if len(basePaths) == 0 {
		return primaryPath, nil, locatedObjects
	}

	allPaths := make([]string, 0, len(basePaths))
	for _, path := range basePaths {
		if path == "" {
			continue
		}
		allPaths = append(allPaths, joinJSONPath(path, fieldPath))
	}
	if len(allPaths) == 0 {
		return primaryPath, nil, locatedObjects
	}
	return primaryPath, allPaths, locatedObjects
}

func terminalFieldPathSegment(fieldPath string) string {
	segments, err := vacuumUtils.ParseFieldPath(fieldPath)
	if err != nil || len(segments) == 0 {
		return fieldPath
	}

	last := segments[len(segments)-1]
	switch last.Type {
	case vacuumUtils.SegmentArrayIndex:
		return "[" + strconv.Itoa(last.Index) + "]"
	case vacuumUtils.SegmentMapKey:
		return "['" + last.Key + "']"
	default:
		return last.Key
	}
}

func joinJSONPath(basePath, suffix string) string {
	if suffix == "" {
		return basePath
	}
	if basePath == "" {
		return suffix
	}
	if suffix[0] == '[' {
		return basePath + suffix
	}
	return model.GetStringTemplates().BuildJSONPath(basePath, suffix)
}

func extractReferenceValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.MappingNode {
		return ""
	}
	_, refNode := openapiUtils.FindKeyNodeTop("$ref", node.Content)
	if refNode == nil {
		return ""
	}
	return refNode.Value
}

func findReferencedParameterPaths(root *yaml.Node, refValue string) []string {
	if root == nil {
		return nil
	}

	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}

	_, pathsNode := openapiUtils.FindKeyNodeTop("paths", root.Content)
	if pathsNode == nil || !openapiUtils.IsNodeMap(pathsNode) {
		return nil
	}

	var locatedPaths []string
	for i := 0; i+1 < len(pathsNode.Content); i += 2 {
		pathKey := pathsNode.Content[i]
		pathValueNode := pathsNode.Content[i+1]
		if !openapiUtils.IsNodeStringValue(pathKey) || !openapiUtils.IsNodeMap(pathValueNode) {
			continue
		}

		appendReferencedParameterPaths(&locatedPaths, pathKey.Value, "top", pathValueNode, refValue)

		for j := 0; j+1 < len(pathValueNode.Content); j += 2 {
			methodKey := pathValueNode.Content[j]
			methodValue := pathValueNode.Content[j+1]
			if !openapiUtils.IsNodeStringValue(methodKey) || !openapiUtils.IsHttpVerb(methodKey.Value) || !openapiUtils.IsNodeMap(methodValue) {
				continue
			}
			appendReferencedParameterPaths(&locatedPaths, pathKey.Value, methodKey.Value, methodValue, refValue)
		}
	}
	return locatedPaths
}

func appendReferencedParameterPaths(locatedPaths *[]string, pathValue, method string, container *yaml.Node, refValue string) {
	_, paramsNode := openapiUtils.FindKeyNodeTop("parameters", container.Content)
	if paramsNode == nil || !openapiUtils.IsNodeArray(paramsNode) {
		return
	}

	for i, paramNode := range paramsNode.Content {
		if extractReferenceValue(paramNode) != refValue {
			continue
		}
		if method == "top" {
			*locatedPaths = append(*locatedPaths, fmt.Sprintf("$.paths['%s'].parameters[%d]", pathValue, i))
			continue
		}
		*locatedPaths = append(*locatedPaths, fmt.Sprintf("$.paths['%s'].%s.parameters[%d]", pathValue, method, i))
	}
}

func locateNodePathInDocument(context model.RuleFunctionContext, target *yaml.Node) (string, bool) {
	if target == nil || context.Document == nil || context.Document.GetRolodex() == nil {
		return "", false
	}

	rootIndex := context.Document.GetRolodex().GetRootIndex()
	if rootIndex == nil || rootIndex.GetRootNode() == nil {
		return "", false
	}

	return findNodeJSONPath(rootIndex.GetRootNode(), target, "$")
}

func findNodeJSONPath(current, target *yaml.Node, currentPath string) (string, bool) {
	if current == nil || target == nil {
		return "", false
	}
	if current == target {
		return currentPath, true
	}

	switch current.Kind {
	case yaml.DocumentNode:
		for _, child := range current.Content {
			if path, found := findNodeJSONPath(child, target, currentPath); found {
				return path, true
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(current.Content); i += 2 {
			keyNode := current.Content[i]
			valueNode := current.Content[i+1]
			if keyNode == target || valueNode == target {
				return currentPath, true
			}
			childPath := buildJSONPathSegment(currentPath, keyNode.Value)
			if path, found := findNodeJSONPath(valueNode, target, childPath); found {
				return path, true
			}
		}
	case yaml.SequenceNode:
		for i, child := range current.Content {
			childPath := fmt.Sprintf("%s[%d]", currentPath, i)
			if child == target {
				return childPath, true
			}
			if path, found := findNodeJSONPath(child, target, childPath); found {
				return path, true
			}
		}
	}
	return "", false
}

func buildJSONPathSegment(basePath, key string) string {
	if isSimpleJSONPathKey(key) {
		return fmt.Sprintf("%s.%s", basePath, key)
	}
	return fmt.Sprintf("%s['%s']", basePath, key)
}

func isSimpleJSONPathKey(key string) bool {
	if key == "" {
		return false
	}

	first := key[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}

	for i := 1; i < len(key); i++ {
		ch := key[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' {
			continue
		}
		return false
	}
	return true
}
