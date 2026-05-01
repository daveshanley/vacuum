// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/loader"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"os"
)

// LoadFileAsReportOrSpec attempts to load a file as either a pre-compiled vacuum report
// or as a raw OpenAPI specification. It returns a ReportLoadResult with all the necessary
// data for either case. Supports both local file paths and remote URLs (http/https).
func LoadFileAsReportOrSpec(filePath string) (*loader.ReportLoadResult, error) {
	return loader.LoadFileAsReportOrSpecWithClient(filePath, nil)
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
