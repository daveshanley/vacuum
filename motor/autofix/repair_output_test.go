// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/motor"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func outputReview(t *testing.T, source, filename string, options *motor.ExecutionOptions) (*FixReview, []FixChoice) {
	t.Helper()
	execution := &motor.RuleSetExecution{Spec: []byte(source), SpecFileName: filename, Base: filepath.Dir(filename), AllowLookup: true, SilenceLogs: true,
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
	review, err := NewFixReview(execution, options)
	require.NoError(t, err)
	t.Cleanup(review.Close)
	var choices []FixChoice
	for _, option := range review.Options() {
		if option.Available && option.Result.RuleId == "info-description" {
			choices = append(choices, FixChoice{Index: option.Index})
		}
	}
	require.Len(t, choices, 1)
	return review, choices
}

const outputSpec = `{"openapi":"3.1.0","info":{"title":"Output","version":"1"},"paths":{},"components":{"schemas":{"Local":{"type":"string"},"LocalUse":{"$ref":"#/components/schemas/Local"},"Remote":{"$ref":"pet.yaml#/Pet"}}}}`

func TestFixPreviewOutputBundlesFrozenSources(t *testing.T) {
	dir := t.TempDir()
	dependency := filepath.Join(dir, "pet.yaml")
	require.NoError(t, os.WriteFile(dependency, []byte("Pet:\n  type: object\n  properties:\n    id:\n      $ref: './id.yaml#/Id'\n"), 0600))
	nested := filepath.Join(dir, "id.yaml")
	require.NoError(t, os.WriteFile(nested, []byte("Id:\n  type: string\n"), 0600))
	review, choices := outputReview(t, outputSpec, filepath.Join(dir, "api.json"), nil)
	require.NoError(t, os.Remove(dependency))
	require.NoError(t, os.Remove(nested))
	for _, mode := range []string{"inline", "composed"} {
		t.Run(mode, func(t *testing.T) {
			preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: mode, OutputPath: filepath.Join(t.TempDir(), "copy.json")})
			require.NoError(t, err)
			defer preview.Close()
			require.True(t, json.Valid(preview.Candidate), string(preview.Candidate))
			require.NotContains(t, string(preview.Candidate), "pet.yaml")
			require.Contains(t, string(preview.Candidate), "#/components/schemas/Local")
			require.Equal(t, 1, preview.Fixed)
			require.Empty(t, preview.Results)
			require.NotEqual(t, RepairImpactUnknown, preview.Impact.Status, preview.Impact.Reason)
			root, err := repair.ParseDocument(preview.Candidate)
			require.NoError(t, err)
			require.NotEmpty(t, fixScalar(fixNodeAt(root, "/info"), "description"))
		})
	}
	// Both bundles must leave the baseline and its dependency snapshots intact.
	preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: "keep", OutputPath: filepath.Join(dir, "copy.json")})
	require.NoError(t, err)
	defer preview.Close()
	require.Contains(t, string(preview.Candidate), "pet.yaml")
	require.Equal(t, outputSpec, string(review.execution.Spec))
	_, err = review.PreviewOutput(choices, FixOutputOptions{OutputPath: filepath.Join(t.TempDir(), "copy.json")})
	require.ErrorContains(t, err, "source directory")
}

