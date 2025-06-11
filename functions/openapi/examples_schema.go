// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	v3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	v3Base "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/sourcegraph/conc"
	"gopkg.in/yaml.v3"
)

// ExamplesSchema will check anything that has an example, has a schema and it's valid.
type ExamplesSchema struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (es ExamplesSchema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasExampleSchema"}
}

// GetCategory returns the category of the ExamplesMissing rule.
func (es ExamplesSchema) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

var bannedErrors = []string{"if-then failed", "if-else failed", "allOf failed", "oneOf failed"}

// Helper function to check if a validation error should be filtered out for null values
// Only applies to OpenAPI versions before 3.1 (which use nullable: true)
func shouldFilterNullError(errorReason string, openAPIVersion string, schema *v3.Schema, example any) bool {
	// Only filter null errors for OpenAPI versions before 3.1
	// OpenAPI 3.1 uses JSON Schema draft 2019-09 which handles nullable differently
	if !strings.HasPrefix(openAPIVersion, "3.0") {
		return false
	}

	// Only filter if the error is about null values
	if !strings.Contains(errorReason, "got null") {
		return false
	}

	// Check if the schema itself is nullable
	if schema != nil && schema.Value != nil &&
		schema.Value.Nullable != nil && *schema.Value.Nullable {
		return true
	}

	// Check if this is an object with nullable properties
	if schema != nil && schema.Value != nil &&
		schema.Value.Type != nil && len(schema.Value.Type) > 0 &&
		schema.Value.Type[0] == "object" && schema.Value.Properties != nil {

		if exampleMap, ok := example.(map[string]interface{}); ok {
			// Check if any property in the example is null and that property is nullable
			for propPair := schema.Value.Properties.First(); propPair != nil; propPair = propPair.Next() {
				propName := propPair.Key()
				propSchemaProxy := propPair.Value()

				if propValue, exists := exampleMap[propName]; exists && propValue == nil {
					if propSchemaProxy != nil {
						// Get the actual schema from the proxy
						propSchema := propSchemaProxy.Schema()
						if propSchema != nil && propSchema.Nullable != nil && *propSchema.Nullable {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (es ExamplesSchema) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	buildResult := func(message, path string, key, node *yaml.Node, component v3.AcceptsRuleResults) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: key,
			EndNode:   vacuumUtils.BuildEndNode(key),
			Path:      path,
			Rule:      context.Rule,
		}
		component.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		return result
	}
	wg := conc.WaitGroup{}
	var expLock sync.Mutex

	validator := schema_validation.NewSchemaValidator()

	// Get OpenAPI version from the document
	var openAPIVersion string
	if context.Document != nil && context.Document.GetVersion() != "" {
		openAPIVersion = context.Document.GetVersion()
	}

	validateSchema := func(iKey *int,
		sKey, label string,
		s *v3.Schema,
		obj v3.AcceptsRuleResults,
		node *yaml.Node,
		keyNode *yaml.Node,
		example any) []model.RuleFunctionResult {

		var rx []model.RuleFunctionResult
		if s != nil && s.Value != nil {
			valid, validationErrors := validator.ValidateSchemaObject(s.Value, example)
			if !valid {
				var path string
				if iKey == nil && sKey == "" {
					path = fmt.Sprintf("%s.%s", obj.(v3.Foundational).GenerateJSONPath(), label)
				}
				if iKey != nil && sKey == "" {
					path = fmt.Sprintf("%s.%s[%d]", obj.(v3.Foundational).GenerateJSONPath(), label, *iKey)
				}
				if iKey == nil && sKey != "" {
					path = fmt.Sprintf("%s.%s['%s']", obj.(v3.Foundational).GenerateJSONPath(), label, sKey)
				}
				for _, r := range validationErrors {
					for _, err := range r.SchemaValidationErrors {
						// Skip any error that mentions "got null" for OpenAPI versions before 3.1
						// and only if the schema is actually marked as nullable
						if shouldFilterNullError(err.Reason, openAPIVersion, s, example) {
							continue
						}

						result := buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, err.Reason),
							path, keyNode, node, s)

						banned := false
						for g := range bannedErrors {
							if strings.Contains(err.Reason, bannedErrors[g]) {
								banned = true
								continue
							}
						}
						if !banned {
							rx = append(rx, result)
						}
					}
				}
			}
		}
		return rx
	}

	if context.DrDocument != nil && context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			s := context.DrDocument.Schemas[i]
			wg.Go(func() {
				if s.Value.Examples != nil {
					for x, ex := range s.Value.Examples {

						isRef, _, _ := utils.IsNodeRefValue(ex)
						if isRef {
							// extract node
							fNode, _, _, _ := low.LocateRefNodeWithContext(s.Value.ParentProxy.GoLow().GetContext(),
								ex, context.Index)
							if fNode != nil {
								ex = fNode
							} else {
								continue
							}
						}

						var example any
						_ = ex.Decode(&example)
						result := validateSchema(&x, "", "examples",
							s, s, s.Value.GoLow().Examples.Value[x].ValueNode,
							s.Value.GoLow().Examples.GetKeyNode(), example)

						if result != nil {
							expLock.Lock()
							results = append(results, result...)
							expLock.Unlock()
						}
					}
				}

				if s.Value.Example != nil {

					isRef, _, _ := utils.IsNodeRefValue(s.Value.Example)
					ref := s.Value.Example
					if isRef {
						// extract node
						fNode, _, _, _ := low.LocateRefNodeWithContext(s.Value.ParentProxy.GoLow().GetContext(),
							s.Value.Example, context.Index)
						if fNode != nil {
							ref = fNode
						}
					}
					changeKeys(0, ref)
					var example interface{}
					_ = ref.Decode(&example)

					result := validateSchema(nil, "", "example", s, s, s.Value.Example,
						s.Value.GoLow().Example.GetKeyNode(), example)
					if result != nil {
						expLock.Lock()
						results = append(results, result...)
						expLock.Unlock()
					}
				}
			})
		}
	}

	parseExamples := func(s *v3.Schema,
		obj v3.AcceptsRuleResults,
		examples *orderedmap.Map[string,
			*v3Base.Example]) []model.RuleFunctionResult {

		var rx []model.RuleFunctionResult
		for examplesPairs := examples.First(); examplesPairs != nil; examplesPairs = examplesPairs.Next() {

			example := examplesPairs.Value()
			exampleKey := examplesPairs.Key()

			var ex any
			if example.Value != nil {
				_ = example.Value.Decode(&ex)
				result := validateSchema(nil, exampleKey, "examples", s, obj, example.Value, example.GoLow().KeyNode, ex)
				if result != nil {
					rx = append(rx, result...)
				}
			}
		}
		return rx
	}

	parseExample := func(s *v3.Schema, node, key *yaml.Node) []model.RuleFunctionResult {

		var rx []model.RuleFunctionResult
		var ex any
		_ = node.Decode(&ex)

		result := validateSchema(nil, "", "example", s, s, node, key, ex)
		if result != nil {
			rx = append(rx, result...)
		}
		return rx
	}

	if context.DrDocument != nil && context.DrDocument.Parameters != nil {
		for i := range context.DrDocument.Parameters {
			p := context.DrDocument.Parameters[i]
			wg.Go(func() {
				if p.Value.Examples.Len() >= 1 && p.SchemaProxy != nil {
					expLock.Lock()
					results = append(results, parseExamples(p.SchemaProxy.Schema, p, p.Value.Examples)...)
					expLock.Unlock()
				} else {
					if p.Value.Example != nil && p.SchemaProxy != nil {
						expLock.Lock()
						results = append(results, parseExample(p.SchemaProxy.Schema, p.Value.Example,
							p.Value.GoLow().Example.GetKeyNode())...)
						expLock.Unlock()
					}
				}
			})
		}
	}

	if context.DrDocument != nil && context.DrDocument.Headers != nil {
		for i := range context.DrDocument.Headers {
			h := context.DrDocument.Headers[i]
			wg.Go(func() {
				if h.Value.Examples.Len() >= 1 && h.Schema != nil {
					expLock.Lock()
					results = append(results, parseExamples(h.Schema.Schema, h, h.Value.Examples)...)
					expLock.Unlock()
				} else {
					if h.Value.Example != nil && h.Schema != nil {
						expLock.Lock()
						results = append(results, parseExample(h.Schema.Schema, h.Value.Example,
							h.Value.GoLow().Example.GetKeyNode())...)
						expLock.Unlock()
					}
				}
			})
		}
	}

	if context.DrDocument != nil && context.DrDocument.MediaTypes != nil {

		for i := range context.DrDocument.MediaTypes {
			mt := context.DrDocument.MediaTypes[i]
			wg.Go(func() {
				if mt.Value.Examples.Len() >= 1 && mt.SchemaProxy != nil {
					expLock.Lock()
					results = append(results, parseExamples(mt.SchemaProxy.Schema, mt, mt.Value.Examples)...)
					expLock.Unlock()
				} else {
					if mt.Value.Example != nil && mt.SchemaProxy != nil {
						expLock.Lock()
						results = append(results, parseExample(mt.SchemaProxy.Schema, mt.Value.Example,
							mt.Value.GoLow().Example.GetKeyNode())...)
						expLock.Unlock()
					}
				}
			})
		}

	}
	wg.Wait()
	return results
}

// all keys need to be strings, anything else and we're going to have a bad time.
func changeKeys(depth int, node *yaml.Node) {
	if depth > 500 {
		return
	}
	if node.Tag == "!!timestamp" {
		node.Tag = "!!str"
	}
	for i, no := range node.Content {
		if i%2 != 0 {
			continue // keys only.
		}
		if node.Tag != "!!seq" && no.Tag != "!!str" {
			no.Tag = "!!str"
		}
		if len(no.Content) > 0 {
			depth++
			changeKeys(depth, no)
		}
	}
}
