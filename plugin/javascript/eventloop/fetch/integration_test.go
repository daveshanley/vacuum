// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package fetch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventLoop implements a minimal event loop for testing
type mockEventLoop struct {
	vm         *goja.Runtime
	jobQueue   []func()
	jobMutex   sync.Mutex
	pendingOps int32
}

func newMockEventLoop() *mockEventLoop {
	return &mockEventLoop{
		vm: goja.New(),
	}
}

func (m *mockEventLoop) Runtime() *goja.Runtime {
	return m.vm
}

func (m *mockEventLoop) RegisterCallback() func(func()) {
	atomic.AddInt32(&m.pendingOps, 1)
	called := int32(0)

	return func(callback func()) {
		if !atomic.CompareAndSwapInt32(&called, 0, 1) {
			return
		}

		m.jobMutex.Lock()
		m.jobQueue = append(m.jobQueue, func() {
			callback()
			atomic.AddInt32(&m.pendingOps, -1)
		})
		m.jobMutex.Unlock()
	}
}

func (m *mockEventLoop) runUntilDone(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		m.jobMutex.Lock()
		if len(m.jobQueue) == 0 && atomic.LoadInt32(&m.pendingOps) == 0 {
			m.jobMutex.Unlock()
			return nil
		}

		if len(m.jobQueue) > 0 {
			job := m.jobQueue[0]
			m.jobQueue = m.jobQueue[1:]
			m.jobMutex.Unlock()
			job()
			// Process Goja's internal job queue
			_, _ = m.vm.RunString("")
		} else {
			m.jobMutex.Unlock()
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// JSON endpoint
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Hello, World!",
			"method":  r.Method,
		})
	})

	// Text endpoint
	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Plain text response"))
	})

	// Echo endpoint - returns request details
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		headers := make(map[string]string)
		for name, values := range r.Header {
			if len(values) > 0 {
				headers[name] = values[0]
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"method":  r.Method,
			"path":    r.URL.Path,
			"headers": headers,
		})
	})

	// POST endpoint
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"received": body,
			"success":  true,
		})
	})

	// Status endpoint - returns specific status code
	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		switch r.URL.Path {
		case "/status/400":
			status = http.StatusBadRequest
		case "/status/401":
			status = http.StatusUnauthorized
		case "/status/403":
			status = http.StatusForbidden
		case "/status/404":
			status = http.StatusNotFound
		case "/status/500":
			status = http.StatusInternalServerError
		}
		w.WriteHeader(status)
		w.Write([]byte(`{"error": "status test"}`))
	})

	// Redirect endpoint
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/json", http.StatusFound)
	})

	// Slow endpoint for timeout testing
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("slow response"))
	})

	return httptest.NewServer(mux)
}

func TestFetchIntegration_BasicGET(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true // Test server uses HTTP
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	// Execute fetch in JavaScript
	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/json");
			const data = await response.json();
			return {
				status: response.status,
				ok: response.ok,
				message: data.message,
				method: data.method
			};
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	// Get Promise result
	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.Equal(t, int64(200), obj.Get("status").ToInteger())
	assert.True(t, obj.Get("ok").ToBoolean())
	assert.Equal(t, "Hello, World!", obj.Get("message").String())
	assert.Equal(t, "GET", obj.Get("method").String())
}

func TestFetchIntegration_POST(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/post", {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({ name: "test" })
			});
			return await response.json();
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.True(t, obj.Get("success").ToBoolean())

	received := obj.Get("received").ToObject(loop.vm)
	assert.Equal(t, "test", received.Get("name").String())
}

func TestFetchIntegration_CustomHeaders(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/echo", {
				headers: {
					"X-Custom-Header": "custom-value",
					"Authorization": "Bearer token123"
				}
			});
			const data = await response.json();
			return data.headers;
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	headers := promise.Result().ToObject(loop.vm)
	assert.Equal(t, "custom-value", headers.Get("X-Custom-Header").String())
	assert.Equal(t, "Bearer token123", headers.Get("Authorization").String())
}

func TestFetchIntegration_ResponseText(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/text");
			return await response.text();
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	assert.Equal(t, "Plain text response", promise.Result().String())
}

