// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	doctorModel "github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// PathSpecificityOrder checks that overlapping paths are ordered from most specific to least specific.
type PathSpecificityOrder struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PathSpecificityOrder rule.
func (pso PathSpecificityOrder) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "pathsSpecificityOrder"}
}

// GetCategory returns the category of the PathSpecificityOrder rule.
func (pso PathSpecificityOrder) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

type orderedPathEntry struct {
	path     string
	keyNode  *yaml.Node
	methods  map[string]struct{}
	pathItem *doctorModel.PathItem
	segs     []segment
}

// RunRule will execute the PathSpecificityOrder rule using the source document order of the paths object.
func (pso PathSpecificityOrder) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) <= 0 || context.Index == nil {
		return nil
	}

	entries := collectOrderedPathEntries(context)
	if len(entries) == 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	message := context.Rule.Message

	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			sharedMethods := intersectPathMethods(entries[i].methods, entries[j].methods)
			if len(sharedMethods) == 0 {
				continue
			}

			if !pathShouldPrecede(entries[j], entries[i]) {
				continue
			}

			methodLabel := strings.Join(sharedMethods, ", ")
			result := model.RuleFunctionResult{
				Message: vacuumUtils.SuppliedOrDefault(message,
					fmt.Sprintf(
						"path `%s` should be declared before `%s` for method(s) %s because a static segment at the first differing position is more specific than a templated segment",
						entries[j].path, entries[i].path, methodLabel,
					),
				),
				StartNode: entries[j].keyNode,
				EndNode:   vacuumUtils.BuildEndNode(entries[j].keyNode),
				Path:      fmt.Sprintf("$.paths['%s']", entries[j].path),
				Rule:      context.Rule,
			}
			results = append(results, result)
		}
	}

	return results
}

func collectOrderedPathEntries(context model.RuleFunctionContext) []orderedPathEntry {
	pathsNode := context.Index.GetPathsNode()
	if pathsNode == nil {
		return nil
	}

	doctorPathItems := make(map[string]*doctorModel.PathItem)
	if context.DrDocument != nil && context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Paths != nil {
		for path, pathItem := range context.DrDocument.V3Document.Paths.PathItems.FromOldest() {
			doctorPathItems[path] = pathItem
		}
	}

	entries := make([]orderedPathEntry, 0, len(pathsNode.Content)/2)
	for i := 0; i+1 < len(pathsNode.Content); i += 2 {
		keyNode := pathsNode.Content[i]
		valueNode := pathsNode.Content[i+1]
		path := keyNode.Value
		pathItem := doctorPathItems[path]
		entries = append(entries, orderedPathEntry{
			path:     path,
			keyNode:  keyNode,
			methods:  getMethodsFromPathItemNode(valueNode),
			pathItem: pathItem,
			segs:     parseSegments(path, pathItem),
		})
	}
	return entries
}

func getMethodsFromPathItemNode(node *yaml.Node) map[string]struct{} {
	methods := make(map[string]struct{})
	if node == nil {
		return methods
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		switch strings.ToLower(node.Content[i].Value) {
		case "get", "post", "put", "delete", "options", "head", "patch", "trace":
			methods[strings.ToUpper(node.Content[i].Value)] = struct{}{}
		}
	}
	return methods
}

func intersectPathMethods(a, b map[string]struct{}) []string {
	ordered := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE"}
	var shared []string
	for _, method := range ordered {
		if _, ok := a[method]; !ok {
			continue
		}
		if _, ok := b[method]; ok {
			shared = append(shared, method)
		}
	}
	return shared
}

func pathShouldPrecede(candidate, current orderedPathEntry) bool {
	if len(candidate.segs) == 0 || len(candidate.segs) != len(current.segs) {
		return false
	}

	firstDiff := 0

	for i := range candidate.segs {
		a := candidate.segs[i]
		b := current.segs[i]

		switch {
		case a.isVar && b.isVar:
			if a.operator != b.operator {
				return false
			}
			if a.paramType != "" && b.paramType != "" && !areTypesCompatible(a.paramType, b.paramType) {
				return false
			}
		case !a.isVar && !b.isVar:
			if a.value != b.value {
				return false
			}
		default:
			var literal, varType string
			if a.isVar {
				literal, varType = b.value, a.paramType
			} else {
				literal, varType = a.value, b.paramType
			}
			if varType != "" && !canLiteralMatchType(literal, varType) {
				return false
			}
			if firstDiff == 0 {
				if !a.isVar && b.isVar {
					firstDiff = 1
				} else {
					firstDiff = -1
				}
			}
		}
	}

	return firstDiff == 1
}
