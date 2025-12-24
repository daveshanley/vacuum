// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package fetch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default timeout for fetch requests
	DefaultTimeout = 30 * time.Second

	// DefaultMaxResponseSize is the default maximum response body size (10MB)
	DefaultMaxResponseSize = 10 * 1024 * 1024

	// DefaultUserAgent is the default User-Agent header
	DefaultUserAgent = "vacuum/1.0"

	// Transport configuration defaults
	defaultDialTimeout         = 30 * time.Second
	defaultKeepAlive           = 30 * time.Second
	defaultMaxIdleConns        = 100
	defaultIdleConnTimeout     = 90 * time.Second
	defaultTLSHandshakeTimeout = 10 * time.Second
	defaultExpectContinue      = 1 * time.Second
)

// FetchConfig contains configuration options for the fetch module.
// By default, fetch is restrictive: HTTPS-only and no private networks.
type FetchConfig struct {
	// HTTPClient is the underlying HTTP client to use. If nil, a default client is created.
	HTTPClient *http.Client

	// DefaultTimeout is the timeout for fetch requests. Defaults to 30 seconds.
	DefaultTimeout time.Duration

	// MaxResponseSize is the maximum allowed response body size in bytes.
	// Defaults to 10MB. Set to 0 for unlimited (not recommended).
	MaxResponseSize int64

	// AllowedHosts is a list of hosts that are allowed to be fetched.
	// If nil or empty, all hosts are allowed (subject to BlockedHosts).
	// If set, only these hosts can be fetched.
	AllowedHosts []string

	// BlockedHosts is a list of hosts that are blocked from being fetched.
	// This is checked after AllowedHosts.
	BlockedHosts []string

	// AllowInsecure allows HTTP (non-HTTPS) requests.
	// Default is false (HTTPS only).
	AllowInsecure bool

	// AllowPrivateNetworks allows requests to private/local network addresses
	// (localhost, 127.0.0.1, 10.x.x.x, 192.168.x.x, 172.16-31.x.x, etc.)
	// Default is false.
	AllowPrivateNetworks bool

	// UserAgent is the User-Agent header to send with requests.
	// Defaults to "vacuum-fetch/1.0".
	UserAgent string
}

// DefaultFetchConfig returns a FetchConfig with secure defaults.
// - HTTPS only (AllowInsecure = false)
// - No private networks (AllowPrivateNetworks = false)
// - 30 second timeout
// - 10MB max response size
func DefaultFetchConfig() *FetchConfig {
	config := &FetchConfig{
		DefaultTimeout:       DefaultTimeout,
		MaxResponseSize:      DefaultMaxResponseSize,
		AllowInsecure:        false,
		AllowPrivateNetworks: false,
		UserAgent:            DefaultUserAgent,
	}
	// Create HTTP client with secure transport that validates resolved IPs
	config.HTTPClient = &http.Client{
		Transport: config.createSecureTransport(),
	}
	return config
}

// ensureSecureClient ensures the HTTPClient has DNS rebinding protection when
// AllowPrivateNetworks is false. This handles these cases:
// 1. HTTPClient is nil - creates a new client with secure transport
// 2. HTTPClient has nil Transport - sets secure transport
// 3. HTTPClient has *http.Transport - wraps its DialContext with security check
// 4. HTTPClient has custom RoundTripper - replaces with secure transport (custom transport incompatible with private network blocking)
func (c *FetchConfig) ensureSecureClient() *http.Client {
	if c.HTTPClient == nil {
		return &http.Client{
			Transport: c.createSecureTransport(),
		}
	}

	// If private networks are allowed, no need to modify the client
	if c.AllowPrivateNetworks {
		return c.HTTPClient
	}

	// Need to ensure dial-time IP validation
	if c.HTTPClient.Transport == nil {
		c.HTTPClient.Transport = c.createSecureTransport()
		return c.HTTPClient
	}

	// Try to wrap existing transport's DialContext
	if transport, ok := c.HTTPClient.Transport.(*http.Transport); ok {
		c.wrapTransportDialContext(transport)
		return c.HTTPClient
	}

	// Custom RoundTripper provided but AllowPrivateNetworks=false
	// We cannot intercept dial-time IP resolution for custom RoundTrippers,
	// so we must replace with secure transport to enforce private network blocking.
	// Callers who need custom RoundTrippers must set AllowPrivateNetworks=true.
	c.HTTPClient.Transport = c.createSecureTransport()
	return c.HTTPClient
}

