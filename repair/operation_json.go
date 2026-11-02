package repair

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON retains an empty source pointer for root move/copy operations.
// Omitting it would change a valid RFC 6902 operation into a malformed one.
func (o Operation) MarshalJSON() ([]byte, error) {
	type encoded struct {
		Op    string          `json:"op"`
		Path  string          `json:"path"`
		From  *string         `json:"from,omitempty"`
		Value json.RawMessage `json:"value,omitempty"`
	}
	value := encoded{Op: o.Op, Path: o.Path, Value: o.Value}
	if o.Op == "move" || o.Op == "copy" {
		value.From = &o.From
	}
	return json.Marshal(value)
}

// UnmarshalJSON rejects missing required members while accepting extra members,
// as required by RFC 6902. An empty pointer is distinct from an absent member.
func (o *Operation) UnmarshalJSON(data []byte) error {
	type plain Operation
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, name := range []string{"op", "path"} {
		if raw, ok := fields[name]; !ok || string(raw) == "null" {
			return fmt.Errorf("patch operation requires %s", name)
		}
	}
	switch value.Op {
	case "copy", "move":
		if raw, ok := fields["from"]; !ok || string(raw) == "null" {
			return fmt.Errorf("%s requires from", value.Op)
		}
	case "add", "replace", "test":
		if _, ok := fields["value"]; !ok {
			return fmt.Errorf("%s requires value", value.Op)
		}
	}
	*o = Operation(value)
	return nil
}
