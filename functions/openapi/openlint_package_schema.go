// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// Includes adaptations of @openlint/ruleset-openapi 0.1.0.
// Copyright 2018 Stoplight, Inc. and 2026 OpenLint.
// See rulesets/schemas/OPENLINT-OPENAPI-LICENSE.
// SPDX-License-Identifier: MIT AND Apache-2.0

package openapi

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/functions/core"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonschema/v6"
	"github.com/pb33f/jsonschema/v6/kind"
)

type packagedSchemaFunction string

func (fn packagedSchemaFunction) GetCategory() string { return model.FunctionCategoryOpenAPI }
func (fn packagedSchemaFunction) GetSchema() model.RuleFunctionSchema {
	switch fn {
	case "unreferencedReusableObject":
		return core.UnreferencedReusableObject{}.GetSchema()
	case "oasSchema":
		return core.Schema{}.GetSchema()
	case "oasExample":
		return model.RuleFunctionSchema{Name: string(fn), Required: []string{"oasVersion", "schemaField", "type"},
			Properties: []model.RuleFunctionProperty{{Name: "oasVersion"}, {Name: "schemaField"}, {Name: "type"}}}
	default:
		return model.RuleFunctionSchema{Name: string(fn)}
	}
}

func (fn packagedSchemaFunction) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) != 1 || nodes[0] == nil {
		return nil
	}
	node := documentContentNode(nodes[0])
	switch fn {
	case "oasUnusedComponent":
		components := findMappingValue(node, "components")
		var results []model.RuleFunctionResult
		for _, group := range []string{"schemas", "responses", "parameters", "examples", "requestBodies", "headers", "links", "callbacks"} {
			if members := findMappingValue(components, group); members != nil {
				ctx.Options = map[string]any{"reusableObjectsLocation": "#/components/" + group}
				results = append(results, packagedUnusedObjects(members, ctx)...)
			}
		}
		return results
	case "unreferencedReusableObject":
		return packagedUnusedObjects(node, ctx)
	case "oasDocumentSchema":
		return packagedDocumentValidation(node, ctx)
	case "oasExample":
		return packagedExampleValidation(node, ctx)
	case "oasSchema":
		options, _ := ctx.Options.(map[string]any)
		path, _ := ctx.Given.(string)
		return validatePackagedValues([]packagedOperand{{node, path}}, ctx, options["schema"])
	}
	return nil
}

func packagedUnusedObjects(members *yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	results := (core.UnreferencedReusableObject{}).RunRule([]*yaml.Node{members}, ctx)
	if len(results) == 0 {
		return nil
	}
	// The draft locates the member's value, including its complete mapping.
	values := make(map[*yaml.Node]*yaml.Node, len(members.Content)/2)
	for i := 0; i+1 < len(members.Content); i += 2 {
		values[members.Content[i]] = members.Content[i+1]
	}
	for i := range results {
		if value := values[results[i].StartNode]; value != nil {
			results[i].StartNode = value
			results[i].EndNode = vacuumUtils.BuildEndNode(value)
		}
		if ctx.Rule.Message != "" {
			results[i].Message = ctx.Rule.Message
		}
	}
	return results
}

// These resources are taken from @openlint/ruleset-openapi 0.1.0,
// dist/oas/schemas/{oas,json-schema}. The bundled draft-04 variant uses a
// private URI so the Go compiler can retain the canonical draft-04 metaschema.
// See rulesets/schemas/OPENLINT-OPENAPI-LICENSE.
//
//go:embed openlint_document_schemas.json
var packagedSchemaResources []byte

// Each compiler requests only the schemas referenced by its version.
type packagedSchemaLoader map[string]json.RawMessage

func (l packagedSchemaLoader) Load(url string) (any, error) {
	data, ok := l[url]
	if !ok {
		return nil, fmt.Errorf("unknown packaged schema %q", url)
	}
	var schema any
	err := json.Unmarshal(data, &schema)
	return schema, err
}

var packagedResources = sync.OnceValues(func() (packagedSchemaLoader, error) {
	var data packagedSchemaLoader
	err := json.Unmarshal(packagedSchemaResources, &data)
	return data, err
})

var packagedDocumentSchemas = map[string]func() (*jsonschema.Schema, error){
	"2.0": loadPackagedSchema("http://swagger.io/v2/schema.json"),
	"3.0": loadPackagedSchema("https://spec.openapis.org/oas/3.0/schema/2019-04-02"),
	"3.1": loadPackagedSchema("https://spec.openapis.org/oas/3.1/schema/2021-09-28"),
	"3.2": loadPackagedSchema("https://spec.openapis.org/oas/3.2/schema/2025-11-23"),
}

