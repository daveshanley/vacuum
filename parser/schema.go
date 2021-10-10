package parser

import "github.com/xeipuuv/gojsonschema"

// CreateJSONSchemaLoaders will take in two byte arrays of JSON and build loaders read for validation.
func CreateJSONSchemaLoaders(schema []byte, jsonToTest []byte) (sch gojsonschema.JSONLoader, doc gojsonschema.JSONLoader) {
	sch = gojsonschema.NewStringLoader(string(schema))
	doc = gojsonschema.NewStringLoader(string(jsonToTest))
	return
}

func ValidateJSONAgainstSchema(schema []byte, jsonToTest []byte) (*gojsonschema.Result, error) {
	schemaLoader, inputLoader := CreateJSONSchemaLoaders(schema, jsonToTest)
	return gojsonschema.Validate(schemaLoader, inputLoader)
}
