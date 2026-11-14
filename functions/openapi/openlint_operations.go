// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// Includes adaptations of @openlint/ruleset-openapi 0.1.0.
// Copyright 2018 Stoplight, Inc. and 2026 OpenLint.
// See rulesets/packaged/OPENLINT-OPENAPI-LICENSE.
// SPDX-License-Identifier: MIT AND Apache-2.0

package openapi

import (
	"fmt"
	"iter"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// OpenLintPackageFunction returns the pinned OpenAPI package function for an
// upstream action name. It returns nil for unknown names.
func OpenLintPackageFunction(name string) model.RuleFunction {
	switch name {
	case "oasOpSuccessResponse", "oasOpFormDataConsumeCheck", "oasOpIdUnique", "oasOpParams", "oasTagDefined", "oasPathParam", "oasDiscriminator", "oasSecurityDefined", "serverVariables":
		return packagedOperation(name)
	case "refSiblings", "typedEnum", "uniquenessTags":
		return openLintValueFunction(name)
	case "oasUnusedComponent", "oasExample", "oasDocumentSchema", "oasSchema", "unreferencedReusableObject":
		return packagedSchemaFunction(name)
	}
	return nil
}

type packagedOperation string

func (f packagedOperation) GetCategory() string { return model.FunctionCategoryOpenAPI }

func (f packagedOperation) GetSchema() model.RuleFunctionSchema {
	schema := model.RuleFunctionSchema{Name: string(f)}
	switch f {
	case "oasSecurityDefined":
		schema.Properties = []model.RuleFunctionProperty{{Name: "oasVersion"}}
	case "serverVariables":
		schema.Properties = []model.RuleFunctionProperty{{Name: "checkSubstitutions"}, {Name: "requireDefault"}}
	}
	return schema
}

// An operand keeps the selected value and its diagnostic path together. When a
// requested child is missing OpenLint highlights its closest existing parent.
type packagedOperand struct {
	node *yaml.Node
	path string
}

func (v packagedOperand) is(kind yaml.Kind) bool { return v.node != nil && v.node.Kind == kind }

func (v packagedOperand) text() string {
	if v.is(yaml.ScalarNode) {
		return v.node.Value
	}
	return ""
}

func (v packagedOperand) isText() bool { return v.is(yaml.ScalarNode) && v.node.ShortTag() == "!!str" }

func (v packagedOperand) fields() iter.Seq2[string, packagedOperand] {
	return func(yield func(string, packagedOperand) bool) {
		if !v.is(yaml.MappingNode) {
			return
		}
		for i := 0; i+1 < len(v.node.Content); i += 2 {
			key := v.node.Content[i].Value
			if !yield(key, packagedOperand{v.node.Content[i+1], model.AppendResultPathSegment(v.path, key)}) {
				return
			}
		}
	}
}

func (v packagedOperand) field(name string) packagedOperand {
	if child := findMappingValue(v.node, name); child != nil {
		return packagedOperand{child, model.AppendResultPathSegment(v.path, name)}
	}
	return packagedOperand{}
}

func (v packagedOperand) items() iter.Seq[packagedOperand] {
	return func(yield func(packagedOperand) bool) {
		if !v.is(yaml.SequenceNode) {
			return
		}
		for i, n := range v.node.Content {
			if !yield(packagedOperand{n, model.AppendResultPathIndex(v.path, i)}) {
				return
			}
		}
	}
}

func (v packagedOperand) containsText(want string) bool {
	for item := range v.items() {
		if item.isText() && item.text() == want {
			return true
		}
	}
	return false
}

type packagedDiagnostics []model.RuleFunctionResult

func (d *packagedDiagnostics) add(where packagedOperand, message string) {
	*d = append(*d, model.RuleFunctionResult{Path: where.path, StartNode: where.node, EndNode: utils.BuildEndNode(where.node), Message: message})
}

func (v packagedOperand) fieldOrSelf(key string) packagedOperand {
	if child := v.field(key); child.node != nil {
		return child
	}
	return v
}

func (v packagedOperand) operations() iter.Seq[packagedOperand] {
	return func(yield func(packagedOperand) bool) {
		for key, value := range v.fields() {
			switch key {
			case "get", "head", "post", "put", "patch", "delete", "options", "trace":
				if value.is(yaml.MappingNode) && !yield(value) {
					return
				}
			}
		}
	}
}

func (f packagedOperation) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	var results packagedDiagnostics
	document := packagedOperand{documentContentNode(ctx.ExecutionRoot), "$"}
	path, _ := ctx.Given.(string)
	for _, node := range nodes {
		if node == nil || ctx.MissingValue {
			continue
		}
		v := packagedOperand{documentContentNode(node), path}
		switch f {
		case "oasOpSuccessResponse":
			if !v.is(yaml.MappingNode) {
				continue
			}
			isOAS3 := strings.HasPrefix(document.field("openapi").text(), "3.")
			success := false
			for key := range v.fields() {
				code, err := strconv.ParseFloat(strings.TrimSpace(key), 64)
				if (err == nil && code >= 200 && code < 400) || (isOAS3 && (key == "2XX" || key == "3XX")) {
					success = true
					break
				}
			}
			if !success {
				results.add(v, "Operation must define at least a single 2xx or 3xx response")
			}
		case "oasOpFormDataConsumeCheck":
			checkPackagedConsumes(v, &results)
		case "oasOpIdUnique":
			checkPackagedOperationIDs(v, &results)
		case "oasOpParams":
			checkPackagedOperationParams(v, &results)
		case "oasTagDefined":
			checkPackagedTags(v, &results)
		case "oasPathParam":
			checkPackagedPathParams(v, &results)
		case "oasDiscriminator":
			if name := v.field("discriminator"); name.isText() {
				if v.field("properties").field(name.text()).node == nil {
					results.add(v.fieldOrSelf("properties"), "The discriminator property must be defined in this schema.")
				}
				if !v.field("required").containsText(name.text()) {
					results.add(v.fieldOrSelf("required"), "The discriminator property must be in the required property list.")
				}
			}
		case "oasSecurityDefined":
			source := document
			if ctx.AuthoredRoot != nil {
				source.node = documentContentNode(ctx.AuthoredRoot)
			}
			checkPackagedSecurity(source, v, ctx.GetOptionsStringMap()["oasVersion"] == "2", &results)
		case "serverVariables":
			checkPackagedServerVariables(v, ctx, &results)
		}
	}
	return results
}

