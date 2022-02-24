package parser

import (
	"encoding/json"
	"github.com/daveshanley/vacuum/utils"
	yamlAlt "github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

type Schema struct {
	Schema           *string           `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Id               *string           `json:"$id,omitempty" yaml:"$id,omitempty"`
	Title            *string           `json:"title,omitempty" yaml:"title,omitempty"`
	Required         *[]string         `json:"required,omitempty" yaml:"required,omitempty"`
	Enum             *[]string         `json:"enum,omitempty" yaml:"enum,omitempty"`
	Description      *string           `json:"description,omitempty" yaml:"description,omitempty"`
	Type             *string           `json:"type,omitempty" yaml:"type,omitempty"`
	ContentEncoding  *string           `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
	ContentSchema    *string           `json:"contentSchema,omitempty" yaml:"contentSchema,omitempty"`
	Items            *Schema           `json:"items,omitempty" yaml:"items,omitempty"`
	MultipleOf       *int              `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum          *int              `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum *int              `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum          *int              `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum *int              `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	UniqueItems      bool              `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxItems         *int              `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems         *int              `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxLength        *int              `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength        *int              `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern          *string           `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxContains      *int              `json:"maxContains,omitempty" yaml:"maxContains,omitempty"`
	MinContains      *int              `json:"minContains,omitempty" yaml:"minContains,omitempty"`
	MaxProperties    *int              `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties    *int              `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Properties       map[string]Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Format           *string           `json:"format,omitempty" yaml:"format,omitempty"` // OpenAPI
	//Example              *string           `json:"example,omitempty" yaml:"example,omitempty"`         // OpenAPI
	Nullable             bool        `json:"nullable,omitempty" yaml:"nullable,omitempty"`       // OpenAPI
	AdditionalProperties interface{} `json:"additionalProperties,omitempty" yaml:"ad,omitempty"` // OpenAPI
}

// ConvertNodeDefinitionIntoSchema will convert any definition node (components, params, etc.) into a standard
// Schema that can be used with JSONSchema.
func ConvertNodeDefinitionIntoSchema(node *yaml.Node) (*Schema, error) {
	dat, err := yaml.Marshal(node)
	if err != nil {
		return nil, err
	}
	var schema Schema
	err = yaml.Unmarshal(dat, &schema)

	schema.Schema = &utils.SchemaSource
	schema.Id = &utils.SchemaId

	if err != nil {
		return nil, err
	}
	return &schema, nil
}

// ValidateNodeAgainstSchema will accept a schema and a node and check it's valid and return the result, or error.
func ValidateNodeAgainstSchema(schema *Schema, node *yaml.Node) (*gojsonschema.Result, error) {

	// convert node to raw yaml first, then convert to json to be used in schema validation
	d, e := yaml.Marshal(node)
	if e != nil {
		return nil, e
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

	// validate
	return gojsonschema.Validate(schemaToCheck, rawObject)

}
