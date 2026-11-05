//go:build !js

// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"fmt"
	"github.com/pb33f/doctor/changerator"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"os"
	"path/filepath"
)

// GenerateChangeReportWithTree compares two specs using the doctor's changerator
// Returns both the DocumentChanges and the node tree for rendering
// newSpecFilePath is the path to the new spec file - its directory is used for $ref resolution (optional, empty = cwd)
func GenerateChangeReportWithTree(originalSpecPath string, newSpecBytes []byte, newSpecFilePath string) (*ChangeResult, error) {
	originalBytes, err := os.ReadFile(originalSpecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read original spec file '%s': %w", originalSpecPath, err)
	}

	// Configure base path for original spec
	leftConfig := datamodel.NewDocumentConfiguration()
	absPath, err := filepath.Abs(originalSpecPath)
	if err == nil {
		leftConfig.BasePath = filepath.Dir(absPath)
		leftConfig.AllowFileReferences = true
	}

	// Parse original (left) spec
	leftLibDoc, err := libopenapi.NewDocumentWithConfiguration(originalBytes, leftConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original spec: %w", err)
	}
	leftModel, leftErr := leftLibDoc.BuildV3Model()
	if leftModel == nil {
		if leftErr != nil {
			return nil, fmt.Errorf("failed to build original spec model: %w", leftErr)
		}
		return nil, fmt.Errorf("failed to build original spec model")
	}
	leftDrDoc := drModel.NewDrDocumentAndGraph(leftModel)

	// Configure base path for new spec (separate from original)
	rightConfig := datamodel.NewDocumentConfiguration()
	if newSpecFilePath != "" {
		absNewPath, pathErr := filepath.Abs(newSpecFilePath)
		if pathErr == nil {
			rightConfig.BasePath = filepath.Dir(absNewPath)
			rightConfig.AllowFileReferences = true
		}
	}

	// Parse new (right) spec with its own config
	rightLibDoc, err := libopenapi.NewDocumentWithConfiguration(newSpecBytes, rightConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new spec: %w", err)
	}
	rightModel, rightErr := rightLibDoc.BuildV3Model()
	if rightModel == nil {
		if rightErr != nil {
			return nil, fmt.Errorf("failed to build new spec model: %w", rightErr)
		}
		return nil, fmt.Errorf("failed to build new spec model")
	}
	rightDrDoc := drModel.NewDrDocumentAndGraph(rightModel)

	// Create and run the changerator
	cd := changerator.NewChangerator(&changerator.ChangeratorConfig{
		LeftDrDoc:  leftDrDoc.V3Document,
		RightDrDoc: rightDrDoc.V3Document,
		Doctor:     rightDrDoc,
	})

	docChanges := cd.Changerate()

	// Build the node change tree for rendering
	if rightDrDoc.V3Document != nil && rightDrDoc.V3Document.Node != nil {
		cd.BuildNodeChangeTree(rightDrDoc.V3Document.Node)
	}

	return &ChangeResult{
		DocumentChanges: docChanges,
		RootNode:        rightDrDoc.V3Document.Node,
	}, nil
}
