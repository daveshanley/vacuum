// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"fmt"
	"github.com/pb33f/testify/require"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyExactScopeAndCatalog(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "api.yaml")
	data := []byte("openapi: 3.1.0\npaths:\n  /pets:\n    get: {operationId: listPets}\n")
	text := []byte("version: 1\npolicies:\n  - id: docs\n    rule: operation-description\n    diagnostic: description-missing\n    scope: {source: api.yaml, pointer: /paths/~1pets/get}\n    catalog: docs.yaml\n")
	catalog := map[string][]byte{filepath.Join(base, "docs.yaml"): []byte("api.yaml:\n  listPets: List all pets.\n")}
	policy, err := ParsePolicy(text, base, catalog)
	require.NoError(t, err)
	findings := []Finding{{RuleID: "operation-description", Code: "description-missing", Pointer: "/paths/~1pets/get"}}
	plan, err := ApplyPolicy(NewPlan(source, data), data, findings, policy)
	require.NoError(t, err)
	require.Len(t, plan.Operations, 1)
	require.NotEmpty(t, plan.PolicyHash)
	result, err := Apply(source, data, plan)
	require.NoError(t, err)
	require.Contains(t, string(result), "List all pets.")
	for _, wrong := range []Finding{{RuleID: "operation-description", Code: "other", Pointer: "/paths/~1pets/get"}, {RuleID: "operation-description", Pointer: "/paths/~1pets"}} {
		p, err := ApplyPolicy(NewPlan(source, data), data, []Finding{wrong}, policy)
		require.NoError(t, err)
		require.Empty(t, p.Operations)
	}
	p, err := ApplyPolicy(NewPlan(source+".other", data), data, findings, policy)
	require.NoError(t, err)
	require.Empty(t, p.Operations)
	missing, err := ParsePolicy(text, base, map[string][]byte{filepath.Join(base, "docs.yaml"): []byte("api.yaml: {}")})
	require.NoError(t, err)
	p, err = ApplyPolicy(NewPlan(source, data), data, findings, missing)
	require.NoError(t, err)
	require.Empty(t, p.Operations)
}
func TestPolicyRejectsAmbiguousAndUnsupportedAssignments(t *testing.T) {
	entry := "  - id: contact\n    rule: info-contact\n    diagnostic: contact-missing\n    scope: {source: api.yaml, pointer: /info}\n    value: {name: Team}\n"
	for name, body := range map[string]string{
		"unknown-field":    "unknown: true\npolicies:\n" + entry,
		"overlap":          "policies:\n" + entry + strings.Replace(entry, "id: contact", "id: second", 1),
		"wildcard":         "policies:\n" + strings.Replace(entry, "api.yaml", "*.yaml", 1),
		"wrong-diagnostic": "policies:\n" + strings.Replace(entry, "contact-missing", "anything", 1),
		"unsupported":      "policies:\n" + strings.Replace(entry, "info-contact", "owasp-rate-limit", 1),
		"stream":           "policies:\n" + entry + "---\nversion: 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParsePolicy([]byte(fmt.Sprint("version: 1\n", body)), t.TempDir(), nil)
			require.Error(t, err)
		})
	}
}
