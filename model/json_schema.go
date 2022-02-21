package model

type Schema struct {
	Schema           *string           `json:"$schema,omitempty"`
	Id               *string           `json:"$id,omitempty"`
	Title            *string           `json:"title,omitempty"`
	Required         *[]string         `json:"required,omitempty"`
	Enum             *[]string         `json:"enum,omitempty"`
	Description      *string           `json:"description,omitempty"`
	Type             *string           `json:"type,omitempty"`
	ContentEncoding  *string           `json:"contentEncoding,omitempty"`
	ContentSchema    *string           `json:"contentSchema,omitempty"`
	Items            *Schema           `json:"items,omitempty"`
	MultipleOf       *int              `json:"multipleOf,omitempty"`
	Maximum          *int              `json:"maximum,omitempty"`
	ExclusiveMaximum *int              `json:"exclusiveMaximum,omitempty"`
	Minimum          *int              `json:"minimum,omitempty"`
	ExclusiveMinimum *int              `json:"exclusiveMinimum,omitempty"`
	UniqueItems      bool              `json:"uniqueItems,omitempty"`
	MaxItems         *int              `json:"maxItems,omitempty"`
	MinItems         *int              `json:"minItems,omitempty"`
	MaxLength        *int              `json:"maxLength,omitempty"`
	MinLength        *int              `json:"minLength,omitempty"`
	Pattern          *string           `json:"pattern,omitempty"`
	MaxContains      *int              `json:"maxContains,omitempty"`
	MinContains      *int              `json:"minContains,omitempty"`
	MaxProperties    *int              `json:"maxProperties,omitempty"`
	MinProperties    *int              `json:"minProperties,omitempty"`
	Properties       map[string]Schema `json:"properties,omitempty"`
}

//var objectL = "object"
//var integerL = "integer"
//var numberL = "number"
//var stringL = "string"
//var binaryL = "binary"
//var arrayL = "array"
//var booleanL = "boolean"
//var schemaS = "https://json-schema.org/draft/2020-12/schema"
