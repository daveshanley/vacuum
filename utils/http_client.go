// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import (
	"context"
	"fmt"
	"net/http"

	"github.com/daveshanley/vacuum/config"
)

// HTTPClientConfig is an alias for config.HTTPClientConfig for backwards compatibility.
type HTTPClientConfig = config.HTTPClientConfig

// FetchConfig is an alias for config.FetchConfig for backwards compatibility.
type FetchConfig = config.FetchConfig

// CreateCustomHTTPClient creates a client using the shared TLS configuration.
func CreateCustomHTTPClient(cfg HTTPClientConfig) (*http.Client, error) {
	return config.CreateCustomHTTPClient(cfg)
}

// CreateRemoteURLHandler creates a RemoteURLHandler function for use with libopenapi
// that uses the provided HTTP client for all remote requests.
func CreateRemoteURLHandler(client *http.Client) func(url string) (*http.Response, error) {
	return CreateRemoteURLHandlerWithContext(context.Background(), client)
}

// CreateRemoteURLHandlerWithContext binds remote requests to the supplied context.
// Cancellation also interrupts response body reads. A nil context uses context.Background.
func CreateRemoteURLHandlerWithContext(ctx context.Context, client *http.Client) func(url string) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return func(url string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
		}
		return client.Do(req)
	}
}

// ShouldUseCustomHTTPClient reports whether custom TLS settings are present.
func ShouldUseCustomHTTPClient(cfg HTTPClientConfig) bool {
	return config.ShouldUseCustomHTTPClient(cfg)
}

// CreateHTTPClientIfNeeded creates a custom HTTP client only if TLS configuration is provided.
// Returns nil (no error) if no custom configuration is needed, allowing use of default client.
func CreateHTTPClientIfNeeded(config HTTPClientConfig) (*http.Client, error) {
	if !ShouldUseCustomHTTPClient(config) {
		return nil, nil
	}
	return CreateCustomHTTPClient(config)
}
