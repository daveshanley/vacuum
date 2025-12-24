// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package fetch

import (
	"testing"
	"time"

	"github.com/daveshanley/vacuum/config"
	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFetchConfig(t *testing.T) {
	cfg := DefaultFetchConfig()

	assert.NotNil(t, cfg.HTTPClient)
	assert.Equal(t, DefaultTimeout, cfg.DefaultTimeout)
	assert.Equal(t, int64(DefaultMaxResponseSize), cfg.MaxResponseSize)
	assert.False(t, cfg.AllowInsecure)
	assert.False(t, cfg.AllowPrivateNetworks)
	assert.Equal(t, DefaultUserAgent, cfg.UserAgent)
}

func TestFetchConfig_ValidateURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *FetchConfig
		url      string
		wantErr  error
		wantPass bool
	}{
		{
			name:     "HTTPS URL with default config",
			config:   DefaultFetchConfig(),
			url:      "https://example.com/api",
			wantPass: true,
		},
		{
			name:     "HTTP URL with default config (blocked)",
			config:   DefaultFetchConfig(),
			url:      "http://example.com/api",
			wantErr:  ErrInsecureNotAllowed,
			wantPass: false,
		},
		{
			name: "HTTP URL with AllowInsecure",
			config: func() *FetchConfig {
				c := DefaultFetchConfig()
				c.AllowInsecure = true
				return c
			}(),
			url:      "http://example.com/api",
			wantPass: true,
		},
		{
			name:     "localhost with default config (blocked)",
			config:   DefaultFetchConfig(),
			url:      "https://localhost/api",
			wantErr:  ErrPrivateNetworkNotAllowed,
			wantPass: false,
		},
		{
			name:     "127.0.0.1 with default config (blocked)",
			config:   DefaultFetchConfig(),
			url:      "https://127.0.0.1/api",
			wantErr:  ErrPrivateNetworkNotAllowed,
			wantPass: false,
		},
		{
			name:     "10.x.x.x with default config (blocked)",
			config:   DefaultFetchConfig(),
			url:      "https://10.0.0.1/api",
			wantErr:  ErrPrivateNetworkNotAllowed,
			wantPass: false,
		},
		{
			name:     "192.168.x.x with default config (blocked)",
			config:   DefaultFetchConfig(),
			url:      "https://192.168.1.1/api",
			wantErr:  ErrPrivateNetworkNotAllowed,
			wantPass: false,
		},
		{
			name:     "172.16.x.x with default config (blocked)",
			config:   DefaultFetchConfig(),
			url:      "https://172.16.0.1/api",
			wantErr:  ErrPrivateNetworkNotAllowed,
			wantPass: false,
		},
		{
			name: "localhost with AllowPrivateNetworks",
			config: func() *FetchConfig {
				c := DefaultFetchConfig()
				c.AllowPrivateNetworks = true
				return c
			}(),
			url:      "https://localhost/api",
			wantPass: true,
		},
		{
			name: "Blocked host",
			config: func() *FetchConfig {
				c := DefaultFetchConfig()
				c.BlockedHosts = []string{"blocked.example.com"}
				return c
			}(),
			url:      "https://blocked.example.com/api",
			wantErr:  ErrHostBlocked,
			wantPass: false,
		},
		{
			name: "Wildcard blocked host",
			config: func() *FetchConfig {
				c := DefaultFetchConfig()
				c.BlockedHosts = []string{"*.example.com"}
				return c
			}(),
			url:      "https://api.example.com/data",
			wantErr:  ErrHostBlocked,
			wantPass: false,
		},
		{
			name: "AllowedHosts - allowed",
			config: func() *FetchConfig {
				c := DefaultFetchConfig()
				c.AllowedHosts = []string{"api.example.com"}
				return c
			}(),
			url:      "https://api.example.com/data",
			wantPass: true,
		},
		{
			name: "AllowedHosts - not allowed",
			config: func() *FetchConfig {
				c := DefaultFetchConfig()
				c.AllowedHosts = []string{"api.example.com"}
				return c
			}(),
			url:      "https://other.example.com/data",
			wantErr:  ErrHostNotAllowed,
			wantPass: false,
		},
		{
			name:     "Invalid URL",
			config:   DefaultFetchConfig(),
			url:      "not a valid url",
			wantErr:  ErrInvalidURL,
			wantPass: false,
		},
		{
			name:     "FTP URL (invalid scheme)",
			config:   DefaultFetchConfig(),
			url:      "ftp://example.com/file",
			wantErr:  ErrInvalidURL,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateURL(tt.url)
			if tt.wantPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
				}
			}
		})
	}
}

