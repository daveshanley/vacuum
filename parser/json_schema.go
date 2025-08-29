// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
	yamlAlt "github.com/ghodss/yaml"
	validationErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type Schema struct {
	Schema               *string            `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Id                   *string            `json:"$id,omitempty" yaml:"$id,omitempty"`
	Title                *string            `json:"title,omitempty" yaml:"title,omitempty"`
	Required             *[]string          `json:"required,omitempty" yaml:"required,omitempty"`
	Enum                 *[]string          `json:"enum,omitempty" yaml:"enum,omitempty"`
	Description          *string            `json:"description,omitempty" yaml:"description,omitempty"`
	Type                 *string            `json:"type,omitempty" yaml:"type,omitempty"`
	ContentEncoding      *string            `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
	ContentSchema        *string            `json:"contentSchema,omitempty" yaml:"contentSchema,omitempty"`
	Items                *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	MultipleOf           *int               `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum              *int               `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum     *int               `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum              *int               `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum     *int               `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems             *int               `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength            *int               `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern              *string            `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxContains          *int               `json:"maxContains,omitempty" yaml:"maxContains,omitempty"`
	MinContains          *int               `json:"minContains,omitempty" yaml:"minContains,omitempty"`
	MaxProperties        *int               `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties        *int               `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Format               *string            `json:"format,omitempty" yaml:"format,omitempty"`           // OpenAPI
	Example              interface{}        `json:"example,omitempty" yaml:"example,omitempty"`         // OpenAPI
	Nullable             bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`       // OpenAPI
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty" yaml:"ad,omitempty"` // OpenAPI
}

type ExampleValidation struct {
	Message string
}

// ValidateExample will check if a schema has a valid type and example, and then perform a simple validation on the
// value that has been set.
func ValidateExample(jc *highBase.Schema) []*ExampleValidation {
	var examples []*ExampleValidation
	if orderedmap.Len(jc.Properties) > 0 {
		for pair := orderedmap.First(jc.Properties); pair != nil; pair = pair.Next() {
			propName := pair.Key()
			prop := pair.Value()

			sc := prop.Schema()
			if sc.Type != nil && sc.Example != nil {

				// todo: replace this with reflection.
				isInt := false
				isBool := false
				isFloat := false
				isString := false

				var example any
				_ = sc.Example.Decode(&example)

				if _, ok := example.(string); ok {
					isString = true
				}

				if _, ok := example.(bool); ok {
					isBool = true
				}

				if _, ok := example.(float64); ok {
					isFloat = true
				}
				if _, ok := example.(float32); ok {
					isFloat = true
				}

				if _, ok := example.(int); ok {
					isInt = true
				}
				if _, ok := example.(int64); ok {
					isInt = true
				}

				invalidMessage := "example value '%v' in '%s' is not a valid %v"

				for y := range sc.Type {
					switch sc.Type[y] {
					case utils.StringLabel:
						if !isString {
							examples = append(examples, &ExampleValidation{
								Message: fmt.Sprintf(invalidMessage, sc.Example, propName, utils.IntegerLabel),
							})
						}
					case utils.IntegerLabel:
						if !isInt {
							examples = append(examples, &ExampleValidation{
								Message: fmt.Sprintf(invalidMessage, sc.Example, propName, utils.IntegerLabel),
							})
						}
					case utils.NumberLabel:
						if !isFloat {
							examples = append(examples, &ExampleValidation{
								Message: fmt.Sprintf(invalidMessage, sc.Example, propName, utils.NumberLabel),
							})
						}
					case utils.BooleanLabel:
						if !isBool {
							examples = append(examples, &ExampleValidation{
								Message: fmt.Sprintf(invalidMessage, sc.Example, propName, utils.BooleanLabel),
							})
						}
					}
				}
				switch sc.Type {
				}
			} else {
				if orderedmap.Len(sc.Properties) > 0 {
					examples = append(examples, ValidateExample(sc)...)
				}
			}
		}
	}
	return examples
}

func ConvertYAMLIntoJSONSchema(str string, index *index.SpecIndex) (*highBase.Schema, error) {
	node := yaml.Node{}
	err := yaml.Unmarshal([]byte(str), &node)
	if err != nil {
		return nil, err
	}
	return ConvertNodeIntoJSONSchema(node.Content[0], index)
}

