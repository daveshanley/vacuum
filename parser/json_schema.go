// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package parser

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	validationErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
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
	mbErr := low.BuildModel(node, &sch)
	if mbErr != nil {
		return nil, mbErr
	}

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

	schErr := sch.Build(ctx, node, idx)
	if schErr != nil {
		return nil, schErr
	}
	highSch := highBase.NewSchema(&sch)
	return highSch, nil
}

// Global validator instance and mutex to ensure thread-safe schema validation
// This fixes issue #512 where concurrent validations cause non-deterministic results
var (
	globalValidator     schema_validation.SchemaValidator
	globalValidatorOnce sync.Once
	globalValidatorMu   sync.Mutex
)

// getGlobalValidator returns a singleton validator instance
func getGlobalValidator(ctx *model.RuleFunctionContext) schema_validation.SchemaValidator {
	globalValidatorOnce.Do(func() {
		if ctx != nil && ctx.Logger != nil {
			globalValidator = schema_validation.NewSchemaValidatorWithLogger(ctx.Logger)
		} else {
			globalValidator = schema_validation.NewSchemaValidator()
		}
	})
	return globalValidator
}

// ValidateNodeAgainstSchema will accept a schema and a node and check it's valid and return the result, or error.
func ValidateNodeAgainstSchema(ctx *model.RuleFunctionContext, schema *highBase.Schema, node *yaml.Node, isArray bool) (bool, []*validationErrors.ValidationError) {
	// convert node to raw yaml first, then convert to json to be used in schema validation
	var d []byte
	var e error
	if !isArray {
		d, e = yaml.Marshal(node)
	} else {
		if !utils.IsNodeArray(node) {
			d, e = yaml.Marshal([]*yaml.Node{node})
		} else {
			d, e = yaml.Marshal(node)
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

	// Use global validator with mutex protection to prevent concurrent schema mutations
	// This ensures thread-safe validation when multiple goroutines validate schemas
	validator := getGlobalValidator(ctx)

	// Lock to ensure only one validation happens at a time
	// This prevents race conditions in schema.RenderInline() which mutates internal state
	globalValidatorMu.Lock()
	defer globalValidatorMu.Unlock()

	return validator.ValidateSchemaObject(schema, decoded)
}