func TestFetchIntegration_HTTPError(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	// HTTP errors should resolve, not reject
	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/status/404");
			return {
				status: response.status,
				ok: response.ok
			};
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.Equal(t, int64(404), obj.Get("status").ToInteger())
	assert.False(t, obj.Get("ok").ToBoolean())
}

func TestFetchIntegration_Redirect(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/redirect");
			const data = await response.json();
			return {
				redirected: response.redirected,
				url: response.url,
				message: data.message
			};
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.True(t, obj.Get("redirected").ToBoolean())
	assert.Contains(t, obj.Get("url").String(), "/json")
	assert.Equal(t, "Hello, World!", obj.Get("message").String())
}

func TestFetchIntegration_SecurityBlockHTTP(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	// Default config blocks HTTP
	config := DefaultFetchConfig()

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			try {
				await fetch("` + server.URL + `/json");
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	assert.Contains(t, errorMsg, "insecure HTTP not allowed")
}

func TestFetchIntegration_Timeout(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true
	config.DefaultTimeout = 100 * time.Millisecond // Very short timeout

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			try {
				await fetch("` + server.URL + `/slow");
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	assert.Contains(t, errorMsg, "timed out")
}

func TestFetchIntegration_ResponseClone(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/json");
			const cloned = response.clone();

			// Read original
			const data1 = await response.json();

			// Read clone
			const data2 = await cloned.json();

			return {
				original: data1.message,
				cloned: data2.message,
				originalBodyUsed: response.bodyUsed,
				clonedBodyUsed: cloned.bodyUsed
			};
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.Equal(t, "Hello, World!", obj.Get("original").String())
	assert.Equal(t, "Hello, World!", obj.Get("cloned").String())
	assert.True(t, obj.Get("originalBodyUsed").ToBoolean())
	assert.True(t, obj.Get("clonedBodyUsed").ToBoolean())
}

func TestFetchIntegration_HeadersForEach(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/json");
			const headers = [];
			response.headers.forEach((value, name) => {
				headers.push(name + ": " + value);
			});
			return { headerCount: headers.length, hasContentType: headers.some(h => h.includes("content-type")) };
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.Greater(t, obj.Get("headerCount").ToInteger(), int64(0))
	assert.True(t, obj.Get("hasContentType").ToBoolean())
}

func TestFetchIntegration_DefaultContentType(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	// When sending a string body without Content-Type, it should default to text/plain
	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/echo", {
				method: "POST",
				body: "plain text content"
			});
			const data = await response.json();
			return { contentType: data.headers["Content-Type"] };
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	contentType := obj.Get("contentType").String()
	assert.Contains(t, contentType, "text/plain")
}

