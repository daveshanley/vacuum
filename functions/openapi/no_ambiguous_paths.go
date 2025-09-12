// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	doctorModel "github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"regexp"
	"strings"
)

// AmbiguousPaths will determine if paths can be confused by a compiler.
type AmbiguousPaths struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the AmbiguousPaths rule.
func (ap AmbiguousPaths) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "noAmbiguousPaths"}
}

// GetCategory returns the category of the AmbiguousPaths rule.
func (ap AmbiguousPaths) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the AmbiguousPaths rule, based on supplied context and a supplied []*yaml.Node slice.
func (ap AmbiguousPaths) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// Try to use doctor model if available for more accurate checking
	if context.DrDocument != nil && context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Paths != nil {
		return ap.checkWithDoctorModel(context)
	}

	// Fallback to simple path checking without parameter type information
	var seen []string
	ops := context.Index.GetPathsNode()
	var opPath string

	if ops != nil {
		var opNode *yaml.Node
		for i, op := range ops.Content {
			if i%2 == 0 {
				opPath = op.Value
				opNode = op
				continue
			}
			path := fmt.Sprintf("$.paths['%s']", opPath)
			for _, p := range seen {
				ambiguous := checkPaths(p, opPath, nil, nil)
				if ambiguous {
					results = append(results, model.RuleFunctionResult{
						Message:   fmt.Sprintf("paths are ambiguous with one another: `%s` and `%s`", p, opPath),
						StartNode: opNode,
						EndNode:   vacuumUtils.BuildEndNode(opNode),
						Path:      path,
						Rule:      context.Rule,
					})
				}
			}
			seen = append(seen, opPath)
		}
	}
	return results
}

// checkWithDoctorModel uses the doctor model to check for ambiguous paths with parameter type checking
func (ap AmbiguousPaths) checkWithDoctorModel(context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	paths := context.DrDocument.V3Document.Paths

	if paths == nil || paths.PathItems == nil {
		return results
	}

	// Build a slice of path entries with their path items
	type pathEntry struct {
		path string
		item *doctorModel.PathItem
	}

	var pathEntries []pathEntry
	for path, pathItem := range paths.PathItems.FromOldest() {
		pathEntries = append(pathEntries, pathEntry{path: path, item: pathItem})
	}

	// Compare each pair of paths
	for i := 0; i < len(pathEntries); i++ {
		for j := i + 1; j < len(pathEntries); j++ {
			pathA := pathEntries[i]
			pathB := pathEntries[j]

			// Check if paths are potentially ambiguous
			if checkPaths(pathA.path, pathB.path, pathA.item, pathB.item) {
				// Paths are ambiguous based on structure and parameter types
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("paths are ambiguous with one another: `%s` and `%s`", pathA.path, pathB.path),
					StartNode: pathB.item.KeyNode,
					EndNode:   vacuumUtils.BuildEndNode(pathB.item.KeyNode),
					Path:      fmt.Sprintf("$.paths['%s']", pathB.path),
					Rule:      context.Rule,
				})
			}
		}
	}

	return results
}

var reggie, _ = regexp.Compile(`^{(.+?)}$`)

type segment struct {
	value     string
	isVar     bool
	paramName string
	paramType string
}

func parseSegments(path string, pathItem *doctorModel.PathItem) []segment {
	parts := strings.Split(path, "/")[1:]
	segments := make([]segment, len(parts))

	for i, part := range parts {
		seg := segment{value: part}
		if matches := reggie.FindStringSubmatch(part); len(matches) > 1 {
			seg.isVar = true
			seg.paramName = matches[1]
			if pathItem != nil {
				seg.paramType = getParameterType(pathItem, seg.paramName)
			}
		}
		segments[i] = seg
	}
	return segments
}

func getParameterType(pathItem *doctorModel.PathItem, paramName string) string {
	if pathItem == nil {
		return ""
	}

	for _, param := range pathItem.Parameters {
		if param.Value != nil && param.Value.In == "path" && param.Value.Name == paramName {
			if param.Value.Schema != nil {
				if schema := param.Value.Schema.Schema(); schema != nil && len(schema.Type) > 0 {
					return schema.Type[0]
				}
			}
		}
	}

	for _, op := range pathItem.GetOperations().FromOldest() {
		if op.Value == nil {
			continue
		}
		for _, param := range op.Value.Parameters {
			if param != nil && param.In == "path" && param.Name == paramName {
				if param.Schema != nil {
					if schema := param.Schema.Schema(); schema != nil && len(schema.Type) > 0 {
						return schema.Type[0]
					}
				}
			}
		}
	}

	return ""
}

func checkPaths(pA, pB string, pathItemA, pathItemB *doctorModel.PathItem) bool {
	segsA := parseSegments(pA, pathItemA)
	segsB := parseSegments(pB, pathItemB)

	if len(segsA) != len(segsB) {
		return false
	}

	// Track variable vs literal mismatches
	varLiteralPositions := make([]int, 0, len(segsA))

	for i := range segsA {
		a, b := &segsA[i], &segsB[i]

		if a.isVar && b.isVar {
			if a.paramType != "" && b.paramType != "" && !areTypesCompatible(a.paramType, b.paramType) {
				return false
			}
		} else if !a.isVar && !b.isVar {
			if a.value != b.value {
				return false
			}
		} else {
			// Variable vs literal
			varLiteralPositions = append(varLiteralPositions, i)

			var varType, literal string
			if a.isVar {
				varType, literal = a.paramType, b.value
			} else {
				varType, literal = b.paramType, a.value
			}

			if varType != "" && !canLiteralMatchType(literal, varType) {
				return false
			}
		}
	}

	// Key logic for issue #504: paths with conflicting variable/literal patterns
	// Example: /a/{x}/b/c/{y} vs /a/{x}/b/{z}/d
	// Position 3: c vs {z} (literal vs var)
	// Position 4: {y} vs d (var vs literal)
	// These conflict - no URL can match both patterns
	if len(varLiteralPositions) >= 2 {
		// Check if we have opposite patterns (literal-var in one position, var-literal in another)
		for i := 0; i < len(varLiteralPositions)-1; i++ {
			pos1 := varLiteralPositions[i]
			for j := i + 1; j < len(varLiteralPositions); j++ {
				pos2 := varLiteralPositions[j]
				// If the variable/literal pattern is reversed, paths cannot be ambiguous
				if segsA[pos1].isVar != segsA[pos2].isVar {
					return false
				}
			}
		}
	}

	return true
}

func areTypesCompatible(typeA, typeB string) bool {
	if typeA == typeB {
		return true
	}
	return (typeA == "integer" && typeB == "number") || (typeA == "number" && typeB == "integer")
}

func canLiteralMatchType(literal, paramType string) bool {
	switch paramType {
	case "integer":
		if literal == "" || (literal[0] == '-' && len(literal) == 1) {
			return false
		}
		start := 0
		if literal[0] == '-' {
			start = 1
		}
		for i := start; i < len(literal); i++ {
			if literal[i] < '0' || literal[i] > '9' {
				return false
			}
		}
		return true
	case "number":
		if literal == "" || literal == "." || literal == "-" || literal == "-." {
			return false
		}
		hasDot := false
		start := 0
		if literal[0] == '-' {
			start = 1
		}
		for i := start; i < len(literal); i++ {
			if literal[i] == '.' {
				if hasDot {
					return false
				}
				hasDot = true
			} else if literal[i] < '0' || literal[i] > '9' {
				return false
			}
		}
		return true
	case "boolean":
		return literal == "true" || literal == "false"
	case "string":
		return true
	default:
		return true
	}
}