func checkPackagedConsumes(v packagedOperand, results *packagedDiagnostics) {
	consumes := v.field("consumes")
	if !consumes.is(yaml.SequenceNode) {
		return
	}
	for entry := range consumes.items() {
		if strings.Contains(entry.text(), "application/x-www-form-urlencoded") || strings.Contains(entry.text(), "multipart/form-data") {
			return
		}
	}
	for p := range v.field("parameters").items() {
		if p.field("in").text() == "formData" {
			results.add(v, "Consumes must include urlencoded, multipart, or form-data media type when using formData parameter.")
			return
		}
	}
}

func checkPackagedOperationIDs(paths packagedOperand, results *packagedDiagnostics) {
	type scalar struct{ tag, value string }
	seen := make(map[scalar]bool)
	for _, item := range paths.fields() {
		for op := range item.operations() {
			id := op.field("operationId")
			if !id.is(yaml.ScalarNode) {
				continue
			}
			key := scalar{id.node.ShortTag(), id.text()}
			if seen[key] {
				results.add(id, "operationId must be unique.")
			}
			seen[key] = true
		}
	}
}

func operationFingerprintPart(v packagedOperand) string {
	if v.node == nil {
		return "undefined"
	}
	if v.node.ShortTag() == "!!null" {
		return "null"
	}
	return v.text()
}

func checkPackagedOperationParams(params packagedOperand, results *packagedDiagnostics) {
	if !params.is(yaml.SequenceNode) || len(params.node.Content) < 2 {
		return
	}
	seen := make(map[string]bool)
	var body []packagedOperand
	form := false
	for p := range params.items() {
		if !p.is(yaml.MappingNode) || p.field("$ref").node != nil {
			continue
		}
		in := p.field("in")
		key := operationFingerprintPart(in) + "-" + operationFingerprintPart(p.field("name"))
		if seen[key] {
			results.add(p, "A parameter in this operation already exposes the same combination of \"name\" and \"in\" values.")
		}
		seen[key] = true
		switch in.text() {
		case "body":
			body = append(body, p)
		case "formData":
			form = true
		}
	}
	if len(body) > 0 && form {
		results.add(params, "Operation must not have both \"in:body\" and \"in:formData\" parameters.")
	}
	for i := 1; i < len(body); i++ {
		results.add(body[i], "Operation must not have more than a single instance of the \"in:body\" parameter.")
	}
}

