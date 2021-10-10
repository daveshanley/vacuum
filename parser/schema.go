package parser

import "github.com/xeipuuv/gojsonschema"

func CreateJSONSchemaLoaders(schema []byte, jsonToTest []byte) (sch gojsonschema.JSONLoader, doc gojsonschema.JSONLoader) {
	sch = gojsonschema.NewStringLoader(string(schema))
	doc = gojsonschema.NewStringLoader(string(jsonToTest))
	return
}

func CreateJSONSchemaLoadersFromFiles(schema, jsonToTest string) (sch gojsonschema.JSONLoader, doc gojsonschema.JSONLoader) {
	sch = gojsonschema.NewReferenceLoader(schema)
	doc = gojsonschema.NewReferenceLoader(jsonToTest)
	return
}

func ValidateJSONAgainstSchema(schema []byte, jsonToTest []byte) (*gojsonschema.Result, error) {
	schemaLoader, inputLoader := CreateJSONSchemaLoaders(schema, jsonToTest)
	return gojsonschema.Validate(schemaLoader, inputLoader)
}

func ValidateJSONAgainstSchemaUsingFiles(schema, jsonToTest string) (*gojsonschema.Result, error) {
	schemaLoader, inputLoader := CreateJSONSchemaLoadersFromFiles(schema, jsonToTest)
	return gojsonschema.Validate(schemaLoader, inputLoader)
}
