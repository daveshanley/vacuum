package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/testify/require"
)

func TestBuiltinRepairCLIAndPreview(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.json")
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["cat","cat","dog"]}}}}`)
	require.NoError(t, os.WriteFile(spec, original, 0600))
	planPath := filepath.Join(dir, "plan.json")
	_, _, err := exportCommand(t, "lint", spec, "--fix-plan", planPath, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
	encoded, err := os.ReadFile(planPath)
	require.NoError(t, err)
	var plan repair.Plan
	require.NoError(t, json.Unmarshal(encoded, &plan))
	require.NotEmpty(t, plan.Targets)
	_, _, err = exportCommand(t, "lint", spec, "--fix", "--fix-builtins", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err = os.ReadFile(spec)
	require.NoError(t, err)
	require.True(t, json.Valid(saved))
	require.NotEqual(t, original, saved)
	var document map[string]any
	require.NoError(t, json.Unmarshal(saved, &document))
	enum := document["components"].(map[string]any)["schemas"].(map[string]any)["Pet"].(map[string]any)["enum"].([]any)
	require.Equal(t, []any{"cat", "dog"}, enum)
	_, _, err = exportCommand(t, "lint", spec, "--fix", "--fix-builtins", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	again, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, saved, again)
}

func TestBuiltinRepairRespectsIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	ignore := filepath.Join(dir, "ignore.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {type: string, enum: [cat, cat]}\n")
	require.NoError(t, os.WriteFile(spec, original, 0600))
	require.NoError(t, os.WriteFile(ignore, []byte("duplicated-entry-in-enum:\n  - $.components.schemas.Pet.enum\n"), 0600))
	_, _, err := exportCommand(t, "lint", spec, "--fix", "--fix-builtins", "--ignore-file", ignore, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
}

func TestPolicyRepairCLI(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	policy := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), 0600))
	require.NoError(t, os.WriteFile(policy, []byte("version: 1\npolicies:\n  - id: contact\n    rule: info-contact\n    diagnostic: contact-missing\n    scope: {source: api.yaml, pointer: /info}\n    value: {name: Team, email: api@example.com, url: https://example.com}\n"), 0600))
	_, _, err := exportCommand(t, "lint", spec, "--fix", "--fix-builtins", "--fix-policy", policy, "--hard-mode", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Contains(t, string(saved), "api@example.com")
}

func TestBuiltinRepairRejectsDocumentStream(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stream.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {enum: [a, a]}\n---\nx-keep: true\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	_, _, err := exportCommand(t, "lint", path, "--fix", "--fix-builtins", "--format", "json", "--fail-severity", "none")
	require.Error(t, err)
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, saved)
}

func TestBuiltinRepairSchemaCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","enum":["a","a"]}`), 0600))
	_, _, err := exportCommand(t, "schema", path, "--fix", "--fix-builtins", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(saved, &doc))
	require.Len(t, doc["enum"], 1)
}
