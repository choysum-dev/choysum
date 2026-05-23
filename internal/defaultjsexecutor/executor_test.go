// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultjsexecutor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type JsScript = jsengine.JsScript
type JsRequest = jsengine.JsRequest
type JsResponse = jsengine.JsResponse
type JsEngine = jsengine.JsEngine
type JsEngineFactory = jsengine.JsEngineFactory

type executorTestScope struct {
	ctx context.Context
	cfg *config.Config
}

func (e *executorTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *executorTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *executorTestScope) Session() *scope.Session { return nil }
func (e *executorTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *executorTestScope) Context() context.Context { return e.ctx }
func (e *executorTestScope) Logger() *slog.Logger     { return slog.Default() }
func (e *executorTestScope) Config() *config.Config   { return e.cfg }
func (e *executorTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newExecutorTestScope() *executorTestScope {
	return &executorTestScope{
		ctx: context.Background(),
		cfg: &config.Config{Server: config.NewDefaultServerConfig()},
	}
}

func newExecutorForTest(opts ...func(*JsExecutor)) (*JsExecutor, error) {
	return newExecutor(newExecutorTestScope(), nil, opts...)
}

func nilTaskContext() context.Context { return nil }

type nilFactoryInputScope struct {
	*executorTestScope
}

func (s *nilFactoryInputScope) FactoryInput() scope.FactoryInput { return nil }

func newLifecycleBranchTestExecutor() *JsExecutor {
	executor := &JsExecutor{
		options: &JsExecutorOption{
			minPoolSize: 1,
			maxPoolSize: 1,
			queueSize:   1,
		},
		engineFactory: mockEngineFactory(),
		state:         uint32(executorStateNew),
	}
	executor.pool = newPool(executor)
	return executor
}

// mockEngine is a simple mock implementation of JsEngine for testing.
type mockEngine struct {
	mu          sync.Mutex  // Mutex for concurrent access
	loadCalled  bool        // Whether Reload was called
	closeCalled bool        // Whether Close was called
	initScripts []*JsScript // Scripts passed to Init/Reload
	executedReq *JsRequest  // Last executed request
	executeResp *JsResponse // Response to return from Execute
	executeErr  error       // Error to return from Execute

	loadFunc    func(scripts []*JsScript) error           // Custom Load behavior (if set)
	executeFunc func(req *JsRequest) (*JsResponse, error) // Custom Execute behavior (if set)
	closeFunc   func() error                              // Custom Close behavior (if set)
}

// Load mocks the initialization of the JavaScript engine.
func (m *mockEngine) Load(scripts []*JsScript) error {
	m.mu.Lock()
	m.loadCalled = true
	m.initScripts = scripts
	m.mu.Unlock()
	// Check for a specific "bad" script to simulate a load error.
	for _, script := range scripts {
		if script.FileName == "bad.js" {
			fmt.Printf("Mock load failed due to bad script: %s\n", script.FileName)
			return errors.New("load failed due to bad script")
		}
	}
	if m.loadFunc != nil {
		return m.loadFunc(scripts)
	}
	return nil
}

// Execute mocks executing a JavaScript request.
func (m *mockEngine) Execute(ctx context.Context, req *JsRequest) (*JsResponse, error) {
	m.mu.Lock()
	m.executedReq = req
	m.mu.Unlock()
	_ = ctx
	if m.executeFunc != nil {
		return m.executeFunc(req)
	}
	return m.executeResp, m.executeErr
}

// Close mocks closing the JavaScript engine.
func (m *mockEngine) Close() error {
	m.closeCalled = true
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockEngineFactory returns a new mockEngine instance as JsEngineFactory.
func mockEngineFactory() JsEngineFactory {
	return func() (JsEngine, error) {
		return &mockEngine{
			executeFunc: func(req *JsRequest) (*JsResponse, error) {
				return &JsResponse{Id: req.Id, Result: "ok"}, nil
			},
		}, nil
	}
}

type reloadTracker struct {
	mu      sync.Mutex
	created int
	closed  int
}

func (r *reloadTracker) newEngine() jsengine.JsEngine {
	r.mu.Lock()
	r.created++
	r.mu.Unlock()
	return &singleLoadEngine{tracker: r}
}

func (r *reloadTracker) snapshot() (created int, closed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.created, r.closed
}

type singleLoadEngine struct {
	tracker *reloadTracker
	loaded  bool
}

func (e *singleLoadEngine) Load(_ []*jsengine.JsScript) error {
	if e.loaded {
		return fmt.Errorf("load called twice on same engine")
	}
	e.loaded = true
	return nil
}

func (e *singleLoadEngine) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{Id: req.Id, Context: map[string]interface{}{}}, nil
}

func (e *singleLoadEngine) Close() error {
	if e.tracker != nil {
		e.tracker.mu.Lock()
		e.tracker.closed++
		e.tracker.mu.Unlock()
	}
	return nil
}

type blockingEngine struct{}

func (b *blockingEngine) Load(_ []*jsengine.JsScript) error { return nil }

func (b *blockingEngine) Execute(ctx context.Context, _ *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (b *blockingEngine) Close() error { return nil }

type blockingReloadTracker struct {
	mu                sync.Mutex
	created           int
	reloadLoadReady   chan struct{}
	releaseReloadLoad chan struct{}
	reloadOnce        sync.Once
	blockReloadLoads  atomic.Bool
}

func (r *blockingReloadTracker) newEngine() jsengine.JsEngine {
	r.mu.Lock()
	r.created++
	id := r.created
	r.mu.Unlock()
	return &blockingReloadEngine{id: id, tracker: r}
}

func (r *blockingReloadTracker) enableReloadBlocking() {
	r.blockReloadLoads.Store(true)
}

type blockingReloadEngine struct {
	id      int
	tracker *blockingReloadTracker
}

func (e *blockingReloadEngine) Load(_ []*jsengine.JsScript) error {
	if e.tracker.blockReloadLoads.Load() {
		e.tracker.reloadOnce.Do(func() {
			close(e.tracker.reloadLoadReady)
		})
		<-e.tracker.releaseReloadLoad
	}
	return nil
}

func (e *blockingReloadEngine) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{Id: req.Id}, nil
}

func (e *blockingReloadEngine) Close() error { return nil }

type startWarmupBlocker struct {
	loadStarted chan struct{}
	releaseLoad chan struct{}
	startedOnce sync.Once
}

func newStartWarmupBlocker() *startWarmupBlocker {
	return &startWarmupBlocker{
		loadStarted: make(chan struct{}),
		releaseLoad: make(chan struct{}),
	}
}

func (b *startWarmupBlocker) factory() JsEngineFactory {
	return func() (JsEngine, error) {
		return &startWarmupBlockingEngine{blocker: b}, nil
	}
}

type startWarmupBlockingEngine struct {
	blocker *startWarmupBlocker
}

func (e *startWarmupBlockingEngine) Load(_ []*JsScript) error {
	e.blocker.startedOnce.Do(func() {
		close(e.blocker.loadStarted)
	})
	<-e.blocker.releaseLoad
	return nil
}

func (e *startWarmupBlockingEngine) Execute(_ context.Context, req *JsRequest) (*JsResponse, error) {
	return &JsResponse{Id: req.Id}, nil
}

func (e *startWarmupBlockingEngine) Close() error { return nil }

// TestJsExecutor_Start_Stop tests starting and stopping the executor.
func TestJsExecutor_Start_Stop(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Start the executor
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}

	// Stop the executor
	if err := executor.Stop(); err != nil {
		t.Fatalf("Failed to stop executor: %v", err)
	}
}

