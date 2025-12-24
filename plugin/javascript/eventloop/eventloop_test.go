// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package eventloop

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventLoop_SyncFunction(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(`(function() { return 42; })()`)
	})

	require.NoError(t, err)
	assert.Equal(t, int64(42), result.Export())
}

func TestEventLoop_SyncFunctionReturnsArray(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(`(function() { return [{ message: "test" }]; })()`)
	})

	require.NoError(t, err)
	exported := result.Export()
	arr, ok := exported.([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 1)
}

func TestEventLoop_PromiseResolve(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(`Promise.resolve(42)`)
	})

	require.NoError(t, err)

	// The result should be a Promise
	promise := GetPromise(result)
	require.NotNil(t, promise)
	assert.Equal(t, goja.PromiseStateFulfilled, promise.State())
	assert.Equal(t, int64(42), promise.Result().Export())
}

func TestEventLoop_PromiseResolveWithArray(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(`Promise.resolve([{ message: "async result" }])`)
	})

	require.NoError(t, err)

	// Extract the Promise value
	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)

	arr, ok := exported.([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 1)
}

func TestEventLoop_PromiseReject(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(`Promise.reject("error message")`)
	})

	require.NoError(t, err)

	// The result should be a rejected Promise
	promise := GetPromise(result)
	require.NotNil(t, promise)
	assert.Equal(t, goja.PromiseStateRejected, promise.State())

	// ExtractPromiseValue should return an error
	_, extractErr := ExtractPromiseValue(result)
	require.Error(t, extractErr)
	assert.Contains(t, extractErr.Error(), "promise rejected")
}

func TestEventLoop_AsyncAwait(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		async function asyncFunc() {
			const a = await Promise.resolve(10);
			const b = await Promise.resolve(20);
			return a + b;
		}
		asyncFunc();
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	// Extract the Promise value
	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)
	assert.Equal(t, int64(30), exported)
}

func TestEventLoop_AsyncAwaitWithArray(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		async function runRule(input) {
			await Promise.resolve();
			return [{ message: "validation result for " + input.name }];
		}
		runRule({ name: "test" });
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	// Extract the Promise value
	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)

	arr, ok := exported.([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 1)

	item := arr[0].(map[string]interface{})
	assert.Equal(t, "validation result for test", item["message"])
}

func TestEventLoop_SetTimeout(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		new Promise(function(resolve) {
			setTimeout(function() {
				resolve("delayed result");
			}, 10);
		});
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	// Extract the Promise value
	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)
	assert.Equal(t, "delayed result", exported)
}

func TestEventLoop_SetTimeoutChained(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		new Promise(function(resolve) {
			setTimeout(function() {
				setTimeout(function() {
					resolve("double delayed");
				}, 5);
			}, 5);
		});
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)
	assert.Equal(t, "double delayed", exported)
}

func TestEventLoop_ClearTimeout(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		var result = "not called";
		var timerId = setTimeout(function() {
			result = "was called";
		}, 100);
		clearTimeout(timerId);
		result;
	`

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)
	assert.Equal(t, "not called", result.Export())
}

func TestEventLoop_Timeout(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		new Promise(function(resolve) {
			setTimeout(function() {
				resolve("should not reach");
			}, 5000);
		});
	`

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestEventLoop_RegisterCallback(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	var callbackExecuted int32

	ctx := context.Background()
	_, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		// Simulate an async Go operation
		enqueue := loop.RegisterCallback()

		go func() {
			time.Sleep(10 * time.Millisecond)
			enqueue(func() {
				atomic.StoreInt32(&callbackExecuted, 1)
			})
		}()

		return vm.RunString(`"started"`)
	})

	require.NoError(t, err)

	// The callback should have been executed
	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))
}

func TestEventLoop_MultipleCallbacks(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	var counter int32

	ctx := context.Background()
	_, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		// Simulate multiple async operations
		for i := 0; i < 5; i++ {
			enqueue := loop.RegisterCallback()
			go func(n int) {
				time.Sleep(time.Duration(n) * time.Millisecond)
				enqueue(func() {
					atomic.AddInt32(&counter, 1)
				})
			}(i)
		}

		return vm.RunString(`"started"`)
	})

	require.NoError(t, err)
	assert.Equal(t, int32(5), atomic.LoadInt32(&counter))
}

