// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"os"
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/table"
	"github.com/pb33f/testify/require"
)

const watchRelintTestSpec = `openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
`

func TestViolationResultTableModel_PerformRelint(t *testing.T) {
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(watchRelintTestSpec), 0o600))

	model := &ViolationResultTableModel{
		table:    table.New(),
		fileName: specPath,
		watchConfig: &WatchConfig{
			TimeoutFlag: 1,
		},
	}

	msg := model.performRelint()

	complete, ok := msg.(relintCompleteMsg)
	require.True(t, ok)
	require.NotNil(t, complete.specContent)
}

func TestViolationResultTableModel_RelativeRulesetOnRelint(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.yaml")
	parent := filepath.Join(dir, "parent.yaml")
	require.NoError(t, os.WriteFile(spec, []byte(watchRelintTestSpec), 0600))
	require.NoError(t, os.WriteFile(parent, []byte("extends: [./child.yaml]\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child.yaml"), []byte("rules:\n  child-rule:\n    given: $.info.title\n    severity: error\n    then: {function: falsy}\n"), 0600))
	model := &ViolationResultTableModel{table: table.New(), fileName: spec, watchConfig: &WatchConfig{TimeoutFlag: 5, RulesetFlag: parent}}
	msg := model.performRelint()
	complete, ok := msg.(relintCompleteMsg)
	require.True(t, ok, "%#v", msg)
	require.NotNil(t, complete.specContent)
	require.Len(t, complete.results, 1)
	require.Equal(t, "child-rule", complete.results[0].RuleId)
}