func checkPackagedTags(v packagedOperand, results *packagedDiagnostics) {
	names := make(map[string]bool)
	for tag := range v.field("tags").items() {
		if name := tag.field("name"); name.isText() {
			names[name.text()] = true
		}
	}
	for _, item := range v.field("paths").fields() {
		for op := range item.operations() {
			for tag := range op.field("tags").items() {
				if !tag.isText() || !names[tag.text()] {
					results.add(tag, "Operation tags must be defined in global tags.")
				}
			}
		}
	}
}

var packagedPathVariable = regexp.MustCompile(`\{;?\??([a-zA-Z0-9_-]+)\*?\}`)

func collectPackagedPathParams(v packagedOperand, results *packagedDiagnostics) map[string]packagedOperand {
	params := make(map[string]packagedOperand)
	for p := range v.field("parameters").items() {
		name := p.field("name")
		if p.field("in").text() != "path" || !name.isText() {
			continue
		}
		n := name.text()
		required := p.field("required")
		if !required.is(yaml.ScalarNode) || required.node.ShortTag() != "!!bool" || !strings.EqualFold(required.text(), "true") {
			results.add(p, fmt.Sprintf("Path parameter %q must have \"required\" property that is set to \"true\".", n))
		}
		if _, exists := params[n]; exists {
			results.add(p, fmt.Sprintf("Path parameter %q must not be defined multiple times.", n))
			continue
		}
		params[n] = p
	}
	return params
}

func checkPackagedPathParams(paths packagedOperand, results *packagedDiagnostics) {
	unique := make(map[string]string)
	for path, item := range paths.fields() {
		if !item.is(yaml.MappingNode) {
			continue
		}
		normalized := packagedPathVariable.ReplaceAllString(path, "%")
		if earlier, exists := unique[normalized]; exists {
			results.add(item, fmt.Sprintf("Paths %q and %q must not be equivalent.", earlier, path))
		} else {
			unique[normalized] = path
		}
		var expected []string
		for _, match := range packagedPathVariable.FindAllStringSubmatch(path, -1) {
			if slices.Contains(expected, match[1]) {
				results.add(item, fmt.Sprintf("Path %q must not use parameter \"{%s}\" multiple times.", path, match[1]))
				continue
			}
			expected = append(expected, match[1])
		}
		top := collectPackagedPathParams(item, results)
		for op := range item.operations() {
			local := collectPackagedPathParams(op, results)
			for name, value := range top {
				if _, overridden := local[name]; !overridden && !slices.Contains(expected, name) {
					results.add(value, fmt.Sprintf("Parameter %q must be used in path %q.", name, path))
				}
			}
			for name, value := range local {
				if !slices.Contains(expected, name) {
					results.add(value, fmt.Sprintf("Parameter %q must be used in path %q.", name, path))
				}
			}
			for _, name := range expected {
				if top[name].node == nil && local[name].node == nil {
					results.add(op, fmt.Sprintf("Operation must define parameter \"{%s}\" as expected by path %q.", name, path))
				}
			}
		}
	}
}

func checkPackagedSecurity(document, requirement packagedOperand, oas2 bool, results *packagedDiagnostics) {
	defs := document.field("components").field("securitySchemes")
	location := "components.securitySchemes"
	if oas2 {
		defs = document.field("securityDefinitions")
		location = "securityDefinitions"
	}
	for _, scopes := range requirement.fields() {
		if !scopes.is(yaml.SequenceNode) {
			return
		}
		for scope := range scopes.items() {
			if !scope.isText() {
				return
			}
		}
	}
	for key, scopes := range requirement.fields() {
		scheme := defs.field(key)
		if scheme.node == nil {
			owner := "Operation"
			if strings.HasPrefix(requirement.path, "$.security[") {
				owner = "API"
			}
			results.add(scopes, fmt.Sprintf("%s \"security\" values must match a scheme defined in the %q object.", owner, location))
			continue
		}
		for scope := range scopes.items() {
			found := false
			if oas2 {
				found = scheme.field("scopes").field(scope.text()).node != nil
			} else {
				for _, flow := range scheme.field("flows").fields() {
					if flow.field("scopes").field(scope.text()).node != nil {
						found = true
						break
					}
				}
			}
			if !found {
				results.add(scope, fmt.Sprintf("%q must be listed among scopes.", scope.text()))
			}
		}
	}
}