// TestJsExecutor_Execute tests executing a JavaScript request.
func TestJsExecutor_Execute(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer executor.Stop()

	req := &JsRequest{
		Id:      "1",
		Service: "testService",
		Args:    []interface{}{"foo"},
	}
	resp, err := executor.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp == nil || resp.Result != "ok" {
		t.Errorf("Unexpected response: %+v", resp)
	}
}

// TestJsExecutor_Reload tests reloading initialization scripts.
func TestJsExecutor_Reload(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer executor.Stop()

	scripts := []*JsScript{
		{FileName: "a.js", Content: "var a = 1;"},
	}
	if err := executor.Reload(scripts...); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	got := executor.GetJsScripts()
	if !reflect.DeepEqual(got, scripts) {
		t.Errorf("Reload did not set scripts correctly, got: %+v, want: %+v", got, scripts)
	}
}

func TestJsExecutor_ReloadRecreatesEngine(t *testing.T) {
	tracker := &reloadTracker{}

	exec := &JsExecutor{
		options: &JsExecutorOption{
			minPoolSize:     1,
			maxPoolSize:     1,
			queueSize:       1,
			threadTTL:       0,
			maxExecutions:   0,
			executeTimeout:  0,
			createThreshold: 0.5,
			selectThreshold: 0.75,
		},
		engineFactory: func() (jsengine.JsEngine, error) {
			return tracker.newEngine(), nil
		},
		logger: nil,
	}
	exec.SetJsScripts([]*jsengine.JsScript{{FileName: "init.js", Content: ""}})
	exec.pool = newPool(exec)

	if err := exec.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := exec.Reload(); err != nil {
		_ = exec.Stop()
		t.Fatalf("reload: %v", err)
	}

	created, closed := tracker.snapshot()
	if created < 2 {
		_ = exec.Stop()
		t.Fatalf("expected reload to create a new engine, created=%d", created)
	}
	if closed < 1 {
		_ = exec.Stop()
		t.Fatalf("expected old engine to be closed during reload, closed=%d", closed)
	}

	if err := exec.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestJsExecutor_ExecuteTimeoutInterruptsJS(t *testing.T) {
	exec := &JsExecutor{
		options: &JsExecutorOption{
			minPoolSize:     1,
			maxPoolSize:     1,
			queueSize:       1,
			threadTTL:       0,
			maxExecutions:   0,
			executeTimeout:  50 * time.Millisecond,
			createThreshold: 0.5,
			selectThreshold: 0.75,
		},
		engineFactory: func() (jsengine.JsEngine, error) { return &blockingEngine{}, nil },
		logger:        nil,
	}
	exec.pool = newPool(exec)

	if err := exec.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = exec.Stop() }()

	start := time.Now()
	_, err := exec.Execute(context.Background(), &jsengine.JsRequest{Id: "1", Service: "x"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("timeout took too long: %s", elapsed)
	}
}

// TestJsExecutor_WithLogger tests setting a custom logger.
func TestJsExecutor_WithLogger(t *testing.T) {
	logger := slog.Default()
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
		WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if executor.logger != logger {
		t.Errorf("Logger not set correctly")
	}
}

// TestJsExecutor_WithThresholds tests setting create and select thresholds.
func TestJsExecutor_WithThresholds(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
		WithCreateThreshold(0.7),
		WithSelectThreshold(0.9),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if executor.options.createThreshold != 0.7 || executor.options.selectThreshold != 0.9 {
		t.Errorf("Thresholds not set correctly: got %+v", executor.options)
	}
}

// TestJsExecutor_WithInitScripts tests setting initialization scripts via option.
func TestJsExecutor_WithInitScripts(t *testing.T) {
	scripts := []*JsScript{
		{FileName: "init.js", Content: "var x = 1;"},
	}
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
		WithJsScripts(scripts...),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	got := executor.GetJsScripts()
	if !reflect.DeepEqual(got, scripts) {
		t.Errorf("Init scripts not set correctly, got: %+v, want: %+v", got, scripts)
	}
}

// TestJsExecutor_WithThreadTTL tests setting the thread time-to-live option.
func TestJsExecutor_WithThreadTTL(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
		WithThreadTTL(10*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if executor.options.threadTTL != 10*time.Second {
		t.Errorf("threadTTL not set correctly, got: %v, want: %v", executor.options.threadTTL, 10*time.Second)
	}
}

// TestJsExecutor_WithMaxExecutions tests setting the maxExecutions option.
func TestJsExecutor_WithMaxExecutions(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
		WithMaxExecutions(123),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if executor.options.maxExecutions != 123 {
		t.Errorf("maxExecutions not set correctly, got: %v, want: %v", executor.options.maxExecutions, 123)
	}
}

// TestJsExecutor_WithExecuteTimeout tests setting the executeTimeout option.
func TestJsExecutor_WithExecuteTimeout(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
		WithExecuteTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if executor.options.executeTimeout != 5*time.Second {
		t.Errorf("executeTimeout not set correctly, got: %v, want: %v", executor.options.executeTimeout, 5*time.Second)
	}
}

// TestJsExecutor_Execute_ErrorWhenPoolNotStarted tests error when executing without starting the pool.
func TestJsExecutor_Execute_ErrorWhenPoolNotStarted(t *testing.T) {
	executor := &JsExecutor{}
	_, err := executor.Execute(context.Background(), &JsRequest{Id: "1"})
	if err == nil {
		t.Error("Expected error when pool is not initialized")
	}
}

func TestJsExecutor_Execute_ErrorWhenNotStarted(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	_, err = executor.Execute(context.Background(), &JsRequest{Id: "1"})
	if !errors.Is(err, errExecutorNotStarted) {
		t.Fatalf("expected errExecutorNotStarted, got %v", err)
	}
}

// TestJsExecutor_Reload_ErrorWhenPoolNotStarted tests error when reloading without starting the pool.
func TestJsExecutor_Reload_ErrorWhenPoolNotStarted(t *testing.T) {
	executor := &JsExecutor{}
	err := executor.Reload(&JsScript{FileName: "a.js", Content: "var a = 1;"})
	if err == nil {
		t.Error("Expected error when pool is not initialized")
	}
}

func TestJsExecutor_Reload_ErrorWhenNotStarted(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	err = executor.Reload(&JsScript{FileName: "a.js", Content: "var a = 1;"})
	if !errors.Is(err, errExecutorNotStarted) {
		t.Fatalf("expected errExecutorNotStarted, got %v", err)
	}
}

// TestJsExecutor_Stop_ErrorWhenPoolNotStarted tests error when stopping without starting the pool.
func TestJsExecutor_Stop_ErrorWhenPoolNotStarted(t *testing.T) {
	executor := &JsExecutor{}
	err := executor.Stop()
	if err == nil {
		t.Error("Expected error when pool is not initialized")
	}
}

func TestJsExecutor_Stop_Idempotent(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	if err := executor.Stop(); err != nil {
		t.Fatalf("first stop failed: %v", err)
	}
	if err := executor.Stop(); err != nil {
		t.Fatalf("second stop failed: %v", err)
	}
	if state := executor.getState(); state != executorStateStopped {
		t.Fatalf("expected stopped state, got %s", state)
	}
}

func TestJsExecutor_StopRejectsNewExecute(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	if err := executor.Stop(); err != nil {
		t.Fatalf("Failed to stop executor: %v", err)
	}
	_, err = executor.Execute(context.Background(), &JsRequest{Id: "after-stop"})
	if !errors.Is(err, errExecutorStopped) {
		t.Fatalf("expected errExecutorStopped, got %v", err)
	}
}

func TestJsExecutor_ReloadRejectedDuringStop(t *testing.T) {
	tracker := &blockingReloadTracker{
		reloadLoadReady:   make(chan struct{}),
		releaseReloadLoad: make(chan struct{}),
	}
	executor, err := newExecutorForTest(
		WithJsEngine(func() (JsEngine, error) { return tracker.newEngine(), nil }),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	tracker.enableReloadBlocking()

	reloadDone := make(chan error, 1)
	go func() {
		reloadDone <- executor.Reload(&JsScript{FileName: "during-stop.js", Content: "var a = 1;"})
	}()

	select {
	case <-tracker.reloadLoadReady:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for reload to enter engine load")
	}

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- executor.Stop()
	}()
	close(tracker.releaseReloadLoad)

	if err := <-reloadDone; !errors.Is(err, errExecutorStopping) {
		t.Fatalf("expected errExecutorStopping, got %v", err)
	}
	if err := <-stopDone; err != nil {
		t.Fatalf("expected stop to succeed, got %v", err)
	}
}

func TestJsExecutor_StartRejectedAfterStop(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	if err := executor.Stop(); err != nil {
		t.Fatalf("Failed to stop executor: %v", err)
	}
	if err := executor.Start(); !errors.Is(err, errExecutorStopped) {
		t.Fatalf("expected errExecutorStopped, got %v", err)
	}
}

// TestJsExecutor_Start_ErrorWhenPoolNotInitialized tests error when starting without initializing the pool.
func TestJsExecutor_Start_ErrorWhenPoolNotInitialized(t *testing.T) {
	executor := &JsExecutor{}
	err := executor.Start()
	if err == nil {
		t.Error("Expected error when pool is not initialized")
	}
}

// TestJsExecutor_EngineErrorPropagation tests error propagation from the engine.
func TestJsExecutor_EngineErrorPropagation(t *testing.T) {
	engine := &mockEngine{
		executeErr: errors.New("mock execute error"),
	}
	executor, err := newExecutorForTest(
		WithJsEngine(func() (JsEngine, error) { return engine, nil }),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer executor.Stop()
	_, err = executor.Execute(context.Background(), &JsRequest{Id: "err"})
	if err == nil || err.Error() != "mock execute error" {
		t.Errorf("Expected engine error to propagate, got: %v", err)
	}
}

// TestNewExecutor_ErrorWhenNoEngineFactory tests error when no engine factory is provided.
func TestNewExecutor_ErrorWhenNoEngineFactory(t *testing.T) {
	runtimeScope := newExecutorTestScope()
	runtimeScope.cfg.Server.JsEngineFactory = "missing"
	_, err := newExecutor(runtimeScope, nil)
	if err == nil {
		t.Error("Expected error when engineFactory is nil")
	}
}

// TestJsExecutor_SetInitScripts_EmptyScripts tests setting and clearing init scripts.
func TestJsExecutor_SetInitScripts_EmptyScripts(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	// Set non-empty first
	scripts := []*JsScript{{FileName: "a.js", Content: "var a = 1;"}}
	executor.SetJsScripts(scripts)
	if got := executor.GetJsScripts(); !reflect.DeepEqual(got, scripts) {
		t.Errorf("Expected scripts to be set")
	}
	// Now set empty
	executor.SetJsScripts([]*JsScript{})
	if got := executor.GetJsScripts(); got != nil {
		t.Errorf("Expected GetJsScripts to return nil when set with empty slice, got: %+v", got)
	}
}

// TestJsExecutor_WithInitScripts_Empty tests WithJsScripts with no scripts.
func TestJsExecutor_WithInitScripts_Empty(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	WithJsScripts()(
		executor,
	)
	if got := executor.GetJsScripts(); got != nil {
		t.Errorf("Expected GetJsScripts to return nil when WithJsScripts is called with no scripts")
	}
}

// TestJsExecutor_AppendInitScripts tests appending initialization scripts.
func TestJsExecutor_AppendInitScripts(t *testing.T) {
	executor, err := newExecutorForTest(
		WithJsEngine(mockEngineFactory()),
	)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// 1. Append to an empty list
	script1 := &JsScript{FileName: "1.js"}
	executor.AppendJsScripts(script1)
	got := executor.GetJsScripts()
	if len(got) != 1 || got[0].FileName != "1.js" {
		t.Errorf("Append to empty failed. Got: %+v, Want: [%+v]", got, script1)
	}

	// 2. Append to an existing list
	script2 := &JsScript{FileName: "2.js"}
	executor.AppendJsScripts(script2)
	got = executor.GetJsScripts()
	expected := []*JsScript{script1, script2}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Append to existing failed. Got: %+v, Want: %+v", got, expected)
	}

	// 3. Append nothing
	executor.AppendJsScripts()
	got = executor.GetJsScripts()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Append nothing should not change scripts. Got: %+v, Want: %+v", got, expected)
	}
}

// TestJsExecutor_AppendInitScripts_Concurrent tests concurrent appends to init scripts.
func TestJsExecutor_AppendInitScripts_Concurrent(t *testing.T) {
	executor, _ := newExecutorForTest(WithJsEngine(mockEngineFactory()))
	var wg sync.WaitGroup
	numGoroutines := 100
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			executor.AppendJsScripts(&JsScript{FileName: fmt.Sprintf("s%d.js", i)})
		}(i)
	}
	wg.Wait()
	finalScripts := executor.GetJsScripts()
	if len(finalScripts) != numGoroutines {
		t.Fatalf("Expected %d scripts after concurrent appends, but got %d", numGoroutines, len(finalScripts))
	}
}

