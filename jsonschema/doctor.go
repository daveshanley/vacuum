// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"

	doctorModel "github.com/pb33f/doctor/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

const schemaDoctorPathRoot = "$"

// RolodexDoctorBuildConfig configures schema-only Doctor document construction from a libopenapi rolodex index.
type RolodexDoctorBuildConfig struct {
	BuildGraph         bool
	DeterministicPaths bool
	Logger             *slog.Logger
	RenderChanges      bool
	StorageRoot        string
	UseSchemaCache     bool
}

// NewDoctorDocumentFromRolodexIndex builds a Doctor document from a libopenapi rolodex root index.
//
// JSON Schema references are foundationally owned by libopenapi's index and rolodex. This builder does not resolve,
// discover, or reinterpret references; it only turns the already-indexed schema root into Doctor models while Doctor
// walks the schema graph.
func NewDoctorDocumentFromRolodexIndex(idx *index.SpecIndex, config RolodexDoctorBuildConfig) (*doctorModel.DrDocument, error) {
	if idx == nil {
		return nil, errors.New("schema index is nil")
	}
	if idx.GetRolodex() == nil {
		return nil, errors.New("schema index is not attached to a libopenapi rolodex")
	}
	root := RootNode(idx.GetRootNode())
	if root == nil {
		return nil, errors.New("schema rolodex root is nil")
	}

	schemaProxy, err := buildSchemaProxyFromRolodexIndex(root, idx)
	if err != nil {
		return nil, err
	}

	return walkDoctorSchema(root, schemaProxy, idx, config), nil
}

func buildSchemaProxyFromRolodexIndex(root *yaml.Node, idx *index.SpecIndex) (*highbase.SchemaProxy, error) {
	var lowProxy lowbase.SchemaProxy
	if err := lowProxy.Build(context.Background(), root, root, idx); err != nil {
		return nil, err
	}
	return highbase.NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{
		Value:     &lowProxy,
		KeyNode:   root,
		ValueNode: root,
	}), nil
}

func walkDoctorSchema(root *yaml.Node, schemaProxy *highbase.SchemaProxy, idx *index.SpecIndex, config RolodexDoctorBuildConfig) *doctorModel.DrDocument {
	schemaChan := make(chan *drV3.WalkedSchema)
	skippedSchemaChan := make(chan *drV3.WalkedSchema)
	objectChan := make(chan any)
	buildErrorChan := make(chan *drV3.BuildError)
	nodeChan := make(chan *drV3.Node)
	edgeChan := make(chan *drV3.Edge)
	done := make(chan struct{})
	complete := make(chan struct{})

	var schemaCache sync.Map
	var canonicalPathCache sync.Map
	var stringCache sync.Map
	var schemas []*drV3.Schema
	var skippedSchemas []*drV3.Schema
	var buildErrors []*drV3.BuildError

	go drainDoctorBuildChannels(
		schemaChan,
		skippedSchemaChan,
		objectChan,
		buildErrorChan,
		nodeChan,
		edgeChan,
		done,
		complete,
		&schemas,
		&skippedSchemas,
		&buildErrors,
	)

	drCtx := &drV3.DrContext{
		SchemaChan:         schemaChan,
		SkippedSchemaChan:  skippedSchemaChan,
		ObjectChan:         objectChan,
		ErrorChan:          buildErrorChan,
		NodeChan:           nodeChan,
		EdgeChan:           edgeChan,
		Index:              idx,
		BuildGraph:         config.BuildGraph,
		RenderChanges:      config.RenderChanges,
		UseSchemaCache:     config.UseSchemaCache,
		SyncWalk:           true,
		DeterministicPaths: config.DeterministicPaths,
		SchemaCache:        &schemaCache,
		CanonicalPathCache: &canonicalPathCache,
		HashCache:          drV3.NewHashCache(),
		StringCache:        &stringCache,
		StorageRoot:        config.StorageRoot,
		Logger:             config.Logger,
	}
	ctx := context.WithValue(context.Background(), "drCtx", drCtx)
	rootFoundation := &drV3.Foundation{
		PathSegment: schemaDoctorPathRoot,
		KeyNode:     root,
		ValueNode:   root,
	}
	drSchemaProxy := &drV3.SchemaProxy{
		Value: schemaProxy,
	}
	drSchemaProxy.Parent = rootFoundation
	drSchemaProxy.NodeParent = rootFoundation
	drSchemaProxy.KeyNode = root
	drSchemaProxy.ValueNode = root

	drSchemaProxy.Walk(ctx, schemaProxy, 0)
	if err := schemaProxy.GetBuildError(); err != nil {
		buildErrorChan <- &drV3.BuildError{
			Error:         err,
			SchemaProxy:   schemaProxy,
			DrSchemaProxy: drSchemaProxy,
		}
	}

	done <- struct{}{}
	<-complete

	drDoc := &doctorModel.DrDocument{
		BuildErrors:    buildErrors,
		SkippedSchemas: skippedSchemas,
		StorageRoot:    config.StorageRoot,
	}
	sortDoctorSchemas(schemas)
	sortDoctorSchemas(skippedSchemas)

	drDoc.Schemas = schemas
	return drDoc
}

