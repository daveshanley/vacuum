// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

// Package arazzo bridges Arazzo validation to Vacuum's rule execution model.
package arazzo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	validation "github.com/pb33f/libopenapi-validator/arazzo"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
)

// IncompleteRule is the rule ID for checks that require unavailable source data
// or syntax that the validator cannot check statically.
const IncompleteRule = model.ArazzoValidationIncomplete

var supportedVersion = regexp.MustCompile(`^1\.(0|1)\.[0-9]+(-.+)?$`)

// Context holds one immutable document and its validation findings. It is owned
// by one execution and can be read by concurrent rule functions.
type Context struct {
	RootNode   *yaml.Node
	SpecInfo   *datamodel.SpecInfo
	Index      *index.SpecIndex
	Validation *validation.Result
	// Sources contains loaded source roots keyed by their report origin.
	Sources  map[string]*yaml.Node
	results  map[string][]model.RuleFunctionResult
	pathOnce sync.Once
	paths    *utils.NodePathIndex
}

// DetectFormat identifies root Arazzo markers. Unsupported versions retain the
// family format so callers cannot send them through the OpenAPI parser.
func DetectFormat(spec []byte) (string, error) {
	if !bytes.Contains(spec, []byte("arazzo")) {
		return "", nil
	}
	root, err := parse(spec)
	if err != nil {
		return "", err
	}
	version, found := rootVersion(root)
	if !found {
		return "", nil
	}
	return formatForVersion(version)
}

func formatForVersion(version string) (string, error) {
	if parts := supportedVersion.FindStringSubmatch(version); parts != nil {
		if parts[1] == "0" {
			return model.Arazzo10, nil
		}
		return model.Arazzo11, nil
	}
	return model.Arazzo, fmt.Errorf("unsupported Arazzo version %q: expected 1.0.x or 1.1.x", version)
}

func rootVersion(root *yaml.Node) (string, bool) {
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return "", false
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "arazzo" {
			return root.Content[i+1].Value, true
		}
	}
	return "", false
}

func parse(spec []byte) (*yaml.Node, error) {
	var root yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(spec))
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("expected one Arazzo document")
	}
	return &root, nil
}

// NewContext parses original nodes and creates parser-neutral report metadata.
// It performs no I/O. Validate must finish before rules read the context.
func NewContext(spec []byte, filename string) (*Context, error) {
	if len(spec) > validation.DefaultLimits().MaxBytes {
		return nil, fmt.Errorf("Arazzo document exceeds the %d byte limit", validation.DefaultLimits().MaxBytes)
	}
	root, err := parse(spec)
	if err != nil {
		return nil, err
	}
	version, found := rootVersion(root)
	if !found {
		return nil, fmt.Errorf("missing Arazzo version")
	}
	format, err := formatForVersion(version)
	if err != nil {
		return nil, err
	}
	fileType := datamodel.YAMLFileType
	if bytes.HasPrefix(bytes.TrimSpace(spec), []byte("{")) {
		fileType = datamodel.JSONFileType
	}
	config := index.CreateClosedAPIIndexConfig()
	config.SpecFilePath = filename
	config.AvoidBuildIndex = true
	return &Context{
		RootNode: root,
		Index:    index.NewSpecIndexWithConfig(root, config),
		SpecInfo: &datamodel.SpecInfo{SpecType: "arazzo", Version: version, SpecFormat: format,
			SpecFileType: fileType, SpecBytes: &spec, RootNode: root, NumLines: bytes.Count(spec, []byte{'\n'}) + 1},
		results: make(map[string][]model.RuleFunctionResult),
	}, nil
}

