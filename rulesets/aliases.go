// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
)

// SimpleAlias is a list of JSONPath expressions (format-independent).
type SimpleAlias []string

// AliasTarget pairs a set of spec formats with a set of JSONPath expressions.
type AliasTarget struct {
	Formats []string
	Given   []string
}

// TargetedAlias is an alias with format-specific targets.
type TargetedAlias struct {
	Description string
	Targets     []AliasTarget
}

// ParsedAlias wraps either a simple or targeted alias to avoid interface boxing.
type ParsedAlias struct {
	Simple   SimpleAlias
	Targeted *TargetedAlias
}

// ParseAliases converts the raw aliases map from YAML/JSON into concrete ParsedAlias structs.
func ParseAliases(raw map[string]interface{}) (map[string]*ParsedAlias, error) {
	result := make(map[string]*ParsedAlias, len(raw))
	for name, value := range raw {
		pa, err := parseOneAlias(name, value)
		if err != nil {
			return nil, err
		}
		result[name] = pa
	}
	return result, nil
}

func parseOneAlias(name string, value interface{}) (*ParsedAlias, error) {
	switch v := value.(type) {
	case string:
		return &ParsedAlias{Simple: SimpleAlias{v}}, nil
	case []interface{}:
		paths := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				paths = append(paths, s)
			} else {
				return nil, fmt.Errorf("alias %q: array element is not a string", name)
			}
		}
		return &ParsedAlias{Simple: SimpleAlias(paths)}, nil
	case map[string]interface{}:
		return parseTargetedAlias(name, v)
	default:
		return nil, fmt.Errorf("alias %q: unsupported type %T", name, value)
	}
}

func parseTargetedAlias(name string, m map[string]interface{}) (*ParsedAlias, error) {
	ta := &TargetedAlias{}
	if desc, ok := m["description"].(string); ok {
		ta.Description = desc
	}
	targetsRaw, ok := m["targets"]
	if !ok {
		return nil, fmt.Errorf("alias %q: targeted alias missing 'targets' key", name)
	}
	targetsSlice, ok := targetsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("alias %q: 'targets' must be an array", name)
	}
	for _, tRaw := range targetsSlice {
		tMap, ok := tRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("alias %q: target entry must be an object", name)
		}
		at := AliasTarget{}
		// parse formats
		if fmts, ok := tMap["formats"]; ok {
			if fmtSlice, ok := fmts.([]interface{}); ok {
				for _, f := range fmtSlice {
					if fs, ok := f.(string); ok {
						at.Formats = append(at.Formats, fs)
					}
				}
			}
		}
		// given can be a string or array of strings per Spectral schema
		if given, ok := tMap["given"]; ok {
			switch g := given.(type) {
			case string:
				at.Given = []string{g}
			case []interface{}:
				for _, item := range g {
					if s, ok := item.(string); ok {
						at.Given = append(at.Given, s)
					}
				}
			}
		}
		ta.Targets = append(ta.Targets, at)
	}
	return &ParsedAlias{Targeted: ta}, nil
}

// ResolveAliasesForFormat resolves all parsed aliases for a specific spec format.
// Simple aliases pass through as-is (format-independent).
// Targeted aliases collect paths from matching format targets.
func ResolveAliasesForFormat(parsed map[string]*ParsedAlias, specFormat string) map[string][]string {
	result := make(map[string][]string, len(parsed))
	for name, pa := range parsed {
		if pa.Simple != nil {
			result[name] = []string(pa.Simple)
		} else if pa.Targeted != nil {
			var paths []string
			for _, target := range pa.Targeted.Targets {
				if formatMatchesAny(target.Formats, specFormat) {
					paths = append(paths, target.Given...)
				}
			}
			result[name] = paths
		}
	}
	return result
}

// formatMatchesAny checks if specFormat matches any of the alias target formats.
func formatMatchesAny(formats []string, specFormat string) bool {
	if len(formats) == 0 {
		return true // no format restriction
	}
	for _, f := range formats {
		if model.FormatMatches(f, specFormat) {
			return true
		}
	}
	return false
}

// ExpandAliasReferences recursively expands #AliasName references within alias paths.
// Detects circular references and returns an error if found.
func ExpandAliasReferences(aliases map[string][]string) (map[string][]string, error) {
	result := make(map[string][]string, len(aliases))
	visited := make(map[string]bool, len(aliases))
	visiting := make(map[string]bool)

	var expand func(name string) ([]string, error)
	expand = func(name string) ([]string, error) {
		if visiting[name] {
			return nil, fmt.Errorf("circular alias reference: %s", name)
		}
		if visited[name] {
			return result[name], nil
		}
		visiting[name] = true

		paths := aliases[name]
		expanded := make([]string, 0, len(paths))
		for _, p := range paths {
			if refName, suffix, isRef := parseAliasRef(p); isRef {
				if _, ok := aliases[refName]; !ok {
					return nil, fmt.Errorf("unknown alias reference: #%s", refName)
				}
				refResult, err := expand(refName)
				if err != nil {
					return nil, err
				}
				for _, rp := range refResult {
					expanded = append(expanded, rp+suffix)
				}
			} else {
				expanded = append(expanded, p)
			}
		}
		visiting[name] = false
		visited[name] = true
		result[name] = expanded
		return expanded, nil
	}

	for name := range aliases {
		if _, err := expand(name); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// parseAliasRef extracts alias name and suffix from a #AliasRef path.
// Returns (aliasName, suffix, true) if the path starts with #, or ("", "", false) otherwise.
func parseAliasRef(path string) (string, string, bool) {
	if len(path) == 0 || path[0] != '#' {
		return "", "", false
	}
	// Find end of alias name: [A-Za-z0-9_-]
	end := 1
	for end < len(path) {
		ch := path[end]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			end++
		} else {
			break
		}
	}
	if end <= 1 {
		return "", "", false
	}
	return path[1:end], path[end:], true
}

// ExpandRuleGivenPaths expands #AliasRef references in a rule's given paths.
// Returns the input slice directly (zero allocation) when no path contains '#'.
// Does NOT mutate the input slice.
func ExpandRuleGivenPaths(givenPaths []string, expandedAliases map[string][]string) ([]string, error) {
	// Early-exit: scan for any '#' reference
	hasRef := false
	for _, p := range givenPaths {
		if strings.IndexByte(p, '#') >= 0 {
			hasRef = true
			break
		}
	}
	if !hasRef {
		return givenPaths, nil
	}

	result := make([]string, 0, len(givenPaths))
	for _, p := range givenPaths {
		if refName, suffix, isRef := parseAliasRef(p); isRef {
			aliasPaths, ok := expandedAliases[refName]
			if !ok {
				return nil, fmt.Errorf("unknown alias: #%s in rule given path", refName)
			}
			for _, ap := range aliasPaths {
				result = append(result, ap+suffix)
			}
		} else {
			result = append(result, p)
		}
	}
	return result, nil
}
