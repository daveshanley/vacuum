package jsruleset

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/config"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin/javascript/eventloop"
	"github.com/daveshanley/vacuum/plugin/javascript/eventloop/fetch"
	"github.com/dop251/goja"
	"github.com/pb33f/go-yaml"
)

// Scope owns one module instance and serializes JavaScript calls for a document.
// Close cancels pending work and waits for the VM owner before releasing it.
type Scope struct {
	paths      map[*yaml.Node]map[*yaml.Node][]any
	native     map[string]model.RuleFunction
	vm         *goja.Runtime
	loop       *eventloop.EventLoop
	ctx        context.Context
	cancel     context.CancelFunc
	gate       chan struct{}
	once       sync.Once
	functions  map[string]goja.Callable
	formats    map[string]goja.Callable
	normalized []byte
	document   goja.Value
	source     string
	current    *model.RuleFunctionContext
}

// Open initializes a fresh module instance and checks its exported contract
// against preparation. Module initialization must be deterministic.
func (p *Prepared) Open(ctx context.Context, fetchConfig *config.FetchConfig) (*Scope, error) {
	s, err := p.open(ctx, fetchConfig)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(s.normalized, p.normalized) {
		s.Close()
		return nil, fmt.Errorf("JavaScript ruleset changed its exported contract during initialization: %s", p.entry)
	}
	return s, nil
}
func (p *Prepared) open(parent context.Context, fetchConfig *config.FetchConfig) (*Scope, error) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	s := &Scope{vm: goja.New(), ctx: ctx, cancel: cancel, gate: make(chan struct{}, 1), functions: map[string]goja.Callable{}, formats: map[string]goja.Callable{}}
	s.paths = make(map[*yaml.Node]map[*yaml.Node][]any)
	s.gate <- struct{}{}
	s.loop = eventloop.New(s.vm)
	fm, err := fetch.NewFetchModuleFromConfig(s.loop, fetchConfig)
	if err != nil {
		s.Close()
		return nil, err
	}
	fm.Register()
	timeout, stop := context.WithTimeout(ctx, initializationTimeout)
	defer stop()
	err = s.run(timeout, func() error {
		vm := s.vm
		console := vm.NewObject()
		log := func(call goja.FunctionCall) goja.Value {
			values := make([]any, len(call.Arguments))
			for i, v := range call.Arguments {
				values[i] = v.String()
			}
			fmt.Fprintln(os.Stderr, values...)
			return goja.Undefined()
		}
		for _, name := range []string{"log", "info", "warn", "error", "debug"} {
			_ = console.Set(name, log)
		}
		_ = vm.Set("console", console)
		if _, err := vm.RunString("var module={exports:{}};var exports=module.exports;"); err != nil {
			return err
		}
		_ = vm.Set("require", func(goja.FunctionCall) goja.Value { panic(vm.NewTypeError("runtime require is unsupported")) })
		ids := map[*goja.Object]string{}
		_ = vm.Set("__register", func(call goja.FunctionCall) goja.Value {
			fn, ok := goja.AssertFunction(call.Argument(0))
			if !ok {
				panic(vm.NewTypeError("expected callable"))
			}
			object := call.Argument(0).ToObject(vm)
			kind := call.Argument(1).String()
			key := ids[object]
			if key == "" {
				key = fmt.Sprintf("__vacuum_js_%s_%d", kind, len(ids))
				ids[object] = key
			}
			if kind == "format" {
				s.formats[key] = fn
			} else {
				s.functions[key] = fn
			}
			return vm.ToValue(key)
		})
		s.registerCompatibility()
		if _, e := s.loop.Run(timeout, func(vm *goja.Runtime) (goja.Value, error) { return vm.RunProgram(p.program) }); e != nil {
			return e
		}
		value := vm.Get("module").ToObject(vm).Get("exports")
		if obj, ok := value.(*goja.Object); ok && obj.Get("__esModule") != nil && obj.Get("__esModule").ToBoolean() {
			value = obj.Get("default")
		}
		normalizer, e := vm.RunString(normalizeSource)
		if e != nil {
			return e
		}
		normalize, _ := goja.AssertFunction(normalizer)
		normalized, e := s.loop.Run(timeout, func(*goja.Runtime) (goja.Value, error) { return normalize(goja.Undefined(), value) })
		if e != nil {
			return e
		}
		s.normalized = []byte(normalized.String())
		return nil
	})
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("initialize ruleset %s: %w", p.entry, err)
	}
	return s, nil
}

// run includes time spent waiting for the VM in the caller's deadline. Interrupt
// completion is joined before clearing it, so it cannot reach the next caller.
func (s *Scope) run(ctx context.Context, fn func() error) error {
	if ctx == nil {
		ctx = s.ctx
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.gate:
	}
	defer func() { s.gate <- struct{}{} }()
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	combined, cancel := context.WithCancel(ctx)
	defer cancel()
	stopScope := context.AfterFunc(s.ctx, cancel)
	defer stopScope()
	interrupted := make(chan struct{})
	stop := context.AfterFunc(combined, func() { s.vm.Interrupt(combined.Err()); close(interrupted) })
	defer func() {
		if !stop() {
			<-interrupted
		}
		s.vm.ClearInterrupt()
	}()
	err := fn()
	if combined.Err() != nil {
		s.cancel()
		return combined.Err()
	}
	if err != nil {
		s.cancel()
	}
	return err
}

// Close stops work owned by this scope. It is safe to call more than once.
func (s *Scope) Close() {
	s.once.Do(func() {
		s.cancel()
		<-s.gate
		s.functions = nil
		s.formats = nil
		s.document = nil
		s.paths = nil
		s.native = nil
		s.current = nil
		s.vm = nil
		s.loop = nil
		s.gate <- struct{}{}
	})
}

// Functions returns adapters bound to this document scope.
func (s *Scope) Functions() map[string]model.RuleFunction {
	out := make(map[string]model.RuleFunction, len(s.functions))
	for name := range s.functions {
		out[name] = &adapter{name: name, scope: s}
	}
	return out
}

// Functions returns validation-only callable adapters for a prepared ruleset.
func (p *Prepared) Functions() map[string]model.RuleFunction {
	out := make(map[string]model.RuleFunction, len(p.names))
	for _, name := range p.names {
		out[name] = &adapter{name: name}
	}
	return out
}

// MatchFormats evaluates callable predicates against the original document once.
func (s *Scope) MatchFormats(ctx context.Context, data any, source string) (map[string]bool, error) {
	matches := map[string]bool{}
	deadline, cancel := context.WithTimeout(ctx, initializationTimeout)
	defer cancel()
	err := s.run(deadline, func() error {
		s.document = s.vm.ToValue(data)
		s.source = source
		for name, fn := range s.formats {
			v, e := s.loop.Run(deadline, func(vm *goja.Runtime) (goja.Value, error) {
				return fn(goja.Undefined(), s.document, vm.ToValue(source))
			})
			if e != nil {
				return e
			}
			resolved := s.loop.AwaitPromise(deadline, v)
			if resolved.Error != nil {
				return resolved.Error
			}
			if _, ok := resolved.Value.Export().(bool); !ok {
				return fmt.Errorf("format predicate %s must return boolean", name)
			}
			matches[name] = resolved.Value.ToBoolean()
		}
		return nil
	})
	return matches, err
}

func invocationContext(rfc model.RuleFunctionContext) (context.Context, context.CancelFunc) {
	ctx := rfc.ExecutionContext
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := rfc.RuleTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}
