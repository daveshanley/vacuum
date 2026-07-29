// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

// Package protocol contains the Language Server Protocol types used by Vacuum.
//
// It intentionally models only Vacuum's supported LSP surface. Unknown JSON
// fields are ignored by encoding/json, allowing clients to send newer
// capabilities without coupling Vacuum to the complete protocol schema.
package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type (
	// Method is an LSP or JSON-RPC method name.
	Method = string
	// Integer is the signed integer range used by LSP.
	Integer = int32
	// UInteger is the unsigned integer range used by LSP.
	UInteger = uint32
	// DocumentURI identifies a text document.
	DocumentURI = string
	// DocumentUri preserves the former GLSP spelling for source compatibility.
	DocumentUri = string // Kept for source compatibility with the former GLSP type.
	// URI identifies a general protocol resource.
	URI = string
)

// IntegerOrString represents protocol fields that accept an integer or string.
type IntegerOrString struct {
	Value any
}

func (v IntegerOrString) MarshalJSON() ([]byte, error) {
	switch v.Value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, string:
		return json.Marshal(v.Value)
	default:
		return nil, fmt.Errorf("protocol integer-or-string contains %T", v.Value)
	}
}

func (v *IntegerOrString) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("protocol integer-or-string is nil")
	}
	var integer Integer
	if err := json.Unmarshal(data, &integer); err == nil {
		v.Value = integer
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("protocol integer-or-string: %w", err)
	}
	v.Value = text
	return nil
}

// BoolOrString represents protocol fields that accept a boolean or string.
type BoolOrString struct {
	Value any
}

func (v BoolOrString) MarshalJSON() ([]byte, error) {
	switch v.Value.(type) {
	case bool, string:
		return json.Marshal(v.Value)
	default:
		return nil, fmt.Errorf("protocol bool-or-string contains %T", v.Value)
	}
}

func (v *BoolOrString) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("protocol bool-or-string is nil")
	}
	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		v.Value = boolean
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("protocol bool-or-string: %w", err)
	}
	v.Value = text
	return nil
}

// String returns the represented boolean or string value.
func (v BoolOrString) String() string {
	switch value := v.Value.(type) {
	case bool:
		return strconv.FormatBool(value)
	case string:
		return value
	default:
		return ""
	}
}

// ID is a JSON-RPC request identifier. It preserves its encoded form so
// responses can echo string and numeric identifiers exactly.
type ID struct {
	raw json.RawMessage
}

// NewIntegerID creates a numeric JSON-RPC request identifier.
func NewIntegerID(value int64) ID {
	return ID{raw: json.RawMessage(strconv.FormatInt(value, 10))}
}

// ParseID creates an identifier from its encoded JSON representation.
func ParseID(data json.RawMessage) (ID, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ID{}, fmt.Errorf("JSON-RPC id is missing")
	}
	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return ID{}, fmt.Errorf("invalid JSON-RPC string id: %w", err)
		}
	} else {
		var value json.Number
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return ID{}, fmt.Errorf("invalid JSON-RPC numeric id: %w", err)
		}
		if err := ensureJSONEnd(decoder); err != nil {
			return ID{}, fmt.Errorf("invalid JSON-RPC numeric id: %w", err)
		}
	}
	return ID{raw: append(json.RawMessage(nil), trimmed...)}, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	if id == nil {
		return fmt.Errorf("JSON-RPC id is nil")
	}
	parsed, err := ParseID(data)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// IsZero reports whether the identifier is unset.
func (id ID) IsZero() bool {
	return len(id.raw) == 0
}

// Key returns the encoded identifier used for request correlation.
func (id ID) Key() string {
	return string(id.raw)
}

func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsZero() {
		return []byte("null"), nil
	}
	return append([]byte(nil), id.raw...), nil
}
