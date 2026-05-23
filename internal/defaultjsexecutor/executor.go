// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultjsexecutor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	jsexecutorcontract "github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// JsExecutorOption contains configuration options for the JavaScript executor.
type JsExecutorOption struct {
	minPoolSize     uint32        // Minimum number of threads in the pool
	maxPoolSize     uint32        // Maximum number of threads in the pool
	queueSize       uint32        // Size of the task queue per thread
	threadTTL       time.Duration // Thread time-to-live for idle cleanup
	maxExecutions   uint32        // Maximum executions per thread before cleanup
	executeTimeout  time.Duration // Timeout for task execution
	createThreshold float64       // Queue load threshold for creating new threads (0.0-1.0)
	selectThreshold float64       // Queue load threshold for skipping busy threads (0.0-1.0)
}

var (
	errExecutorNotStarted = errors.New("js executor is not started")
	errExecutorReloading  = errors.New("js executor is reloading")
	errExecutorStopping   = errors.New("js executor is stopping")
	errExecutorStopped    = errors.New("js executor is stopped")
)

type executorState uint32

const (
	executorStateNew executorState = iota
	executorStateStarting
	executorStateRunning
	executorStateReloading
	executorStateStopping
	executorStateStopped
)

func (s executorState) String() string {
	switch s {
	case executorStateNew:
		return "new"
	case executorStateStarting:
		return "starting"
	case executorStateRunning:
		return "running"
	case executorStateReloading:
		return "reloading"
	case executorStateStopping:
		return "stopping"
	case executorStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// JsExecutor manages a pool of JavaScript execution threads.
type JsExecutor struct {
	options       *JsExecutorOption        // Configuration options
	pool          *pool                    // Thread pool
	engineFactory jsengine.JsEngineFactory // JavaScript engine factory function

	// Use atomic pointer for lock-free, zero-copy reads of jsScripts.
	jsScriptsPtr unsafe.Pointer // Points to []*JsScript (atomic access)

	logger *slog.Logger // Logger instance

	state uint32

	controlMu     sync.Mutex
	stopRequested bool
	stopDone      chan struct{}
	stopErr       error
}

func (e *JsExecutor) getState() executorState {
	return executorState(atomic.LoadUint32(&e.state))
}

func (e *JsExecutor) setState(state executorState) {
	atomic.StoreUint32(&e.state, uint32(state))
}

func (e *JsExecutor) ensureStopDoneLocked() chan struct{} {
	if e.stopDone == nil {
		e.stopDone = make(chan struct{})
	}
	return e.stopDone
}

func (e *JsExecutor) completeStopLocked(err error) {
	e.stopErr = err
	e.stopRequested = false
	e.setState(executorStateStopped)
	if e.stopDone != nil {
		close(e.stopDone)
		e.stopDone = nil
	}
}

func (e *JsExecutor) executeStateError() error {
	switch e.getState() {
	case executorStateNew, executorStateStarting:
		return errExecutorNotStarted
	case executorStateReloading:
		return errExecutorReloading
	case executorStateStopping:
		return errExecutorStopping
	case executorStateStopped:
		return errExecutorStopped
	default:
		return nil
	}
}

// GetJsScripts returns the current initialization scripts (no copy, read-only).
func (e *JsExecutor) GetJsScripts() []*jsengine.JsScript {
	ptr := atomic.LoadPointer(&e.jsScriptsPtr)
	if ptr == nil {
		return nil
	}
	return *(*[]*jsengine.JsScript)(ptr)
}

func (e *JsExecutor) RuntimeInfo() jsexecutorcontract.RuntimeInfo {
	if e == nil || e.options == nil {
		return jsexecutorcontract.RuntimeInfo{}
	}
	return jsexecutorcontract.RuntimeInfo{
		MinPoolSize: e.options.minPoolSize,
		MaxPoolSize: e.options.maxPoolSize,
	}
}

// SetJsScripts atomically sets new initialization scripts.
func (e *JsExecutor) SetJsScripts(scripts []*jsengine.JsScript) {
	if len(scripts) == 0 {
		atomic.StorePointer(&e.jsScriptsPtr, nil)
		return
	}

	// Create immutable copy once during write.
	newScripts := make([]*jsengine.JsScript, len(scripts))
	copy(newScripts, scripts)

	// Atomically replace the pointer.
	atomic.StorePointer(&e.jsScriptsPtr, unsafe.Pointer(&newScripts))
}

// AppendJsScripts appends new scripts to the existing initialization scripts.
func (e *JsExecutor) AppendJsScripts(scripts ...*jsengine.JsScript) {
	if len(scripts) == 0 {
		return
	}

	for {
		oldPtr := atomic.LoadPointer(&e.jsScriptsPtr)

		var oldScripts []*jsengine.JsScript
		if oldPtr != nil {
			oldScripts = *(*[]*jsengine.JsScript)(oldPtr)
		}

		// Create a new slice with the combined content.
		newScripts := make([]*jsengine.JsScript, len(oldScripts)+len(scripts))
		copy(newScripts, oldScripts)
		copy(newScripts[len(oldScripts):], scripts)

		// Attempt to atomically swap the pointer.
		if atomic.CompareAndSwapPointer(&e.jsScriptsPtr, oldPtr, unsafe.Pointer(&newScripts)) {
			break // Success
		}
		// If CAS fails, another goroutine updated the pointer. Retry the loop.
	}
}

// Start initializes and starts the executor thread pool.
func (e *JsExecutor) Start() error {
	if e.pool == nil {
		return fmt.Errorf("thread pool is not initialized")
	}

	e.controlMu.Lock()
	switch e.getState() {
	case executorStateNew:
		e.stopErr = nil
		e.stopRequested = false
		e.setState(executorStateStarting)
	case executorStateStarting, executorStateRunning, executorStateReloading:
		e.controlMu.Unlock()
		return nil
	case executorStateStopping:
		e.controlMu.Unlock()
		return errExecutorStopping
	case executorStateStopped:
		e.controlMu.Unlock()
		return errExecutorStopped
	default:
		e.controlMu.Unlock()
		return nil
	}
	e.controlMu.Unlock()

	err := e.pool.start()

	e.controlMu.Lock()
	if err != nil {
		e.setState(executorStateNew)
		e.controlMu.Unlock()
		return err
	}
	if e.stopRequested {
		e.ensureStopDoneLocked()
		e.setState(executorStateStopping)
		e.controlMu.Unlock()
		stopErr := e.pool.stop()
		e.controlMu.Lock()
		e.completeStopLocked(stopErr)
		e.controlMu.Unlock()
		if stopErr != nil {
			return stopErr
		}
		return errExecutorStopping
	}
	e.stopErr = nil
	e.setState(executorStateRunning)
	e.controlMu.Unlock()
	return nil
}

// Execute executes a JavaScript request and returns the response.
func (e *JsExecutor) Execute(ctx context.Context, request *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.pool == nil {
		return nil, fmt.Errorf("thread pool is not initialized")
	}
	if err := e.executeStateError(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Apply a hard timeout via context so the JS runtime can be interrupted.
	// If the caller already provided an earlier deadline, keep it.
	execCtx := ctx
	var cancel context.CancelFunc
	if e.options.executeTimeout > 0 {
		if dl, ok := ctx.Deadline(); !ok || time.Until(dl) > e.options.executeTimeout {
			execCtx, cancel = context.WithTimeout(ctx, e.options.executeTimeout)
		}
	}
	if cancel != nil {
		defer cancel()
	}

	start := time.Now()
	defer func() {
		if e.logger != nil {
			elapsed := time.Since(start)
			requestID := ""
			service := ""
			if request != nil {
				requestID = request.Id
				service = request.Service
			}
			e.logger.Debug("js executor request completed",
				"request_id", requestID,
				"service", service,
				"duration_ms", elapsed.Milliseconds(),
			)
		}
	}()
	task := newTask(execCtx, request)
	return e.pool.execute(task)
}

// Stop stops the executor and shuts down all threads.
func (e *JsExecutor) Stop() error {
	if e.pool == nil {
		return fmt.Errorf("thread pool is not initialized")
	}

	e.controlMu.Lock()
	switch e.getState() {
	case executorStateNew:
		e.stopErr = nil
		e.stopRequested = false
		e.setState(executorStateStopped)
		e.controlMu.Unlock()
		return nil
	case executorStateStarting, executorStateReloading:
		done := e.ensureStopDoneLocked()
		e.stopRequested = true
		if e.getState() == executorStateReloading && e.pool != nil && e.pool.getState() == poolStateRunning {
			e.pool.setState(poolStateStopping)
		}
		e.controlMu.Unlock()
		<-done
		e.controlMu.Lock()
		err := e.stopErr
		e.controlMu.Unlock()
		return err
	case executorStateRunning:
		e.stopRequested = true
		e.ensureStopDoneLocked()
		e.setState(executorStateStopping)
		e.controlMu.Unlock()
		err := e.pool.stop()
		e.controlMu.Lock()
		e.completeStopLocked(err)
		e.controlMu.Unlock()
		return err
	case executorStateStopping:
		done := e.ensureStopDoneLocked()
		e.controlMu.Unlock()
		<-done
		e.controlMu.Lock()
		err := e.stopErr
		e.controlMu.Unlock()
		return err
	case executorStateStopped:
		err := e.stopErr
		e.controlMu.Unlock()
		return err
	default:
		e.controlMu.Unlock()
		return nil
	}
}

// Reload reloads all threads with new initialization scripts.
func (e *JsExecutor) Reload(scripts ...*jsengine.JsScript) error {
	if e.pool == nil {
		return fmt.Errorf("thread pool is not initialized")
	}

	e.controlMu.Lock()
	switch e.getState() {
	case executorStateNew, executorStateStarting:
		e.controlMu.Unlock()
		return errExecutorNotStarted
	case executorStateRunning:
		if e.stopRequested {
			e.controlMu.Unlock()
			return errExecutorStopping
		}
		e.setState(executorStateReloading)
	case executorStateReloading:
		e.controlMu.Unlock()
		return errExecutorReloading
	case executorStateStopping:
		e.controlMu.Unlock()
		return errExecutorStopping
	case executorStateStopped:
		e.controlMu.Unlock()
		return errExecutorStopped
	default:
		e.controlMu.Unlock()
		return nil
	}

	if scripts != nil {
		e.SetJsScripts(scripts) // Use safe setter method.
	}
	e.controlMu.Unlock()

	err := e.pool.reload()

	e.controlMu.Lock()
	if e.stopRequested {
		e.ensureStopDoneLocked()
		e.setState(executorStateStopping)
		e.controlMu.Unlock()
		stopErr := e.pool.stop()
		e.controlMu.Lock()
		e.completeStopLocked(stopErr)
		e.controlMu.Unlock()
		if stopErr != nil {
			return stopErr
		}
		return errExecutorStopping
	}
	e.setState(executorStateRunning)
	e.controlMu.Unlock()
	return err
}

// WithJsEngine configures the JavaScript engine builder and options.
func WithJsEngine(engineFactory jsengine.JsEngineFactory) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		executor.engineFactory = engineFactory
	}
}

// WithLogger configures the logger for the executor.
func WithLogger(logger *slog.Logger) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		executor.logger = logger
	}
}

