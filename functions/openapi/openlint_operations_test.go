// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestOpenLintPackagedOperations(t *testing.T) {
	tests := []struct {
		name, function, input, document string
		options                         map[string]any
		issues                          []string
	}{
		{name: "OAS3 response range", function: "oasOpSuccessResponse", input: `{"2XX":{}}`, document: `{openapi: 3.1.0}`},
		{name: "Swagger response range", function: "oasOpSuccessResponse", input: `{"2XX":{}, "400":{}}`, document: `{swagger: "2.0"}`, issues: []string{"$|Operation must define at least a single 2xx or 3xx response"}},
		{name: "wrong consumes", function: "oasOpFormDataConsumeCheck", input: `{consumes: [application/json], parameters: [{in: formData}]}`, issues: []string{"$|Consumes must include urlencoded, multipart, or form-data media type when using formData parameter."}},
		{name: "missing consumes is skipped", function: "oasOpFormDataConsumeCheck", input: `{parameters: [{in: formData}]}`},
		{name: "duplicate operation ID", function: "oasOpIdUnique", input: `{/a: {get: {operationId: first}}, /b: {post: {operationId: first}}}`, issues: []string{"$['/b'].post.operationId|operationId must be unique."}},
		{name: "repeated body parameter", function: "oasOpParams", input: `[{in: body, name: a}, {in: body, name: a}, {in: formData, name: a}, {$ref: "#/local"}]`, issues: []string{
			"$[1]|A parameter in this operation already exposes the same combination of \"name\" and \"in\" values.",
			"$|Operation must not have both \"in:body\" and \"in:formData\" parameters.",
			"$[1]|Operation must not have more than a single instance of the \"in:body\" parameter.",
		}},
		{name: "undefined tag", function: "oasTagDefined", input: `{tags: [{name: good}], paths: {/a: {get: {tags: [good, missing]}}}}`, issues: []string{"$.paths['/a'].get.tags[1]|Operation tags must be defined in global tags."}},
		{name: "path parameters", function: "oasPathParam", input: `{"/a/{;id*}": {parameters: [{in: path, name: id, required: true}], get: {parameters: [{in: path, name: typo}]}}, "/a/{other}": {get: {}}}`, issues: []string{
			"$['/a/{;id*}'].get.parameters[0]|Path parameter \"typo\" must have \"required\" property that is set to \"true\".",
			"$['/a/{;id*}'].get.parameters[0]|Parameter \"typo\" must be used in path \"/a/{;id*}\".",
			"$['/a/{other}']|Paths \"/a/{;id*}\" and \"/a/{other}\" must not be equivalent.",
			"$['/a/{other}'].get|Operation must define parameter \"{other}\" as expected by path \"/a/{other}\".",
		}},
		{name: "discriminator", function: "oasDiscriminator", input: `{discriminator: animal, properties: {kind: {}}, required: [kind]}`, issues: []string{
			"$.properties|The discriminator property must be defined in this schema.", "$.required|The discriminator property must be in the required property list.",
		}},
		{name: "OAS2 scopes", function: "oasSecurityDefined", document: `{securityDefinitions: {auth: {scopes: {read: ok}}}}`, input: `{auth: [read, write]}`, options: map[string]any{"oasVersion": 2.0}, issues: []string{"$.auth[1]|\"write\" must be listed among scopes."}},
		{name: "OAS3 scopes", function: "oasSecurityDefined", document: `{components: {securitySchemes: {auth: {flows: {implicit: {scopes: {read: ok}}}}}}}`, input: `{auth: [read, write]}`, options: map[string]any{"oasVersion": 3.0}, issues: []string{"$.auth[1]|\"write\" must be listed among scopes."}},
		{name: "invalid scope array skips validation", function: "oasSecurityDefined", document: `{}`, input: `{auth: "read"}`, options: map[string]any{"oasVersion": 3.0}},
		{name: "missing and unused server variables", function: "serverVariables", input: `{url: "https://{host}:{port}/{resource}", variables: {host: {enum: [localhost], default: example.org}, unused: {}, resource: {}}}`, options: map[string]any{"checkSubstitutions": true, "requireDefault": true}, issues: []string{
			"$.variables.host.default|Server Variable \"host\" has a default not listed in the enum.", "$.variables.unused|Server's \"variables\" object has unused defined \"unused\" url variable.", "$.variables.resource|Server Variable \"resource\" has a missing default.", "$.variables|Not all server's variables are described with \"variables\" object. Missed: port.",
		}},
		{name: "invalid server", function: "serverVariables", input: `{url: 42}`, issues: []string{"$|Invalid Server Object"}},
		{name: "IPv4 parts exceed 255", function: "serverVariables", input: `{url: "https://{host}/api", variables: {host: {default: 999.300.0.1}}}`, options: map[string]any{"checkSubstitutions": true}, issues: []string{"$.variables|A few substitutions of server variables resulted in invalid URLs: https://999.300.0.1/api"}},
		{name: "WHATWG special scheme", function: "serverVariables", input: `{url: "https:{host}", variables: {host: {default: example.com}}}`, options: map[string]any{"checkSubstitutions": true}},
		{name: "invalid server port", function: "serverVariables", input: `{url: "https://example.org:{port}", variables: {port: {enum: ["42", "65536"], default: "42"}}}`, options: map[string]any{"checkSubstitutions": true}, issues: []string{"$.variables|A few substitutions of server variables resulted in invalid URLs: https://example.org:65536"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var doc, input yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.input), &input))
			require.NoError(t, yaml.Unmarshal([]byte(tc.document), &doc))
			fn := OpenLintPackageFunction(tc.function)
			require.NotNil(t, fn)
			results := fn.RunRule([]*yaml.Node{input.Content[0]}, model.RuleFunctionContext{Given: "$", ExecutionRoot: &doc, Contract: model.OpenLintV1Draft, Options: tc.options})
			var issues []string
			for _, r := range results {
				require.NotNil(t, r.StartNode)
				require.True(t, strings.HasPrefix(r.Path, "$"))
				issues = append(issues, r.Path+"|"+r.Message)
			}
			require.ElementsMatch(t, tc.issues, issues)
		})
	}
	require.Nil(t, OpenLintPackageFunction("oasOther"))
}