func TestIsPrivateNetwork(t *testing.T) {
	tests := []struct {
		host    string
		private bool
	}{
		// Localhost variants
		{"localhost", true},
		{"LOCALHOST", true},
		{"localhost.", true},

		// IPv4 loopback
		{"127.0.0.1", true},
		{"127.0.0.255", true},
		{"127.255.255.255", true},

		// IPv4 private ranges
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.0.1", true},
		{"192.168.255.255", true},

		// Link-local
		{"169.254.0.1", true},
		{"169.254.255.254", true},

		// Public IPs
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false}, // example.com

		// IPv6 loopback
		{"::1", true},

		// Public hostnames (not IPs - can't check without DNS)
		{"example.com", false},
		{"api.github.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isPrivateNetwork(tt.host)
			assert.Equal(t, tt.private, got)
		})
	}
}

func TestIsHostInList(t *testing.T) {
	tests := []struct {
		host  string
		list  []string
		match bool
	}{
		// Exact match
		{"example.com", []string{"example.com"}, true},
		{"Example.COM", []string{"example.com"}, true}, // case insensitive
		{"other.com", []string{"example.com"}, false},

		// Wildcard match
		{"api.example.com", []string{"*.example.com"}, true},
		{"sub.api.example.com", []string{"*.example.com"}, true},
		{"example.com", []string{"*.example.com"}, false}, // wildcard requires subdomain
		{"notexample.com", []string{"*.example.com"}, false},

		// Empty list
		{"example.com", []string{}, false},
		{"example.com", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isHostInList(tt.host, tt.list)
			assert.Equal(t, tt.match, got)
		})
	}
}