// WithJsScripts configures the initialization scripts.
func WithJsScripts(scripts ...*jsengine.JsScript) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if len(scripts) > 0 {
			executor.SetJsScripts(scripts) // Use safe setter method.
		}
	}
}

// WithMinPoolSize sets the minimum number of threads in the pool.
func WithMinPoolSize(size uint32) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if size > 0 {
			executor.options.minPoolSize = size
		}
	}
}

// WithMaxPoolSize sets the maximum number of threads in the pool.
func WithMaxPoolSize(size uint32) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if size > 0 {
			executor.options.maxPoolSize = size
		}
	}
}

// WithQueueSize sets the size of the task queue per thread.
func WithQueueSize(size uint32) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if size > 0 {
			executor.options.queueSize = size
		}
	}
}

// WithThreadTTL sets the time-to-live for idle threads.
func WithThreadTTL(ttl time.Duration) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if ttl > 0 {
			executor.options.threadTTL = ttl
		}
	}
}

// WithMaxExecutions sets the maximum executions per thread before cleanup.
func WithMaxExecutions(max uint32) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if max > 0 {
			executor.options.maxExecutions = max
		}
	}
}

// WithExecuteTimeout sets the timeout for task execution.
func WithExecuteTimeout(timeout time.Duration) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if timeout > 0 {
			executor.options.executeTimeout = timeout
		}
	}
}

