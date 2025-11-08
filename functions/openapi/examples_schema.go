// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	v3Base "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// ExamplesSchema will check anything that has an example, has a schema and it's valid.
type ExamplesSchema struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (es ExamplesSchema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasExampleSchema"}
}

// GetCategory returns the category of the ExamplesMissing rule.
func (es ExamplesSchema) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

var bannedErrors = []string{"if-then failed", "if-else failed", "allOf failed", "oneOf failed"}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (es ExamplesSchema) RunRule(_ []*yaml.Node, ruleContext model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if ruleContext.DrDocument == nil {
		return results
	}

	// get configuration values from context, use defaults if not set
	maxConcurrentValidations := ruleContext.MaxConcurrentValidations
	if maxConcurrentValidations <= 0 {
		maxConcurrentValidations = 10 // Default: 10 parallel validations
	}

	validationTimeout := ruleContext.ValidationTimeout
	if validationTimeout <= 0 {
		validationTimeout = 10 * time.Second // Default: 10 seconds
	}

	// create a timeout context for the entire validation process
	ctx, cancel := context.WithTimeout(context.Background(), validationTimeout)
	defer cancel()

	// create semaphore for concurrency limiting
	sem := make(chan struct{}, maxConcurrentValidations)

	// track active workers
	var activeWorkers int32
	var completedWorkers int32

	buildResult := func(message, path string, key, node *yaml.Node, component v3.AcceptsRuleResults) model.RuleFunctionResult {
		// try to find all paths for this node if it's a schema
		var allPaths []string
		if schema, ok := component.(*v3.Schema); ok {
			_, allPaths = vacuumUtils.LocateSchemaPropertyPaths(ruleContext, schema, key, node)
		}

		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: key,
			EndNode:   vacuumUtils.BuildEndNode(key),
			Path:      path,
			Rule:      ruleContext.Rule,
		}

		// set the Paths array if we found multiple locations
		if len(allPaths) > 1 {
			result.Paths = allPaths
		}

		component.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		return result
	}

	var expLock sync.Mutex
	var wg sync.WaitGroup

	// helper function to spawn workers with context and concurrency control
	spawnWorker := func(work func()) {
		// check if context is already cancelled before spawning
		select {
		case <-ctx.Done():
			return
		default:
		}

		atomic.AddInt32(&activeWorkers, 1)
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer atomic.AddInt32(&completedWorkers, 1)
			defer atomic.AddInt32(&activeWorkers, -1)

			// recover from panics to prevent crashes
			defer func() {
				if r := recover(); r != nil {
					// log panic if logger available
					if ruleContext.Logger != nil {
						ruleContext.Logger.Error("ExamplesSchema validation panic", "error", r)
					}
				}
			}()

			// try to acquire semaphore with context
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				// context cancelled while waiting for semaphore
				return
			}

			// check context again before starting work
			select {
			case <-ctx.Done():
				return
			default:
				work()
			}
		}()
	}

	validator := schema_validation.NewSchemaValidator()
	xmlValidator := schema_validation.NewXMLValidator()
	version := ruleContext.Document.GetSpecInfo().VersionNumeric
	validateSchema := func(iKey *int,
		sKey, label string,
		s *v3.Schema,
		obj v3.AcceptsRuleResults,
		node *yaml.Node,
		keyNode *yaml.Node,
		example any,
	) []model.RuleFunctionResult {
		var rx []model.RuleFunctionResult
		if s != nil && s.Value != nil {
			var valid bool
			var validationErrors []*errors.ValidationError
			if version > 0 {
				valid, validationErrors = validator.ValidateSchemaObjectWithVersion(s.Value, example, version)
			} else {
				valid, validationErrors = validator.ValidateSchemaObject(s.Value, example)
			}
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
						result := buildResult(vacuumUtils.SuppliedOrDefault(ruleContext.Rule.Message, err.Reason),
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

	if ruleContext.DrDocument != nil && ruleContext.DrDocument.Schemas != nil {
		for i := range ruleContext.DrDocument.Schemas {
			s := ruleContext.DrDocument.Schemas[i]
			spawnWorker(func() {
				// check context at start of work
				select {
				case <-ctx.Done():
					return
				default:
				}

				if s.Value.Examples != nil {
					for x, ex := range s.Value.Examples {
						// check context in loop
						select {
						case <-ctx.Done():
							return
						default:
						}

						isRef, _, _ := utils.IsNodeRefValue(ex)
						if isRef {
							// extract node
							fNode, _, _, _ := low.LocateRefNodeWithContext(s.Value.ParentProxy.GoLow().GetContext(),
								ex, ruleContext.Index)
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
							s.Value.Example, ruleContext.Index)
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

	// exampleValidatorFunc defines the function signature for validating examples
	type exampleValidatorFunc func(example any) (bool, []*errors.ValidationError)

	// processValidationErrors converts validation errors to rule function results
	processValidationErrors := func(
		validationErrors []*errors.ValidationError,
		path string,
		keyNode, valueNode *yaml.Node,
		schema *v3.Schema,
	) []model.RuleFunctionResult {
		var rx []model.RuleFunctionResult
		for _, r := range validationErrors {
			for _, err := range r.SchemaValidationErrors {
				result := buildResult(
					vacuumUtils.SuppliedOrDefault(ruleContext.Rule.Message, err.Reason),
					path, keyNode, valueNode, schema)

				// check if this is a banned error
				banned := false
				for g := range bannedErrors {
					if strings.Contains(err.Reason, bannedErrors[g]) {
						banned = true
						break
					}
				}
				if !banned {
					rx = append(rx, result)
				}
			}
		}
		return rx
	}

	// createJSONValidator creates a validator for JSON examples
	createJSONValidator := func(s *v3.Schema, ver float32) exampleValidatorFunc {
		return func(example any) (bool, []*errors.ValidationError) {
			if s != nil && s.Value != nil {
				if ver > 0 {
					return validator.ValidateSchemaObjectWithVersion(s.Value, example, ver)
				}
				return validator.ValidateSchemaObject(s.Value, example)
			}
			return true, nil
		}
	}

	// createXMLValidator creates a validator for XML examples
	createXMLValidator := func(s *v3.Schema, ver float32) exampleValidatorFunc {
		return func(example any) (bool, []*errors.ValidationError) {
			if xmlStr, ok := example.(string); ok {
				if ver > 0 {
					return xmlValidator.ValidateXMLStringWithVersion(s.Value, xmlStr, ver)
				}
				return xmlValidator.ValidateXMLString(s.Value, xmlStr)
			}
			return true, nil
		}
	}

	parseExamples := func(s *v3.Schema,
		obj v3.AcceptsRuleResults,
		examples *orderedmap.Map[string, *v3Base.Example],
		validatorFunc exampleValidatorFunc,
	) []model.RuleFunctionResult {
		var rx []model.RuleFunctionResult
		for examplesPairs := examples.First(); examplesPairs != nil; examplesPairs = examplesPairs.Next() {
			example := examplesPairs.Value()
			exampleKey := examplesPairs.Key()

			var ex any
			if example.Value != nil {
				_ = example.Value.Decode(&ex)
				valid, validationErrors := validatorFunc(ex)

				if !valid {
					path := fmt.Sprintf("%s.examples['%s']", obj.(v3.Foundational).GenerateJSONPath(), exampleKey)
					rx = append(rx, processValidationErrors(validationErrors, path,
						example.GoLow().KeyNode, example.Value, s)...)
				}
			}
		}
		return rx
	}

	parseExample := func(s *v3.Schema, node, key *yaml.Node, validatorFunc exampleValidatorFunc) []model.RuleFunctionResult {
		var rx []model.RuleFunctionResult
		var ex any
		_ = node.Decode(&ex)

		valid, validationErrors := validatorFunc(ex)
		if !valid {
			path := ""
			rx = append(rx, processValidationErrors(validationErrors, path, key, node, s)...)
		}
		return rx
	}

	if ruleContext.DrDocument != nil && ruleContext.DrDocument.Parameters != nil {
		for i := range ruleContext.DrDocument.Parameters {
			p := ruleContext.DrDocument.Parameters[i]
			spawnWorker(func() {
				// check context at start of work
				select {
				case <-ctx.Done():
					return
				default:
				}

				if p.Value.Examples.Len() >= 1 && p.SchemaProxy != nil {
					jsonValidator := createJSONValidator(p.SchemaProxy.Schema, version)
					expLock.Lock()
					if p.Value.Examples != nil && p.Value.Examples.Len() > 0 {
						results = append(results, parseExamples(p.SchemaProxy.Schema, p, p.Value.Examples, jsonValidator)...)
					}
					expLock.Unlock()
				} else {
					if p.Value.Example != nil && p.SchemaProxy != nil {
						jsonValidator := createJSONValidator(p.SchemaProxy.Schema, version)
						expLock.Lock()
						if p.Value.Examples != nil && p.Value.Examples.Len() > 0 {
							results = append(results, parseExample(p.SchemaProxy.Schema, p.Value.Example,
								p.Value.GoLow().Example.GetKeyNode(), jsonValidator)...)
						}
						expLock.Unlock()
					}
				}
			})
		}
	}

	if ruleContext.DrDocument != nil && ruleContext.DrDocument.Headers != nil {
		for i := range ruleContext.DrDocument.Headers {
			h := ruleContext.DrDocument.Headers[i]
			spawnWorker(func() {
				// check context at start of work
				select {
				case <-ctx.Done():
					return
				default:
				}

				if h.Value.Examples.Len() >= 1 && h.Schema != nil {
					jsonValidator := createJSONValidator(h.Schema.Schema, version)
					expLock.Lock()
					results = append(results, parseExamples(h.Schema.Schema, h, h.Value.Examples, jsonValidator)...)
					expLock.Unlock()
				} else {
					if h.Value.Example != nil && h.Schema != nil {
						jsonValidator := createJSONValidator(h.Schema.Schema, version)
						expLock.Lock()
						results = append(results, parseExample(h.Schema.Schema, h.Value.Example,
							h.Value.GoLow().Example.GetKeyNode(), jsonValidator)...)
						expLock.Unlock()
					}
				}
			})
		}
	}

	if ruleContext.DrDocument != nil && ruleContext.DrDocument.MediaTypes != nil {
		for i := range ruleContext.DrDocument.MediaTypes {
			mt := ruleContext.DrDocument.MediaTypes[i]
			spawnWorker(func() {
				// check context at start of work
				select {
				case <-ctx.Done():
					return
				default:
				}

				// check if this is xml content type
				mediaTypeStr := mt.GetKeyValue()
				isXML := schema_validation.IsXMLContentType(mediaTypeStr)

				if mt.Value.Examples.Len() >= 1 && mt.SchemaProxy != nil {
					var exampleValidator exampleValidatorFunc
					if isXML {
						exampleValidator = createXMLValidator(mt.SchemaProxy.Schema, version)
					} else {
						exampleValidator = createJSONValidator(mt.SchemaProxy.Schema, version)
					}
					expLock.Lock()
					results = append(results, parseExamples(mt.SchemaProxy.Schema, mt, mt.Value.Examples, exampleValidator)...)
					expLock.Unlock()
				} else {
					if mt.Value.Example != nil && mt.SchemaProxy != nil {
						var exampleValidator exampleValidatorFunc
						if isXML {
							exampleValidator = createXMLValidator(mt.SchemaProxy.Schema, version)
						} else {
							exampleValidator = createJSONValidator(mt.SchemaProxy.Schema, version)
						}
						expLock.Lock()
						results = append(results, parseExample(mt.SchemaProxy.Schema, mt.Value.Example,
							mt.Value.GoLow().Example.GetKeyNode(), exampleValidator)...)
						expLock.Unlock()
					}
				}
			})
		}
	}

	// wait for all workers to complete or context to timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// all workers completed normally
		if ruleContext.Logger != nil && atomic.LoadInt32(&completedWorkers) > 0 {
			ruleContext.Logger.Debug("ExamplesSchema completed validations",
				"completed", atomic.LoadInt32(&completedWorkers))
		}
	case <-ctx.Done():
		// timeout occurred - return whatever results we have
		if ruleContext.Logger != nil {
			ruleContext.Logger.Warn("ExamplesSchema validation timeout",
				"timeout", validationTimeout,
				"active", atomic.LoadInt32(&activeWorkers),
				"completed", atomic.LoadInt32(&completedWorkers),
				"results", len(results))
		}
	}

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
