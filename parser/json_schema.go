// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package parser

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	validationErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
)

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

	// Schema.Build performs the low-model extraction; do not pre-build the
	// model here or validation pays that cost twice.
	schErr := sch.Build(ctx, node, idx)
	if schErr != nil {
		return nil, schErr
	}
	highSch := highBase.NewSchema(&sch)
	return highSch, nil
}

// ValidateNodeAgainstSchema will accept a schema and a node and check it's valid and return the result, or error.
func ValidateNodeAgainstSchema(ctx *model.RuleFunctionContext, schema *highBase.Schema, node *yaml.Node, isArray bool) (bool, []*validationErrors.ValidationError) {
	var validator schema_validation.SchemaValidator
	if ctx != nil {
		validator = ctx.SchemaValidator
	}
	if validator == nil {
		if ctx != nil && ctx.Logger != nil {
			validator = schema_validation.NewSchemaValidatorWithLogger(ctx.Logger)
		} else {
			validator = schema_validation.NewSchemaValidator()
		}
		defer validator.Release()
	}

	// yaml.Marshal's desolver mutates node metadata while rendering. Marshal a
	// deep clone so validation never changes caller-owned nodes.
	validationNode := utils.CloneYAMLNode(node)

	// convert node to raw yaml first, then convert to json to be used in schema validation
	var d []byte
	var e error
	if !isArray {
		d, e = yaml.Marshal(validationNode)
	} else {
		if !utils.IsNodeArray(validationNode) {
			d, e = yaml.Marshal([]*yaml.Node{validationNode})
		} else {
			d, e = yaml.Marshal(validationNode)
		}
	}
	if e != nil {
		return false, []*validationErrors.ValidationError{{Message: e.Error()}}
	}

	// safely convert yaml to JSON using standard library
	var yamlObj interface{}
	err := yaml.Unmarshal(d, &yamlObj)
	if err != nil {
		return false, []*validationErrors.ValidationError{{Message: err.Error()}}
	}

	n, err := json.Marshal(yamlObj)
	if err != nil {
		return false, []*validationErrors.ValidationError{{Message: err.Error()}}
	}

	var decoded any
	_ = json.Unmarshal(n, &decoded)

	return validator.ValidateSchemaObject(schema, decoded)
}
