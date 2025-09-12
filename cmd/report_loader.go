// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/daveshanley/vacuum/model"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
)

// ReportLoadResult contains the results of attempting to load a file as either
// a pre-compiled vacuum report or raw OpenAPI spec
type ReportLoadResult struct {
	// If the file was a pre-compiled report
	IsReport bool
	Report   *vacuum_report.VacuumReport
	
	// The raw spec bytes (either from file or extracted from report)
	SpecBytes []byte
	
	// The filename/path for display
	FileName string
	
	// Pre-processed results if from a report
	ResultSet *model.RuleResultSet
}

// LoadFileAsReportOrSpec attempts to load a file as either a pre-compiled vacuum report
// or as a raw OpenAPI specification. It returns a ReportLoadResult with all the necessary
// data for either case.
func LoadFileAsReportOrSpec(filePath string) (*ReportLoadResult, error) {
	// Get the absolute path for consistent handling
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}
	
	result := &ReportLoadResult{
		FileName: filePath,
	}
	
	// Try to load as a vacuum report first
	vacuumReport, bytes, err := vacuum_report.BuildVacuumReportFromFile(absPath)
	if err != nil {
		// If we can't read the file at all, that's an error
		if bytes == nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		// File was read but isn't a report - treat as spec
		result.SpecBytes = bytes
		result.IsReport = false
		return result, nil
	}
	
	// Check if it's actually a report
	if vacuumReport != nil && vacuumReport.ResultSet != nil {
		result.IsReport = true
		result.Report = vacuumReport
		result.ResultSet = vacuumReport.ResultSet
		
		// Extract spec bytes from the report if available
		if vacuumReport.SpecInfo != nil && vacuumReport.SpecInfo.SpecBytes != nil {
			result.SpecBytes = *vacuumReport.SpecInfo.SpecBytes
		}
		
		// Use the original filename from the report's execution if available
		if vacuumReport.Execution != nil && vacuumReport.Execution.SpecFileName != "" {
			result.FileName = vacuumReport.Execution.SpecFileName
		}
		
		return result, nil
	}
	
	// Not a report, treat as regular spec file
	result.SpecBytes = bytes
	result.IsReport = false
	return result, nil
}

// LoadReportOnly attempts to load a file specifically as a vacuum report.
// Returns an error if the file is not a valid vacuum report.
func LoadReportOnly(filePath string) (*vacuum_report.VacuumReport, error) {
	result, err := LoadFileAsReportOrSpec(filePath)
	if err != nil {
		return nil, err
	}
	
	if !result.IsReport {
		return nil, fmt.Errorf("file '%s' is not a vacuum report", filePath)
	}
	
	return result.Report, nil
}

// ExtractSpecFromReport extracts the specification bytes from a vacuum report.
// Returns the spec bytes and the original filename if available.
func ExtractSpecFromReport(report *vacuum_report.VacuumReport) ([]byte, string, error) {
	if report == nil {
		return nil, "", fmt.Errorf("report is nil")
	}
	
	if report.SpecInfo == nil || report.SpecInfo.SpecBytes == nil {
		return nil, "", fmt.Errorf("report does not contain specification data")
	}
	
	fileName := "specification.yaml"
	if report.Execution != nil && report.Execution.SpecFileName != "" {
		fileName = report.Execution.SpecFileName
	}
	
	return *report.SpecInfo.SpecBytes, fileName, nil
}

// IsVacuumReport checks if a file is a vacuum report without fully loading it
func IsVacuumReport(filePath string) bool {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	
	report, err := vacuum_report.CheckFileForVacuumReport(bytes)
	return err == nil && report != nil && report.ResultSet != nil
}