func TestFixPreviewOutputFormats(t *testing.T) {
	source := `{"openapi":"3.1.0","info":{"title":"Output","version":"1"},"paths":{},"x-values":[9007199254740993,0.12345678901234567890123456789]}`
	for _, mode := range []string{"keep", "inline", "composed"} {
		for _, format := range []string{"", "json", "yaml"} {
			t.Run(mode+"/"+format, func(t *testing.T) {
				review, choices := outputReview(t, source, filepath.Join(t.TempDir(), "api.json"), nil)
				preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: mode, Format: format})
				require.NoError(t, err)
				defer preview.Close()
				require.Equal(t, format != "yaml", json.Valid(preview.Candidate))
				require.Contains(t, string(preview.Candidate), "9007199254740993")
				require.Contains(t, string(preview.Candidate), "0.12345678901234567890123456789")
			})
		}
	}
	yamlSource := "openapi: 3.1.0\ninfo:\n  title: Output\n  version: '1'\npaths: {}\nx-value: 0.12345678901234567890123456789\n"
	review, choices := outputReview(t, yamlSource, filepath.Join(t.TempDir(), "api.yaml"), nil)
	preview, err := review.PreviewOutput(choices, FixOutputOptions{Format: "json"})
	require.NoError(t, err)
	defer preview.Close()
	require.True(t, json.Valid(preview.Candidate))
	require.Contains(t, string(preview.Candidate), "0.12345678901234567890123456789")
}

func TestFixPreviewOutputRejectsOptionsAndCancellation(t *testing.T) {
	source := strings.Replace(outputSpec, `,"Remote":{"$ref":"pet.yaml#/Pet"}`, "", 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	review, choices := outputReview(t, source, filepath.Join(t.TempDir(), "api.json"), &motor.ExecutionOptions{Context: ctx})
	for _, opts := range []FixOutputOptions{{BundleMode: "unknown"}, {Format: "xml"}} {
		preview, err := review.PreviewOutput(choices, opts)
		require.ErrorContains(t, err, "unsupported")
		require.Nil(t, preview)
	}
	preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: "composed"})
	require.NoError(t, err)
	preview.Close()
	cancel()
	preview, err = review.PreviewOutput(choices, FixOutputOptions{BundleMode: "inline"})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, preview)
	review.Close()
	_, err = review.PreviewOutput(choices, FixOutputOptions{})
	require.ErrorContains(t, err, "closed")
}

func TestFixPreviewOutputRejectsUnsupportedBundleDocument(t *testing.T) {
	for _, source := range []string{
		`{"swagger":"2.0","info":{"title":"Output","version":"1"},"paths":{}}`,
		`{"asyncapi":"3.0.0","info":{"title":"Output","version":"1"},"channels":{}}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`,
	} {
		for _, mode := range []string{"inline", "composed"} {
			_, err := bundleRepairOutput([]byte(source), nil, nil, nil, mode)
			require.ErrorContains(t, err, "OpenAPI 3")
		}
	}
}

func TestRepairOutputRelocationChecksFinalReferences(t *testing.T) {
	for _, field := range []string{"$ref: './model.yaml'", "$dynamicRef: './model.yaml#anchor'", "$recursiveRef: './model.yaml#'", "$id: './schema.json'", "$self: './api.json'", "$schema: './dialect.json'", "externalValue: './example.json'", "sourceDescriptions: [{name: api, url: './api.yaml', type: openapi}]", "discriminator: {propertyName: type, mapping: {pet: './pet.yaml'}}", "servers: [{url: './api'}]", "server: {url: './api'}", "authorizationUrl: './authorize'", "tokenUrl: './token'", "refreshUrl: './refresh'", "operationRef: './api.yaml#/paths/~1pets/get'", "externalDocs: {url: './guide'}", "info: {termsOfService: './terms'}"} {
		t.Run(field, func(t *testing.T) {
			require.ErrorContains(t, validateRepairOutputLocation("/source/api.yaml", "/copy/api.yaml", []byte(field)), "relative references")
			require.NoError(t, validateRepairOutputLocation("/source/api.yaml", "/source/copy.yaml", []byte(field)))
		})
	}
	for _, field := range []string{"$ref: '#/components/schemas/Pet'", "$ref: 'https://example.com/pet.yaml'", "discriminator: {propertyName: type, mapping: {pet: Pet}}"} {
		require.NoError(t, validateRepairOutputLocation("/source/api.yaml", "/copy/api.yaml", []byte(field)), field)
	}
}

