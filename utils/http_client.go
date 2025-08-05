// Copyright 2024 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

// HTTPClientConfig holds configuration for creating a custom HTTP client
type HTTPClientConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
	Insecure bool
}

// CreateCustomHTTPClient creates an HTTP client with custom TLS configuration
// for certificate-based authentication and custom CA certificates.
func CreateCustomHTTPClient(config HTTPClientConfig) (*http.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.Insecure,
	}

	// Load client certificate if both cert and key files are provided
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	} else if config.CertFile != "" || config.KeyFile != "" {
		// Only one of cert or key file is provided - this is an error
		return nil, fmt.Errorf("both cert-file and key-file must be provided together")
	}

	// Load custom CA certificate if provided
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate from file: %s", config.CAFile)
		}
		tlsConfig.RootCAs = caCertPool
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// CreateRemoteURLHandler creates a RemoteURLHandler function for use with libopenapi
// that uses the provided HTTP client for all remote requests.
func CreateRemoteURLHandler(client *http.Client) func(url string) (*http.Response, error) {
	return func(url string) (*http.Response, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
		}
		return client.Do(req)
	}
}

// ShouldUseCustomHTTPClient returns true if any TLS-related configuration is provided
func ShouldUseCustomHTTPClient(config HTTPClientConfig) bool {
	return config.CertFile != "" || config.KeyFile != "" || config.CAFile != "" || config.Insecure
}