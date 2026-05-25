// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"

	"github.com/daveshanley/vacuum/motor"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
)

func buildReportErrors(errs []error) *vacuum_report.ReportErrors {
	if len(errs) == 0 {
		return nil
	}

	items := make([]vacuum_report.ReportError, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		item := vacuum_report.ReportError{Message: err.Error()}
		var lookupErr *motor.RuleLookupError
		if errors.As(err, &lookupErr) {
			item.Type = vacuum_report.ReportErrorTypeRuleLookup
			item.RuleId = lookupErr.RuleId
			item.Given = lookupErr.Given
			item.Message = lookupErr.Error()
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil
	}
	return &vacuum_report.ReportErrors{Items: items}
}
