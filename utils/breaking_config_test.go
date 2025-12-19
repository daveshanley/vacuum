// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"os"
	"path/filepath"
	"testing"

	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBreakingRulesConfig_ExplicitPath_Exists(t *testing.T) {
	// Create a temporary valid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `schema:
  type:
    modified: false
pathItem:
  get:
    removed: false`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadBreakingRulesConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify config was loaded correctly
	assert.NotNil(t, config.Schema)
	assert.NotNil(t, config.Schema.Type)
	assert.NotNil(t, config.Schema.Type.Modified)
	assert.False(t, *config.Schema.Type.Modified)
}

func TestLoadBreakingRulesConfig_ExplicitPath_NotExists(t *testing.T) {
	config, err := LoadBreakingRulesConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "breaking config file not found")
}

func TestLoadBreakingRulesConfig_DefaultCWD(t *testing.T) {
	// Save current directory and change to temp dir
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Create default config file in CWD
	configContent := `schema:
  description:
    modified: true`

	err = os.WriteFile(DefaultBreakingConfigFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load without explicit path - should find default
	config, err := LoadBreakingRulesConfig("")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.NotNil(t, config.Schema)
	assert.NotNil(t, config.Schema.Description)
	assert.NotNil(t, config.Schema.Description.Modified)
	assert.True(t, *config.Schema.Description.Modified)
}

func TestLoadBreakingRulesConfig_NoConfigFound(t *testing.T) {
	// Save current directory and change to temp dir with no config
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Load without explicit path - should return nil (use defaults)
	config, err := LoadBreakingRulesConfig("")
	assert.NoError(t, err)
	assert.Nil(t, config)
}

func TestLoadBreakingRulesConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	config, err := LoadBreakingRulesConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
	// libopenapi's validation catches YAML errors first, returning a ConfigValidationError
	var validationErr *ConfigValidationError
	if assert.ErrorAs(t, err, &validationErr) {
		assert.NotNil(t, validationErr.Result)
	}
}

func TestLoadBreakingRulesConfig_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-structure.yaml")

	// Write structurally invalid config (nested component that should be at root)
	// This should trigger validation errors in libopenapi
	configContent := `schema:
  discriminator:
    propertyName:
      modified: false`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadBreakingRulesConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)

	// Check it's a ConfigValidationError
	var validationErr *ConfigValidationError
	if assert.ErrorAs(t, err, &validationErr) {
		assert.Equal(t, configPath, validationErr.FilePath)
		assert.NotNil(t, validationErr.Result)
		assert.Greater(t, len(validationErr.Result.Errors), 0)

		// Test FormatValidationErrors
		formatted := validationErr.FormatValidationErrors()
		assert.NotEmpty(t, formatted)
	}
}

func TestExpandUserPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde expansion",
			input:    "~/config.yaml",
			expected: filepath.Join(home, "config.yaml"),
		},
		{
			name:     "tilde with subdir",
			input:    "~/.config/rules.yaml",
			expected: filepath.Join(home, ".config/rules.yaml"),
		},
		{
			name:     "absolute path unchanged",
			input:    "/absolute/path/config.yaml",
			expected: "/absolute/path/config.yaml",
		},
		{
			name:     "relative path unchanged",
			input:    "./relative/config.yaml",
			expected: "./relative/config.yaml",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandUserPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyBreakingRulesConfig(t *testing.T) {
	// Reset to known state
	wcModel.ResetActiveBreakingRulesConfig()
	defer wcModel.ResetActiveBreakingRulesConfig()

	// Get default config to compare
	defaultConfig := wcModel.GetActiveBreakingRulesConfig()
	require.NotNil(t, defaultConfig)

	// Create a custom config that overrides one rule
	boolTrue := true
	boolFalse := false
	customConfig := &wcModel.BreakingRulesConfig{
		Schema: &wcModel.SchemaRules{
			Type: &wcModel.BreakingChangeRule{
				Modified: &boolFalse, // Override: type changes not breaking
			},
		},
		PathItem: &wcModel.PathItemRules{
			Get: &wcModel.BreakingChangeRule{
				Removed: &boolTrue, // Keep as breaking
			},
		},
	}

	// Apply custom config
	ApplyBreakingRulesConfig(customConfig)

	// Verify the config was applied (merged with defaults)
	activeConfig := wcModel.GetActiveBreakingRulesConfig()
	require.NotNil(t, activeConfig)

	// Check our override was applied
	assert.NotNil(t, activeConfig.Schema)
	assert.NotNil(t, activeConfig.Schema.Type)
	assert.NotNil(t, activeConfig.Schema.Type.Modified)
	assert.False(t, *activeConfig.Schema.Type.Modified)
}

func TestApplyBreakingRulesConfig_Nil(t *testing.T) {
	// Reset to known state
	wcModel.ResetActiveBreakingRulesConfig()
	defer wcModel.ResetActiveBreakingRulesConfig()

	// Apply nil config should reset to defaults
	ApplyBreakingRulesConfig(nil)

	// Should have default config
	config := wcModel.GetActiveBreakingRulesConfig()
	assert.NotNil(t, config)
}

func TestResetBreakingRulesConfig(t *testing.T) {
	// Apply some custom config first
	boolFalse := false
	customConfig := &wcModel.BreakingRulesConfig{
		Schema: &wcModel.SchemaRules{
			Type: &wcModel.BreakingChangeRule{
				Modified: &boolFalse,
			},
		},
	}
	ApplyBreakingRulesConfig(customConfig)

	// Reset
	ResetBreakingRulesConfig()

	// Should be back to defaults
	config := wcModel.GetActiveBreakingRulesConfig()
	assert.NotNil(t, config)
}

func TestConfigValidationError_FormatValidationErrors_Empty(t *testing.T) {
	err := &ConfigValidationError{
		FilePath: "/test/path",
		Result:   nil,
	}
	assert.Empty(t, err.FormatValidationErrors())

	err2 := &ConfigValidationError{
		FilePath: "/test/path",
		Result:   &wcModel.ConfigValidationResult{},
	}
	assert.Empty(t, err2.FormatValidationErrors())
}

func TestGetDefaultConfigPaths(t *testing.T) {
	paths := getDefaultConfigPaths()
	assert.GreaterOrEqual(t, len(paths), 1)
	assert.Equal(t, "./"+DefaultBreakingConfigFile, paths[0])

	// If home directory is available, should have second path
	if home, err := os.UserHomeDir(); err == nil {
		assert.Equal(t, 2, len(paths))
		assert.Equal(t, filepath.Join(home, ".config", DefaultBreakingConfigFile), paths[1])
	}
}
