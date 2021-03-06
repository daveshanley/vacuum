package parser

import (
	_ "embed" // using embed, throws off golint.
	"encoding/json"
	"errors"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/utils"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	"sync"
)

var openAPISchemaGrab sync.Once

//go:embed schemas/oas3-schema.yaml
var OpenAPI3SchemaData string

//go:embed schemas/swagger2-schema.yaml
var OpenAPI2SchemaData string

var openAPI3Schema gojsonschema.JSONLoader
var openAPI2Schema gojsonschema.JSONLoader

// CheckSpecIsValidOpenAPI will check if a supplied specification is a valid OpenAPI spec or not, it runs a JSON
// Schema check against the supplied spec against the known standard. This is not yet linted, it's just validated as
// Being a valid spec against the schema.
func CheckSpecIsValidOpenAPI(spec []byte) (*gojsonschema.Result, error) {

	openAPISchemaGrab.Do(func() {
		// render yaml as JSON, YAML v3 is smart enough to do this for us.
		openAPI3JSON, _ := utils.ConvertYAMLtoJSON([]byte(OpenAPI3SchemaData))
		openAPI2JSON, _ := utils.ConvertYAMLtoJSON([]byte(OpenAPI2SchemaData))
		openAPI3Schema = gojsonschema.NewStringLoader(string(openAPI3JSON))
		openAPI2Schema = gojsonschema.NewStringLoader(string(openAPI2JSON))
	})

	if len(spec) <= 0 {
		return nil, errors.New("specification is empty")
	}

	specString := string(spec)

	if utils.IsJSON(specString) {

		return processJSONSpec(spec)

	}

	if utils.IsYAML(specString) {

		// convert to JSON (we don't need to worry about losing fidelity at this point).
		// there is little point capturing the errors here as have already unmarshalled data at least once.

		var yamlData map[string]interface{}
		uErr := yaml.Unmarshal(spec, &yamlData)
		if uErr != nil {
			return nil, uErr
		}

		jsonData, _ := json.Marshal(yamlData)

		return processJSONSpec(jsonData)

	}
	return nil, errors.New("spec is neither YAML nor JSON, unable to process")
}

func processJSONSpec(spec []byte) (*gojsonschema.Result, error) {

	// create loader
	doc := gojsonschema.NewStringLoader(string(spec))

	// which version is the spec?
	info, err := datamodel.ExtractSpecInfo(spec)
	if err != nil {
		return nil, err
	}
	var schemaToValidate gojsonschema.JSONLoader
	switch info.SpecType {
	case utils.OpenApi2:
		schemaToValidate = openAPI2Schema

	case utils.OpenApi3:
		schemaToValidate = openAPI3Schema
	}

	// validate spec
	res, err := gojsonschema.Validate(schemaToValidate, doc)

	// at this point, it's hard to trigger an error on validation.
	// but let's catch what ever could be thrown out.
	if err != nil {
		return nil, err
	}

	return res, nil
}
