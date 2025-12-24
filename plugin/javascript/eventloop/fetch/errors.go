// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package fetch provides a Web Fetch API compliant implementation for use
// with Goja JavaScript runtime and the vacuum event loop.
package fetch

import "errors"

var (
	// ErrFetchTimeout is returned when a fetch request exceeds the configured timeout
	ErrFetchTimeout = errors.New("fetch request timed out")

	// ErrNetworkFailure is returned for network-level errors (DNS, connection refused, etc.)
	ErrNetworkFailure = errors.New("network error")

	// ErrBodyAlreadyUsed is returned when attempting to read a Response body that has already been consumed
	ErrBodyAlreadyUsed = errors.New("body has already been consumed")

	// ErrInvalidURL is returned when the provided URL cannot be parsed
	ErrInvalidURL = errors.New("invalid URL")

	// ErrInsecureNotAllowed is returned when attempting HTTP (non-HTTPS) and AllowInsecure is false
	ErrInsecureNotAllowed = errors.New("insecure HTTP not allowed")

	// ErrPrivateNetworkNotAllowed is returned when attempting to access private/local networks
	// and AllowPrivateNetworks is false
	ErrPrivateNetworkNotAllowed = errors.New("private network access not allowed")

	// ErrHostBlocked is returned when the target host is in the BlockedHosts list
	ErrHostBlocked = errors.New("host is blocked")

	// ErrHostNotAllowed is returned when AllowedHosts is set and the target host is not in the list
	ErrHostNotAllowed = errors.New("host not in allowed list")

	// ErrResponseTooLarge is returned when the response body exceeds MaxResponseSize
	ErrResponseTooLarge = errors.New("response body exceeds maximum allowed size")

	// ErrRedirectNotAllowed is returned when redirect mode is "error" and a redirect is encountered
	ErrRedirectNotAllowed = errors.New("redirects not allowed")
)
