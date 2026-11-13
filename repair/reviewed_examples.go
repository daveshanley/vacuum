// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"context"
	"encoding/json"

	"strings"

	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/pb33f/go-yaml"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/renderer"
)

// reviewedExample generates only validated instances; it never changes schema constraints.
func reviewedExample(root *yaml.Node, f Finding, nodes map[string]*yaml.Node) ([]Operation, string, string) {
	if f.RuleID == "asyncapi-message-examples" {
		return reviewedAsyncExample(root, f, nodes)
	}
	if version := field(root, "openapi"); version != nil && !strings.HasPrefix(version.Value, "3.0.") && !strings.HasPrefix(version.Value, "3.1.") && !strings.HasPrefix(version.Value, "3.2.") {
		return nil, "", "example generation requires a supported JSON Schema dialect"
	}
	path := f.Pointer
	var schema *yaml.Node
	var target string
	for {
		n := nodes[path]
		if n != nil && n.Kind == yaml.MappingNode && (field(n, "type") != nil || field(n, "properties") != nil || field(n, "items") != nil || field(n, "enum") != nil || field(n, "const") != nil || field(n, "allOf") != nil || field(n, "oneOf") != nil || field(n, "anyOf") != nil) {
			schema = n
			if f.RuleID == "json-schema-examples-valid" {
				prefix := path + "/examples/"
				if !strings.HasPrefix(f.Pointer, prefix) {
					return nil, "", "authored defaults are not replaceable examples"
				}
				target = f.Pointer
			} else {
				target = path + "/example"
				if field(n, "examples") != nil {
					return nil, "", "named or array examples need an exact example owner"
				}
			}
			break
		}
		if path == "" {
			break
		}
		path = path[:strings.LastIndex(path, "/")]
	}
	if schema == nil {
		return nil, "", "example schema could not be identified"
	}
	// Subschema extraction cannot change identifier or reference scope. Directional
	// schemas also need request/response ownership, not a guessed neutral sample.
	var unsupported func(*yaml.Node) bool
	unsupported = func(n *yaml.Node) bool {
		if n == nil {
			return false
		}
		if n.Kind == yaml.MappingNode {
			for i := 0; i < len(n.Content); i += 2 {
				switch n.Content[i].Value {
				case "$ref", "$dynamicRef", "$recursiveRef", "$id", "readOnly", "writeOnly":
					return true
				}
			}
		}
		for _, c := range n.Content {
			if unsupported(c) {
				return true
			}
		}
		return false
	}
	if unsupported(schema) {
		return nil, "", "reference scope or request/response direction needs a resolved schema"
	}
	compiled, err := compileExampleSchema(root, schema)
	if err != nil || compiled == nil {
		return nil, "", "example schema cannot be compiled independently"
	}
	sample, valid := exampleFromSchema(root, schema, compiled)
	if !valid {
		// The review UI can supply a value for a scalar string. Other shapes
		// need authored domain data before a useful example can be proposed.
		if kind := field(schema, "type"); kind != nil && kind.Value == "string" {
			if f.ExampleValue == nil {
				return nil, "", ExampleInputRequired
			}
			sample = *f.ExampleValue
			if compiled.Validate(sample) != nil {
				return nil, "", "entered example does not satisfy the schema"
			}
		} else {
			return nil, "", "schema has no supported representative example"
		}
	}
	encoded, err := json.Marshal(sample)
	if err != nil {
		return nil, "", err.Error()
	}
	op := "add"
	if nodes[target] != nil {
		op = "replace"
	}
	ops := []Operation{{Op: op, Path: target, Value: encoded}}
	if f.RuleID == "oas3-missing-example" {
		ops = append(ops, childExampleOperations(schema, path, sample)...)
	}
	return ops, "example-generated", ""
}

// ExampleInputRequired identifies a string example that needs a value from the author.
const ExampleInputRequired = "enter a representative string example"

var formatExamples = map[string]string{
	"email": "user@example.com", "idn-email": "user@example.com",
	"hostname": "api.example.com", "idn-hostname": "api.example.com",
	"uri": "https://example.com/resources/1", "uri-reference": "/resources/1",
	"date": "2026-01-15", "time": "12:00:00Z", "date-time": "2026-01-15T12:00:00Z",
	"duration": "PT1H", "uuid": "550e8400-e29b-41d4-a716-446655440000",
	"ipv4": "192.0.2.1", "ipv6": "2001:db8::1",
}