// WithCreateThreshold sets the queue load threshold for creating new threads.
func WithCreateThreshold(threshold float64) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if threshold > 0 && threshold <= 1.0 {
			executor.options.createThreshold = threshold
		}
	}
}

// WithSelectThreshold sets the queue load threshold for skipping busy threads.
func WithSelectThreshold(threshold float64) func(*JsExecutor) {
	return func(executor *JsExecutor) {
		if threshold > 0 && threshold <= 1.0 {
			executor.options.selectThreshold = threshold
		}
	}
}

// NewRuntimeExecutor creates an unstarted executor for long-lived runtime owners.
// Runtime plugin scope resolution is delegated to the provider-aware engine
// factory seam, so callers can keep a stable executor while executions rebind to
// the active runtime context.
func NewRuntimeExecutor(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...func(*JsExecutor)) (*JsExecutor, error) {
	return newExecutor(runtimeScope, authenticator, opts...)
}

// NewCompilerExecutor creates an unstarted executor for compiler/bundling owners.
// Callers that need transactional ownership should pass the tx-bound scope
// before starting the executor.
func NewCompilerExecutor(runtimeScope scope.Scope, opts ...func(*JsExecutor)) (*JsExecutor, error) {
	return newExecutor(runtimeScope, nil, opts...)
}