// TestJsExecutor_ConcurrentReloadAndExecute tests concurrent reload and execute calls.
func TestJsExecutor_ConcurrentReloadAndExecute(t *testing.T) {
	executor, _ := newExecutorForTest(WithJsEngine(mockEngineFactory()))
	_ = executor.Start()
	defer executor.Stop()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = executor.Reload(&JsScript{FileName: "a.js", Content: "var a=1;"})
		}()
		go func() {
			defer wg.Done()
			_, _ = executor.Execute(context.Background(), &JsRequest{Id: "c"})
		}()
	}
	wg.Wait()
}

func TestJsexecutor_StateStringers(t *testing.T) {
	t.Run("executorState", func(t *testing.T) {
		tests := []struct {
			state executorState
			want  string
		}{
			{executorStateNew, "new"},
			{executorStateStarting, "starting"},
			{executorStateRunning, "running"},
			{executorStateReloading, "reloading"},
			{executorStateStopping, "stopping"},
			{executorStateStopped, "stopped"},
			{executorState(999), "unknown"},
		}
		for _, tt := range tests {
			if got := tt.state.String(); got != tt.want {
				t.Fatalf("executorState(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		}
	})

	t.Run("poolState", func(t *testing.T) {
		tests := []struct {
			state poolState
			want  string
		}{
			{poolStateCold, "cold"},
			{poolStateRunning, "running"},
			{poolStateStopping, "stopping"},
			{poolStateStopped, "stopped"},
			{poolState(999), "unknown"},
		}
		for _, tt := range tests {
			if got := tt.state.String(); got != tt.want {
				t.Fatalf("poolState(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		}
	})

	t.Run("threadState", func(t *testing.T) {
		tests := []struct {
			state threadState
			want  string
		}{
			{threadStateInit, "init"},
			{threadStateRunning, "running"},
			{threadStateDraining, "draining"},
			{threadStateReloading, "reloading"},
			{threadStateRetiring, "retiring"},
			{threadStateStopped, "stopped"},
			{threadState(999), "unknown"},
		}
		for _, tt := range tests {
			if got := tt.state.String(); got != tt.want {
				t.Fatalf("threadState(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		}
	})
}

func TestJsExecutor_WithPoolSizingOptions(t *testing.T) {
	executor := &JsExecutor{options: &JsExecutorOption{minPoolSize: 1, maxPoolSize: 2, queueSize: 3}}

	WithMinPoolSize(0)(executor)
	WithMaxPoolSize(0)(executor)
	WithQueueSize(0)(executor)
	if executor.options.minPoolSize != 1 || executor.options.maxPoolSize != 2 || executor.options.queueSize != 3 {
		t.Fatalf("zero-value pool options should be ignored, got %+v", executor.options)
	}

	WithMinPoolSize(4)(executor)
	WithMaxPoolSize(5)(executor)
	WithQueueSize(6)(executor)
	if executor.options.minPoolSize != 4 || executor.options.maxPoolSize != 5 || executor.options.queueSize != 6 {
		t.Fatalf("positive pool options should be applied, got %+v", executor.options)
	}
}

func TestJsExecutor_RuntimeInfo(t *testing.T) {
	executor := &JsExecutor{options: &JsExecutorOption{minPoolSize: 3, maxPoolSize: 7}}
	runtimeInfo := executor.RuntimeInfo()
	if runtimeInfo.MinPoolSize != 3 || runtimeInfo.MaxPoolSize != 7 {
		t.Fatalf("RuntimeInfo() = %#v, want min=3 max=7", runtimeInfo)
	}
}

func TestJsExecutor_PublicConstructors(t *testing.T) {
	runtimeExecutor, err := NewRuntimeExecutor(newExecutorTestScope(), nil, WithJsEngine(mockEngineFactory()))
	if err != nil {
		t.Fatalf("NewRuntimeExecutor() error = %v", err)
	}
	if runtimeExecutor.pool == nil {
		t.Fatal("NewRuntimeExecutor() should initialize thread pool")
	}
	if runtimeExecutor.getState() != executorStateNew {
		t.Fatalf("NewRuntimeExecutor() state = %s, want new", runtimeExecutor.getState())
	}

	compilerExecutor, err := NewCompilerExecutor(newExecutorTestScope(), WithJsEngine(mockEngineFactory()))
	if err != nil {
		t.Fatalf("NewCompilerExecutor() error = %v", err)
	}
	if compilerExecutor.pool == nil {
		t.Fatal("NewCompilerExecutor() should initialize thread pool")
	}

	runtimeScope := newExecutorTestScope()
	runtimeScope.cfg.Server.JsEngineFactory = "missing"
	if _, err := NewRuntimeExecutor(runtimeScope, nil); err == nil {
		t.Fatal("expected NewRuntimeExecutor() to fail when engine factory is missing")
	}
}

func TestRuntimeOptionsFromScope(t *testing.T) {
	if got := runtimeOptionsFromScope(nil); got.serverJsEngineFactory != "" {
		t.Fatalf("runtimeOptionsFromScope(nil) = %+v, want empty", got)
	}

	noFactoryInputScope := &nilFactoryInputScope{executorTestScope: newExecutorTestScope()}
	if got := runtimeOptionsFromScope(noFactoryInputScope); got.serverJsEngineFactory != "" {
		t.Fatalf("runtimeOptionsFromScope(noFactoryInputScope) = %+v, want empty", got)
	}

	runtimeScope := newExecutorTestScope()
	runtimeScope.cfg.Server.JsEngineFactory = "quickjs"
	if got := runtimeOptionsFromScope(runtimeScope); got.serverJsEngineFactory != "quickjs" {
		t.Fatalf("runtimeOptionsFromScope(runtimeScope) = %+v, want quickjs", got)
	}
}

func TestNewTask_DefaultContext(t *testing.T) {
	req := &JsRequest{Id: "task-default-context"}
	task := newTask(nilTaskContext(), req)
	if task.ctx == nil {
		t.Fatal("newTask(nil, req) should install a background context")
	}
	if task.request != req {
		t.Fatal("newTask(nil, req) should preserve the request pointer")
	}
	if cap(task.resultChan) != 1 {
		t.Fatalf("newTask(nil, req) resultChan cap = %d, want 1", cap(task.resultChan))
	}
	if task.status != taskStatusPending {
		t.Fatalf("newTask(nil, req) status = %d, want pending", task.status)
	}
}

func TestJsExecutor_Start_StateBranches(t *testing.T) {
	t.Run("idempotent transient states", func(t *testing.T) {
		for _, state := range []executorState{executorStateStarting, executorStateRunning, executorStateReloading} {
			executor := newLifecycleBranchTestExecutor()
			executor.setState(state)
			if err := executor.Start(); err != nil {
				t.Fatalf("Start() in state %s returned error: %v", state, err)
			}
		}
	})

	t.Run("stopping rejected", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		executor.setState(executorStateStopping)
		if err := executor.Start(); !errors.Is(err, errExecutorStopping) {
			t.Fatalf("Start() in stopping state = %v, want %v", err, errExecutorStopping)
		}
	})

	t.Run("stopped rejected", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		executor.setState(executorStateStopped)
		if err := executor.Start(); !errors.Is(err, errExecutorStopped) {
			t.Fatalf("Start() in stopped state = %v, want %v", err, errExecutorStopped)
		}
	})

	t.Run("warmup failure resets to new", func(t *testing.T) {
		executor := &JsExecutor{
			options: &JsExecutorOption{minPoolSize: 1, maxPoolSize: 1, queueSize: 1},
			engineFactory: func() (JsEngine, error) {
				return nil, errors.New("factory error")
			},
		}
		executor.pool = newPool(executor)
		if err := executor.Start(); err == nil {
			t.Fatal("expected Start() to fail when warmup thread init fails")
		}
		if executor.getState() != executorStateNew {
			t.Fatalf("Start() warmup failure should reset state to new, got %s", executor.getState())
		}
	})

	t.Run("stop requested during warmup", func(t *testing.T) {
		blocker := newStartWarmupBlocker()
		executor := &JsExecutor{
			options:       &JsExecutorOption{minPoolSize: 1, maxPoolSize: 1, queueSize: 1},
			engineFactory: blocker.factory(),
			state:         uint32(executorStateNew),
		}
		executor.pool = newPool(executor)

		startDone := make(chan error, 1)
		go func() {
			startDone <- executor.Start()
		}()

		select {
		case <-blocker.loadStarted:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for Start() warmup to begin")
		}

		stopDone := make(chan error, 1)
		go func() {
			stopDone <- executor.Stop()
		}()

		close(blocker.releaseLoad)

		if err := <-startDone; !errors.Is(err, errExecutorStopping) {
			t.Fatalf("Start() during concurrent Stop() = %v, want %v", err, errExecutorStopping)
		}
		if err := <-stopDone; err != nil {
			t.Fatalf("Stop() during concurrent Start() = %v, want nil", err)
		}
		if executor.getState() != executorStateStopped {
			t.Fatalf("expected stopped state after concurrent Start/Stop, got %s", executor.getState())
		}
	})

	t.Run("unknown state no-op", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		executor.setState(executorState(999))
		if err := executor.Start(); err != nil {
			t.Fatalf("Start() in unknown state returned error: %v", err)
		}
	})
}

func TestJsExecutor_Stop_StateBranches(t *testing.T) {
	t.Run("new transitions to stopped", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		if err := executor.Stop(); err != nil {
			t.Fatalf("Stop() on new executor returned error: %v", err)
		}
		if executor.getState() != executorStateStopped {
			t.Fatalf("Stop() on new executor should transition to stopped, got %s", executor.getState())
		}
	})

	t.Run("starting joins existing stop", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		executor.setState(executorStateStarting)
		executor.stopErr = errors.New("joined stop")
		executor.stopDone = make(chan struct{})
		go close(executor.stopDone)
		if err := executor.Stop(); !errors.Is(err, executor.stopErr) {
			t.Fatalf("Stop() on starting executor = %v, want %v", err, executor.stopErr)
		}
		if !executor.stopRequested {
			t.Fatal("Stop() on starting executor should mark stopRequested")
		}
	})

	t.Run("reloading marks pool stopping and joins", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		executor.setState(executorStateReloading)
		executor.pool.setState(poolStateRunning)
		executor.stopErr = errors.New("reload stop")
		executor.stopDone = make(chan struct{})
		go close(executor.stopDone)
		if err := executor.Stop(); !errors.Is(err, executor.stopErr) {
			t.Fatalf("Stop() on reloading executor = %v, want %v", err, executor.stopErr)
		}
		if executor.pool.getState() != poolStateStopping {
			t.Fatalf("Stop() on reloading executor should force pool stopping, got %s", executor.pool.getState())
		}
	})

	t.Run("stopping joins existing stop", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		executor.setState(executorStateStopping)
		executor.stopErr = errors.New("already stopping")
		executor.stopDone = make(chan struct{})
		go close(executor.stopDone)
		if err := executor.Stop(); !errors.Is(err, executor.stopErr) {
			t.Fatalf("Stop() on stopping executor = %v, want %v", err, executor.stopErr)
		}
	})

	t.Run("stopped returns prior error", func(t *testing.T) {
		executor := newLifecycleBranchTestExecutor()
		executor.setState(executorStateStopped)
		executor.stopErr = errors.New("prior stop error")
		if err := executor.Stop(); !errors.Is(err, executor.stopErr) {
			t.Fatalf("Stop() on stopped executor = %v, want %v", err, executor.stopErr)
		}
	})
}

func TestJsExecutor_Reload_StateBranches(t *testing.T) {
	tests := []struct {
		name          string
		state         executorState
		stopRequested bool
		want          error
	}{
		{name: "starting", state: executorStateStarting, want: errExecutorNotStarted},
		{name: "reloading", state: executorStateReloading, want: errExecutorReloading},
		{name: "running stop requested", state: executorStateRunning, stopRequested: true, want: errExecutorStopping},
		{name: "stopping", state: executorStateStopping, want: errExecutorStopping},
		{name: "stopped", state: executorStateStopped, want: errExecutorStopped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := newLifecycleBranchTestExecutor()
			executor.setState(tt.state)
			executor.stopRequested = tt.stopRequested
			if err := executor.Reload(); !errors.Is(err, tt.want) {
				t.Fatalf("Reload() in state %s returned %v, want %v", tt.state, err, tt.want)
			}
		})
	}
}

func TestJsExecutor_Execute_StateBranches(t *testing.T) {
	tests := []struct {
		state executorState
		want  error
	}{
		{executorStateStarting, errExecutorNotStarted},
		{executorStateReloading, errExecutorReloading},
		{executorStateStopping, errExecutorStopping},
		{executorStateStopped, errExecutorStopped},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			executor := newLifecycleBranchTestExecutor()
			executor.setState(tt.state)
			_, err := executor.Execute(context.Background(), &JsRequest{Id: "state-guard"})
			if !errors.Is(err, tt.want) {
				t.Fatalf("Execute() in state %s returned %v, want %v", tt.state, err, tt.want)
			}
		})
	}
}

