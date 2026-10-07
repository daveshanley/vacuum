// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package utils

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/testify/require"
)

type contextTestTransport func(*http.Request) (*http.Response, error)

func (f contextTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRemoteURLHandlerWithContextCancelsRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	client := &http.Client{Transport: contextTestTransport(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "GET", request.Method)
		require.Equal(t, "https://example.test/schema.yaml", request.URL.String())
		close(started)
		<-request.Context().Done()
		return nil, request.Context().Err()
	})}
	handler := CreateRemoteURLHandlerWithContext(ctx, client)
	done := make(chan error, 1)
	go func() {
		_, err := handler("https://example.test/schema.yaml")
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("request ignored cancellation")
	}
}

func TestRemoteURLHandlerBackgroundCompatibility(t *testing.T) {
	client := &http.Client{Transport: contextTestTransport(func(request *http.Request) (*http.Response, error) {
		require.NoError(t, request.Context().Err())
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("type: string"))}, nil
	})}
	for _, handler := range []func(string) (*http.Response, error){
		CreateRemoteURLHandler(client),
		CreateRemoteURLHandlerWithContext(nil, client),
	} {
		response, err := handler("https://example.test/schema.yaml")
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, "type: string", string(body))
		_, err = handler(":invalid")
		require.ErrorContains(t, err, "failed to create request")
	}
}
