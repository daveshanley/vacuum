package repair

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
)

// ValidateReviewedExamples checks the final instance values after user edits.
// It is independent of the selected lint rules and cannot be waived by accepting
// an unknown contract impact. Current providers only emit local schema instances.
func ValidateReviewedExamples(data []byte, plan Plan) error {
	var targets []Target
	for _, t := range plan.Targets {
		switch t.RuleID {
		case "oas3-missing-example", "oas3-valid-schema-example", "json-schema-examples-valid", "asyncapi-message-examples":
			targets = append(targets, t)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	doc, err := ParseDocument(data)
	if err != nil {
		return err
	}
	root := doc.Content[0]
	nodes := sourceNodes(root)
	checked := make(map[string]bool)
	for _, op := range plan.Operations {
		if op.Op != "add" && op.Op != "replace" {
			continue
		}
		var schema *yaml.Node
		schemaRoot := root
		if field(root, "asyncapi") != nil {
			split := strings.LastIndex(op.Path, "/examples/")
			if split < 0 {
				continue
			}
			rest := strings.Split(op.Path[split+len("/examples/"):], "/")
			if len(rest) != 2 || (rest[1] != "payload" && rest[1] != "headers") {
				continue
			}
			schema = field(nodes[op.Path[:split]], rest[1])
			schemaRoot = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "$schema"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: "http://json-schema.org/draft-07/schema#"}}}
		} else if strings.HasSuffix(op.Path, "/example") {
			schema = nodes[strings.TrimSuffix(op.Path, "/example")]
		} else if split := strings.LastIndex(op.Path, "/examples/"); split >= 0 {
			if _, err := strconv.Atoi(op.Path[split+len("/examples/"):]); err != nil {
				continue
			}
			schema = nodes[op.Path[:split]]
		}
		if schema == nil {
			continue
		}
		compiled, err := compileExampleSchema(schemaRoot, schema)
		if err != nil || compiled == nil {
			return fmt.Errorf("cannot validate proposed example at %s", op.Path)
		}
		sample := nodes[op.Path]
		if sample == nil {
			return fmt.Errorf("proposed example is missing at %s", op.Path)
		}
		value, err := schemautil.NodeToInterface(sample)
		if err != nil {
			return err
		}
		if err := compiled.Validate(value); err != nil {
			return fmt.Errorf("proposed example at %s does not satisfy its schema: %w", op.Path, err)
		}
		checked[op.Path] = true
	}
	for _, target := range targets {
		covered := false
		for path := range checked {
			if overlaps(path, target.Pointer) {
				covered = true
				break
			}
		}
		if !covered {
			return fmt.Errorf("proposed example target cannot be validated: %s", target.Pointer)
		}
	}
	return nil
}

type exampleValidator interface{ Validate(any) error }

type oas30ExampleValidator struct {
	schema    *highbase.Schema
	validator schema_validation.SchemaValidator
}

func (v oas30ExampleValidator) Validate(value any) error {
	valid, errs := v.validator.ValidateSchemaObjectWithVersion(v.schema, value, 3.0)
	if !valid || len(errs) > 0 {
		return fmt.Errorf("example violates OpenAPI 3.0 schema constraints")
	}
	return nil
}

func compileExampleSchema(root, schema *yaml.Node) (exampleValidator, error) {
	if version := field(root, "openapi"); version != nil && strings.HasPrefix(version.Value, "3.0.") {
		proxy := new(lowbase.SchemaProxy)
		if err := proxy.Build(context.Background(), nil, schema, nil); err != nil {
			return nil, err
		}
		high := highbase.NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{Value: proxy, ValueNode: schema}).Schema()
		if high == nil {
			return nil, fmt.Errorf("OpenAPI example schema is unavailable")
		}
		return oas30ExampleValidator{schema: high, validator: schema_validation.NewSchemaValidator()}, nil
	}
	copy := clone(schema)
	if field(copy, "$schema") == nil {
		dialect := field(root, "$schema")
		if oasDialect := field(root, "jsonSchemaDialect"); oasDialect != nil {
			dialect = oasDialect
		}
		if dialect != nil {
			copy.Content = append(copy.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "$schema"}, clone(dialect))
		}
	}
	compiled, err := schemautil.CompileSchema(copy)
	if compiled == nil {
		return nil, err
	}
	return compiled, err
}
