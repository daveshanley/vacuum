// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"go.yaml.in/yaml/v4"
)

// DefaultBreakingConfigFile is the default filename for breaking rules configuration
const DefaultBreakingConfigFile = "changes-rules.yaml"

// ConfigValidationError wraps validation errors with file context
type ConfigValidationError struct {
	FilePath string
	Result   *wcModel.ConfigValidationResult
}

func (e *ConfigValidationError) Error() string {
	return fmt.Sprintf("invalid breaking config in %s: %d errors", e.FilePath, len(e.Result.Errors))
}

// FormatValidationErrors returns a formatted string of all validation errors
func (e *ConfigValidationError) FormatValidationErrors() string {
	if e.Result == nil || len(e.Result.Errors) == 0 {
		return ""
	}

	var result string
	for _, err := range e.Result.Errors {
		result += fmt.Sprintf("  - %s (line %d, column %d)\n", err.Message, err.Line, err.Column)
		if err.SuggestedPath != "" {
			result += fmt.Sprintf("    Suggestion: use '%s' instead of '%s'\n", err.SuggestedPath, err.Path)
		}
	}
	return result
}

// LoadBreakingRulesConfig loads breaking rules from specified path or default locations.
// If configPath is specified, it must exist or an error is returned.
// If configPath is empty, searches default locations (CWD and ~/.config/).
// Returns nil with no error if no config is found (use libopenapi defaults).
func LoadBreakingRulesConfig(configPath string) (*wcModel.BreakingRulesConfig, error) {
	// If user specified a path, it must exist
	if configPath != "" {
		return loadConfigFromPath(configPath, true)
	}

	// Check default locations: ./changes-rules.yaml, ~/.config/changes-rules.yaml
	defaultPaths := getDefaultConfigPaths()
	for _, path := range defaultPaths {
		config, err := loadConfigFromPath(path, false)
		if err != nil {
			// Return YAML parse errors and validation errors, but not "file not found"
			return nil, err
		}
		if config != nil {
			return config, nil
		}
	}

	// No config found, use defaults
	return nil, nil
}

// loadConfigFromPath loads and validates a breaking rules config from a file path.
// If required is true, returns an error if the file doesn't exist.
// If required is false, returns nil (no error) if the file doesn't exist.
func loadConfigFromPath(configPath string, required bool) (*wcModel.BreakingRulesConfig, error) {
	expandedPath := expandUserPath(configPath)

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if required {
				return nil, fmt.Errorf("breaking config file not found: %s", expandedPath)
			}
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate config structure before parsing
	if validationResult := wcModel.ValidateBreakingRulesConfigYAML(data); validationResult != nil {
		return nil, &ConfigValidationError{
			FilePath: expandedPath,
			Result:   validationResult,
		}
	}

	var config wcModel.BreakingRulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", expandedPath, err)
	}

	return &config, nil
}

// getDefaultConfigPaths returns the list of default paths to search for breaking config
func getDefaultConfigPaths() []string {
	paths := []string{"./" + DefaultBreakingConfigFile}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", DefaultBreakingConfigFile))
	}

	return paths
}

// expandUserPath expands ~ to the user's home directory
func expandUserPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// ApplyBreakingRulesConfig applies config to libopenapi's active config.
// The config is merged on top of the default rules, so users only need to
// specify the rules they want to override.
// Call with nil to reset to defaults.
func ApplyBreakingRulesConfig(config *wcModel.BreakingRulesConfig) {
	if config == nil {
		wcModel.ResetActiveBreakingRulesConfig()
		return
	}

	defaults := wcModel.GenerateDefaultBreakingRules()
	defaults.Merge(config)
	wcModel.SetActiveBreakingRulesConfig(defaults)
}

// ResetBreakingRulesConfig resets the active breaking rules to defaults.
// This is a convenience wrapper around wcModel.ResetActiveBreakingRulesConfig().
func ResetBreakingRulesConfig() {
	wcModel.ResetActiveBreakingRulesConfig()
}