func TestFixPreviewOutputRejectsLocationDependentAPIURLs(t *testing.T) {
	for _, suffix := range []string{
		`,"servers":[{"url":"./api"}]`,
		`,"components":{"links":{"Next":{"operationRef":"./api.yaml#/paths/~1pets/get"}}}`,
	} {
		source := `{"openapi":"3.1.0","info":{"title":"Output","version":"1"},"paths":{}` + suffix + `}`
		review, choices := outputReview(t, source, filepath.Join(t.TempDir(), "api.json"), nil)
		for _, mode := range []string{"keep", "inline", "composed"} {
			preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: mode, OutputPath: filepath.Join(t.TempDir(), "copy.json")})
			require.Nil(t, preview)
			require.ErrorContains(t, err, "source directory")
		}
	}
}

func TestFixPreviewOutputRetainsVerificationGates(t *testing.T) {
	for _, mode := range []string{"keep", "inline", "composed"} {
		t.Run(mode, func(t *testing.T) {
			review, choices := outputReview(t, integritySpec, filepath.Join(t.TempDir(), "api.json"), nil)
			plan, err := review.planChoices(choices)
			require.NoError(t, err)
			plan.Operations = append(plan.Operations, repair.Operation{Op: "replace", Path: "/paths/~1b/get/operationId", Value: json.RawMessage(`"a"`)})
			verified, err := validatePreparedPlanOutput(review.execution, plan, nil, review.baseline, FixOutputOptions{BundleMode: mode, Format: "yaml"}, nil)
			require.Nil(t, verified)
			require.ErrorContains(t, err, "duplicate operationId")
			plan, err = review.planChoices(choices)
			require.NoError(t, err)
			plan.Operations = nil
			verified, err = validatePreparedPlanOutput(review.execution, plan, nil, review.baseline, FixOutputOptions{BundleMode: mode}, nil)
			require.Nil(t, verified)
			require.ErrorContains(t, err, "target remains")
			// A failed preview does not consume the user's selected proposal.
			preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: mode})
			require.NoError(t, err)
			preview.Close()
		})
	}
}

func TestFixPreviewOutputRejectsIncompleteBundle(t *testing.T) {
	review, choices := outputReview(t, integritySpec, filepath.Join(t.TempDir(), "api.json"), nil)
	for _, mode := range []string{"inline", "composed"} {
		plan, err := review.planChoices(choices)
		require.NoError(t, err)
		plan.Operations = append(plan.Operations, repair.Operation{Op: "add", Path: "/components/schemas/Item/$ref", Value: json.RawMessage(`"#/components/schemas/Missing"`)})
		verified, err := validatePreparedPlanOutput(review.execution, plan, nil, review.baseline, FixOutputOptions{BundleMode: mode}, nil)
		require.Nil(t, verified)
		require.ErrorContains(t, err, "incomplete document")
	}
}

func TestFixPreviewOutputRejectsStaleDependencySnapshot(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pet.yaml"), []byte("Pet: {type: string}"), 0600))
	review, choices := outputReview(t, outputSpec, filepath.Join(dir, "api.json"), nil)
	for name := range review.proposal.Sources {
		review.proposal.Sources[name] = "stale"
	}
	preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: "composed"})
	require.Nil(t, preview)
	require.ErrorContains(t, err, "dependency snapshots changed")
}

func TestFixPreviewOutputBundlesFrozenRemoteSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Pet: {type: string}"))
	}))
	defer server.Close()
	source := strings.Replace(outputSpec, "pet.yaml#/Pet", server.URL+"/pet.yaml#/Pet", 1)
	review, choices := outputReview(t, source, filepath.Join(t.TempDir(), "api.json"), nil)
	server.Close()
	for _, mode := range []string{"inline", "composed"} {
		preview, err := review.PreviewOutput(choices, FixOutputOptions{BundleMode: mode, OutputPath: filepath.Join(t.TempDir(), "copy.json")})
		require.NoError(t, err)
		require.NotContains(t, string(preview.Candidate), server.URL)
		preview.Close()
	}
}
