// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/stretchr/testify/assert"
)

func TestBuildReportErrors(t *testing.T) {
	reportErrors := buildReportErrors([]error{
		&motor.RuleLookupError{
			RuleId: "check-min-length",
			Given:  "$['a']",
			Err:    errors.New("node lookup timeout exceeded (500ms)"),
		},
		nil,
		errors.New("failed to evaluate JSONPath"),
	})

	assert.NotNil(t, reportErrors)
	assert.Len(t, reportErrors.Items, 2)
	assert.Equal(t, "rule check-min-length lookup failed for given $['a']: node lookup timeout exceeded (500ms)", reportErrors.Items[0].Message)
	assert.Equal(t, vacuum_report.ReportErrorTypeRuleLookup, reportErrors.Items[0].Type)
	assert.Equal(t, "check-min-length", reportErrors.Items[0].RuleId)
	assert.Equal(t, "$['a']", reportErrors.Items[0].Given)
	assert.Equal(t, "failed to evaluate JSONPath", reportErrors.Items[1].Message)
}

func TestBuildReportErrors_Empty(t *testing.T) {
	assert.Nil(t, buildReportErrors(nil))
	assert.Nil(t, buildReportErrors([]error{nil}))
}
