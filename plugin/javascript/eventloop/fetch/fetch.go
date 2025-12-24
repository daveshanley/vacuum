// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/daveshanley/vacuum/config"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dop251/goja"
)

// EventLoopInterface defines the minimal interface needed from the event loop
type EventLoopInterface interface {
	RegisterCallback() func(func())
	Runtime() *goja.Runtime
}

// FetchModule provides the fetch() function for JavaScript
type FetchModule struct {
	config *FetchConfig
	loop   EventLoopInterface
	vm     *goja.Runtime
}

// NewFetchModule creates a new FetchModule with the given configuration
func NewFetchModule(loop EventLoopInterface, config *FetchConfig) *FetchModule {
	if config == nil {
		config = DefaultFetchConfig()
	}

	// Ensure the HTTPClient has secure transport when private networks are blocked
	config.HTTPClient = config.ensureSecureClient()

	return &FetchModule{
		config: config,
		loop:   loop,
		vm:     loop.Runtime(),
	}
}

// NewFetchModuleFromConfig creates a FetchModule from config.FetchConfig.
// Converts CLI/config system configuration to the internal FetchConfig.
func NewFetchModuleFromConfig(loop EventLoopInterface, cfg *config.FetchConfig) *FetchModule {
	if cfg == nil {
		return NewFetchModule(loop, nil)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	fetchCfg := &FetchConfig{
		AllowInsecure:        cfg.Insecure,
		AllowPrivateNetworks: cfg.AllowPrivateNetworks,
		DefaultTimeout:       timeout,
		MaxResponseSize:      DefaultMaxResponseSize,
		UserAgent:            DefaultUserAgent,
	}

	if utils.ShouldUseCustomHTTPClient(cfg.HTTPClientConfig) {
		if httpClient, err := utils.CreateCustomHTTPClient(cfg.HTTPClientConfig); err == nil {
			fetchCfg.HTTPClient = httpClient
		}
	}

	return NewFetchModule(loop, fetchCfg)
}

// Register registers the fetch function in the JavaScript runtime
func (m *FetchModule) Register() {
	m.vm.Set("fetch", m.fetch)
}

// fetch implements the Web Fetch API fetch() function
// https://developer.mozilla.org/en-US/docs/Web/API/fetch
func (m *FetchModule) fetch(call goja.FunctionCall) goja.Value {
	// Get URL (required first argument)
	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("fetch requires at least 1 argument"))
	}

	url := call.Argument(0).String()

	// Parse options (optional second argument)
	opts := m.parseOptions(call)

	// Validate URL based on config
	if err := m.config.ValidateURL(url); err != nil {
		return m.rejectPromise(err)
	}

	// Create Promise
	promise, resolve, reject := m.vm.NewPromise()

	// Register async callback with event loop
	callback := m.loop.RegisterCallback()

	// Execute request in goroutine
	go func() {
		resp, err := m.executeRequest(url, opts)

		callback(func() {
			if err != nil {
				reject(m.vm.NewTypeError(err.Error()))
				return
			}

			// Create Response object
			response := NewResponse(m.vm, ResponseInit{
				Status:     resp.statusCode,
				StatusText: resp.statusText,
				Headers:    NewHeadersFromHTTP(m.vm, resp.headers),
				URL:        resp.url,
				Body:       resp.body,
				Redirected: resp.redirected,
			})

			// Set Promise constructor for body methods
			response.SetPromiseConstructor(m.vm.Get("Promise"))

			resolve(response.ToGojaObject())
		})
	}()

	return m.vm.ToValue(promise)
}

// fetchOptions contains parsed fetch options
type fetchOptions struct {
	method   string
	headers  http.Header
	body     string
	redirect string // "follow", "error", "manual"
}

