// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

// Package asyncapi contains vacuum's AsyncAPI execution context and document
// detection helpers. It intentionally keeps the parsing contract small: command
// code can cheaply detect AsyncAPI, while motor and rule functions receive the
// full libasyncapi document graph when linting AsyncAPI 3.x documents.
package asyncapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	vacuumModel "github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libasyncapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
)

var asyncAPIMarker = []byte("asyncapi")

// Context is the AsyncAPI equivalent of vacuum's OpenAPI document context.
// It carries the raw YAML tree, parsed document and index metadata used by
// AsyncAPI rules.
type Context struct {
	Spec         []byte
	SpecFileName string
	SpecInfo     *datamodel.SpecInfo
	Document     libasyncapi.Document
	RootNode     *yaml.Node
	Version      string
	Format       string
	Index        *index.SpecIndex
	Rolodex      *index.Rolodex

	pathIndexOnce sync.Once
	pathIndex     *vacuumUtils.NodePathIndex
}

// DetectFormat returns vacuum's AsyncAPI format for spec bytes. It only
// recognizes AsyncAPI 3.x and returns an error for AsyncAPI 2.x or invalid
// AsyncAPI version strings. Non-AsyncAPI documents return an empty format.
func DetectFormat(spec []byte) (string, error) {
	if !hasAsyncAPIMarker(spec) {
		return "", nil
	}
	version, ok, err := detectVersion(spec)
	if err != nil || !ok {
		return "", err
	}
	return FormatForVersion(version)
}

// HasMarker checks for an AsyncAPI marker without building a document model.
// Escaped JSON field names are decoded until a root format marker is found.
func HasMarker(spec []byte) bool {
	return hasAsyncAPIMarker(spec)
}

// IsDocument reports whether spec bytes have an AsyncAPI root marker. It does
// not require the version to be supported, making it suitable for command
// surfaces that simply need to reject AsyncAPI input.
func IsDocument(spec []byte) (bool, error) {
	if !hasAsyncAPIMarker(spec) {
		return false, nil
	}
	_, ok, err := detectVersion(spec)
	return ok, err
}

// FormatForVersion maps an AsyncAPI version string onto vacuum's format
// constants. AsyncAPI 2.x is intentionally unsupported.
func FormatForVersion(version string) (string, error) {
	clean := strings.TrimSpace(version)
	if clean == "" {
		return "", libasyncapi.ErrNoAsyncAPIVersion
	}
	info, err := libasyncapi.ParseAsyncAPIVersion(clean)
	if err != nil {
		return "", err
	}
	if info == nil || info.VersionParsed == nil || info.VersionParsed.Major() != 3 {
		return "", libasyncapi.ErrInvalidAsyncAPIVersion
	}
	minor := int(info.VersionParsed.Minor())
	switch minor {
	case 0:
		return vacuumModel.AsyncAPI30, nil
	case 1:
		return vacuumModel.AsyncAPI31, nil
	default:
		return vacuumModel.AsyncAPI3, nil
	}
}

// NewContext parses spec bytes into a libasyncapi document and builds the
// metadata vacuum needs for format filtering, locations and reports.
func NewContext(spec []byte, specFileName string, config *libasyncapi.DocumentConfiguration) (*Context, error) {
	doc, err := libasyncapi.NewDocumentWithConfiguration(spec, config)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("unable to parse AsyncAPI document")
	}

	format, err := FormatForVersion(doc.GetVersion())
	if err != nil {
		return nil, err
	}

	specInfo := BuildSpecInfo(spec, doc.RootNode(), doc.GetVersion(), format)
	return &Context{
		Spec:         spec,
		SpecFileName: specFileName,
		SpecInfo:     specInfo,
		Document:     doc,
		RootNode:     doc.RootNode(),
		Version:      doc.GetVersion(),
		Format:       format,
		Index:        doc.Index(),
		Rolodex:      doc.Rolodex(),
	}, nil
}

// BuildSpecInfo creates libopenapi-compatible metadata for AsyncAPI documents.
// The metadata is intentionally limited to parser-neutral fields used by vacuum:
// raw bytes, root node, document type, version and format.
func BuildSpecInfo(spec []byte, root *yaml.Node, version, format string) *datamodel.SpecInfo {
	fileType := datamodel.YAMLFileType
	trimmed := bytes.TrimSpace(spec)
	if len(trimmed) > 0 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		fileType = datamodel.JSONFileType
	}
	return &datamodel.SpecInfo{
		SpecType:     "asyncapi",
		NumLines:     bytes.Count(spec, []byte{'\n'}) + 1,
		Version:      version,
		SpecFormat:   format,
		SpecFileType: fileType,
		SpecBytes:    &spec,
		RootNode:     root,
	}
}

// Root returns the parsed YAML document root.
func (c *Context) Root() *yaml.Node {
	if c == nil {
		return nil
	}
	return c.RootNode
}

// DocumentErrors returns libasyncapi document validation errors.
func (c *Context) DocumentErrors() []error {
	if c == nil || c.Document == nil {
		return nil
	}
	return c.Document.Errors()
}

// NodePath returns the exact vacuum JSONPath for node. The path index is built
// once per AsyncAPI context and reused by all rule functions.
func (c *Context) NodePath(node *yaml.Node) (string, bool) {
	if c == nil || node == nil {
		return "", false
	}
	c.pathIndexOnce.Do(func() {
		c.pathIndex = vacuumUtils.BuildNodePathIndex(c.RootNode)
	})
	if c.pathIndex == nil {
		return "", false
	}
	return c.pathIndex.Lookup(node)
}

func hasAsyncAPIMarker(spec []byte) bool {
	if bytes.Contains(spec, asyncAPIMarker) {
		return true
	}
	// JSON permits escaped field names. Inspect root keys without copying all
	// top-level values, and stop as soon as the document format is known.
	if !bytes.Contains(spec, []byte(`\u`)) {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(spec))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return false
	}
	var discard discardedJSONValue
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return false
		}
		switch key {
		case "asyncapi":
			return true
		case "openapi", "swagger":
			return false
		}
		if decoder.Decode(&discard) != nil {
			return false
		}
	}
	return false
}

// discardedJSONValue lets the decoder skip a value without retaining a copy.
type discardedJSONValue struct{}

func (*discardedJSONValue) UnmarshalJSON([]byte) error { return nil }

func detectVersion(spec []byte) (string, bool, error) {
	version, err := libasyncapi.DetectAsyncAPIVersion(spec)
	if errors.Is(err, libasyncapi.ErrNoAsyncAPIVersion) {
		return "", false, nil
	}
	return version, err == nil, err
}
