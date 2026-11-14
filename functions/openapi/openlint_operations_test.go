// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"strings"
	"testing"

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
		{name: "invalid server port", function: "serverVariables", input: `{url: "https://example.org:{port}", variables: {port: {enum: ["42", "65536"], default: "42"}}}`, options: map[string]any{"checkSubstitutions": true}, issues: []string{"$.variables|A few substitutions of server variables resulted in invalid URLs: https://example.org:65536"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var doc, input yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.input), &input))
			require.NoError(t, yaml.Unmarshal([]byte(tc.document), &doc))
			fn := OpenLintPackageOperation(tc.function)
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
	require.Nil(t, OpenLintPackageOperation("oasOther"))
}
