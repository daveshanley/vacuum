// Copyright 2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
    "fmt"
    "regexp"
    "strings"

    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    v3 "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
)

// CamelCaseProperties validates that all schema property names use camelCase.
// It inspects all schemas extracted by the Doctor model and checks their properties map keys.
// Vendor extensions (keys beginning with x-) are ignored.
type CamelCaseProperties struct{}

// GetSchema returns the function schema metadata.
func (cc CamelCaseProperties) GetSchema() model.RuleFunctionSchema {
    return model.RuleFunctionSchema{Name: "camelCaseProperties"}
}

// GetCategory returns the category for this function.
func (cc CamelCaseProperties) GetCategory() string {
    return model.FunctionCategoryOpenAPI
}

var (
    // Acceptable camelCase, allowing numbers and acronyms (consecutive capitals) after the first lowercase letter
    camelCaseRegex   = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
    pascalCaseRegex  = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

// RunRule executes the rule against all schemas in the document.
func (cc CamelCaseProperties) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
    var results []model.RuleFunctionResult

    if context.DrDocument == nil {
        return results
    }

    // Deduplicate schemas that can appear multiple times through different contexts
    seen := make(map[string]bool)

    for i := range context.DrDocument.Schemas {
        s := context.DrDocument.Schemas[i]
        if s == nil || s.Value == nil || s.Value.Properties == nil || s.Value.Properties.Len() == 0 {
            continue
        }

        if hash := extractHash(s); hash != "" {
            if _, ok := seen[hash]; ok {
                continue
            }
            seen[hash] = true
        }

        for propName, prop := range s.Properties.FromOldest() {
            // Ignore vendor extensions
            if strings.HasPrefix(propName, "x-") || strings.HasPrefix(propName, "X-") {
                continue
            }

            if isCamelCase(propName) {
                continue
            }

            detected := detectCaseType(propName)

            // Determine nodes and path for accurate reporting
            keyNode := prop.Schema.KeyNode
            if keyNode == nil && prop.Schema.Value != nil && prop.Schema.Value.GoLow() != nil {
                keyNode = prop.Schema.Value.GoLow().KeyNode
            }
            if keyNode == nil {
                // Fallback to a dummy node if we cannot find a key node
                keyNode = &yaml.Node{Line: 1, Column: 1}
            }

            path := prop.Schema.GenerateJSONPath()

            msg := fmt.Sprintf("schema property `%s` is %s; use camelCase", propName, detected)

            result := model.RuleFunctionResult{
                Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, msg),
                StartNode: keyNode,
                EndNode:   vacuumUtils.BuildEndNode(keyNode),
                Path:      path,
                Rule:      context.Rule,
            }
            // Attach to model for cross-tooling
            prop.Schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
            results = append(results, result)
        }
    }

    return results
}

func isCamelCase(s string) bool {
    // Accept single all-lowercase words and camelCase with digits/acronyms
    return camelCaseRegex.MatchString(s)
}

func detectCaseType(s string) string {
    switch {
    case strings.Contains(s, "_"):
        return "snake_case"
    case strings.Contains(s, "-"):
        return "kebab-case"
    case pascalCaseRegex.MatchString(s):
        return "PascalCase"
    default:
        return "non-camelCase"
    }
}


