package harness

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pb33f/go-yaml"
)

// Decode the complete fixture. The engine consumes one document, so accepting
// a stream here would silently leave part of an agent's fixture unchecked.
func decodeSingleDocument(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid document: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("fixture must contain exactly one YAML or JSON document")
	}
	return nil
}