// parseOptions parses the optional init object
func (m *FetchModule) parseOptions(call goja.FunctionCall) *fetchOptions {
	opts := &fetchOptions{
		method:   "GET",
		headers:  make(http.Header),
		redirect: "follow",
	}

	if len(call.Arguments) < 2 {
		return opts
	}

	initArg := call.Argument(1)
	if goja.IsUndefined(initArg) || goja.IsNull(initArg) {
		return opts
	}

	initObj := initArg.ToObject(m.vm)

	// Method
	if method := initObj.Get("method"); method != nil && !goja.IsUndefined(method) {
		opts.method = strings.ToUpper(method.String())
	}

	// Headers
	if headers := initObj.Get("headers"); headers != nil && !goja.IsUndefined(headers) {
		headersObj := headers.ToObject(m.vm)
		for _, key := range headersObj.Keys() {
			val := headersObj.Get(key)
			if val != nil && !goja.IsUndefined(val) {
				opts.headers.Set(key, val.String())
			}
		}
	}

	// Body - only accept strings to avoid silent coercion of objects to "[object Object]"
	if body := initObj.Get("body"); body != nil && !goja.IsUndefined(body) && !goja.IsNull(body) {
		exported := body.Export()
		bodyStr, ok := exported.(string)
		if !ok {
			panic(m.vm.NewTypeError("body must be a string"))
		}
		opts.body = bodyStr

		if opts.headers.Get("Content-Type") == "" {
			opts.headers.Set("Content-Type", "text/plain; charset=utf-8")
		}
	}

	// Redirect mode
	if redirect := initObj.Get("redirect"); redirect != nil && !goja.IsUndefined(redirect) {
		mode := redirect.String()
		if mode == "follow" || mode == "error" || mode == "manual" {
			opts.redirect = mode
		}
	}

	return opts
}

// httpResponse contains the result of an HTTP request
type httpResponse struct {
	statusCode int
	statusText string
	headers    http.Header
	body       []byte
	url        string
	redirected bool
}

func (m *FetchModule) executeRequest(url string, opts *fetchOptions) (*httpResponse, error) {
	ctx := context.Background()
	var cancel context.CancelFunc

	// Only apply timeout if configured (0 means no timeout)
	if m.config.DefaultTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, m.config.DefaultTimeout)
		defer cancel()
	}

	var bodyReader io.Reader
	if opts.body != "" {
		bodyReader = strings.NewReader(opts.body)
	}

	req, err := http.NewRequestWithContext(ctx, opts.method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	for key, values := range opts.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if req.Header.Get("User-Agent") == "" && m.config.UserAgent != "" {
		req.Header.Set("User-Agent", m.config.UserAgent)
	}

	client := m.createHTTPClient(opts.redirect)

	var redirected bool
	var finalURL string

	// Wrap redirect handler to:
	// 1. Validate redirect targets against security config (AllowInsecure, AllowedHosts, BlockedHosts)
	// 2. Track whether we actually followed a redirect (for response.redirected)
	originalCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if originalCheckRedirect != nil {
			err := originalCheckRedirect(req, via)
			if err != nil {
				// Not following this redirect (error or manual mode)
				return err
			}
		}

		// Validate redirect target URL against security config
		if err := m.config.ValidateURL(req.URL.String()); err != nil {
			return fmt.Errorf("%w: redirect target not allowed: %v", ErrRedirectNotAllowed, err)
		}

		// We're following the redirect
		if len(via) > 0 {
			redirected = true
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w after %v", ErrFetchTimeout, m.config.DefaultTimeout)
		}
		// Check if the error is redirect-related (redirect: "error" mode or blocked redirect target)
		if errors.Is(err, ErrRedirectNotAllowed) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrNetworkFailure, err)
	}
	defer resp.Body.Close()

	finalURL = resp.Request.URL.String()

	var body []byte
	if m.config.MaxResponseSize > 0 {
		limitedReader := io.LimitReader(resp.Body, m.config.MaxResponseSize+1)
		body, err = io.ReadAll(limitedReader)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrNetworkFailure, err)
		}
		if int64(len(body)) > m.config.MaxResponseSize {
			return nil, ErrResponseTooLarge
		}
	} else {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrNetworkFailure, err)
		}
	}

	return &httpResponse{
		statusCode: resp.StatusCode,
		statusText: ParseStatusTextFromHeader(resp.Status),
		headers:    resp.Header,
		body:       body,
		url:        finalURL,
		redirected: redirected,
	}, nil
}

// createHTTPClient creates a copy of the base HTTP client configured for the redirect mode.
// Each call returns a new client to avoid race conditions in concurrent requests.
func (m *FetchModule) createHTTPClient(redirectMode string) *http.Client {
	client := *m.config.HTTPClient

	switch redirectMode {
	case "error":
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return ErrRedirectNotAllowed
		}
	case "manual":
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		// "follow" uses default behavior (set in executeRequest)
	}

	return &client
}

// rejectPromise creates a rejected Promise with the given error
func (m *FetchModule) rejectPromise(err error) goja.Value {
	promise, _, reject := m.vm.NewPromise()
	_ = reject(m.vm.NewTypeError(err.Error()))
	return m.vm.ToValue(promise)
}
