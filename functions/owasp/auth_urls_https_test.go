package owasp

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestAuthURLsHTTPS_AttachesFindingsToEndpointOwner(t *testing.T) {
	spec := `openapi: 3.1.0
info: {title: Test, version: '1'}
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: http://example.com/authorize
          tokenUrl: http://example.com/token
          refreshUrl: http://example.com/refresh
          scopes: {}
    discovery:
      type: openIdConnect
      openIdConnectUrl: http://example.com/discovery
    secure:
      type: openIdConnect
      openIdConnectUrl: https://example.com/discovery
`
	document, err := libopenapi.NewDocument([]byte(spec))
	require.NoError(t, err)
	m, err := document.BuildV3Model()
	require.NoError(t, err)
	doc := drModel.NewDrDocument(m)
	ctx := model.RuleFunctionContext{
		Document:   document,
		DrDocument: doc,
		Rule:       &model.Rule{Id: "transport"},
	}
	results := (AuthURLsHTTPS{}).RunRule(nil, ctx)
	require.Len(t, results, 4)
	schemes := doc.V3Document.Components.SecuritySchemes
	flow := schemes.GetOrZero("oauth").Flows.AuthorizationCode
	assert.Len(t, flow.GetRuleFunctionResults(), 3)
	assert.Empty(t, schemes.GetOrZero("oauth").GetRuleFunctionResults())
	assert.Len(t, schemes.GetOrZero("discovery").GetRuleFunctionResults(), 1)
	assert.Empty(t, schemes.GetOrZero("secure").GetRuleFunctionResults())
	for _, finding := range flow.GetRuleFunctionResults() {
		assert.Contains(t, finding.Path, ".flows.authorizationCode.")
	}
}
