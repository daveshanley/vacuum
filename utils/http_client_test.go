// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestCreateRemoteURLHandler_SetsUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "vacuum-linter/1.0", request.UserAgent())
	}))
	defer server.Close()

	response, err := CreateRemoteURLHandler(http.DefaultClient)(server.URL)
	require.NoError(t, err)
	defer response.Body.Close()
}
