// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsexecutor

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// RuntimeFactory builds runtime executors.
// Implementations are usually wired through side-effect imports.
type RuntimeFactory func(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error)

// CompilerFactory builds compiler executors.
// Implementations are usually wired through side-effect imports.
type CompilerFactory func(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error)

// Option mutates FactoryOptions before the selected factory is invoked.
type Option func(*FactoryOptions)

// FactoryOptions is the normalized option payload resolved from public
// With* option functions and consumed by concrete factory implementations.
type FactoryOptions struct {
	JsEngineFactory jsengine.JsEngineFactory
	Logger          *slog.Logger
	JsScripts       []*jsengine.JsScript
	MinPoolSize     uint32
	MaxPoolSize     uint32
	QueueSize       uint32
	ThreadTTL       time.Duration
	MaxExecutions   uint32
	ExecuteTimeout  time.Duration
	CreateThreshold float64
	SelectThreshold float64
}

const (
	defaultRuntimeFactoryName        = "default"
	defaultCompilerFactoryName       = "default"
	errorPrefixFactoryNotRegistered  = "factory-not-registered"
	errorPrefixFactoryPairIncomplete = "factory-pair-incomplete"
)

var (
	factoryMu         sync.RWMutex
	runtimeFactories  = make(map[string]RuntimeFactory)
	compilerFactories = make(map[string]CompilerFactory)
)

func registerFactory[T any](name string, isNil bool, factories map[string]T, factory T) {
	if name == "" || isNil {
		return
	}
	factoryMu.Lock()
	defer factoryMu.Unlock()
	factories[name] = factory
}

// RegisterRuntimeFactory registers a named runtime executor factory.
func RegisterRuntimeFactory(name string, factory RuntimeFactory) {
	registerFactory(name, factory == nil, runtimeFactories, factory)
}

// RegisterCompilerFactory registers a named compiler executor factory.
func RegisterCompilerFactory(name string, factory CompilerFactory) {
	registerFactory(name, factory == nil, compilerFactories, factory)
}

// RuntimeFactoryExists reports whether a runtime factory is registered.
func RuntimeFactoryExists(name string) bool {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	return factoryExists(runtimeFactories, name)
}

// CompilerFactoryExists reports whether a compiler factory is registered.
func CompilerFactoryExists(name string) bool {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	return factoryExists(compilerFactories, name)
}

func factoryExists[T any](factories map[string]T, name string) bool {
	_, ok := factories[name]
	return ok
}

func sortedFactoryKeys[T any](factories map[string]T) []string {
	keys := make([]string, 0, len(factories))
	for name := range factories {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

// RuntimeFactoryKeys returns registered runtime factory names.
func RuntimeFactoryKeys() []string {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	return sortedFactoryKeys(runtimeFactories)
}

// CompilerFactoryKeys returns registered compiler factory names.
func CompilerFactoryKeys() []string {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	return sortedFactoryKeys(compilerFactories)
}

func lookupFactoryPair(name string, runtime map[string]RuntimeFactory, compiler map[string]CompilerFactory) (RuntimeFactory, CompilerFactory) {
	runtimeFactory, _ := runtime[name]
	compilerFactory, _ := compiler[name]
	return runtimeFactory, compilerFactory
}

func lookupRegisteredFactoryPair(name string) (RuntimeFactory, CompilerFactory) {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	return lookupFactoryPair(name, runtimeFactories, compilerFactories)
}

func newFactoryNotRegisteredError(kind, name string) error {
	return fmt.Errorf("%s: %s executor factory is not registered: %s", errorPrefixFactoryNotRegistered, kind, name)
}

func newFactoryPairIncompleteError(kind, missingKind, name string) error {
	return fmt.Errorf("%s: %s executor factory pair is incomplete: %s executor factory is not registered: %s", errorPrefixFactoryPairIncomplete, kind, missingKind, name)
}

func resolveRuntimeFactory(name string) (RuntimeFactory, error) {
	runtimeFactory, compilerFactory := lookupRegisteredFactoryPair(name)
	if runtimeFactory == nil {
		return nil, newFactoryNotRegisteredError("runtime", name)
	}
	if compilerFactory == nil {
		return nil, newFactoryPairIncompleteError("runtime", "compiler", name)
	}
	return runtimeFactory, nil
}

func resolveCompilerFactory(name string) (CompilerFactory, error) {
	runtimeFactory, compilerFactory := lookupRegisteredFactoryPair(name)
	if compilerFactory == nil {
		return nil, newFactoryNotRegisteredError("compiler", name)
	}
	if runtimeFactory == nil {
		return nil, newFactoryPairIncompleteError("compiler", "runtime", name)
	}
	return compilerFactory, nil
}

func resolveFactoryName(runtimeScope scope.Scope, defaultFactoryName string) string {
	if runtimeScope == nil {
		return defaultFactoryName
	}
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	if !hasServerOpts {
		return defaultFactoryName
	}
	factoryName := strings.TrimSpace(serverOpts.JsExecutorFactory)
	if factoryName == "" {
		return defaultFactoryName
	}
	return factoryName
}

func resolveRuntimeFactoryName(runtimeScope scope.Scope) string {
	return resolveFactoryName(runtimeScope, defaultRuntimeFactoryName)
}

func resolveCompilerFactoryName(runtimeScope scope.Scope) string {
	return resolveFactoryName(runtimeScope, defaultCompilerFactoryName)
}

func cloneScripts(scripts []*jsengine.JsScript) []*jsengine.JsScript {
	if len(scripts) == 0 {
		return nil
	}
	cloned := make([]*jsengine.JsScript, len(scripts))
	copy(cloned, scripts)
	return cloned
}

func cloneFactoryOptions(options FactoryOptions) FactoryOptions {
	cloned := options
	cloned.JsScripts = cloneScripts(options.JsScripts)
	return cloned
}

// ResolveFactoryOptions normalizes With* options into a stable option payload
// consumed by registered runtime/compiler factories.
func ResolveFactoryOptions(opts ...Option) FactoryOptions {
	options := FactoryOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&options)
	}
	return cloneFactoryOptions(options)
}

