// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestHostsHttps_RunRule(t *testing.T) {

	yml := `openapi: "3.1.0"
servers:
  - url: http://api.pb33f.io
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "hosts_https", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := HostsHttps{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "server URLs should use TLS (https)", res[0].Message)
	assert.Equal(t, "$.servers[0].url", res[0].Path)
}

func TestHostsHttps_RunRule_Pass(t *testing.T) {

	yml := `openapi: "3.1.0"
servers:
  - url: https://api.pb33f.io
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "hosts_https", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := HostsHttps{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestHostsHttps_ServerLocationsAndVariables(t *testing.T) {
	for _, tc := range []struct{ name, servers, path string }{
		{"root", "servers: [{url: 'http://example.com'}]", "$.servers[0].url"},
		{"path", "servers: [{url: 'https://example.com'}]\npaths:\n  /items:\n    servers: [{url: 'http://example.com'}]", "$.paths['/items'].servers[0].url"},
		{"operation", "servers: [{url: 'https://example.com'}]\npaths:\n  /items:\n    get:\n      servers: [{url: 'http://example.com'}]\n      responses: {'200': {description: OK}}", "$.paths['/items'].get.servers[0].url"},
		{"webhook path", "webhooks:\n  event:\n    servers: [{url: 'http://example.com'}]", "$.webhooks['event'].servers[0].url"},
		{"webhook operation", "webhooks:\n  event:\n    post:\n      servers: [{url: 'http://example.com'}]", "$.webhooks['event'].post.servers[0].url"},
		{"callback operation", "paths:\n  /items:\n    post:\n      callbacks:\n        event:\n          '{$request.body#/callbackUrl}':\n            post:\n              servers: [{url: 'http://example.com'}]", "$.paths['/items'].post.callbacks['event']['{$request.body#/callbackUrl}'].post.servers[0].url"},
		{"relative", "servers: [{url: '/v1'}, {url: '//example.com/v1'}]", ""},
		{"https prefix is not scheme", "servers: [{url: 'https-insecure://example.com'}]", "$.servers[0].url"},
		{"scheme default", "servers:\n  - url: '{scheme}://example.com'\n    variables:\n      scheme: {default: http}", "$.servers[0].url"},
		{"scheme enum", "servers:\n  - url: '{scheme}://{host}'\n    variables:\n      scheme: {default: https, enum: [https, http]}\n      host: {default: example.com}", "$.servers[0].url"},
		{"host variable", "servers:\n  - url: 'https://{host}'\n    variables:\n      host: {default: example.com, enum: [example.com, api.example.com]}", ""},
		{"whole URL variable", "servers:\n  - url: '{endpoint}'\n    variables:\n      endpoint: {default: 'https://example.com', enum: ['http://example.com', 'https://example.com']}", "$.servers[0].url"},
		{"missing servers", "paths: {}", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := libopenapi.NewDocument([]byte("openapi: 3.1.0\n" + tc.servers))
			require.NoError(t, err)
			m, err := document.BuildV3Model()
			require.NoError(t, err)
			rule := buildOpenApiTestRuleAction("$", "owaspHostsHttps", "", nil)
			rule.Message = "custom transport message"
			ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
			ctx.Document, ctx.DrDocument, ctx.Rule = document, drModel.NewDrDocument(m), &rule
			res := (HostsHttps{}).RunRule(nil, ctx)
			if tc.path == "" {
				require.Empty(t, res)
				return
			}
			require.Len(t, res, 1)
			assert.Equal(t, tc.path, res[0].Path)
			assert.Equal(t, rule.Message, res[0].Message)
			require.NotNil(t, res[0].StartNode)
			assert.Positive(t, res[0].StartNode.Line)
		})
	}
}