func ConvertNodeIntoJSONSchema(node *yaml.Node, idx *index.SpecIndex) (*highBase.Schema, error) {
	sch := lowBase.Schema{}
	mbErr := low.BuildModel(node, &sch)
	if mbErr != nil {
		return nil, mbErr
	}

	path := ""

	isRef, _, ref := utils.IsNodeRefValue(node)
	if isRef {
		r := strings.Split(ref, "#")
		if len(r) == 2 {
			if r[0] != "" {
				path = r[0]
			}
		} else {
			path = r[0]
		}
	}

	if path == "" && idx != nil {
		path = idx.GetSpecAbsolutePath()
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, path)

	schErr := sch.Build(ctx, node, idx)
	if schErr != nil {
		return nil, schErr
	}
	highSch := highBase.NewSchema(&sch)
	return highSch, nil
}

// ConvertNodeDefinitionIntoSchema will convert any definition node (components, params, etc.) into a standard
// Schema that can be used with JSONSchema. This will auto-timeout of th
func ConvertNodeDefinitionIntoSchema(node *yaml.Node) (*Schema, error) {
	schChan := make(chan Schema, 1)
	errChan := make(chan error, 1)

	go func() {
		var dat []byte
		var err error
		dat, err = yaml.Marshal(node)
		if dat == nil && err != nil {
			errChan <- err
		}
		var schema Schema
		err = yaml.Unmarshal(dat, &schema)

		if err != nil {
			errChan <- err
		}
		schChan <- schema
	}()

	var schema Schema
	select {
	case err := <-errChan:
		return nil, err
	case schema = <-schChan:
		schema.Schema = &utils.SchemaSource
		schema.Id = &utils.SchemaId
		return &schema, nil
	case <-time.After(500 * time.Millisecond): // even this seems long to me.
		return nil, errors.New("schema is too big! It failed to unpack in a reasonable timeframe")
	}
}

// Global validator instance and mutex to ensure thread-safe schema validation
// This fixes issue #512 where concurrent validations cause non-deterministic results
var (
	globalValidator     schema_validation.SchemaValidator
	globalValidatorOnce sync.Once
	globalValidatorMu   sync.Mutex
)

// getGlobalValidator returns a singleton validator instance
func getGlobalValidator(ctx *model.RuleFunctionContext) schema_validation.SchemaValidator {
	globalValidatorOnce.Do(func() {
		if ctx != nil && ctx.Logger != nil {
			globalValidator = schema_validation.NewSchemaValidatorWithLogger(ctx.Logger)
		} else {
			globalValidator = schema_validation.NewSchemaValidator()
		}
	})
	return globalValidator
}

// ValidateNodeAgainstSchema will accept a schema and a node and check it's valid and return the result, or error.
func ValidateNodeAgainstSchema(ctx *model.RuleFunctionContext, schema *highBase.Schema, node *yaml.Node, isArray bool) (bool, []*validationErrors.ValidationError) {
	// convert node to raw yaml first, then convert to json to be used in schema validation
	var d []byte
	var e error
	if !isArray {
		d, e = yaml.Marshal(node)
	} else {
		if !utils.IsNodeArray(node) {
			d, e = yaml.Marshal([]*yaml.Node{node})
		} else {
			d, e = yaml.Marshal(node)
		}
	}
	if e != nil {
		return false, []*validationErrors.ValidationError{{Message: e.Error()}}
	}

	// safely convert yaml to JSON.
	n, err := yamlAlt.YAMLToJSON(d)
	if err != nil {
		return false, []*validationErrors.ValidationError{{Message: err.Error()}}
	}

	var decoded any
	_ = json.Unmarshal(n, &decoded)

	// Use global validator with mutex protection to prevent concurrent schema mutations
	// This ensures thread-safe validation when multiple goroutines validate schemas
	validator := getGlobalValidator(ctx)
	
	// Lock to ensure only one validation happens at a time
	// This prevents race conditions in schema.RenderInline() which mutates internal state
	globalValidatorMu.Lock()
	defer globalValidatorMu.Unlock()
	
	return validator.ValidateSchemaObject(schema, decoded)
}
