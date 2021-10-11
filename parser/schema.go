package parser

import (
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	"sync"
)

var openAPISchemaGrab sync.Once

//go:embed schemas/oas3-schema.yaml
var openAPI3SchemaData string

//go:embed schemas/swagger2-schema.yaml
var openAPI2SchemaData string

var openAPI3Schema gojsonschema.JSONLoader
var openAPI2Schema gojsonschema.JSONLoader

func CheckSpecIsValidOpenAPI(spec []byte) (*gojsonschema.Result, error) {

	openAPISchemaGrab.Do(func() {
		// render yaml as JSON, YAML v3 is smart enough to do this for us.
		openAPI3JSON, _ := utils.ConvertYAMLtoJSON([]byte(openAPI3SchemaData))
		openAPI2JSON, _ := utils.ConvertYAMLtoJSON([]byte(openAPI2SchemaData))
		openAPI3Schema = gojsonschema.NewStringLoader(string(openAPI3JSON))
		openAPI2Schema = gojsonschema.NewStringLoader(string(openAPI2JSON))
	})

	if len(spec) <= 0 {
		return nil, errors.New("specification is empty")
	}

	specString := string(spec)

	if utils.IsJSON(specString) {

		return processJSONSpec(spec)

	} else if utils.IsYAML(specString) {

		// convert to JSON (we don't need to worry about losing fidelity at this point).
		var yamlData map[string]interface{}
		if err := yaml.Unmarshal(spec, &yamlData); err != nil {
			return nil, err
		}

		jsonData, err := json.Marshal(yamlData)
		if err != nil {
			return nil, err
		}

		return processJSONSpec(jsonData)

	} else {
		return nil, errors.New("spec is neither YAML nor JSON, unable to process")
	}
}

func processJSONSpec(spec []byte) (*gojsonschema.Result, error) {

	// create loader
	doc := gojsonschema.NewStringLoader(string(spec))

	// which version is the spec?
	info, err := model.ExtractSpecInfo(spec)
	if err != nil {
		return nil, err
	}
	var schemaToValidate gojsonschema.JSONLoader
	switch info.SpecType {
	case utils.OpenApi2:
		schemaToValidate = openAPI2Schema
		break

	case utils.OpenApi3:
		schemaToValidate = openAPI3Schema
		break
	}

	if schemaToValidate == nil {
		return nil, errors.New("unable to determine specification type")
	}

	// validate spec
	res, err := gojsonschema.Validate(schemaToValidate, doc)

	if err != nil {
		return nil, err
	}

	return res, nil
}