// OpenLint resolves the requirement selector but looks up schemes in the authored document.
func TestOpenLintPackagedSecurityUsesAuthoredDefinitions(t *testing.T) {
	var resolved, authored, requirement yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`{components: {securitySchemes: {api: {flows: {implicit: {scopes: {read: ok}}}}}}}`), &resolved))
	require.NoError(t, yaml.Unmarshal([]byte(`{components: {securitySchemes: {api: {$ref: "#/components/securitySchemes/other"}}}}`), &authored))
	require.NoError(t, yaml.Unmarshal([]byte(`{api: [read]}`), &requirement))
	ctx := model.RuleFunctionContext{ExecutionRoot: &resolved, AuthoredRoot: &authored, Given: "$.security[0]", Contract: model.OpenLintV1Draft, Options: map[string]any{"oasVersion": 3.0}}
	results := OpenLintPackageFunction("oasSecurityDefined").RunRule(requirement.Content, ctx)
	require.Len(t, results, 1)
	require.Equal(t, "$.security[0].api[0]", results[0].Path)
	require.Equal(t, `"read" must be listed among scopes.`, results[0].Message)
}

// Enum combinations multiply. The substitution check must stop at its cap
// instead of parsing every combination.
func TestOpenLintServerVariableSubstitutionsAreBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{url: "https://{a}{b}{c}{d}{e}.example.com/{f}", variables: {`)
	for i, name := range []string{"a", "b", "c", "d", "e", "f"} {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(name + ": {enum: [")
		for v := 0; v < 20; v++ {
			if v > 0 {
				b.WriteString(", ")
			}
			b.WriteString(`"v` + string(rune('a'+v)) + `"`)
		}
		b.WriteString("]}")
	}
	b.WriteString("}}")
	var input yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(b.String()), &input))
	started := time.Now()
	results := OpenLintPackageFunction("serverVariables").RunRule(input.Content, model.RuleFunctionContext{Given: "$", ExecutionRoot: &yaml.Node{}, Contract: model.OpenLintV1Draft, Options: map[string]any{"checkSubstitutions": true}})
	require.Len(t, results, 1)
	require.Equal(t, "Server URL validation is incomplete after 4096 substitutions.", results[0].Message)
	require.Less(t, time.Since(started), 5*time.Second, "64,000,000 combinations must not all be parsed")
}

// Expected values come from Node's new URL(), the parser OpenLint uses.
func TestPackagedServerURLMatchesWHATWG(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		valid bool
	}{
		{"https://example.com/api", true},
		{"https:example.com", true},
		{"https:\\\\example.com", true},
		{"HTTPS://EXAMPLE.com", true},
		{"https://999.300.0.1/api", false},
		{"https://1.2.3", true},
		{"https://0x7f.1", true},
		{"https://1.2.3.4.5", false},
		{"https://a.09", false},
		{"https://4294967296", false},
		{"https://example.org:65535", true},
		{"https://example.org:65536", false},
		{"https://example.org:8a", false},
		{"https://[::1]:8080", true},
		{"https://[::1", false},
		{"https://[fe80::1%25eth0]", false},
		{"https://ex%41mple.com", true},
		{"https://ex%2Fample.com", false},
		{"https://ex ample.com", false},
		{"https://ex^ample.com", false},
		{"https://bücher.de", true},
		{"https://é_b", true},
		{"https://xn--zz", false},
		{"https://user:pass@example.com", true},
		{"https://", false},
		{"https://:80", false},
		{"file:///etc/hosts", true},
		{"file://C:/x", true},
		{"file://example.com:80", false},
		{"custom:anything at all", true},
		{"custom://:", false},
		{"custom://user@", false},
		{"custom://host:99999", false},
		{"mailto:dev@example.com", true},
		{"1http://example.com", false},
		{"relative/path", false},
	} {
		require.Equal(t, tc.valid, validPackagedServerURL(tc.raw), tc.raw)
	}
}
