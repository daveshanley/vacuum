// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package fetch

import (
	"encoding/json"
	"strings"
	"sync/atomic"

	"github.com/dop251/goja"
)

// Response implements the Web Fetch API Response interface.
// https://developer.mozilla.org/en-US/docs/Web/API/Response
type Response struct {
	vm         *goja.Runtime
	status     int
	statusText string
	headers    *Headers
	url        string
	bodyBytes  []byte
	bodyUsed   int32 // atomic: 0 = not used, 1 = used
	redirected bool
	ok         bool

	// promiseConstructor is used to create Promises for body methods
	promiseConstructor goja.Value
}

// ResponseInit contains initialization options for creating a Response
type ResponseInit struct {
	Status     int
	StatusText string
	Headers    *Headers
	URL        string
	Body       []byte
	Redirected bool
}

// NewResponse creates a new Response object
func NewResponse(vm *goja.Runtime, init ResponseInit) *Response {
	statusText := init.StatusText
	if statusText == "" {
		statusText = parseStatusText(init.Status)
	}

	return &Response{
		vm:         vm,
		status:     init.Status,
		statusText: statusText,
		headers:    init.Headers,
		url:        init.URL,
		bodyBytes:  init.Body,
		redirected: init.Redirected,
		ok:         init.Status >= 200 && init.Status < 300,
	}
}

// parseStatusText returns the default status text for a given status code
func parseStatusText(status int) string {
	// Common status texts per HTTP spec
	texts := map[int]string{
		100: "Continue",
		101: "Switching Protocols",
		200: "OK",
		201: "Created",
		202: "Accepted",
		204: "No Content",
		301: "Moved Permanently",
		302: "Found",
		303: "See Other",
		304: "Not Modified",
		307: "Temporary Redirect",
		308: "Permanent Redirect",
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		409: "Conflict",
		422: "Unprocessable Entity",
		429: "Too Many Requests",
		500: "Internal Server Error",
		502: "Bad Gateway",
		503: "Service Unavailable",
		504: "Gateway Timeout",
	}
	if text, ok := texts[status]; ok {
		return text
	}
	return ""
}

// ParseStatusTextFromHeader extracts just the status text from Go's resp.Status
// which is formatted as "200 OK"
func ParseStatusTextFromHeader(status string) string {
	parts := strings.SplitN(status, " ", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// SetPromiseConstructor sets the Promise constructor for creating Promises
func (r *Response) SetPromiseConstructor(ctor goja.Value) {
	r.promiseConstructor = ctor
}

// ToGojaObject creates the JavaScript Response object
func (r *Response) ToGojaObject() *goja.Object {
	obj := r.vm.NewObject()

	// Read-only properties
	_ = obj.DefineDataProperty("status", r.vm.ToValue(r.status), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)
	_ = obj.DefineDataProperty("statusText", r.vm.ToValue(r.statusText), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)
	_ = obj.DefineDataProperty("ok", r.vm.ToValue(r.ok), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)
	_ = obj.DefineDataProperty("url", r.vm.ToValue(r.url), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)
	_ = obj.DefineDataProperty("redirected", r.vm.ToValue(r.redirected), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)

	// Headers object
	if r.headers != nil {
		_ = obj.DefineDataProperty("headers", r.headers.ToGojaObject(), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)
	}

	// bodyUsed as a getter (dynamically returns current state)
	_ = obj.DefineAccessorProperty("bodyUsed",
		r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return r.vm.ToValue(atomic.LoadInt32(&r.bodyUsed) == 1)
		}),
		nil,
		goja.FLAG_FALSE, goja.FLAG_TRUE)

	// Methods
	_ = obj.Set("json", r.jsonMethod)
	_ = obj.Set("text", r.textMethod)
	_ = obj.Set("clone", r.cloneMethod)

	return obj
}

// markBodyUsed attempts to mark the body as used.
// Returns true if successful, false if body was already used.
func (r *Response) markBodyUsed() bool {
	return atomic.CompareAndSwapInt32(&r.bodyUsed, 0, 1)
}

// jsonMethod implements Response.json()
// Returns a Promise that resolves to the parsed JSON
func (r *Response) jsonMethod(call goja.FunctionCall) goja.Value {
	// Check if body already used
	if !r.markBodyUsed() {
		return r.rejectWithError(ErrBodyAlreadyUsed)
	}

	// Parse the body as JSON
	var result interface{}
	if err := json.Unmarshal(r.bodyBytes, &result); err != nil {
		return r.rejectWithError(err)
	}

	return r.resolveWithValue(result)
}

// textMethod implements Response.text()
// Returns a Promise that resolves to the body as a string
func (r *Response) textMethod(call goja.FunctionCall) goja.Value {
	// Check if body already used
	if !r.markBodyUsed() {
		return r.rejectWithError(ErrBodyAlreadyUsed)
	}

	return r.resolveWithValue(string(r.bodyBytes))
}

// cloneMethod implements Response.clone()
// Returns a new Response with a copy of the body
func (r *Response) cloneMethod(call goja.FunctionCall) goja.Value {
	// Per spec, cannot clone if body has been used
	if atomic.LoadInt32(&r.bodyUsed) == 1 {
		panic(r.vm.NewTypeError("Cannot clone a response whose body has been used"))
	}

	// Create independent copy
	cloned := &Response{
		vm:                 r.vm,
		status:             r.status,
		statusText:         r.statusText,
		headers:            r.headers.clone(),
		url:                r.url,
		bodyBytes:          append([]byte(nil), r.bodyBytes...), // copy slice
		bodyUsed:           0,                                   // independent tracking
		redirected:         r.redirected,
		ok:                 r.ok,
		promiseConstructor: r.promiseConstructor,
	}

	return cloned.ToGojaObject()
}

// resolveWithValue creates a resolved Promise with the given value
func (r *Response) resolveWithValue(value interface{}) goja.Value {
	if r.promiseConstructor == nil {
		// Fallback: return value directly if no Promise constructor
		return r.vm.ToValue(value)
	}

	// Create Promise using Promise.resolve()
	promiseObj := r.promiseConstructor.ToObject(r.vm)
	resolveFunc, ok := goja.AssertFunction(promiseObj.Get("resolve"))
	if !ok {
		return r.vm.ToValue(value)
	}

	result, err := resolveFunc(r.promiseConstructor, r.vm.ToValue(value))
	if err != nil {
		return r.vm.ToValue(value)
	}
	return result
}

// rejectWithError creates a rejected Promise with the given error
func (r *Response) rejectWithError(err error) goja.Value {
	if r.promiseConstructor == nil {
		// Fallback: throw error if no Promise constructor
		panic(r.vm.NewTypeError(err.Error()))
	}

	// Create Promise using Promise.reject()
	promiseObj := r.promiseConstructor.ToObject(r.vm)
	rejectFunc, ok := goja.AssertFunction(promiseObj.Get("reject"))
	if !ok {
		panic(r.vm.NewTypeError(err.Error()))
	}

	errObj := r.vm.NewTypeError(err.Error())
	result, callErr := rejectFunc(r.promiseConstructor, errObj)
	if callErr != nil {
		panic(r.vm.NewTypeError(err.Error()))
	}
	return result
}
