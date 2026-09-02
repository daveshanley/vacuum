package transform

import (
	"fmt"
	"net/url"
	"strings"

	"go.yaml.in/yaml/v4"
)

var documentReferenceKeys = map[string]bool{
	"$ref":          true,
	"$dynamicRef":   true,
	"$recursiveRef": true,
	"operationRef":  true,
}

// ValidateBundled rejects document references whose URI portion is external.
// Runtime URLs such as Example Object externalValue values are ignored.
func ValidateBundled(root *yaml.Node) error {
	root, err := requireMappingRoot(root)
	if err != nil {
		return err
	}
	return walkExternalReferences(root, nil)
}

func walkExternalReferences(node *yaml.Node, path []string) error {
	if isWithinArbitraryExample(path) || isWithinExtension(path) {
		return nil
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i].Value, node.Content[i+1]
			path = append(path, key)
			next := path
			if documentReferenceKeys[key] && value.Kind == yaml.ScalarNode && isExternalReference(value.Value) {
				return fmt.Errorf("inclusive filtering and unused-component pruning require a bundled OpenAPI document; external reference found at %s: %s", jsonPath(next), value.Value)
			}
			if key == "discriminator" {
				if err := validateDiscriminatorReferences(value, next); err != nil {
					return err
				}
			}
			if err := walkExternalReferences(value, next); err != nil {
				return err
			}
			path = path[:len(path)-1]
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			path = append(path, fmt.Sprintf("%d", i))
			if err := walkExternalReferences(child, path); err != nil {
				return err
			}
			path = path[:len(path)-1]
		}
	}
	return nil
}

func validateDiscriminatorReferences(node *yaml.Node, path []string) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		switch key {
		case "mapping":
			if value.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j+1 < len(value.Content); j += 2 {
				target := value.Content[j+1]
				if target.Kind == yaml.ScalarNode && isExternalDiscriminatorReference(target.Value) {
					p := appendPath(appendPath(path, key), value.Content[j].Value)
					return fmt.Errorf("inclusive filtering and unused-component pruning require a bundled OpenAPI document; external reference found at %s: %s", jsonPath(p), target.Value)
				}
			}
		case "defaultMapping":
			if value.Kind == yaml.ScalarNode && isExternalDiscriminatorReference(value.Value) {
				return fmt.Errorf("inclusive filtering and unused-component pruning require a bundled OpenAPI document; external reference found at %s: %s", jsonPath(appendPath(path, key)), value.Value)
			}
		}
	}
	return nil
}

func isExternalReference(value string) bool {
	if value == "" || strings.HasPrefix(value, "#") {
		return false
	}
	u, err := url.Parse(value)
	return err != nil || u.Scheme != "" || u.Host != "" || u.Path != ""
}

func isExternalDiscriminatorReference(value string) bool {
	if value == "" || strings.HasPrefix(value, "#") {
		return false
	}
	// A bare value is a schema component name.
	lower := strings.ToLower(value)
	return strings.ContainsAny(value, "/:\\") ||
		strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json")
}

func appendPath(path []string, part string) []string {
	next := make([]string, len(path)+1)
	copy(next, path)
	next[len(path)] = part
	return next
}

func localPointerTokens(value string) ([]string, error) {
	if !strings.HasPrefix(value, "#") {
		return nil, fmt.Errorf("reference is not local: %s", value)
	}
	fragment, err := url.PathUnescape(strings.TrimPrefix(value, "#"))
	if err != nil {
		return nil, fmt.Errorf("invalid URI fragment in reference %q: %w", value, err)
	}
	if fragment == "" {
		return nil, nil
	}
	if fragment[0] != '/' {
		return []string{fragment}, nil
	}
	raw := strings.Split(fragment[1:], "/")
	tokens := make([]string, len(raw))
	for i, token := range raw {
		decoded, err := decodePointerToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON Pointer in reference %q: %w", value, err)
		}
		tokens[i] = decoded
	}
	return tokens, nil
}

func decodePointerToken(token string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(token); i++ {
		if token[i] != '~' {
			b.WriteByte(token[i])
			continue
		}
		if i+1 >= len(token) || (token[i+1] != '0' && token[i+1] != '1') {
			return "", fmt.Errorf("invalid escape in token %q", token)
		}
		i++
		if token[i] == '0' {
			b.WriteByte('~')
		} else {
			b.WriteByte('/')
		}
	}
	return b.String(), nil
}
