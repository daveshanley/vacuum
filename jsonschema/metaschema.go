// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"fmt"
	"sync"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	santhoshjsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v4"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type ValidationIssue struct {
	Message  string
	Pointer  string
	Path     string
	Node     *yaml.Node
	EndNode  *yaml.Node
	Location []string
}

var (
	metaOnce    sync.Once
	metaSchemas map[string]*santhoshjsonschema.Schema
	metaErr     error
)

func ValidateAgainstMetaschema(root *yaml.Node) ([]ValidationIssue, error) {
	root = RootNode(root)
	if root == nil {
		return nil, fmt.Errorf("schema document is empty")
	}
	dialect := DetectDialect(root)
	if !IsSupportedDialect(dialect.Format) {
		return nil, nil
	}
	meta, err := metaschemaForFormat(dialect.Format)
	if err != nil {
		return nil, err
	}
	data, err := NodeToInterface(root)
	if err != nil {
		return nil, err
	}
	err = meta.Validate(data)
	if err == nil {
		return nil, nil
	}
	var validationErr *santhoshjsonschema.ValidationError
	if !asValidationError(err, &validationErr) {
		return []ValidationIssue{issueForLocation(root, nil, err.Error())}, nil
	}
	leaves := flattenValidationErrors(validationErr)
	issues := make([]ValidationIssue, 0, len(leaves))
	printer := message.NewPrinter(language.Tag{})
	for _, leaf := range leaves {
		msg := leaf.Error()
		if leaf.ErrorKind != nil {
			msg = leaf.ErrorKind.LocalizedString(printer)
		}
		issues = append(issues, issueForLocation(root, leaf.InstanceLocation, msg))
	}
	return issues, nil
}

func CompileSchema(root *yaml.Node) (*santhoshjsonschema.Schema, error) {
	root = RootNode(root)
	dialect := DetectDialect(root)
	if !IsSupportedDialect(dialect.Format) {
		return nil, nil
	}
	data, err := NodeToInterface(root)
	if err != nil {
		return nil, err
	}
	compiler := santhoshjsonschema.NewCompiler()
	compiler.DefaultDraft(dialect.Draft)
	compiler.UseLoader(noopLoader{})
	if err := compiler.AddResource("schema.json", data); err != nil {
		return nil, err
	}
	return compiler.Compile("schema.json")
}

type noopLoader struct{}

func (noopLoader) Load(loadURL string) (any, error) {
	return nil, fmt.Errorf("remote schema loading is disabled: %s", loadURL)
}

func metaschemaForFormat(format string) (*santhoshjsonschema.Schema, error) {
	metaOnce.Do(func() {
		metaSchemas = make(map[string]*santhoshjsonschema.Schema)
		for _, dialect := range []Dialect{
			dialectForFormat(model.JSONSchemaDraft2020),
			dialectForFormat(model.JSONSchemaDraft2019),
			dialectForFormat(model.JSONSchemaDraft07),
		} {
			compiler := santhoshjsonschema.NewCompiler()
			compiler.DefaultDraft(dialect.Draft)
			compiler.UseLoader(noopLoader{})
			compiled, err := compiler.Compile(dialect.URL)
			if err != nil {
				metaErr = err
				return
			}
			metaSchemas[dialect.Format] = compiled
		}
	})
	if metaErr != nil {
		return nil, metaErr
	}
	return metaSchemas[format], nil
}

func issueForLocation(root *yaml.Node, location []string, msg string) ValidationIssue {
	node, path := FindNodeByLocation(root, location)
	if node == nil {
		node = RootNode(root)
		path = "$"
	}
	return ValidationIssue{
		Message:  msg,
		Pointer:  InstanceLocationPointer(location),
		Path:     path,
		Node:     node,
		EndNode:  vacuumUtils.BuildEndNode(node),
		Location: location,
	}
}

func asValidationError(err error, target **santhoshjsonschema.ValidationError) bool {
	if validationErr, ok := err.(*santhoshjsonschema.ValidationError); ok {
		*target = validationErr
		return true
	}
	if schemaErr, ok := err.(*santhoshjsonschema.SchemaValidationError); ok {
		if validationErr, ok := schemaErr.Err.(*santhoshjsonschema.ValidationError); ok {
			*target = validationErr
			return true
		}
	}
	return false
}

func flattenValidationErrors(err *santhoshjsonschema.ValidationError) []*santhoshjsonschema.ValidationError {
	if err == nil {
		return nil
	}
	if len(err.Causes) == 0 {
		return []*santhoshjsonschema.ValidationError{err}
	}
	var leaves []*santhoshjsonschema.ValidationError
	for _, cause := range err.Causes {
		leaves = append(leaves, flattenValidationErrors(cause)...)
	}
	return leaves
}
