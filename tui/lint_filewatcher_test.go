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