func TestPool_HelperBranches(t *testing.T) {
	t.Run("admissionError", func(t *testing.T) {
		pool := newLifecycleBranchTestExecutor().pool
		pool.setState(poolStateStopping)
		if err := pool.admissionError(); !errors.Is(err, errPoolStopping) {
			t.Fatalf("admissionError() in stopping state = %v, want %v", err, errPoolStopping)
		}
		pool.setState(poolStateStopped)
		if err := pool.admissionError(); !errors.Is(err, errPoolStopped) {
			t.Fatalf("admissionError() in stopped state = %v, want %v", err, errPoolStopped)
		}
	})

	t.Run("requestReplenish only when running", func(t *testing.T) {
		pool := newLifecycleBranchTestExecutor().pool
		pool.setState(poolStateRunning)
		pool.requestReplenish()
		select {
		case <-pool.replenishChan:
		default:
			t.Fatal("requestReplenish() should signal when pool is running")
		}

		pool.setState(poolStateStopping)
		pool.requestReplenish()
		select {
		case <-pool.replenishChan:
			t.Fatal("requestReplenish() should not signal when pool is stopping")
		default:
		}
	})

	t.Run("beginThreadRetire guards and success", func(t *testing.T) {
		pool := newLifecycleBranchTestExecutor().pool
		if pool.beginThreadRetire(nil) {
			t.Fatal("beginThreadRetire(nil) should return false")
		}

		pool.setState(poolStateStopping)
		thread := newThread(pool.executor, "retire-guard", 99)
		if pool.beginThreadRetire(thread) {
			t.Fatal("beginThreadRetire() should reject when pool is stopping")
		}

		pool = newLifecycleBranchTestExecutor().pool
		pool.setState(poolStateRunning)
		thread = newThread(pool.executor, "retire-success", 100)
		pool.threads.Store(thread.threadId, thread)
		ids := []uint32{thread.threadId}
		pool.threadIds.Store(&ids)
		atomic.StoreUint32(&pool.threadCount, 1)
		if !pool.beginThreadRetire(thread) {
			t.Fatal("beginThreadRetire() should succeed for attached running thread")
		}
		if atomic.LoadUint32(&pool.threadCount) != 0 {
			t.Fatalf("beginThreadRetire() should detach thread, threadCount=%d", atomic.LoadUint32(&pool.threadCount))
		}
		select {
		case <-pool.replenishChan:
		default:
			t.Fatal("beginThreadRetire() should request replenish after detach")
		}
	})

	t.Run("enqueueTask guards", func(t *testing.T) {
		pool := newLifecycleBranchTestExecutor().pool
		thread := newThread(pool.executor, "enqueue-guard", 101)
		task := newTask(context.Background(), &JsRequest{Id: "enqueue-guard"})

		pool.setState(poolStateStopping)
		if err := pool.enqueueTask(thread, task); !errors.Is(err, errPoolStopping) {
			t.Fatalf("enqueueTask() in stopping state = %v, want %v", err, errPoolStopping)
		}

		pool.setState(poolStateRunning)
		close(thread.doneCh)
		if err := pool.enqueueTask(thread, task); !errors.Is(err, errPoolStopping) {
			t.Fatalf("enqueueTask() with stopped thread = %v, want %v", err, errPoolStopping)
		}
	})
}

