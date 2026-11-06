package openapi

import (
	"net/url"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
)

const securitySchemePrefix = "#/components/securitySchemes/"

// securitySchemeKey returns the index key of a declared scheme. Component names
// take precedence over URI references. Only local component fragments are resolved;
// document-prefixed URIs require a separate document-resolution policy.
func securitySchemeKey(name string, schemes, localSchemes map[string]*index.Reference, allowLocalURI bool) string {
	key := securitySchemePrefix + name
	if schemes[key] != nil {
		return key
	}
	if !allowLocalURI || !strings.HasPrefix(name, "#") {
		return ""
	}
	fragment, err := url.PathUnescape(name)
	if err != nil || !strings.HasPrefix(fragment, securitySchemePrefix) {
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
