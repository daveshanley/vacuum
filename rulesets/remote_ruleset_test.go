// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	ctx "context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCheckForRemoteExtends_Fail(t *testing.T) {
	extends := make(map[string]string)
	extends["bing"] = "bong"
	assert.False(t, CheckForRemoteExtends(extends))
}

func TestCheckForRemoteExtends_Success(t *testing.T) {
	extends := make(map[string]string)
	extends["http://quobix.com"] = "bong"
	assert.True(t, CheckForRemoteExtends(extends))
}

func TestDownloadRemoteRuleSet(t *testing.T) {

	mockRemote := func() *httptest.Server {
		bs, _ := os.ReadFile("examples/custom-ruleset.yaml")
		return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write(bs)
		}))
	}

	server := mockRemote()
	defer server.Close()

	rs, err := DownloadRemoteRuleSet(ctx.Background(), server.URL)

	assert.NoError(t, err)
	assert.NotNil(t, rs)
	assert.NotNil(t, rs.RuleDefinitions["check-title-is-exactly-this"])
}

func TestLoadLocalCompositeRuleSet(t *testing.T) {

	rs, err := LoadLocalRuleSet(ctx.Background(), "examples/composite-ruleset.yaml")

	assert.NoError(t, err)
	assert.Len(t, rs.RuleDefinitions, 2)
	assert.Len(t, rs.Rules, 2)
	assert.Contains(t, rs.Rules, "rule-from-subset-1")
	assert.Contains(t, rs.Rules, "rule-from-subset-2")
	assert.Contains(t, rs.RuleDefinitions, "rule-from-subset-1")
	assert.Contains(t, rs.RuleDefinitions, "rule-from-subset-2")
}