func TestThread_HelperBranches(t *testing.T) {
	t.Run("isStopped", func(t *testing.T) {
		thread := newThread(newLifecycleBranchTestExecutor(), "stopped-check", 201)
		if thread.isStopped() {
			t.Fatal("isStopped() should be false before doneCh is closed")
		}
		close(thread.doneCh)
		if !thread.isStopped() {
			t.Fatal("isStopped() should be true after doneCh is closed")
		}
	})

	t.Run("completeAction handles nil and sends error", func(t *testing.T) {
		thread := newThread(newLifecycleBranchTestExecutor(), "complete-action", 202)
		thread.completeAction(nil, errors.New("ignored"))
		thread.completeAction(&threadActionRequest{}, errors.New("ignored"))

		done := make(chan error, 1)
		expected := errors.New("action failure")
		thread.completeAction(&threadActionRequest{done: done}, expected)
		if err := <-done; !errors.Is(err, expected) {
			t.Fatalf("completeAction() sent %v, want %v", err, expected)
		}
	})

	t.Run("sendActionRequest guards", func(t *testing.T) {
		thread := newThread(newLifecycleBranchTestExecutor(), "send-action", 203)
		if err := thread.sendActionRequest(nil); err != nil {
			t.Fatalf("sendActionRequest(nil) returned error: %v", err)
		}

		thread.actionQueue = nil
		close(thread.doneCh)
		if err := thread.sendActionRequest(&threadActionRequest{done: make(chan error, 1)}); err == nil {
			t.Fatal("sendActionRequest() should fail when thread is already stopped")
		}

		thread = newThread(newLifecycleBranchTestExecutor(), "send-action-recover", 204)
		close(thread.actionQueue)
		if err := thread.sendActionRequest(&threadActionRequest{done: make(chan error, 1)}); err == nil {
			t.Fatal("sendActionRequest() should recover and return error when actionQueue is closed")
		}
	})

	t.Run("requestAction stopped", func(t *testing.T) {
		thread := newThread(newLifecycleBranchTestExecutor(), "request-action", 205)
		close(thread.doneCh)
		if err := thread.requestAction(actionStop); err == nil {
			t.Fatal("requestAction() should fail when thread is already stopped")
		}
	})

	t.Run("executeAction nil and invalid request", func(t *testing.T) {
		thread := newThread(newLifecycleBranchTestExecutor(), "execute-action", 206)
		if terminal := thread.executeAction(nil); terminal {
			t.Fatal("executeAction(nil) should not terminate thread")
		}

		done := make(chan error, 1)
		if terminal := thread.executeAction(&threadActionRequest{action: threadAction(999), done: done}); terminal {
			t.Fatal("executeAction(unknown) should not terminate thread")
		}
		if err := <-done; err != nil {
			t.Fatalf("executeAction(unknown) returned error: %v", err)
		}
	})
}
