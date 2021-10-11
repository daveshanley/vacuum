package model

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

const (
	OpenApi3 = "openapi"
	OpenApi2 = "swagger"
	AsyncApi = "asyncapi"
)

func ExtractSpecInfo(spec []byte) (*SpecInfo, error) {
	var parsedSpec map[string]interface{}
	specVersion := &SpecInfo{}
	runes := []rune(strings.TrimSpace(string(spec)))
	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		// try JSON
		err := json.Unmarshal(spec, &parsedSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
		}
		specVersion.Version = "json"
	} else {
		// try YAML
		err := yaml.Unmarshal(spec, &parsedSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to parse specification: %s", err.Error())
		}
		specVersion.Version = "yaml"
	}

	// check for specific keys
	if parsedSpec[OpenApi3] != nil {
		specVersion.SpecType = OpenApi3
		specVersion.Version = parseVersionTypeData(parsedSpec[OpenApi3])
	}
	if parsedSpec[OpenApi2] != nil {
		specVersion.SpecType = OpenApi2
		specVersion.Version = parseVersionTypeData(parsedSpec[OpenApi2])
	}
	if parsedSpec[AsyncApi] != nil {
		specVersion.SpecType = AsyncApi
		specVersion.Version = parseVersionTypeData(parsedSpec[AsyncApi])
	}

	if specVersion.SpecType == "" {
		specVersion.SpecType = "unknown specification type, unsupported schema"
		specVersion.Version = "-1"
	}
	return specVersion, nil
}

func parseVersionTypeData(d interface{}) string {
	switch d.(type) {
	case int:
		return strconv.Itoa(d.(int))
	case float64:
		return strconv.FormatFloat(d.(float64), 'f', 2, 32)
	case bool:
		if d.(bool) {
			return "true"
		}
		return "false"
	case []string:
		return "multiple versions detected"
	}
	return fmt.Sprintf("%v", d)
}
