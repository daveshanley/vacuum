// Copyright 2024-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	"encoding/json"
	"fmt"
)

// LSPConfig represents the complete configuration for the vacuum language server.
// This struct serves as the canonical representation of all configurable options,
// regardless of whether they come from a config file, InitializationOptions, or
// workspace/didChangeConfiguration.
type LSPConfig struct {
	// Ruleset specifies a path to a custom ruleset file (local or remote URL)
	Ruleset string `json:"ruleset,omitempty"`

	// Functions specifies a path to custom function definitions
	Functions string `json:"functions,omitempty"`

	// Base overrides the base URL/path for resolving references
	Base string `json:"base,omitempty"`

	// Remote controls whether remote HTTP references are resolved (default: true)
	Remote *bool `json:"remote,omitempty"`

	// SkipCheck skips OpenAPI document validation
	SkipCheck *bool `json:"skipCheck,omitempty"`

	// Timeout is the rule execution timeout in seconds (default: 5)
	Timeout *int `json:"timeout,omitempty"`

	// LookupTimeout is the node lookup timeout in milliseconds
	LookupTimeout *int `json:"lookupTimeout,omitempty"`

	// HardMode enables all built-in rules including OWASP
	HardMode *bool `json:"hardMode,omitempty"`

	// IgnoreArrayCircleRef ignores circular array references
	IgnoreArrayCircleRef *bool `json:"ignoreArrayCircleRef,omitempty"`

	// IgnorePolymorphCircleRef ignores circular polymorphic references
	IgnorePolymorphCircleRef *bool `json:"ignorePolymorphCircleRef,omitempty"`

	// ExtensionRefs enables $ref lookups for extension objects
	ExtensionRefs *bool `json:"extensionRefs,omitempty"`

	// IgnoreFile specifies a path to the ignore file
	IgnoreFile string `json:"ignoreFile,omitempty"`

	// TLS configuration for remote references
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
	CAFile   string `json:"caFile,omitempty"`
	Insecure *bool  `json:"insecure,omitempty"`
}

// MergeConfig merges source into target. Non-nil/non-empty values from source
// override corresponding values in target.
func MergeConfig(target, source *LSPConfig) {
	if source == nil {
		return
	}

	if source.Ruleset != "" {
		target.Ruleset = source.Ruleset
	}
	if source.Functions != "" {
		target.Functions = source.Functions
	}
	if source.Base != "" {
		target.Base = source.Base
	}
	if source.Remote != nil {
		target.Remote = source.Remote
	}
	if source.SkipCheck != nil {
		target.SkipCheck = source.SkipCheck
	}
	if source.Timeout != nil {
		target.Timeout = source.Timeout
	}
	if source.LookupTimeout != nil {
		target.LookupTimeout = source.LookupTimeout
	}
	if source.HardMode != nil {
		target.HardMode = source.HardMode
	}
	if source.IgnoreArrayCircleRef != nil {
		target.IgnoreArrayCircleRef = source.IgnoreArrayCircleRef
	}
	if source.IgnorePolymorphCircleRef != nil {
		target.IgnorePolymorphCircleRef = source.IgnorePolymorphCircleRef
	}
	if source.ExtensionRefs != nil {
		target.ExtensionRefs = source.ExtensionRefs
	}
	if source.IgnoreFile != "" {
		target.IgnoreFile = source.IgnoreFile
	}
	if source.CertFile != "" {
		target.CertFile = source.CertFile
	}
	if source.KeyFile != "" {
		target.KeyFile = source.KeyFile
	}
	if source.CAFile != "" {
		target.CAFile = source.CAFile
	}
	if source.Insecure != nil {
		target.Insecure = source.Insecure
	}
}

// ParseLSPConfig parses configuration from an arbitrary JSON value.
// Supports both direct format {"ruleset": "..."} and nested format {"vacuum": {"ruleset": "..."}}.
func ParseLSPConfig(data any) (*LSPConfig, error) {
	if data == nil {
		return nil, nil
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config data: %w", err)
	}

	// Try direct parse
	var config LSPConfig
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// If empty, try nested format
	if config.isEmpty() {
		var nested struct {
			Vacuum *LSPConfig `json:"vacuum"`
		}
		if err := json.Unmarshal(jsonBytes, &nested); err == nil && nested.Vacuum != nil {
			return nested.Vacuum, nil
		}
	}

	return &config, nil
}

// isEmpty returns true if no configuration values are set
func (c *LSPConfig) isEmpty() bool {
	return c.Ruleset == "" &&
		c.Functions == "" &&
		c.Base == "" &&
		c.Remote == nil &&
		c.SkipCheck == nil &&
		c.Timeout == nil &&
		c.LookupTimeout == nil &&
		c.HardMode == nil &&
		c.IgnoreArrayCircleRef == nil &&
		c.IgnorePolymorphCircleRef == nil &&
		c.ExtensionRefs == nil &&
		c.IgnoreFile == "" &&
		c.CertFile == "" &&
		c.KeyFile == "" &&
		c.CAFile == "" &&
		c.Insecure == nil
}

// Helper functions for creating pointers to primitives
func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }
