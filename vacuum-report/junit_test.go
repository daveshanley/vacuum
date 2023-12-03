// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package vacuum_report

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBuildJUnitReport(t *testing.T) {
	j := testhelp_generateReport()
	j.ResultSet.Results[0].Message = "testing, 123"
	j.ResultSet.Results[0].Path = "$.somewhere.out.there"
	j.ResultSet.Results[0].RuleId = "R0001"
	f := time.Now().Add(-time.Millisecond * 5)
	data := BuildJUnitReport(j.ResultSet, f)
	assert.GreaterOrEqual(t, len(data), 407)
}