func loadPackagedSchema(entry string) func() (*jsonschema.Schema, error) {
	return sync.OnceValues(func() (*jsonschema.Schema, error) {
		resources, err := packagedResources()
		if err != nil {
			return nil, err
		}
		c := jsonschema.NewCompiler()
		c.UseLoader(resources)
		return c.Compile(entry)
	})
}

func packagedDocumentValidation(root *yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	version := "2.0"
	if v := findMappingValue(root, "openapi"); v != nil {
		parts := strings.SplitN(v.Value, ".", 3)
		if len(parts) < 2 {
			return nil
		}
		version = parts[0] + "." + parts[1]
	}
	load := packagedDocumentSchemas[version]
	if load == nil {
		return nil
	}
	compiled, err := load()
	var value any
	if err == nil {
		err = root.Decode(&value)
	}
	if err == nil {
		err = compiled.Validate(normalizeToStringMap(value))
	}
	if err == nil {
		return nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		if ctx.ReportError != nil {
			ctx.ReportError(err)
		}
		return []model.RuleFunctionResult{{Rule: ctx.Rule, StartNode: root, Path: "$", Message: err.Error()}}
	}
	var results []model.RuleFunctionResult
	add := func(path []string, msg string) {
		node := resolveInstanceNode(root, path)
		results = append(results, model.RuleFunctionResult{Rule: ctx.Rule, StartNode: node, EndNode: vacuumUtils.BuildEndNode(node), Path: jsonPointerToJSONPath(instancePointer(path), root), Message: msg})
	}
	for _, leaf := range extractLeafValidationErrors(ve) {
		path := leaf.err.InstanceLocation
		prefix := ""
		if len(path) > 0 {
			prefix = fmt.Sprintf("%q property ", path[len(path)-1])
		}
		switch k := leaf.err.ErrorKind.(type) {
		case *kind.Required:
			for _, field := range k.Missing {
				add(path, prefix+fmt.Sprintf("must have required property %q", field))
			}
		case *kind.AdditionalProperties:
			for _, field := range k.Properties {
				add(append(path[:len(path):len(path)], field), fmt.Sprintf("Property %q is not expected to be here", field))
			}
		case *kind.Type:
			add(path, prefix+"must be "+strings.Join(k.Want, ","))
		default:
			add(path, defaultFormatter.Format(NewErrorContext(leaf.err)))
		}
	}
	return results
}

// OpenLint uses the selected document dialect to check examples and enum arrays.
func validatePackagedValues(values []packagedOperand, ctx model.RuleFunctionContext, schema any) []model.RuleFunctionResult {
	c := jsonschema.NewCompiler()
	c.UseLoader(jsonschema.SchemeURLLoader{}) // examples may only use the resolved input and embedded metaschemas
	c.DefaultDraft(jsonschema.Draft4)
	root := documentContentNode(ctx.ExecutionRoot)
	if v := findMappingValue(root, "openapi"); v != nil && (strings.HasPrefix(v.Value, "3.1.") || strings.HasPrefix(v.Value, "3.2.")) {
		c.DefaultDraft(jsonschema.Draft2020)
		if d := findMappingValue(root, "jsonSchemaDialect"); d != nil {
			switch strings.TrimSuffix(d.Value, "#") {
			case "http://json-schema.org/draft-04/schema":
				c.DefaultDraft(jsonschema.Draft4)
			case "http://json-schema.org/draft-06/schema":
				c.DefaultDraft(jsonschema.Draft6)
			case "http://json-schema.org/draft-07/schema":
				c.DefaultDraft(jsonschema.Draft7)
			case "https://json-schema.org/draft/2019-09/schema":
				c.DefaultDraft(jsonschema.Draft2019)
			}
		}
	}
	const schemaID = "urn:openlint:example"
	var compiled *jsonschema.Schema
	err := c.AddResource(schemaID, schema)
	if err == nil {
		compiled, err = c.Compile(schemaID)
	}
	var results packagedDiagnostics
	if err != nil {
		results.add(values[0], err.Error())
		return results
	}
	for _, v := range values {
		var example any
		if err = v.node.Decode(&example); err != nil {
			results.add(v, err.Error())
			continue
		}
		err = compiled.Validate(normalizeToStringMap(example))
		if err == nil {
			continue
		}
		var failure *jsonschema.ValidationError
		if !errors.As(err, &failure) {
			results.add(v, err.Error())
			continue
		}
		for _, leaf := range extractLeafValidationErrors(failure) {
			loc := v
			property := ""
			if parsed, e := jsonpath.NewPath(v.path); e == nil {
				if segments, e := parsed.GetSegmentInfo(); e == nil && len(segments) > 0 {
					property = segments[len(segments)-1].Key
				}
			}
			for _, key := range leaf.err.InstanceLocation {
				if loc.is(yaml.SequenceNode) {
					idx, e := strconv.Atoi(key)
					if e != nil || idx < 0 || idx >= len(loc.node.Content) {
						break
					}
					loc = packagedOperand{loc.node.Content[idx], model.AppendResultPathIndex(loc.path, idx)}
				} else {
					loc = loc.fieldOrSelf(key)
				}
				property = key
			}
			prefix := ""
			if property != "" {
				prefix = fmt.Sprintf("%q property ", property)
			}
			msg := defaultFormatter.Format(NewErrorContext(leaf.err))
			switch k := leaf.err.ErrorKind.(type) {
			case *kind.Type:
				msg = prefix + "type must be " + strings.Join(k.Want, ",")
			case *kind.Required:
				if len(k.Missing) > 0 {
					msg = prefix + fmt.Sprintf("must have required property %q", k.Missing[0])
				}
			case *kind.UniqueItems:
				msg = prefix + "must not have duplicate items"
			}
			results.add(loc, msg)
			break // the package defaults to allErrors: false
		}
	}
	return results
}

