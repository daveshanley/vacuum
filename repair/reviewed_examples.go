package repair

import (
	"context"
	"encoding/json"
	"fmt"
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
		if n != nil && n.Kind == yaml.MappingNode && (field(n, "type") != nil || field(n, "properties") != nil || field(n, "items") != nil || field(n, "enum") != nil || field(n, "allOf") != nil || field(n, "oneOf") != nil || field(n, "anyOf") != nil) {
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
	// Try existing authored data before synthesis. A default may supply a sample,
	// but this provider never changes the default itself.
	var sample any
	valid := false
	for _, name := range []string{"example", "default", "const"} {
		if n := field(schema, name); n != nil {
			v, e := schemautil.NodeToInterface(n)
			if e == nil && compiled.Validate(v) == nil {
				sample = v
				valid = true
				break
			}
		}
	}
	if !valid {
		clean := clone(schema)
		stripExampleHints(clean)
		proxy := new(lowbase.SchemaProxy)
		if err := proxy.Build(context.Background(), nil, clean, nil); err != nil {
			return nil, "", err.Error()
		}
		high := highbase.NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{Value: proxy, ValueNode: clean}).Schema()
		if high == nil {
			return nil, "", "example schema model is unavailable"
		}
		generator := renderer.CreateRendererUsingDictionary("")
		generator.SetSeed(42)
		sample, err = generator.RenderSchemaWithError(high)
		if err != nil {
			return nil, "", fmt.Sprintf("bounded example generation failed: %v", err)
		}
	}
	encoded, err := json.Marshal(sample)
	if err != nil {
		return nil, "", err.Error()
	}
	// Decode through the same numeric representation as the schema validator.
	node, err := valueNode(encoded)
	if err != nil {
		return nil, "", err.Error()
	}
	value, err := schemautil.NodeToInterface(node)
	if err != nil || compiled.Validate(value) != nil {
		return nil, "", "generated example does not satisfy the schema"
	}
	op := "add"
	if nodes[target] != nil {
		op = "replace"
	}
	ops := []Operation{{Op: op, Path: target, Value: encoded}}
	if f.RuleID == "oas3-missing-example" {
		ops = append(ops, childExampleOperations(schema, path, value)...)
	}
	return ops, "example-generated", ""
}

func stripExampleHints(n *yaml.Node) {
	if n == nil || n.Kind != yaml.MappingNode {
		return
	}
	for i := len(n.Content) - 2; i >= 0; i -= 2 {
		key, value := n.Content[i].Value, n.Content[i+1]
		switch key {
		case "example", "examples", "default":
			n.Content = append(n.Content[:i], n.Content[i+2:]...)
		case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas":
			if value.Kind == yaml.MappingNode {
				for j := 1; j < len(value.Content); j += 2 {
					stripExampleHints(value.Content[j])
				}
			}
		case "allOf", "anyOf", "oneOf", "prefixItems":
			if value.Kind == yaml.SequenceNode {
				for _, child := range value.Content {
					stripExampleHints(child)
				}
			}
		case "items", "additionalProperties", "unevaluatedProperties", "contains", "not", "if", "then", "else", "propertyNames":
			stripExampleHints(value)
		}
	}
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
			return nil, "", reason
		}
		ops = append(ops, Operation{Op: "add", Path: f.Pointer + "/" + name, Value: generated[0].Value})
	}
	if len(ops) == 0 {
		return nil, "", "message has no schema from which to generate an example"
	}
	return ops, "example-generated", ""
}
