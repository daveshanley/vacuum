// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	drModel "github.com/pb33f/doctor/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
)

// ChangeResult holds the results of a change comparison including the tree for rendering
type ChangeResult struct {
	DocumentChanges *wcModel.DocumentChanges
	RootNode        *drV3.Node // Root node for tree rendering (nil when loading from JSON)
}

// LoadChangeReportFromFile loads a what-changed JSON report from a file
func LoadChangeReportFromFile(path string) (*wcModel.DocumentChanges, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read change report file: %w", err)
	}

	var changes wcModel.DocumentChanges
	if err := json.Unmarshal(data, &changes); err != nil {
		return nil, fmt.Errorf("failed to parse change report JSON: %w", err)
	}

	return &changes, nil
}

// GenerateChangeReport compares two specs and generates a DocumentChanges report
// This is the simple version that only returns DocumentChanges (for backwards compatibility)
// newSpecFilePath is the path to the new spec file - its directory is used for $ref resolution (optional, empty = cwd)
func GenerateChangeReport(originalSpecPath string, newSpecBytes []byte, newSpecFilePath string) (*wcModel.DocumentChanges, error) {
	result, err := GenerateChangeReportWithTree(originalSpecPath, newSpecBytes, newSpecFilePath)
	if err != nil {
		return nil, err
	}
	return result.DocumentChanges, nil
}

// CreateChangeFilterFromSpecs creates a ChangeFilter by comparing an original spec file
// with the new spec bytes, using the provided DrDocument for model location
// newSpecFilePath is the path to the new spec file - its directory is used for $ref resolution (optional, empty = cwd)
func CreateChangeFilterFromSpecs(originalSpecPath string, newSpecBytes []byte, newSpecFilePath string, drDoc *drModel.DrDocument) (*ChangeFilter, error) {
	changes, err := GenerateChangeReport(originalSpecPath, newSpecBytes, newSpecFilePath)
	if err != nil {
		return nil, err
	}

	return NewChangeFilter(changes, drDoc), nil
}

// LoadChangeFilterFromReport creates a ChangeFilter from a pre-existing JSON change report file
func LoadChangeFilterFromReport(reportPath string, drDoc *drModel.DrDocument) (*ChangeFilter, error) {
	changes, err := LoadChangeReportFromFile(reportPath)
	if err != nil {
		return nil, err
	}

	return NewChangeFilter(changes, drDoc), nil
}

// IsChangeReportFile checks if a file path appears to be a JSON change report
// based on file extension
func IsChangeReportFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".json"
}

// IsSpecFile checks if a file path appears to be an OpenAPI spec file
func IsSpecFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml" || ext == ".json"
}

// CreateChangeFilterFromFlags creates a ChangeFilter based on the provided flags.
// If originalFlag is set, compares the original spec file with newSpecBytes.
// If changesFlag is set, loads the change report from the JSON file.
// Returns nil filter (no error) if neither flag is set.
// newSpecFilePath is the path to the new spec file - its directory is used for $ref resolution (optional, empty = cwd)
func CreateChangeFilterFromFlags(changesFlag, originalFlag string, newSpecBytes []byte, newSpecFilePath string, drDoc *drModel.DrDocument) (*ChangeFilter, error) {
	if changesFlag == "" && originalFlag == "" {
		return nil, nil
	}

	if originalFlag != "" {
		return CreateChangeFilterFromSpecs(originalFlag, newSpecBytes, newSpecFilePath, drDoc)
	}
	return LoadChangeFilterFromReport(changesFlag, drDoc)
}