// Validate runs libopenapi-validator once and maps its detached findings back to
// source nodes. The optional sources map supplies nodes loaded by the resolver.
func (c *Context) Validate(ctx context.Context, uri string, sources map[string]*SourceDocument, options ...validation.Option) error {
	result, err := validation.Validate(ctx, validation.Document{Root: c.RootNode, URI: uri}, options...)
	c.Validation = result
	c.Sources = make(map[string]*yaml.Node, len(sources))
	for sourceURI, document := range sources {
		c.Sources[sourceOrigin(sourceURI)] = document.Root
	}
	if result == nil {
		return err
	}
	for _, diagnostic := range result.Diagnostics {
		c.add(string(diagnostic.Code), diagnostic.Message, diagnostic.Location, uri, sources)
	}
	for _, check := range result.Checks {
		if check.Status == validation.CheckIncomplete {
			message := fmt.Sprintf("%s: %s", check.Name, check.Reason)
			c.add(IncompleteRule, message, validation.Location{URI: check.URI, Pointer: check.Pointer}, uri, sources)
		}
	}
	return err
}

// Results returns findings for a validator code. Callers must not mutate the
// returned slice; each rule copies findings before attaching its configuration.
func (c *Context) Results(code string) []model.RuleFunctionResult { return c.results[code] }

// NodePath returns the exact authored path, using one index per execution.
func (c *Context) NodePath(node *yaml.Node) (string, bool) {
	c.pathOnce.Do(func() { c.paths = utils.BuildNodePathIndex(c.RootNode) })
	return c.paths.Lookup(node)
}

func (c *Context) add(code, message string, location validation.Location, uri string, sources map[string]*SourceDocument) {
	root := c.RootNode
	var source *SourceDocument
	if location.URI != "" && location.URI != uri {
		source = sources[location.URI]
		root = nil
		if source != nil {
			root = source.Root
		}
	}
	node, path := locate(root, location.Pointer)
	if node == nil {
		node = &yaml.Node{Line: location.Line, Column: location.Column}
		if node.Line < 1 {
			node.Line = 1
		}
		if node.Column < 1 {
			node.Column = 1
		}
	}
	result := model.RuleFunctionResult{Message: message, Path: path, Paths: []string{path}, StartNode: node, EndNode: node}
	if location.URI != "" && location.URI != uri {
		result.SourceContext = sourceContext(source, node.Line)
		origin := sourceOrigin(location.URI)
		result.Origin = &index.NodeOrigin{AbsoluteLocation: origin, Node: node, Line: node.Line, Column: node.Column}
	}
	c.results[code] = append(c.results[code], result)
}

func sourceOrigin(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.Scheme == "file" {
		return filepath.FromSlash(u.Path)
	}
	return uri
}

// locate follows an RFC 6901 pointer while building a Vacuum JSONPath. If a
// required field is absent, it retains the nearest parent for a useful snippet.
func locate(root *yaml.Node, pointer string) (*yaml.Node, string) {
	node, path := root, "$"
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if pointer == "" {
		return node, path
	}
	for _, encoded := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(encoded, "~1", "/"), "~0", "~")
		if node != nil && node.Kind == yaml.SequenceNode {
			if n, err := strconv.Atoi(part); err == nil && n >= 0 {
				path = utils.AppendResultPathIndex(path, n)
				if n < len(node.Content) {
					node = node.Content[n]
				}
				continue
			}
		}
		path = utils.AppendResultPathSegment(path, part)
		if node != nil && node.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(node.Content); i += 2 {
				if node.Content[i].Value == part {
					node = node.Content[i+1]
					break
				}
			}
		}
	}
	return node, path
}

// Keep at most seven authored lines and 16 KiB per external finding. Clone the
// lines so a small report does not retain a large source document through slices.
func sourceContext(source *SourceDocument, line int) *model.ResultSourceContext {
	result := &model.ResultSourceContext{}
	if source == nil || source.Content == "" {
		return result
	}
	if source.lines == nil {
		source.lines = strings.Split(source.Content, "\n")
	}
	if line < 1 || line > len(source.lines) || len(source.lines) <= 1 {
		return result
	}
	start, end := max(1, line-3), min(len(source.lines), line+3)
	lines := source.lines[start-1 : end]
	size := 0
	for _, text := range lines {
		size += len(text)
	}
	if size > 16<<10 {
		return result
	}
	result.StartLine = start
	result.Lines = make([]string, len(lines))
	for i, text := range lines {
		result.Lines[i] = strings.Clone(text)
	}
	return result
}
