// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	yamlAlt "github.com/ghodss/yaml"
	"github.com/pb33f/libopenapi/utils"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	"time"
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
func ValidateExample(jc *Schema) []*ExampleValidation {
	var examples []*ExampleValidation
	if len(jc.Properties) > 0 {
		for propName, prop := range jc.Properties {
			if prop.Type != nil && prop.Example != nil {

				isInt := false
				isBool := false
				isFloat := false
				isString := false

				if _, ok := prop.Example.(string); ok {
					isString = true
				}

				if _, ok := prop.Example.(bool); ok {
					isBool = true
				}

				if _, ok := prop.Example.(float64); ok {
					isFloat = true
				}

				if _, ok := prop.Example.(int); ok {
					isInt = true
				}

				invalidMessage := "example value '%v' in '%s' is not a valid %v"

				switch *prop.Type {
				case utils.StringLabel:
					if !isString {
						examples = append(examples, &ExampleValidation{
							Message: fmt.Sprintf(invalidMessage, prop.Example, propName, utils.IntegerLabel),
						})
					}
				case utils.IntegerLabel:
					if !isInt {
						examples = append(examples, &ExampleValidation{
							Message: fmt.Sprintf(invalidMessage, prop.Example, propName, utils.IntegerLabel),
						})
					}
				case utils.NumberLabel:
					if !isFloat {
						examples = append(examples, &ExampleValidation{
							Message: fmt.Sprintf(invalidMessage, prop.Example, propName, utils.NumberLabel),
						})
					}
				case utils.BooleanLabel:
					if !isBool {
						examples = append(examples, &ExampleValidation{
							Message: fmt.Sprintf(invalidMessage, prop.Example, propName, utils.BooleanLabel),
						})
					}
				}
			} else {
				if len(prop.Properties) > 0 {
					examples = append(examples, ValidateExample(prop)...)
				}
			}
		}
	}
	return examples
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

// ValidateNodeAgainstSchema will accept a schema and a node and check it's valid and return the result, or error.
func ValidateNodeAgainstSchema(schema *Schema, node *yaml.Node, isArray bool) (*gojsonschema.Result, error) {

	//convert node to raw yaml first, then convert to json to be used in schema validation
	var d []byte
	var e error
	if !isArray {
		d, e = yaml.Marshal(node)
	} else {
		d, e = yaml.Marshal([]*yaml.Node{node})
	}

	// safely convert yaml to JSON.
	n, err := yamlAlt.YAMLToJSON(d)
	if err != nil {
		return nil, e
	}

	// convert schema to JSON.
	sJson, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	// create loaders
	rawObject := gojsonschema.NewStringLoader(string(n))
	schemaToCheck := gojsonschema.NewStringLoader(string(sJson))

	//validate
	return gojsonschema.Validate(schemaToCheck, rawObject)

}