func TestHeaders(t *testing.T) {
	vm := goja.New()

	t.Run("get and set", func(t *testing.T) {
		h := NewHeaders(vm)
		obj := h.ToGojaObject()

		// Set via method call
		setFn, _ := goja.AssertFunction(obj.Get("set"))
		_, _ = setFn(goja.Undefined(), vm.ToValue("Content-Type"), vm.ToValue("application/json"))

		// Get via method call
		getFn, _ := goja.AssertFunction(obj.Get("get"))
		result, _ := getFn(goja.Undefined(), vm.ToValue("content-type")) // case insensitive

		assert.Equal(t, "application/json", result.String())
	})

	t.Run("has", func(t *testing.T) {
		h := NewHeaders(vm)
		obj := h.ToGojaObject()

		setFn, _ := goja.AssertFunction(obj.Get("set"))
		_, _ = setFn(goja.Undefined(), vm.ToValue("X-Custom"), vm.ToValue("value"))

		hasFn, _ := goja.AssertFunction(obj.Get("has"))

		result, _ := hasFn(goja.Undefined(), vm.ToValue("x-custom"))
		assert.True(t, result.ToBoolean())

		result, _ = hasFn(goja.Undefined(), vm.ToValue("x-missing"))
		assert.False(t, result.ToBoolean())
	})

	t.Run("append", func(t *testing.T) {
		h := NewHeaders(vm)
		obj := h.ToGojaObject()

		appendFn, _ := goja.AssertFunction(obj.Get("append"))
		_, _ = appendFn(goja.Undefined(), vm.ToValue("Accept"), vm.ToValue("text/html"))
		_, _ = appendFn(goja.Undefined(), vm.ToValue("Accept"), vm.ToValue("application/json"))

		getFn, _ := goja.AssertFunction(obj.Get("get"))
		result, _ := getFn(goja.Undefined(), vm.ToValue("Accept"))

		// First value is returned
		assert.Equal(t, "text/html", result.String())
	})

	t.Run("delete", func(t *testing.T) {
		h := NewHeaders(vm)
		obj := h.ToGojaObject()

		setFn, _ := goja.AssertFunction(obj.Get("set"))
		_, _ = setFn(goja.Undefined(), vm.ToValue("X-Remove"), vm.ToValue("value"))

		deleteFn, _ := goja.AssertFunction(obj.Get("delete"))
		_, _ = deleteFn(goja.Undefined(), vm.ToValue("X-Remove"))

		hasFn, _ := goja.AssertFunction(obj.Get("has"))
		result, _ := hasFn(goja.Undefined(), vm.ToValue("X-Remove"))
		assert.False(t, result.ToBoolean())
	})

	t.Run("forEach", func(t *testing.T) {
		h := NewHeaders(vm)
		obj := h.ToGojaObject()

		setFn, _ := goja.AssertFunction(obj.Get("set"))
		_, _ = setFn(goja.Undefined(), vm.ToValue("Content-Type"), vm.ToValue("application/json"))
		_, _ = setFn(goja.Undefined(), vm.ToValue("Accept"), vm.ToValue("*/*"))

		forEachFn, _ := goja.AssertFunction(obj.Get("forEach"))

		// Collect values via callback
		var collected []string
		callback := vm.ToValue(func(call goja.FunctionCall) goja.Value {
			value := call.Argument(0).String()
			name := call.Argument(1).String()
			collected = append(collected, name+": "+value)
			return goja.Undefined()
		})

		_, err := forEachFn(goja.Undefined(), callback)
		require.NoError(t, err)

		assert.Len(t, collected, 2)
	})

	t.Run("keys", func(t *testing.T) {
		h := NewHeaders(vm)
		obj := h.ToGojaObject()

		setFn, _ := goja.AssertFunction(obj.Get("set"))
		_, _ = setFn(goja.Undefined(), vm.ToValue("Content-Type"), vm.ToValue("application/json"))
		_, _ = setFn(goja.Undefined(), vm.ToValue("Accept"), vm.ToValue("*/*"))

		keysFn, _ := goja.AssertFunction(obj.Get("keys"))
		result, _ := keysFn(goja.Undefined())

		keys := result.Export().([]string)
		assert.Len(t, keys, 2)
	})

	t.Run("clone", func(t *testing.T) {
		h := NewHeaders(vm)
		h.headers.Set("X-Original", "value")

		cloned := h.clone()
		cloned.headers.Set("X-Cloned", "value")

		// Original should not have the cloned header
		assert.Empty(t, h.headers.Get("X-Cloned"))
		// Cloned should have both
		assert.Equal(t, "value", cloned.headers.Get("X-Original"))
		assert.Equal(t, "value", cloned.headers.Get("X-Cloned"))
	})
}

func TestNewHeadersFromHTTP(t *testing.T) {
	vm := goja.New()

	httpHeaders := make(map[string][]string)
	httpHeaders["Content-Type"] = []string{"application/json"}
	httpHeaders["X-Request-Id"] = []string{"123"}

	h := NewHeadersFromHTTP(vm, httpHeaders)

	assert.Equal(t, "application/json", h.headers.Get("Content-Type"))
	assert.Equal(t, "123", h.headers.Get("X-Request-Id"))
}

func TestNewHeadersFromObject(t *testing.T) {
	vm := goja.New()

	obj := vm.NewObject()
	_ = obj.Set("Content-Type", "application/json")
	_ = obj.Set("Authorization", "Bearer token")

	h := NewHeadersFromObject(vm, obj)

	assert.Equal(t, "application/json", h.headers.Get("Content-Type"))
	assert.Equal(t, "Bearer token", h.headers.Get("Authorization"))
}