// newExecutor is the package-local constructor shared by the public runtime and
// compiler entry points.
func newExecutor(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...func(*JsExecutor)) (*JsExecutor, error) {
	cpuCount := runtime.GOMAXPROCS(0)

	executor := &JsExecutor{
		logger: slog.Default(), // Default logger
		options: &JsExecutorOption{
			minPoolSize:     uint32(cpuCount),     // Default to CPU count
			maxPoolSize:     uint32(cpuCount * 2), // Default to 2x CPU count
			queueSize:       256,                  // Default queue size
			threadTTL:       0,                    // No TTL by default
			maxExecutions:   0,                    // No execution limit by default
			executeTimeout:  0,                    // No execution timeout by default
			createThreshold: 0.5,                  // Create new thread at 50% load
			selectThreshold: 0.75,                 // Skip thread at 75% load
		},
		state: uint32(executorStateNew),
	}

	// Apply configuration options.
	for _, opt := range opts {
		opt(executor)
	}

	if executor.engineFactory == nil {
		engineFactory := jsengine.NewJsEngineFactory(runtimeScope, authenticator)
		if engineFactory == nil {
			return nil, fmt.Errorf("JavaScript engine factory is not registered: %s", runtimeOptionsFromScope(runtimeScope).serverJsEngineFactory)
		}
		executor.engineFactory = engineFactory
	}

	executor.pool = newPool(executor)

	return executor, nil
}