func packagedExampleValidation(node *yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	opts, _ := ctx.Options.(map[string]any)
	path, _ := ctx.Given.(string)
	target := packagedOperand{node, path}
	version2 := opts["oasVersion"] == float64(2)
	media := opts["type"] == "media"
	schemaNode := node
	if opts["schemaField"] != "$" {
		schemaNode = target.field("schema").node
	}
	if schemaNode == nil {
		return nil
	}
	var values []packagedOperand
	if !media {
		for _, name := range []string{"example", "x-example", "default"} {
			if name == "x-example" && !version2 {
				continue
			}
			if item := target.field(name); item.node != nil {
				values = append(values, item)
			}
		}
	} else if example := target.field("example"); !version2 && example.node != nil {
		values = append(values, example)
	} else {
		for _, value := range target.field("examples").fields() {
			if !version2 {
				if value.field("externalValue").node != nil {
					continue
				}
				value = value.field("value")
			}
			if value.node != nil {
				values = append(values, value)
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	var schema any
	if err := schemaNode.Decode(&schema); err != nil {
		return []model.RuleFunctionResult{{Rule: ctx.Rule, StartNode: schemaNode, Message: err.Error()}}
	}
	schema = normalizeToStringMap(schema)
	access := ""
	if media {
		access = packagedExampleAccess(path)
	}
	if obj, ok := schema.(map[string]any); ok {
		if _, ok := obj["required"].(bool); ok && version2 {
			delete(obj, "required")
		}
		prepareExampleSchema(obj, access, version2, false)
	}
	return validatePackagedValues(values, ctx, schema)
}

func packagedExampleAccess(path string) string {
	parsed, err := jsonpath.NewPath(path)
	if err != nil {
		return ""
	}
	s, err := parsed.GetSegmentInfo()
	if err != nil || len(s) == 0 {
		return ""
	}
	if s[0].Key == "responses" {
		return "writeOnly"
	}
	at := 1
	switch s[0].Key {
	case "paths":
		at = 3
	case "components":
	default:
		return ""
	}
	if len(s) > at {
		switch s[at].Key {
		case "requestBodies", "requestBody":
			return "readOnly"
		case "responses":
			return "writeOnly"
		}
	}
	return ""
}

// Only schema-bearing keywords are traversed. A property named "not" or an
// example containing "required" must not be interpreted as a schema keyword.
func prepareExampleSchema(obj map[string]any, access string, swagger, negated bool) {
	nullable := obj["nullable"] == true || swagger && obj["x-nullable"] == true
	if t, ok := obj["type"].(string); ok && nullable && t != "null" {
		obj["type"] = []any{t, "null"}
	}
	if access != "" && !negated {
		props, _ := obj["properties"].(map[string]any)
		if required, ok := obj["required"].([]any); ok {
			required = slices.DeleteFunc(required, func(v any) bool {
				key, ok := v.(string)
				if !ok {
					return false
				}
				p, _ := props[key].(map[string]any)
				return p[access] == true
			})
			if len(required) == 0 {
				delete(obj, "required")
			} else {
				obj["required"] = required
			}
		}
	}
	for key, value := range obj {
		switch key {
		case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas", "dependencies":
			group, _ := value.(map[string]any)
			for _, sub := range group {
				if child, ok := sub.(map[string]any); ok {
					prepareExampleSchema(child, access, swagger, negated)
				}
			}
		case "not", "if", "then", "else", "additionalProperties", "additionalItems", "propertyNames", "contains", "items", "allOf", "anyOf", "oneOf", "prefixItems", "unevaluatedProperties", "unevaluatedItems":
			if child, ok := value.(map[string]any); ok {
				prepareExampleSchema(child, access, swagger, negated != (key == "not"))
			} else if list, ok := value.([]any); ok {
				for _, sub := range list {
					if child, ok := sub.(map[string]any); ok {
						prepareExampleSchema(child, access, swagger, negated)
					}
				}
			}
		}
	}
}
