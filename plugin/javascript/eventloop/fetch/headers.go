// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package fetch

import (
	"net/http"
	"sort"
	"strings"

	"github.com/dop251/goja"
)

// Headers implement the Web Fetch API Headers interface.
// https://developer.mozilla.org/en-US/docs/Web/API/Headers
type Headers struct {
	vm      *goja.Runtime
	headers http.Header
}

// NewHeaders creates a new Headers object
func NewHeaders(vm *goja.Runtime) *Headers {
	return &Headers{
		vm:      vm,
		headers: make(http.Header),
	}
}

// NewHeadersFromHTTP creates a Headers object from an http.Header
func NewHeadersFromHTTP(vm *goja.Runtime, h http.Header) *Headers {
	headers := &Headers{
		vm:      vm,
		headers: make(http.Header),
	}
	// Copy headers
	for k, v := range h {
		headers.headers[k] = append([]string(nil), v...)
	}
	return headers
}

// NewHeadersFromObject creates a Headers object from a JavaScript object
func NewHeadersFromObject(vm *goja.Runtime, obj *goja.Object) *Headers {
	headers := NewHeaders(vm)

	for _, key := range obj.Keys() {
		val := obj.Get(key)
		if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
			headers.headers.Set(key, val.String())
		}
	}

	return headers
}

// ToHTTPHeader converts to http.Header for use with http.Request
func (h *Headers) ToHTTPHeader() http.Header {
	result := make(http.Header)
	for k, v := range h.headers {
		result[k] = append([]string(nil), v...)
	}
	return result
}

// clone creates a deep copy of the Headers
func (h *Headers) clone() *Headers {
	cloned := &Headers{
		vm:      h.vm,
		headers: make(http.Header, len(h.headers)),
	}
	for k, v := range h.headers {
		cloned.headers[k] = append([]string(nil), v...)
	}
	return cloned
}

// sortedKeys returns header names in sorted order for consistent iteration
func (h *Headers) sortedKeys() []string {
	keys := make([]string, 0, len(h.headers))
	for k := range h.headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ToGojaObject creates the JavaScript Headers object
func (h *Headers) ToGojaObject() *goja.Object {
	obj := h.vm.NewObject()

	_ = obj.Set("get", h.get)
	_ = obj.Set("set", h.set)
	_ = obj.Set("has", h.has)
	_ = obj.Set("append", h.append)
	_ = obj.Set("delete", h.delete)
	_ = obj.Set("forEach", h.forEach)
	_ = obj.Set("keys", h.keys)
	_ = obj.Set("values", h.values)
	_ = obj.Set("entries", h.entries)

	return obj
}

func (h *Headers) get(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return goja.Null()
	}

	name := call.Argument(0).String()
	value := h.headers.Get(name)
	if value == "" {
		return goja.Null()
	}
	return h.vm.ToValue(value)
}

func (h *Headers) set(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		return goja.Undefined()
	}

	name := call.Argument(0).String()
	value := call.Argument(1).String()
	h.headers.Set(name, value)
	return goja.Undefined()
}

func (h *Headers) has(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return h.vm.ToValue(false)
	}

	name := call.Argument(0).String()
	_, exists := h.headers[http.CanonicalHeaderKey(name)]
	return h.vm.ToValue(exists)
}

func (h *Headers) append(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		return goja.Undefined()
	}

	name := call.Argument(0).String()
	value := call.Argument(1).String()
	h.headers.Add(name, value)
	return goja.Undefined()
}

func (h *Headers) delete(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return goja.Undefined()
	}

	name := call.Argument(0).String()
	h.headers.Del(name)
	return goja.Undefined()
}

func (h *Headers) forEach(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(h.vm.NewTypeError("forEach requires a callback function"))
	}

	callback, ok := goja.AssertFunction(call.Argument(0))
	if !ok {
		panic(h.vm.NewTypeError("forEach requires a callback function"))
	}

	var thisArg goja.Value = goja.Undefined()
	if len(call.Arguments) > 1 {
		thisArg = call.Argument(1)
	}

	headersObj := h.ToGojaObject()
	for _, name := range h.sortedKeys() {
		combinedValue := strings.Join(h.headers[name], ", ")
		_, _ = callback(thisArg,
			h.vm.ToValue(combinedValue),
			h.vm.ToValue(strings.ToLower(name)),
			headersObj)
	}

	return goja.Undefined()
}

func (h *Headers) keys(call goja.FunctionCall) goja.Value {
	sortedKeys := h.sortedKeys()
	result := make([]string, len(sortedKeys))
	for i, k := range sortedKeys {
		result[i] = strings.ToLower(k)
	}
	return h.vm.ToValue(result)
}

func (h *Headers) values(call goja.FunctionCall) goja.Value {
	sortedKeys := h.sortedKeys()
	values := make([]string, len(sortedKeys))
	for i, k := range sortedKeys {
		values[i] = strings.Join(h.headers[k], ", ")
	}
	return h.vm.ToValue(values)
}

func (h *Headers) entries(call goja.FunctionCall) goja.Value {
	sortedKeys := h.sortedKeys()
	entries := make([][]string, len(sortedKeys))
	for i, k := range sortedKeys {
		entries[i] = []string{
			strings.ToLower(k),
			strings.Join(h.headers[k], ", "),
		}
	}
	return h.vm.ToValue(entries)
}
