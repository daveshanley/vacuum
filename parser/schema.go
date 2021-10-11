package parser

import (
	_ "embed"
	"errors"
	"github.com/daveshanley/vaccum/utils"
	"github.com/xeipuuv/gojsonschema"
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
		openAPI3Schema = gojsonschema.NewStringLoader(openAPI3SchemaData)
		openAPI2Schema = gojsonschema.NewStringLoader(openAPI2SchemaData)
	})

	if len(spec) <= 0 {
		return nil, errors.New("specification is empty")
	}

	specString := string(spec)

	if utils.IsJSON(specString) {

		// create loader
		doc := gojsonschema.NewStringLoader(string(spec))

		// which version is the spec?
		info, err := utils.ExtractSpecInfo(spec)
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
		res, err := gojsonschema.Validate(openAPI3Schema, doc)

		if err != nil {
			return nil, err
		}

		return res, nil

	} else if utils.IsYAML(specString) {

		// convert to JSON (we don't need to worry about losing fidelity at this point).

		return nil, nil

	} else {

		return nil, errors.New("spec is neither YAML nor JSON, unable to process")
	}
}