func TestResponse(t *testing.T) {
	vm := goja.New()

	t.Run("properties", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status:     200,
			StatusText: "OK",
			Headers:    NewHeaders(vm),
			URL:        "https://example.com/api",
			Body:       []byte(`{"result": "success"}`),
			Redirected: false,
		})
		resp.SetPromiseConstructor(vm.Get("Promise"))

		obj := resp.ToGojaObject()

		assert.Equal(t, int64(200), obj.Get("status").ToInteger())
		assert.Equal(t, "OK", obj.Get("statusText").String())
		assert.True(t, obj.Get("ok").ToBoolean())
		assert.Equal(t, "https://example.com/api", obj.Get("url").String())
		assert.False(t, obj.Get("redirected").ToBoolean())
		assert.False(t, obj.Get("bodyUsed").ToBoolean())
	})

	t.Run("ok is false for 4xx", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status: 404,
		})
		obj := resp.ToGojaObject()

		assert.False(t, obj.Get("ok").ToBoolean())
	})

	t.Run("ok is false for 5xx", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status: 500,
		})
		obj := resp.ToGojaObject()

		assert.False(t, obj.Get("ok").ToBoolean())
	})

	t.Run("text method", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status: 200,
			Body:   []byte("Hello, World!"),
		})
		resp.SetPromiseConstructor(vm.Get("Promise"))

		obj := resp.ToGojaObject()
		textFn, _ := goja.AssertFunction(obj.Get("text"))
		result, err := textFn(goja.Undefined())
		require.NoError(t, err)

		// Result is a Promise, resolve it
		promise := result.Export().(*goja.Promise)
		assert.Equal(t, goja.PromiseStateFulfilled, promise.State())
		assert.Equal(t, "Hello, World!", promise.Result().String())
	})

	t.Run("json method", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status: 200,
			Body:   []byte(`{"key": "value"}`),
		})
		resp.SetPromiseConstructor(vm.Get("Promise"))

		obj := resp.ToGojaObject()
		jsonFn, _ := goja.AssertFunction(obj.Get("json"))
		result, err := jsonFn(goja.Undefined())
		require.NoError(t, err)

		// Result is a Promise, resolve it
		promise := result.Export().(*goja.Promise)
		assert.Equal(t, goja.PromiseStateFulfilled, promise.State())

		// Get the parsed JSON
		parsed := promise.Result().ToObject(vm)
		assert.Equal(t, "value", parsed.Get("key").String())
	})

	t.Run("body can only be used once", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status: 200,
			Body:   []byte("content"),
		})
		resp.SetPromiseConstructor(vm.Get("Promise"))

		obj := resp.ToGojaObject()
		textFn, _ := goja.AssertFunction(obj.Get("text"))

		// First call succeeds
		_, err := textFn(goja.Undefined())
		require.NoError(t, err)

		// bodyUsed should now be true
		assert.True(t, obj.Get("bodyUsed").ToBoolean())

		// Second call should return rejected Promise
		result, err := textFn(goja.Undefined())
		require.NoError(t, err)

		promise := result.Export().(*goja.Promise)
		assert.Equal(t, goja.PromiseStateRejected, promise.State())
	})

	t.Run("clone creates independent copy", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status:  200,
			Headers: NewHeaders(vm),
			Body:    []byte("content"),
		})
		resp.SetPromiseConstructor(vm.Get("Promise"))

		obj := resp.ToGojaObject()
		cloneFn, _ := goja.AssertFunction(obj.Get("clone"))

		clonedObj, err := cloneFn(goja.Undefined())
		require.NoError(t, err)

		cloned := clonedObj.ToObject(vm)

		// Both should have same properties
		assert.Equal(t, int64(200), cloned.Get("status").ToInteger())

		// Using body on original should not affect clone
		textFn, _ := goja.AssertFunction(obj.Get("text"))
		_, _ = textFn(goja.Undefined())

		// Original bodyUsed is true
		assert.True(t, obj.Get("bodyUsed").ToBoolean())

		// Clone bodyUsed is still false
		assert.False(t, cloned.Get("bodyUsed").ToBoolean())

		// Clone can still read body
		clonedTextFn, _ := goja.AssertFunction(cloned.Get("text"))
		result, err := clonedTextFn(goja.Undefined())
		require.NoError(t, err)

		promise := result.Export().(*goja.Promise)
		assert.Equal(t, goja.PromiseStateFulfilled, promise.State())
		assert.Equal(t, "content", promise.Result().String())
	})

	t.Run("cannot clone after body used", func(t *testing.T) {
		resp := NewResponse(vm, ResponseInit{
			Status: 200,
			Body:   []byte("content"),
		})
		resp.SetPromiseConstructor(vm.Get("Promise"))

		obj := resp.ToGojaObject()

		// Use the body first
		textFn, _ := goja.AssertFunction(obj.Get("text"))
		_, _ = textFn(goja.Undefined())

		// Now try to clone - should return an error
		cloneFn, _ := goja.AssertFunction(obj.Get("clone"))
		_, err := cloneFn(goja.Undefined())

		// Goja catches panics from exceptions and returns them as errors
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot clone")
	})
}