// wrapTransportDialContext wraps an existing transport's DialContext with
// private network IP validation. If the transport has no DialContext, we set one.
func (c *FetchConfig) wrapTransportDialContext(transport *http.Transport) {
	originalDialContext := transport.DialContext
	dialer := c.newDialer()

	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if err := c.validateResolvedIPs(ctx, addr); err != nil {
			return nil, err
		}

		// Use original DialContext if available, otherwise use default dialer
		if originalDialContext != nil {
			return originalDialContext(ctx, network, addr)
		}
		return dialer.DialContext(ctx, network, addr)
	}
}

// newDialer creates a net.Dialer with configured timeouts.
// Uses config's DefaultTimeout for dial timeout if set, otherwise uses the default.
func (c *FetchConfig) newDialer() *net.Dialer {
	dialTimeout := defaultDialTimeout
	if c.DefaultTimeout > 0 {
		dialTimeout = c.DefaultTimeout
	}
	return &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: defaultKeepAlive,
	}
}

// validateResolvedIPs checks that a host doesn't resolve to private IPs.
func (c *FetchConfig) validateResolvedIPs(ctx context.Context, addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("%w: DNS resolution failed: %v", ErrNetworkFailure, err)
	}

	for _, ip := range ips {
		if isPrivateNetworkIP(ip.IP) {
			return fmt.Errorf("%w: %s resolves to private IP %s", ErrPrivateNetworkNotAllowed, host, ip.IP)
		}
	}
	return nil
}

// createSecureTransport creates an http.Transport with a custom DialContext
// that validates resolved IP addresses against private network restrictions.
// This prevents DNS rebinding attacks where a hostname resolves to a private IP.
func (c *FetchConfig) createSecureTransport() *http.Transport {
	dialer := c.newDialer()

	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if !c.AllowPrivateNetworks {
				if err := c.validateResolvedIPs(ctx, addr); err != nil {
					return nil, err
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          defaultMaxIdleConns,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultExpectContinue,
	}
}

// ValidateURL checks if a URL is allowed based on the configuration.
// Returns nil if allowed, or an appropriate error if not.
func (c *FetchConfig) ValidateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	// Only allow http and https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}

	// Check scheme (HTTPS only by default)
	if !c.AllowInsecure && parsed.Scheme != "https" {
		return ErrInsecureNotAllowed
	}

	host := parsed.Hostname()

	// Check private networks
	if !c.AllowPrivateNetworks && isPrivateNetwork(host) {
		return ErrPrivateNetworkNotAllowed
	}

	// Check allowed hosts
	if len(c.AllowedHosts) > 0 && !isHostInList(host, c.AllowedHosts) {
		return ErrHostNotAllowed
	}

	// Check blocked hosts
	if isHostInList(host, c.BlockedHosts) {
		return ErrHostBlocked
	}

	return nil
}

// isPrivateNetwork checks if a host string is a known private/local network address.
// This handles "localhost" and literal IP addresses. Hostnames that might resolve
// to private IPs are checked at dial time by createSecureTransport.
func isPrivateNetwork(host string) bool {
	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || lowerHost == "localhost." {
		return true
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Hostname - will be checked at dial time when resolved
		return false
	}

	return isPrivateNetworkIP(ip)
}

// isPrivateNetworkIP checks if an IP address is in a private/local network range.
func isPrivateNetworkIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}

	if ip.Equal(net.IPv6loopback) {
		return true
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// IPv6 unique local (fc00::/7)
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}

	return false
}

// isHostInList checks if a host matches any entry in the list.
// Supports exact match and wildcard prefix (*.pb33f.io)
func isHostInList(host string, list []string) bool {
	lowerHost := strings.ToLower(host)
	for _, entry := range list {
		lowerEntry := strings.ToLower(entry)

		// Exact match
		if lowerHost == lowerEntry {
			return true
		}

		// Wildcard match (*.pb33f.io)
		if strings.HasPrefix(lowerEntry, "*.") {
			suffix := lowerEntry[1:] // Remove the *
			if strings.HasSuffix(lowerHost, suffix) {
				return true
			}
		}
	}
	return false
}
