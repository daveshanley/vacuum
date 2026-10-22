package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

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

// ShouldUseCustomHTTPClient returns true if any TLS-related configuration is provided
func ShouldUseCustomHTTPClient(config HTTPClientConfig) bool {
	return config.CertFile != "" || config.KeyFile != "" || config.CAFile != "" || config.Insecure
}