func TestFetchIntegration_BodyMustBeString(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	// Passing an object as body should throw a TypeError, not silently coerce to "[object Object]"
	script := `
		(async function() {
			try {
				await fetch("` + server.URL + `/post", {
					method: "POST",
					body: { key: "value" }
				});
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	assert.Contains(t, errorMsg, "body must be a string")
}

func TestFetchIntegration_ManualRedirectNotMarkedAsRedirected(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	// In manual mode, response.redirected should be false even for 3xx responses
	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/redirect", {
				redirect: "manual"
			});
			return {
				status: response.status,
				redirected: response.redirected
			};
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	// Should return the 302 response directly
	assert.Equal(t, int64(302), obj.Get("status").ToInteger())
	// Should NOT be marked as redirected since we didn't follow it
	assert.False(t, obj.Get("redirected").ToBoolean())
}

func TestFetchIntegration_RedirectErrorMode(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true

	module := NewFetchModule(loop, config)
	module.Register()

	// redirect: "error" should reject with ErrRedirectNotAllowed
	script := `
		(async function() {
			try {
				await fetch("` + server.URL + `/redirect", {
					redirect: "error"
				});
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	assert.Contains(t, errorMsg, "redirects not allowed")
}

func TestFetchIntegration_RedirectToBlockedHost(t *testing.T) {
	// Create a server that redirects to a blocked host
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://blocked.example.com/evil", http.StatusFound)
	}))
	defer server.Close()

	loop := newMockEventLoop()

	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true
	config.BlockedHosts = []string{"blocked.example.com"}

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			try {
				await fetch("` + server.URL + `");
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	// Should fail because redirect target is blocked
	assert.Contains(t, errorMsg, "redirect target not allowed")
	assert.Contains(t, errorMsg, "blocked")
}

func TestFetchIntegration_RedirectToPrivateNetworkBlocked(t *testing.T) {
	// Test that redirect to a private network address (10.x.x.x) is blocked
	// when that address is in BlockedHosts

	// Create a server that redirects to a private network address
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect to 10.x.x.x which we'll block via BlockedHosts
		http.Redirect(w, r, "http://10.0.0.1:9999/evil", http.StatusFound)
	}))
	defer server.Close()

	loop := newMockEventLoop()
	config := DefaultFetchConfig()
	config.AllowInsecure = true
	config.AllowPrivateNetworks = true
	config.BlockedHosts = []string{"10.0.0.1"} // Block the redirect target specifically

	module := NewFetchModule(loop, config)
	module.Register()

	script := `
		(async function() {
			try {
				await fetch("` + server.URL + `");
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	// Should fail because redirect target is blocked
	assert.Contains(t, errorMsg, "redirect target not allowed")
	assert.Contains(t, errorMsg, "blocked")
}

func TestFetchIntegration_CustomConfigWithNilHTTPClient(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	// Create custom config with nil HTTPClient - should still get secure transport
	config := &FetchConfig{
		DefaultTimeout:       5 * time.Second,
		MaxResponseSize:      DefaultMaxResponseSize,
		AllowInsecure:        true,
		AllowPrivateNetworks: true,
		UserAgent:            "custom-agent",
		HTTPClient:           nil, // Explicitly nil
	}

	module := NewFetchModule(loop, config)
	module.Register()

	// Verify the request works (secure transport should be applied)
	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/json");
			const data = await response.json();
			return { message: data.message };
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.Equal(t, "Hello, World!", obj.Get("message").String())

	// Verify HTTPClient was set
	assert.NotNil(t, config.HTTPClient)
}

func TestFetchIntegration_CustomConfigSecureTransportEnforced(t *testing.T) {
	// Test that custom config with nil HTTPClient still enforces private network blocking
	loop := newMockEventLoop()

	config := &FetchConfig{
		DefaultTimeout:       5 * time.Second,
		MaxResponseSize:      DefaultMaxResponseSize,
		AllowInsecure:        true,
		AllowPrivateNetworks: false, // Block private networks
		UserAgent:            "test-agent",
		HTTPClient:           nil, // Will be set by NewFetchModule with secure transport
	}

	module := NewFetchModule(loop, config)
	module.Register()

	// Try to fetch localhost - should be blocked
	script := `
		(async function() {
			try {
				await fetch("http://localhost:9999/test");
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	assert.Contains(t, errorMsg, "private network")
}

func TestFetchIntegration_CustomHTTPClientWithTransport(t *testing.T) {
	// Test that custom HTTPClient with existing Transport gets its DialContext wrapped
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	// Create custom client with existing transport (simulating user-provided client)
	customTransport := &http.Transport{
		MaxIdleConns:    50,
		IdleConnTimeout: 60 * time.Second,
	}
	customClient := &http.Client{
		Transport: customTransport,
		Timeout:   10 * time.Second,
	}

	config := &FetchConfig{
		DefaultTimeout:       5 * time.Second,
		MaxResponseSize:      DefaultMaxResponseSize,
		AllowInsecure:        true,
		AllowPrivateNetworks: false, // Block private networks - should wrap transport
		UserAgent:            "custom-client-test",
		HTTPClient:           customClient,
	}

	module := NewFetchModule(loop, config)
	module.Register()

	// Try to fetch localhost - should be blocked even with custom client
	script := `
		(async function() {
			try {
				await fetch("http://localhost:9999/test");
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	// Should be blocked because DialContext was wrapped
	assert.Contains(t, errorMsg, "private network")
}

func TestFetchIntegration_CustomHTTPClientAllowsPrivateWhenConfigured(t *testing.T) {
	// Test that custom HTTPClient works normally when AllowPrivateNetworks is true
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	customClient := &http.Client{
		Transport: &http.Transport{},
		Timeout:   10 * time.Second,
	}

	config := &FetchConfig{
		DefaultTimeout:       5 * time.Second,
		MaxResponseSize:      DefaultMaxResponseSize,
		AllowInsecure:        true,
		AllowPrivateNetworks: true, // Allow private networks - should NOT wrap transport
		UserAgent:            "custom-client-test",
		HTTPClient:           customClient,
	}

	module := NewFetchModule(loop, config)
	module.Register()

	// Should work fine with localhost when AllowPrivateNetworks is true
	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/json");
			const data = await response.json();
			return { message: data.message };
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.Equal(t, "Hello, World!", obj.Get("message").String())
}

// customRoundTripper is a test RoundTripper that wraps another transport
type customRoundTripper struct {
	wrapped http.RoundTripper
}

func (c *customRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.wrapped.RoundTrip(req)
}

func TestFetchIntegration_CustomRoundTripperReplacedWhenPrivateBlocked(t *testing.T) {
	// Test that custom RoundTripper gets replaced with secure transport when AllowPrivateNetworks=false
	loop := newMockEventLoop()

	// Create a custom RoundTripper (not *http.Transport)
	customRT := &customRoundTripper{
		wrapped: http.DefaultTransport,
	}
	customClient := &http.Client{
		Transport: customRT,
		Timeout:   10 * time.Second,
	}

	config := &FetchConfig{
		DefaultTimeout:       5 * time.Second,
		MaxResponseSize:      DefaultMaxResponseSize,
		AllowInsecure:        true,
		AllowPrivateNetworks: false, // Block private networks - custom RT should be replaced
		UserAgent:            "custom-rt-test",
		HTTPClient:           customClient,
	}

	module := NewFetchModule(loop, config)
	module.Register()

	// The custom RoundTripper should have been replaced, so private network should be blocked
	script := `
		(async function() {
			try {
				await fetch("http://localhost:9999/test");
				return { error: null };
			} catch (e) {
				return { error: e.message };
			}
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	errorMsg := obj.Get("error").String()
	// Should be blocked because custom RT was replaced with secure transport
	assert.Contains(t, errorMsg, "private network")

	// Verify the transport was replaced
	_, isSecureTransport := config.HTTPClient.Transport.(*http.Transport)
	assert.True(t, isSecureTransport, "Custom RoundTripper should have been replaced with *http.Transport")
}

func TestFetchIntegration_CustomRoundTripperPreservedWhenPrivateAllowed(t *testing.T) {
	// Test that custom RoundTripper is preserved when AllowPrivateNetworks=true
	server := setupTestServer()
	defer server.Close()

	loop := newMockEventLoop()

	// Create a custom RoundTripper (not *http.Transport)
	customRT := &customRoundTripper{
		wrapped: http.DefaultTransport,
	}
	customClient := &http.Client{
		Transport: customRT,
		Timeout:   10 * time.Second,
	}

	config := &FetchConfig{
		DefaultTimeout:       5 * time.Second,
		MaxResponseSize:      DefaultMaxResponseSize,
		AllowInsecure:        true,
		AllowPrivateNetworks: true, // Allow private networks - custom RT should be preserved
		UserAgent:            "custom-rt-test",
		HTTPClient:           customClient,
	}

	module := NewFetchModule(loop, config)
	module.Register()

	// Custom RoundTripper should be preserved
	_, isCustomRT := config.HTTPClient.Transport.(*customRoundTripper)
	assert.True(t, isCustomRT, "Custom RoundTripper should be preserved when AllowPrivateNetworks=true")

	// And fetch should work
	script := `
		(async function() {
			const response = await fetch("` + server.URL + `/json");
			const data = await response.json();
			return { message: data.message };
		})()
	`

	result, err := loop.vm.RunString(script)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = loop.runUntilDone(ctx)
	require.NoError(t, err)

	promise := result.Export().(*goja.Promise)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())

	obj := promise.Result().ToObject(loop.vm)
	assert.Equal(t, "Hello, World!", obj.Get("message").String())
}
