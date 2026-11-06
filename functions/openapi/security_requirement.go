// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
)

const securitySchemePrefix = "#/components/securitySchemes/"

// securitySchemeKey returns the index key of a declared scheme. Component names
// take precedence over URI references. URI lookups are restricted to the source
// document, identified by $self or its retrieval URI; no document is fetched.
func securitySchemeKey(name string, schemes, localSchemes map[string]*index.Reference, allowLocalURI bool, documentURI *url.URL) string {
	key := securitySchemePrefix + name
	if schemes[key] != nil {
		return key
	}
	if !allowLocalURI {
		return ""
	}
	reference, err := url.Parse(name)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(name, "#") {
		if documentURI == nil {
			return ""
		}
		target := documentURI.ResolveReference(reference)
		target.Fragment, target.RawFragment = "", ""
		target.Scheme, target.Host = strings.ToLower(target.Scheme), strings.ToLower(target.Host)
		if target.String() != documentURI.String() {
			return ""
		}
	}
	// URL parsing decodes the fragment exactly once; JSON Pointer decoding follows.
	fragment := "#" + reference.Fragment
	if !strings.HasPrefix(fragment, securitySchemePrefix) {
		return ""
	}
	token := strings.TrimPrefix(fragment, securitySchemePrefix)
	if strings.Contains(token, "/") {
		return ""
	}
	// A component name is one JSON Pointer token; reject invalid escape sequences.
	for i := 0; i < len(token); i++ {
		if token[i] == '~' {
			if i+1 == len(token) || (token[i+1] != '0' && token[i+1] != '1') {
				return ""
			}
			i++
		}
	}
	token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
	key = securitySchemePrefix + token
	if localSchemes[key] != nil {
		return key
	}
	return ""
}

// securityDocumentURI establishes the source document's identity without I/O.
// A relative $self is resolved against the retrieval URI before other references.
func securityDocumentURI(source *index.SpecIndex) *url.URL {
	if source == nil {
		return nil
	}
	var base *url.URL
	retrieval := source.GetSpecAbsolutePath()
	if filepath.IsAbs(retrieval) {
		path := filepath.ToSlash(retrieval)
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		base = &url.URL{Scheme: "file", Path: path}
	} else if retrieval != "" {
		base, _ = url.Parse(retrieval)
	}
	root := utils.ResolveMergeNodes(documentContentNode(source.GetRootNode()))
	self := utils.NodeAlias(mappingValueNode(root, "$self"))
	if self != nil {
		if self.Kind != yaml.ScalarNode || self.Tag != "!!str" {
			return nil
		}
		identity, err := url.Parse(self.Value)
		if err != nil {
			return nil
		}
		if identity.IsAbs() {
			base = identity
		} else if base != nil {
			base = base.ResolveReference(identity)
		} else {
			return nil
		}
	}
	if base == nil || !base.IsAbs() {
		return nil
	}
	base.Fragment, base.RawFragment = "", ""
	base.Scheme, base.Host = strings.ToLower(base.Scheme), strings.ToLower(base.Host)
	return base.ResolveReference(&url.URL{})
}

func allowsSecuritySchemeURI(context model.RuleFunctionContext) bool {
	if context.SpecInfo != nil {
		return context.SpecInfo.VersionNumeric == 3.2
	}
	version, ok := context.Index.ResolveDocumentVersion()
	return ok && version == 3.2
}

// securitySchemeSource requires node identity: equal security blocks in two
// documents can share both their source coordinates and their content hash.
func securitySchemeSource(root *index.SpecIndex, node *yaml.Node) *index.SpecIndex {
	if rolodex := root.GetRolodex(); rolodex != nil && len(rolodex.GetIndexes()) > 0 {
		if found, _ := root.GetNode(node.Line, node.Column); found == node {
			return root
		}
		for _, candidate := range rolodex.GetIndexes() {
			if found, _ := candidate.GetNode(node.Line, node.Column); found == node {
				return candidate
			}
		}
		return nil
	}
	return root
}
