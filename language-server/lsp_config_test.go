// Copyright 2024-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeConfig_NilSource(t *testing.T) {
	target := &LSPConfig{
		Ruleset: "original.yaml",
		Timeout: intPtr(10),
	}

	MergeConfig(target, nil)

	assert.Equal(t, "original.yaml", target.Ruleset)
	assert.Equal(t, 10, *target.Timeout)
}

func TestMergeConfig_OverridesNonEmpty(t *testing.T) {
	target := &LSPConfig{
		Ruleset: "original.yaml",
		Timeout: intPtr(10),
		Remote:  boolPtr(true),
	}

	source := &LSPConfig{
		Ruleset: "override.yaml",
		Timeout: intPtr(20),
	}

	MergeConfig(target, source)

	assert.Equal(t, "override.yaml", target.Ruleset)
	assert.Equal(t, 20, *target.Timeout)
	assert.True(t, *target.Remote) // Unchanged since source.Remote is nil
}

func TestMergeConfig_PreservesExistingWhenSourceEmpty(t *testing.T) {
	target := &LSPConfig{
		Ruleset:   "original.yaml",
		Functions: "funcs.js",
		Timeout:   intPtr(10),
	}

	source := &LSPConfig{
		// Empty - nothing set
	}

	MergeConfig(target, source)

	assert.Equal(t, "original.yaml", target.Ruleset)
	assert.Equal(t, "funcs.js", target.Functions)
	assert.Equal(t, 10, *target.Timeout)
}

func TestMergeConfig_BooleanFalseOverrides(t *testing.T) {
	target := &LSPConfig{
		Remote: boolPtr(true),
	}

	source := &LSPConfig{
		Remote: boolPtr(false),
	}

	MergeConfig(target, source)

	assert.False(t, *target.Remote)
}

func TestParseLSPConfig_DirectFormat(t *testing.T) {
	data := map[string]interface{}{
		"ruleset": "custom.yaml",
		"timeout": 15,
		"remote":  false,
	}

	config, err := ParseLSPConfig(data)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "custom.yaml", config.Ruleset)
	assert.Equal(t, 15, *config.Timeout)
	assert.False(t, *config.Remote)
}

func TestParseLSPConfig_NestedFormat(t *testing.T) {
	data := map[string]interface{}{
		"vacuum": map[string]interface{}{
			"ruleset": "nested.yaml",
			"timeout": 30,
		},
	}

	config, err := ParseLSPConfig(data)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "nested.yaml", config.Ruleset)
	assert.Equal(t, 30, *config.Timeout)
}

func TestParseLSPConfig_NilData(t *testing.T) {
	config, err := ParseLSPConfig(nil)

	assert.NoError(t, err)
	assert.Nil(t, config)
}

func TestLSPConfig_IsEmpty(t *testing.T) {
	empty := &LSPConfig{}
	assert.True(t, empty.isEmpty())

	nonEmpty := &LSPConfig{Ruleset: "test.yaml"}
	assert.False(t, nonEmpty.isEmpty())

	withBool := &LSPConfig{Remote: boolPtr(false)}
	assert.False(t, withBool.isEmpty())
}

func TestHelperFunctions(t *testing.T) {
	b := boolPtr(true)
	assert.True(t, *b)

	b2 := boolPtr(false)
	assert.False(t, *b2)

	i := intPtr(42)
	assert.Equal(t, 42, *i)
}