// exampleFromSchema tries authored values in source order, then builds a sample
// from known formats and member schemas. Each candidate must satisfy its schema.
func exampleFromSchema(root, schema *yaml.Node, compiled exampleValidator) (any, bool) {
	if schema == nil || schema.Kind != yaml.MappingNode {
		return nil, false
	}
	if compiled == nil {
		var err error
		compiled, err = compileExampleSchema(root, schema)
		if err != nil || compiled == nil {
			return nil, false
		}
	}
	for _, name := range []string{"example", "const", "default", "examples", "enum"} {
		n := field(schema, name)
		if n == nil {
			continue
		}
		candidates := []*yaml.Node{n}
		if name == "examples" || name == "enum" {
			if n.Kind != yaml.SequenceNode {
				continue
			}
			candidates = n.Content
		}
		for _, candidate := range candidates {
			value, err := schemautil.NodeToInterface(candidate)
			if err == nil && compiled.Validate(value) == nil {
				return value, true
			}
		}
	}
	kind := field(schema, "type")
	var sample any
	switch {
	case field(schema, "properties") != nil:
		properties := field(schema, "properties")
		if properties.Kind != yaml.MappingNode {
			return nil, false
		}
		object := make(map[string]any, len(properties.Content)/2)
		for i := 0; i+1 < len(properties.Content); i += 2 {
			if v, ok := exampleFromSchema(root, properties.Content[i+1], nil); ok {
				object[properties.Content[i].Value] = v
			}
		}
		if len(object) == 0 {
			return nil, false
		}
		sample = object
	case field(schema, "items") != nil:
		item, ok := exampleFromSchema(root, field(schema, "items"), nil)
		if !ok {
			return nil, false
		}
		// A single representative item keeps nested arrays bounded. An authored
		// example can still satisfy schemas that require larger collections.
		sample = []any{item}
	case kind != nil && kind.Value == "string":
		format := field(schema, "format")
		if format == nil {
			return nil, false
		}
		v, ok := formatExamples[format.Value]
		if !ok {
			return nil, false
		}
		sample = v
	case kind != nil && (kind.Value == "integer" || kind.Value == "number" || kind.Value == "boolean"):
		return numericExample(schema, compiled)
	default:
		return nil, false
	}
	return sample, compiled.Validate(sample) == nil
}

// numericExample retains the renderer's numeric range support. Pass only the
// scalar constraints so rendering cannot expand nested collections or invent text.
func numericExample(schema *yaml.Node, validator exampleValidator) (any, bool) {
	numeric := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for i := 0; i < len(schema.Content); i += 2 {
		switch schema.Content[i].Value {
		case "type", "minimum", "exclusiveMinimum", "maximum", "exclusiveMaximum", "multipleOf", "format":
			numeric.Content = append(numeric.Content, schema.Content[i:i+2]...)
		}
	}
	proxy := new(lowbase.SchemaProxy)
	if err := proxy.Build(context.Background(), nil, numeric, nil); err != nil {
		return nil, false
	}
	high := highbase.NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{Value: proxy, ValueNode: numeric}).Schema()
	if high == nil {
		return nil, false
	}
	generator := renderer.CreateRendererUsingDictionary("")
	generator.SetSeed(42)
	sample, err := generator.RenderSchemaWithError(high)
	return sample, err == nil && validator.Validate(sample) == nil
}

// Parent examples can expose missing property examples on the next lint pass.
// Use the corresponding values from the validated parent instance.
func childExampleOperations(schema *yaml.Node, path string, sample any) []Operation {
	object, ok := sample.(map[string]any)
	properties := field(schema, "properties")
	if !ok || properties == nil || properties.Kind != yaml.MappingNode {
		return nil
	}
	var ops []Operation
	for i := 0; i < len(properties.Content); i += 2 {
		name, child := properties.Content[i].Value, properties.Content[i+1]
		value, present := object[name]
		if !present || child.Kind != yaml.MappingNode {
			continue
		}
		childPath := path + "/properties/" + EscapeToken(name)
		if field(child, "example") == nil && field(child, "examples") == nil {
			encoded, err := json.Marshal(value)
			if err == nil {
				ops = append(ops, Operation{Op: "add", Path: childPath + "/example", Value: encoded})
			}
		}
		ops = append(ops, childExampleOperations(child, childPath, value)...)
	}
	return ops
}

func reviewedAsyncExample(root *yaml.Node, f Finding, nodes map[string]*yaml.Node) ([]Operation, string, string) {
	split := strings.LastIndex(f.Pointer, "/examples/")
	if split < 0 {
		return nil, "", "message example owner is unavailable"
	}
	owner := nodes[f.Pointer[:split]]
	example := nodes[f.Pointer]
	if owner == nil || example == nil || example.Kind != yaml.MappingNode {
		return nil, "", "message example must be an object"
	}
	if field(owner, "schemaFormat") != nil {
		return nil, "", "message schema format needs a matching generator"
	}
	var ops []Operation
	for _, name := range []string{"payload", "headers"} {
		schema := field(owner, name)
		if schema == nil {
			continue
		}
		if schema.Kind != yaml.MappingNode || field(schema, "schemaFormat") != nil {
			return nil, "", "message schema format needs a matching generator"
		}
		local := clone(schema)
		if field(local, "$schema") == nil {
			local.Content = append(local.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "$schema"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "http://json-schema.org/draft-07/schema#"})
		}
		generated, _, reason := reviewedExample(local, Finding{RuleID: "oas3-valid-schema-example", Pointer: ""}, sourceNodes(local))
		if len(generated) != 1 {
			if reason == ExampleInputRequired {
				reason = "message schema needs an authored string example"
			}
			return nil, "", reason
		}
		ops = append(ops, Operation{Op: "add", Path: f.Pointer + "/" + name, Value: generated[0].Value})
	}
	if len(ops) == 0 {
		return nil, "", "message has no schema from which to generate an example"
	}
	return ops, "example-generated", ""
}