func TestParseStatusText(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
		{999, ""}, // Unknown status
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := parseStatusText(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseStatusTextFromHeader(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"200 OK", "OK"},
		{"201 Created", "Created"},
		{"404 Not Found", "Not Found"},
		{"500 Internal Server Error", "Internal Server Error"},
		{"200", ""}, // No text
		{"", ""},    // Empty
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := ParseStatusTextFromHeader(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewFetchModuleFromConfig(t *testing.T) {
	t.Run("nil config uses defaults", func(t *testing.T) {
		loop := &simpleEventLoop{vm: goja.New()}
		module := NewFetchModuleFromConfig(loop, nil)

		assert.NotNil(t, module)
		assert.NotNil(t, module.config)
		assert.Equal(t, DefaultTimeout, module.config.DefaultTimeout)
		assert.False(t, module.config.AllowInsecure)
		assert.False(t, module.config.AllowPrivateNetworks)
	})

	t.Run("config values are applied", func(t *testing.T) {
		loop := &simpleEventLoop{vm: goja.New()}
		cfg := &config.FetchConfig{
			AllowPrivateNetworks: true,
			Timeout:              60 * time.Second,
		}
		cfg.Insecure = true

		module := NewFetchModuleFromConfig(loop, cfg)

		assert.NotNil(t, module)
		assert.True(t, module.config.AllowInsecure)
		assert.True(t, module.config.AllowPrivateNetworks)
		assert.Equal(t, 60*time.Second, module.config.DefaultTimeout)
	})

	t.Run("zero timeout uses default", func(t *testing.T) {
		loop := &simpleEventLoop{vm: goja.New()}
		cfg := &config.FetchConfig{
			Timeout: 0,
		}

		module := NewFetchModuleFromConfig(loop, cfg)

		assert.Equal(t, DefaultTimeout, module.config.DefaultTimeout)
	})

	t.Run("negative timeout uses default", func(t *testing.T) {
		loop := &simpleEventLoop{vm: goja.New()}
		cfg := &config.FetchConfig{
			Timeout: -5 * time.Second,
		}

		module := NewFetchModuleFromConfig(loop, cfg)

		assert.Equal(t, DefaultTimeout, module.config.DefaultTimeout)
	})
}

// simpleEventLoop implements EventLoopInterface for simple unit tests
type simpleEventLoop struct {
	vm *goja.Runtime
}

func (m *simpleEventLoop) RegisterCallback() func(func()) {
	return func(f func()) { f() }
}

func (m *simpleEventLoop) Runtime() *goja.Runtime {
	return m.vm
}