func TestEventLoop_NestedPromises(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		async function inner() {
			return await Promise.resolve(5);
		}
		async function outer() {
			const a = await inner();
			const b = await inner();
			const c = await Promise.resolve(10);
			return a + b + c;
		}
		outer();
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)
	assert.Equal(t, int64(20), exported)
}

func TestEventLoop_PromiseAll(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		Promise.all([
			Promise.resolve(1),
			Promise.resolve(2),
			Promise.resolve(3)
		]).then(function(values) {
			return values.reduce(function(a, b) { return a + b; }, 0);
		});
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)
	assert.Equal(t, int64(6), exported)
}

func TestEventLoop_PromiseRace(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		Promise.race([
			new Promise(function(resolve) { setTimeout(function() { resolve("slow"); }, 50); }),
			Promise.resolve("fast")
		]);
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	exported, extractErr := ExtractPromiseValue(result)
	require.NoError(t, extractErr)
	assert.Equal(t, "fast", exported)
}

func TestEventLoop_AlreadyRunning(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	// Manually set running flag
	atomic.StoreInt32(&loop.running, 1)

	ctx := context.Background()
	_, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(`42`)
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLoopAlreadyRunning)

	// Reset for cleanup
	atomic.StoreInt32(&loop.running, 0)
}

func TestIsPromise(t *testing.T) {
	vm := goja.New()

	// Test with non-Promise values
	assert.False(t, IsPromise(nil))
	assert.False(t, IsPromise(goja.Undefined()))
	assert.False(t, IsPromise(goja.Null()))
	assert.False(t, IsPromise(vm.ToValue(42)))
	assert.False(t, IsPromise(vm.ToValue("string")))

	// Test with Promise
	promiseValue, _ := vm.RunString(`Promise.resolve(42)`)
	assert.True(t, IsPromise(promiseValue))
}

func TestGetPromise(t *testing.T) {
	vm := goja.New()

	// Test with non-Promise values
	assert.Nil(t, GetPromise(nil))
	assert.Nil(t, GetPromise(goja.Undefined()))
	assert.Nil(t, GetPromise(vm.ToValue(42)))

	// Test with Promise
	promiseValue, _ := vm.RunString(`Promise.resolve(42)`)
	promise := GetPromise(promiseValue)
	assert.NotNil(t, promise)
	assert.Equal(t, goja.PromiseStateFulfilled, promise.State())
}

func TestExtractPromiseValue_NonPromise(t *testing.T) {
	vm := goja.New()

	// Test with regular value
	value := vm.ToValue(42)
	exported, err := ExtractPromiseValue(value)
	require.NoError(t, err)
	assert.Equal(t, int64(42), exported)
}

func TestExtractPromiseValue_Nil(t *testing.T) {
	exported, err := ExtractPromiseValue(nil)
	require.NoError(t, err)
	assert.Nil(t, exported)
}

func TestEventLoop_AsyncAwaitWithError(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	script := `
		async function failingFunc() {
			throw new Error("async error");
		}
		failingFunc();
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err) // The Run itself doesn't error

	// But the Promise should be rejected
	_, extractErr := ExtractPromiseValue(result)
	require.Error(t, extractErr)
	// Error objects are exported differently, just check we got a rejection
	assert.Contains(t, extractErr.Error(), "promise rejected")
}

func TestEventLoop_AsyncAwaitWithStringError(t *testing.T) {
	vm := goja.New()
	loop := New(vm)

	// Test with a string rejection (which preserves the message)
	script := `
		async function failingFunc() {
			return Promise.reject("string error message");
		}
		failingFunc();
	`

	ctx := context.Background()
	result, err := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return vm.RunString(script)
	})

	require.NoError(t, err)

	_, extractErr := ExtractPromiseValue(result)
	require.Error(t, extractErr)
	assert.Contains(t, extractErr.Error(), "string error message")
}