// WithJsEngine configures the JavaScript engine builder and options.
func WithJsEngine(engineFactory jsengine.JsEngineFactory) Option {
	return func(options *FactoryOptions) {
		if options == nil {
			return
		}
		options.JsEngineFactory = engineFactory
	}
}

// WithLogger configures the logger for the executor.
func WithLogger(logger *slog.Logger) Option {
	return func(options *FactoryOptions) {
		if options == nil {
			return
		}
		options.Logger = logger
	}
}

// WithJsScripts configures the initialization scripts.
func WithJsScripts(scripts ...*jsengine.JsScript) Option {
	return func(options *FactoryOptions) {
		if len(scripts) == 0 {
			return
		}
		if options == nil {
			return
		}
		options.JsScripts = cloneScripts(scripts)
	}
}

// WithMinPoolSize sets the minimum number of threads in the pool.
func WithMinPoolSize(size uint32) Option {
	return func(options *FactoryOptions) {
		if size == 0 {
			return
		}
		if options == nil {
			return
		}
		options.MinPoolSize = size
	}
}

// WithMaxPoolSize sets the maximum number of threads in the pool.
func WithMaxPoolSize(size uint32) Option {
	return func(options *FactoryOptions) {
		if size == 0 {
			return
		}
		if options == nil {
			return
		}
		options.MaxPoolSize = size
	}
}

// WithQueueSize sets the size of the task queue per thread.
func WithQueueSize(size uint32) Option {
	return func(options *FactoryOptions) {
		if size == 0 {
			return
		}
		if options == nil {
			return
		}
		options.QueueSize = size
	}
}

// WithThreadTTL sets the time-to-live for idle threads.
func WithThreadTTL(ttl time.Duration) Option {
	return func(options *FactoryOptions) {
		if ttl <= 0 {
			return
		}
		if options == nil {
			return
		}
		options.ThreadTTL = ttl
	}
}

// WithMaxExecutions sets the maximum executions per thread before cleanup.
func WithMaxExecutions(max uint32) Option {
	return func(options *FactoryOptions) {
		if max == 0 {
			return
		}
		if options == nil {
			return
		}
		options.MaxExecutions = max
	}
}

// WithExecuteTimeout sets the timeout for task execution.
func WithExecuteTimeout(timeout time.Duration) Option {
	return func(options *FactoryOptions) {
		if timeout <= 0 {
			return
		}
		if options == nil {
			return
		}
		options.ExecuteTimeout = timeout
	}
}

// WithCreateThreshold sets the queue load threshold for creating new threads.
func WithCreateThreshold(threshold float64) Option {
	return func(options *FactoryOptions) {
		if threshold <= 0 || threshold > 1.0 {
			return
		}
		if options == nil {
			return
		}
		options.CreateThreshold = threshold
	}
}

// WithSelectThreshold sets the queue load threshold for skipping busy threads.
func WithSelectThreshold(threshold float64) Option {
	return func(options *FactoryOptions) {
		if threshold <= 0 || threshold > 1.0 {
			return
		}
		if options == nil {
			return
		}
		options.SelectThreshold = threshold
	}
}

// NewRuntimeExecutor creates an unstarted executor for long-lived runtime owners.
func NewRuntimeExecutor(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) (JsExecutor, error) {
	factoryName := resolveRuntimeFactoryName(runtimeScope)
	factory, err := resolveRuntimeFactory(factoryName)
	if err != nil {
		return nil, err
	}
	executor, err := factory(runtimeScope, authenticator, opts...)
	if err != nil {
		return nil, err
	}
	if executor == nil {
		return nil, fmt.Errorf("runtime executor factory returned nil executor: %s", factoryName)
	}
	return executor, nil
}

// NewCompilerExecutor creates an unstarted executor for compiler/bundling owners.
func NewCompilerExecutor(runtimeScope scope.Scope, opts ...Option) (JsExecutor, error) {
	factoryName := resolveCompilerFactoryName(runtimeScope)
	factory, err := resolveCompilerFactory(factoryName)
	if err != nil {
		return nil, err
	}
	executor, err := factory(runtimeScope, opts...)
	if err != nil {
		return nil, err
	}
	if executor == nil {
		return nil, fmt.Errorf("compiler executor factory returned nil executor: %s", factoryName)
	}
	return executor, nil
}
