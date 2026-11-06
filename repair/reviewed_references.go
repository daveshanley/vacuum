package repair

import (
	"github.com/pb33f/go-yaml"
	"strings"
)

func reviewedLocalReference(root *yaml.Node, nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	owner, path := reviewedOwner(nodes, f.Pointer)
	ref := field(owner, "$ref")
	if ref == nil || !strings.HasPrefix(ref.Value, "#/") {
		return nil, "a resolved local reference is required"
	}
	targetPath := strings.TrimPrefix(ref.Value, "#")
	target := nodes[targetPath]
	if target == nil || target.Kind != yaml.MappingNode || field(target, "$ref") != nil {
		return nil, "reference target is missing or requires chained resolution"
	}
	switch f.RuleID {
	case "no-$ref-siblings":
		var ops []Operation
		for i := 0; i < len(owner.Content); i += 2 {
			key, value := owner.Content[i], owner.Content[i+1]
			if key.Value == "$ref" {
				continue
			}
			existing := field(target, key.Value)
			if existing == nil || !uncommented(key) || !uncommented(value) {
				return nil, "reference siblings are not redundant or contain comments"
			}
			equal, err := equalNodes(value, existing)
			if err != nil || !equal {
				return nil, "reference siblings differ from the target"
			}
			ops = append(ops, Operation{Op: "remove", Path: path + "/" + EscapeToken(key.Value)})
		}
		return ops, ""
	case "path-item-refs":
		parts, _ := tokens(path)
		targetParts, _ := tokens(targetPath)
		operation := func(p []string) bool {
			return len(p) == 3 && (p[0] == "paths" || p[0] == "webhooks") && reviewedHTTPMethod(p[2]) || len(p) == 4 && p[0] == "components" && p[1] == "pathItems" && reviewedHTTPMethod(p[3])
		}
		if !operation(parts) || !operation(targetParts) || len(owner.Content) != 2 || !uncommented(owner) || !uncommented(target) {
			return nil, "operation inlining requires an exact uncommented local operation"
		}
		// Local root references keep their scope. Nested resource identifiers need a
		// resolver-aware transform instead of copying schema resources.
		for p, n := range nodes {
			if strings.HasPrefix(p, targetPath+"/") && (strings.HasSuffix(p, "/$id") || strings.HasSuffix(p, "/$dynamicAnchor")) && n != nil {
				return nil, "operation contains nested schema resource scope"
			}
		}
		value, err := raw(target)
		if err != nil {
			return nil, err.Error()
		}
		return []Operation{{Op: "replace", Path: path, Value: value}}, ""
	}
	return nil, "unsupported local reference repair"
}