func drainDoctorBuildChannels(
	schemaChan <-chan *drV3.WalkedSchema,
	skippedSchemaChan <-chan *drV3.WalkedSchema,
	objectChan <-chan any,
	buildErrorChan <-chan *drV3.BuildError,
	nodeChan <-chan *drV3.Node,
	edgeChan <-chan *drV3.Edge,
	done <-chan struct{},
	complete chan<- struct{},
	schemas *[]*drV3.Schema,
	skippedSchemas *[]*drV3.Schema,
	buildErrors *[]*drV3.BuildError,
) {
	seenSchemas := make(map[*yaml.Node]struct{})
	seenSkippedSchemas := make(map[*yaml.Node]struct{})
	for {
		select {
		case <-done:
			complete <- struct{}{}
			return
		case walked := <-schemaChan:
			if walked != nil {
				*schemas = appendUniqueDoctorSchema(*schemas, walked.Schema, walked.SchemaNode, seenSchemas)
			}
		case walked := <-skippedSchemaChan:
			if walked != nil {
				*skippedSchemas = appendUniqueDoctorSchema(*skippedSchemas, walked.Schema, walked.SchemaNode, seenSkippedSchemas)
			}
		case buildError := <-buildErrorChan:
			if buildError != nil {
				*buildErrors = append(*buildErrors, buildError)
			}
		case object := <-objectChan:
			if schema, ok := object.(*drV3.Schema); ok {
				*schemas = appendUniqueDoctorSchema(*schemas, schema, nil, seenSchemas)
			}
		case <-nodeChan:
		case <-edgeChan:
		}
	}
}

func appendUniqueDoctorSchema(schemas []*drV3.Schema, schema *drV3.Schema, node *yaml.Node, seen map[*yaml.Node]struct{}) []*drV3.Schema {
	if schema == nil {
		return schemas
	}
	if schema.Value != nil && schema.Value.GoLow() != nil && schema.Value.GoLow().RootNode != nil {
		node = schema.Value.GoLow().RootNode
	}
	if node != nil {
		if _, ok := seen[node]; ok {
			return schemas
		}
		seen[node] = struct{}{}
	}
	return append(schemas, schema)
}

func sortDoctorSchemas(schemas []*drV3.Schema) {
	sort.SliceStable(schemas, func(i, j int) bool {
		left := schemaSortNode(schemas[i])
		right := schemaSortNode(schemas[j])
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Line == right.Line {
			return left.Column < right.Column
		}
		return left.Line < right.Line
	})
}

func schemaSortNode(schema *drV3.Schema) *yaml.Node {
	if schema == nil {
		return nil
	}
	if schema.GetKeyNode() != nil {
		return schema.GetKeyNode()
	}
	if schema.Value != nil && schema.Value.GoLow() != nil {
		return schema.Value.GoLow().RootNode
	}
	return nil
}
