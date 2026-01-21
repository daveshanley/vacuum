// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

// Package config provides shared configuration types that are used across
// multiple packages. This package should have no dependencies on other
// vacuum packages to avoid import cycles.
package config

import "time"

// HTTPClientConfig holds configuration for creating a custom HTTP client.
// This is used for TLS/certificate authentication with remote URLs.
type HTTPClientConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
	Insecure bool
}

// FetchConfig contains configuration for JavaScript fetch() requests.
// Defined in config package to avoid import cycles between model and plugin packages.
type FetchConfig struct {
	HTTPClientConfig                  // TLS configuration reused for fetch() requests
	AllowPrivateNetworks bool         // Allow localhost, 10.x, 192.168.x
	AllowHTTP            bool         // Allow HTTP (non-HTTPS) requests (separate from TLS skip)
	Timeout              time.Duration // Request timeout (default 30s)
}

// DefaultFetchConfig returns secure defaults.
func DefaultFetchConfig() *FetchConfig {
	return &FetchConfig{
		Timeout: 30 * time.Second,
	}
}
