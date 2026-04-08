//go:build !html_report_ui

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLReportCommand_ResolveRefFlags(t *testing.T) {
	specPath := writeResolveAllRefsTestSpec(t)
	rulesetPath := writeResolveAllRefsRuleset(t)
	reportFile := filepath.Join(t.TempDir(), "resolve-all-refs.html")

	cmd := GetHTMLReportCommand()
	registerPersistentFlags(cmd)
	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--ruleset", rulesetPath,
		"--resolve-all-refs",
		"--nested-refs-doc-context",
		"-b",
		specPath,
		reportFile,
	})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "html-report support is not included in this build")
	_, statErr := os.Stat(reportFile)
	assert.True(t, os.IsNotExist(statErr))
}
