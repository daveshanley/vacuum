package repair

import (
	"encoding/json"

	"github.com/pb33f/go-yaml"
)

// RenderJSON converts one YAML or JSON document to JSON without rounding numbers.
// Aliases, duplicate keys and values that cannot be represented exactly fail.
func RenderJSON(data []byte) ([]byte, error) {
	doc, err := ParseDocument(data)
	if err != nil {
		return nil, err
	}
	if err := validateTree(doc.Content[0], make(map[*yaml.Node]bool)); err != nil {
		return nil, err
	}
	value, err := jsonValue(doc)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(value, "", "  ")
}
