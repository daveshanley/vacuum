// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidOpenAPIExtensions defines the valid file extensions for OpenAPI specifications
// Also includes .json.gz for pre-compiled vacuum reports
var ValidOpenAPIExtensions = []string{"yaml", "yml", "json", "json.gz"}

// GetFilesToProcess resolves files from glob patterns and arguments.
// This is a shared utility for commands that need multi-file support.
// It handles glob patterns via --globbed-files flag and direct file arguments.
// Supports simple bash-style brace expansion like *.{json,yaml,yml}.
func GetFilesToProcess(globPattern string, args []string) ([]string, error) {
	var files []string

	// Add files from arguments
	if len(args) > 0 {
		files = append(files, args[0]) // First arg is always the spec file
	}

	// Add files from glob pattern (with brace expansion support)
	if globPattern != "" {
		patterns := expandBracePattern(globPattern)
		seen := make(map[string]bool)

		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
			}
			for _, match := range matches {
				if !seen[match] {
					seen[match] = true
					files = append(files, match)
				}
			}
		}
	}

	// Remove duplicates (for files from args)
	files = deduplicateFiles(files)

	// Validate extensions
	for _, file := range files {
		if !hasValidOpenAPIExtension(file) {
			return nil, fmt.Errorf("file %q has an invalid extension; only %v are supported",
				file, ValidOpenAPIExtensions)
		}
	}

	return files, nil
}

// expandBracePattern expands simple bash-style brace patterns like *.{json,yaml}
// into multiple patterns ["*.json", "*.yaml"]. Only supports single, non-nested
// brace groups. For complex patterns, users should let their shell expand them.
func expandBracePattern(pattern string) []string {
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")

	// No braces, malformed, or empty braces - return as-is
	if start == -1 || end == -1 || end <= start+1 {
		return []string{pattern}
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	alternatives := strings.Split(pattern[start+1:end], ",")

	var expanded []string
	for _, alt := range alternatives {
		alt = strings.TrimSpace(alt)
		if alt != "" {
			expanded = append(expanded, prefix+alt+suffix)
		}
	}

	// If all alternatives were empty, return original
	if len(expanded) == 0 {
		return []string{pattern}
	}

	return expanded
}

// deduplicateFiles removes duplicate file paths from a slice
func deduplicateFiles(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, val := range input {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// hasValidOpenAPIExtension checks if a file has a valid OpenAPI extension
func hasValidOpenAPIExtension(filename string) bool {
	for _, ext := range ValidOpenAPIExtensions {
		if strings.HasSuffix(strings.ToLower(filename), "."+ext) {
			return true
		}
	}
	return false
}

// GenerateReportFileName creates a report filename based on the input spec file.
// For multi-file processing, this ensures each report has a unique, identifiable name.
//
// Parameters:
//   - specFile: the input spec file path (e.g., "specs/api.yaml")
//   - outputDir: directory to write the report to (empty = current directory)
//   - prefix: optional prefix for the report name (e.g., "vacuum-report")
//   - timestamp: timestamp string to append (e.g., "01-02-06-15_04_05")
//   - extension: file extension including dot (e.g., ".json", ".html", ".json.gz")
//
// Returns:
//   - Full path to the output report file
func GenerateReportFileName(specFile, outputDir, prefix, timestamp, extension string) string {
	// Extract base name without extension from spec file
	baseName := filepath.Base(specFile)
	// Remove all extensions (handles .yaml, .yml, .json)
	for _, ext := range ValidOpenAPIExtensions {
		baseName = strings.TrimSuffix(baseName, "."+ext)
	}

	// Build the report filename
	var reportName string
	if prefix != "" {
		reportName = fmt.Sprintf("%s-%s-%s%s", prefix, baseName, timestamp, extension)
	} else {
		reportName = fmt.Sprintf("%s-%s%s", baseName, timestamp, extension)
	}

	// Combine with output directory
	if outputDir != "" {
		return filepath.Join(outputDir, reportName)
	}
	return reportName
}

// EnsureOutputDir creates the output directory if it doesn't exist
func EnsureOutputDir(outputDir string) error {
	if outputDir == "" {
		return nil
	}
	return os.MkdirAll(outputDir, 0755)
}

// MultiFileReportConfig holds configuration for multi-file report generation
type MultiFileReportConfig struct {
	GlobPattern string // --globbed-files pattern
	OutputDir   string // --output-dir directory
	Silent      bool   // Suppress non-essential output
	NoStyle     bool   // Disable styling
}

// AddMultiFileFlags adds the common multi-file flags to a cobra command.
// Call this in your command's flag setup.
func AddMultiFileReportFlags(cmd interface {
	Flags() interface {
		String(name string, value string, usage string)
	}
}) {
	flags := cmd.Flags()
	flags.String("globbed-files", "", "Glob pattern of files to process (e.g., 'specs/*.yaml')")
	flags.String("output-dir", "", "Directory to write report files to (default: current directory)")
}
