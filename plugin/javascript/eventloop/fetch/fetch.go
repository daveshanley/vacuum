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

// EventLoop defines the minimal interface needed from the event loop
type EventLoop interface {
	RegisterCallback() func(func())
	Runtime() *goja.Runtime
}

// Fetch provides the fetch() function for JavaScript
type Fetch struct {
	config *FetchConfig
	loop   EventLoop
	vm     *goja.Runtime
}

// NewFetchModule creates a new Fetch pointer with the given configuration
func NewFetchModule(loop EventLoop, config *FetchConfig) *Fetch {
	if config == nil {
		config = DefaultFetchConfig()
	}

	// Ensure the HTTPClient has secure transport when private networks are blocked
	config.HTTPClient = config.ensureSecureClient()

	return &Fetch{
		config: config,
		loop:   loop,
		vm:     loop.Runtime(),
	}
}

// NewFetchModuleFromConfig creates a Fetch from config.FetchConfig.
// Converts CLI/config system configuration to the internal FetchConfig.
// Returns an error if TLS certificate configuration is invalid.
func NewFetchModuleFromConfig(loop EventLoop, cfg *config.FetchConfig) (*Fetch, error) {
	if cfg == nil {
		return NewFetchModule(loop, nil), nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	fetchCfg := &FetchConfig{
		AllowInsecure:        cfg.AllowHTTP, // Use dedicated flag for HTTP allowance (not TLS skip)
		AllowPrivateNetworks: cfg.AllowPrivateNetworks,
		DefaultTimeout:       timeout,
		MaxResponseSize:      DefaultMaxResponseSize,
		UserAgent:            DefaultUserAgent,
	}

	if utils.ShouldUseCustomHTTPClient(cfg.HTTPClientConfig) {
		httpClient, err := utils.CreateCustomHTTPClient(cfg.HTTPClientConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for fetch(): %w", err)
		}
		fetchCfg.HTTPClient = httpClient
	}

	return NewFetchModule(loop, fetchCfg), nil
}

// Register registers the fetch function in the JavaScript runtime
func (m *Fetch) Register() {
	m.vm.Set("fetch", m.fetch)
}

// fetch implements the Web Fetch API fetch() function
// https://developer.mozilla.org/en-US/docs/Web/API/fetch
func (m *Fetch) fetch(call goja.FunctionCall) goja.Value {

	if len(call.Arguments) < 1 {
		panic(m.vm.NewTypeError("fetch requires at least 1 argument"))
	}

	url := call.Argument(0).String()

	opts := m.parseOptions(call)

	if err := m.config.ValidateURL(url); err != nil {
		return m.rejectPromise(err)
	}

	promise, resolve, reject := m.vm.NewPromise()

	callback := m.loop.RegisterCallback()

	go func() {
		resp, err := m.executeRequest(url, opts)

		callback(func() {
			if err != nil {
				_ = reject(m.vm.NewTypeError(err.Error()))
				return
			}

			response := NewResponse(m.vm, ResponseInit{
				Status:     resp.statusCode,
				StatusText: resp.statusText,
				Headers:    NewHeadersFromHTTP(m.vm, resp.headers),
				URL:        resp.url,
				Body:       resp.body,
				Redirected: resp.redirected,
			})

			// promise constructor for body methods
			response.SetPromiseConstructor(m.vm.Get("Promise"))

			_ = resolve(response.ToGojaObject())
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
func (m *Fetch) parseOptions(call goja.FunctionCall) *fetchOptions {
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

	if method := initObj.Get("method"); method != nil && !goja.IsUndefined(method) {
		opts.method = strings.ToUpper(method.String())
	}

	if headers := initObj.Get("headers"); headers != nil && !goja.IsUndefined(headers) {
		headersObj := headers.ToObject(m.vm)
		for _, key := range headersObj.Keys() {
			val := headersObj.Get(key)
			if val != nil && !goja.IsUndefined(val) {
				opts.headers.Set(key, val.String())
			}
		}
	}

	// only accept strings to avoid silent coercion of objects to "[object Object]"
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

func (m *Fetch) executeRequest(url string, opts *fetchOptions) (*httpResponse, error) {
	ctx := context.Background()
	var cancel context.CancelFunc

	// only apply timeout if configured (0 means no timeout)
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

	// wrap the redirect handler to:
	// - validate redirect targets against security config (AllowInsecure, AllowedHosts, BlockedHosts)
	// - track whether we actually followed a redirect (for response.redirected)
	originalCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if originalCheckRedirect != nil {
			err := originalCheckRedirect(req, via)
			if err != nil {
				// Not following this redirect (error or manual mode)
				return err
			}
		}

		// validate redirect target URL against security config
		if err := m.config.ValidateURL(req.URL.String()); err != nil {
			return fmt.Errorf("%w: redirect target not allowed: %v", ErrRedirectNotAllowed, err)
		}

		// following the redirect
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
		// check if the error is redirect-related (redirect: "error" mode or blocked redirect target)
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
func (m *Fetch) createHTTPClient(redirectMode string) *http.Client {
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
func (m *Fetch) rejectPromise(err error) goja.Value {
	promise, _, reject := m.vm.NewPromise()
	_ = reject(m.vm.NewTypeError(err.Error()))
	return m.vm.ToValue(promise)
}
