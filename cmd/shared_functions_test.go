package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
)

func TestRenderTime(t *testing.T) {
	start := time.Now()
	time.Sleep(1 * time.Millisecond)
	fi, _ := os.Stat("shared_functions.go")
	RenderTime(true, time.Since(start), fi.Size())
}

func TestLoadCustomFunctionsRejectsFlagValueAsPath(t *testing.T) {
	_, err := LoadCustomFunctions("--no-banner", true)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--functions requires a path")
}

func TestLoadCustomFunctionsSkipsSpectralModuleAndKeepsVacuumFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spectral.js"), []byte(`import { createRulesetFunction } from "@stoplight/spectral-core";
export default createRulesetFunction({}, () => []);
`), 0o600))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "vacuumFunc.js"), []byte(`function getSchema() {
    return {"name": "vacuumFunc", "description": "a vacuum function"};
}
function runRule(input) {
    return [];
}
`), 0o600))

	funcs, err := LoadCustomFunctions(tmpDir, true)

	// the unsupported Spectral module is skipped, the vacuum function still loads
	assert.NoError(t, err)
	assert.Contains(t, funcs, "vacuumFunc")
	assert.Len(t, funcs, 1)
}

func TestBuildRuleSetFromUserSuppliedLocationReturnsExternalLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	rulesetPath := filepath.Join(tmpDir, "main.yml")
	assert.NoError(t, os.WriteFile(rulesetPath, []byte(`extends: missing.yml
`), 0o600))

	_, err := BuildRuleSetFromUserSuppliedLocation(rulesetPath, rulesets.BuildDefaultRuleSets(), true, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot open external ruleset")
}
