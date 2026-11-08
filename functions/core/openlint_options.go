// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
)

// ValidateOpenLintOptions checks core function options against the pinned draft.
// Unknown names are left to their registered custom-function validator.
func ValidateOpenLintOptions(name string, options any) error {
	var allowed []string
	switch name {
	case "truthy", "falsy", "defined", "undefined":
	case "pattern":
		allowed = []string{"match", "notMatch"}
	case "casing":
		allowed = []string{"type", "disallowDigits", "separator"}
	case "length":
		allowed = []string{"min", "max"}
	case "alphabetical":
		allowed = []string{"keyedBy"}
	case "enumeration":
		allowed = []string{"values"}
	case "schema":
		allowed = []string{"schema", "dialect", "allErrors"}
	case "xor", "or":
		allowed = []string{"properties"}
	case "unreferencedReusableObject":
		allowed = []string{"reusableObjectsLocation"}
	default:
		return nil
	}
	props := map[string]any{}
	if options != nil {
		var ok bool
		props, ok = options.(map[string]any)
		if !ok {
			return fmt.Errorf("%s functionOptions must be an object", name)
		}
	}
	for key := range props {
		if !slices.Contains(allowed, key) {
			return fmt.Errorf("%s does not accept option %q", name, key)
		}
	}
	switch name {
	case "pattern":
		if len(props) == 0 {
			return fmt.Errorf("pattern requires match or notMatch")
		}
		for key, value := range props {
			pattern, ok := value.(string)
			if !ok {
				return fmt.Errorf("pattern %s must be a string", key)
			}
			if _, err := compilePattern(pattern, model.RuleFunctionContext{Contract: model.OpenLintV1Draft}); err != nil {
				return fmt.Errorf("pattern %s: %w", key, err)
			}
		}
	case "casing":
		kind, ok := props["type"].(string)
		if !ok || !slices.Contains(casingTypes, kind) {
			return fmt.Errorf("casing type must be one of %v", casingTypes)
		}
		if value, exists := props["disallowDigits"]; exists {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("casing disallowDigits must be a boolean")
			}
		}
		if value, exists := props["separator"]; exists {
			separator, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("casing separator must be an object")
			}
			char, ok := separator["char"].(string)
			if !ok || utf8.RuneCountInString(char) != 1 {
				return fmt.Errorf("casing separator.char must be one character")
			}
			for key, value := range separator {
				switch key {
				case "char":
				case "allowLeading":
					if _, ok := value.(bool); !ok {
						return fmt.Errorf("casing separator.allowLeading must be a boolean")
					}
				default:
					return fmt.Errorf("casing separator does not accept %q", key)
				}
			}
		}
	case "length":
		if len(props) == 0 {
			return fmt.Errorf("length requires min or max")
		}
		for key, value := range props {
			if !openLintNumber(value) {
				return fmt.Errorf("length %s must be a number", key)
			}
		}
	case "alphabetical":
		if value, exists := props["keyedBy"]; exists {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("alphabetical keyedBy must be a string")
			}
		}
	case "enumeration":
		values, ok := props["values"].([]any)
		if !ok {
			return fmt.Errorf("enumeration values must be an array")
		}
		for _, value := range values {
			switch value.(type) {
			case nil, string, bool:
			default:
				if !openLintNumber(value) {
					return fmt.Errorf("enumeration values must contain JSON scalars")
				}
			}
		}
	case "schema":
		if _, ok := props["schema"].(map[string]any); !ok {
			return fmt.Errorf("schema option must be an object")
		}
		return (Schema{}).ValidateOptions(props)
	case "xor", "or":
		values, ok := props["properties"].([]any)
		if !ok || len(values) < 2 {
			return fmt.Errorf("%s properties must contain at least two strings", name)
		}
		for _, value := range values {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s properties must contain strings", name)
			}
		}
	case "unreferencedReusableObject":
		return (UnreferencedReusableObject{}).ValidateOptions(props)
	}
	return nil
}

func openLintNumber(value any) bool {
	switch number := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return !math.IsNaN(float64(number)) && !math.IsInf(float64(number), 0)
	case float64:
		return !math.IsNaN(number) && !math.IsInf(number, 0)
	}
	return false
}
