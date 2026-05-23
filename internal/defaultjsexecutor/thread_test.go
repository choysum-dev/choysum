// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultjsexecutor

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestThreadAction_String tests the string representation of threadAction.
func TestThreadAction_String(t *testing.T) {
	tests := []struct {
		action   threadAction
		expected string
	}{
		{actionStop, "stop"},
		{actionReload, "reload"},
		{actionRetire, "retire"},
		{threadAction(999), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.action.String(); got != tt.expected {
			t.Errorf("threadAction(%d).String() = %q, want %q", tt.action, got, tt.expected)
		}
	}
}

// TestThread_InitEngine_Success tests successful initialization of the JS engine.
func TestThread_InitEngine_Success(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t1", 1)
	if err := th.initEngine(); err != nil {
		t.Fatalf("initEngine failed: %v", err)
	}
	if th.jsEngine == nil {
		t.Error("jsEngine should be set after initEngine")
	}
}

// TestThread_InitEngine_FactoryError tests error handling when engineFactory fails.
func TestThread_InitEngine_FactoryError(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return nil, errors.New("factory error") },
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t2", 2)
	err := th.initEngine()
	if err == nil || err.Error() != "failed to create JS engine: factory error" {
		t.Errorf("Expected factory error, got: %v", err)
	}
}

// TestThread_InitEngine_InitError tests error handling when engine.Init fails.
func TestThread_InitEngine_InitError(t *testing.T) {
	engine := &mockEngine{
		loadFunc: func(scripts []*JsScript) error { return errors.New("init error") },
	}
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return engine, nil },
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t3", 3)
	err := th.initEngine()
	if err == nil || err.Error() != "failed to init JS engine: init error" {
		t.Errorf("Expected init error, got: %v", err)
	}
}

// TestThread_ExecuteTask tests execution of a single JavaScript task.
func TestThread_ExecuteTask(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t4", 4)
	_ = th.initEngine()
	req := &JsRequest{Id: "abc"}
	task := newTask(context.Background(), req)
	th.executeTask(task)
	result := <-task.resultChan
	if result.err != nil {
		t.Errorf("executeTask returned error: %v", result.err)
	}
	if result.response == nil || result.response.Id != "abc" {
		t.Errorf("Unexpected response: %+v", result.response)
	}
}

// TestThread_Reload tests the reload operation of a thread.
func TestThread_Reload(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t5", 5)
	_ = th.initEngine()
	go th.run()
	time.Sleep(10 * time.Millisecond)
	err := th.reload()
	if err != nil {
		t.Errorf("reload failed: %v", err)
	}
	th.stop()
}

// TestThread_Stop tests the stop operation and channel closure.
func TestThread_Stop(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t6", 6)
	_ = th.initEngine()
	go th.run()
	time.Sleep(10 * time.Millisecond)
	th.stop()
	if state := th.getState(); state != threadStateStopped {
		t.Fatalf("expected stopped state after stop, got %s", state)
	}
	// After stop, channels should be closed
	select {
	case _, ok := <-th.taskQueue:
		if ok {
			t.Error("taskQueue should be closed after stop")
		}
	default:
	}
	select {
	case _, ok := <-th.actionQueue:
		if ok {
			t.Error("actionQueue should be closed after stop")
		}
	default:
	}
}

func TestThread_Stop_RequestAckOnly(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t_stop_ack", 601)
	go th.run()
	time.Sleep(10 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		th.stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stop request/ack")
	}

	if state := th.getState(); state != threadStateStopped {
		t.Fatalf("expected stopped state after stop ack, got %s", state)
	}
	select {
	case <-th.doneCh:
	default:
		t.Fatal("expected doneCh to be closed after stop")
	}
}

// TestThread_Run_And_Execute tests running the thread and executing a task.
func TestThread_Run_And_Execute(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t7", 7)
	go th.run()
	time.Sleep(10 * time.Millisecond)
	req := &JsRequest{Id: "run1"}
	task := newTask(context.Background(), req)
	th.taskQueue <- task
	result := <-task.resultChan
	if result.err != nil {
		t.Errorf("run/execute returned error: %v", result.err)
	}
	th.stop()
}

// TestThread_ExecuteAction_ReloadError tests error handling in reload action.
func TestThread_ExecuteAction_ReloadError(t *testing.T) {
	engine := &mockEngine{
		loadFunc: func(scripts []*JsScript) error { return errors.New("reload error") },
	}
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return engine, nil },
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t8", 8)
	_ = th.initEngine()
	req := &threadActionRequest{
		action: actionReload,
		done:   make(chan error, 1),
	}
	th.executeAction(req)
	err := <-req.done
	if err == nil || err.Error() != "reload error" {
		t.Errorf("Expected reload error, got: %v", err)
	}
}

func TestThread_RequestAction_ReturnsStoppedWhenDoneClosesBeforeAck(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_request_done_closed", 18)

	errCh := make(chan error, 1)
	go func() {
		errCh <- th.requestAction(actionReload)
	}()

	var req *threadActionRequest
	select {
	case req = <-th.actionQueue:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for requestAction to enqueue action")
	}
	if req == nil || req.action != actionReload {
		t.Fatalf("unexpected action request: %+v", req)
	}

	close(th.doneCh)

	if err := <-errCh; err == nil || err.Error() != "thread t_request_done_closed is stopped" {
		t.Fatalf("expected stopped error after doneCh closes, got %v", err)
	}
}

func TestThread_ExecuteAction_Reload_NoEngineFactory(t *testing.T) {
	exec := &JsExecutor{
		logger:  slog.Default(),
		options: &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_reload_no_factory", 19)
	th.setState(threadStateRunning)
	req := &threadActionRequest{
		action: actionReload,
		done:   make(chan error, 1),
	}

	if terminal := th.executeAction(req); terminal {
		t.Fatal("reload without engine factory should not terminate thread")
	}
	if err := <-req.done; err == nil || err.Error() != "thread t_reload_no_factory has no engine factory" {
		t.Fatalf("expected missing engine factory error, got %v", err)
	}
	if state := th.getState(); state != threadStateRunning {
		t.Fatalf("expected thread to return to running state, got %s", state)
	}
}

func TestThread_ExecuteAction_Reload_FactoryError(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return nil, errors.New("factory error") },
		logger:        slog.Default(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_reload_factory_err", 20)
	th.jsEngine = &mockEngine{}
	req := &threadActionRequest{
		action: actionReload,
		done:   make(chan error, 1),
	}

	if terminal := th.executeAction(req); terminal {
		t.Fatal("reload factory error should not terminate thread")
	}
	if err := <-req.done; err == nil || err.Error() != "factory error" {
		t.Fatalf("expected factory error, got %v", err)
	}
	if state := th.getState(); state != threadStateRunning {
		t.Fatalf("expected thread to return to running state, got %s", state)
	}
}

func TestThread_ExecuteAction_Reload_ClosePreviousEngineError(t *testing.T) {
	newEngine := &mockEngine{}
	oldClosed := false
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return newEngine, nil },
		logger:        slog.Default(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_reload_close_prev_err", 21)
	th.jsEngine = &mockEngine{
		closeFunc: func() error {
			oldClosed = true
			return errors.New("close previous error")
		},
	}
	req := &threadActionRequest{
		action: actionReload,
		done:   make(chan error, 1),
	}

	if terminal := th.executeAction(req); terminal {
		t.Fatal("reload with previous close error should not terminate thread")
	}
	if err := <-req.done; err != nil {
		t.Fatalf("expected reload to succeed, got %v", err)
	}
	if !oldClosed {
		t.Fatal("expected previous engine close to be attempted")
	}
	if th.jsEngine != newEngine {
		t.Fatal("expected new engine to remain active after reload")
	}
}

// TestThread_ExecuteAction_Stop_EngineNil tests the stop action when jsEngine is already nil.
func TestThread_ExecuteAction_Stop_EngineNil(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_stop_nil", 200)
	// Do NOT call th.initEngine(), so th.jsEngine is nil
	req := &threadActionRequest{
		action: actionStop,
		done:   make(chan error, 1),
	}
	th.executeAction(req)
	err := <-req.done
	if err != nil {
		t.Errorf("expected nil error when stopping with nil jsEngine, got %v", err)
	}
}

// TestThread_ExecuteAction_Stop_PanicInClose tests panic in Close function
func TestThread_ExecuteAction_Stop_PanicInClose(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) {
			return &mockEngine{
				closeFunc: func() error {
					panic("panic in Close")
				},
			}, nil
		},
		logger:  slog.Default(),
		options: &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_stop_panic", 201)
	_ = th.initEngine()
	req := &threadActionRequest{
		action: actionStop,
		done:   make(chan error, 1),
	}
	th.executeAction(req)
	select {
	case err := <-req.done:
		if err == nil {
			t.Fatal("expected a panic error, but got nil")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for action stop")
	}
}

// TestThread_GetTaskCountAndLastUsed tests task count and last used timestamp.
func TestThread_GetTaskCountAndLastUsed(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t9", 9)
	th.initEngine()
	before := th.getTaskCount()
	req := &JsRequest{Id: "count"}
	task := newTask(context.Background(), req)
	th.executeTask(task)
	<-task.resultChan
	after := th.getTaskCount()
	if after != before+1 {
		t.Errorf("getTaskCount did not increment, before=%d after=%d", before, after)
	}
	if th.getLastUsed().IsZero() {
		t.Error("getLastUsed should not be zero")
	}
}

// TestThread_Run_PanicRecovery tests panic recovery in the thread's run loop.
func TestThread_Run_PanicRecovery(t *testing.T) {
	engine := &mockEngine{
		executeResp: nil,
		executeErr:  errors.New("panic!"),
	}
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return engine, nil },
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t10", 10)
	go th.run()
	time.Sleep(10 * time.Millisecond)
	req := &JsRequest{Id: "panic"}
	task := newTask(context.Background(), req)
	th.taskQueue <- task
	result := <-task.resultChan
	if result.err == nil {
		t.Error("Expected error from panic in executeTask")
	}
	th.stop()
}

// TestThread_Concurrent_ExecuteTask tests concurrent execution of tasks.
func TestThread_Concurrent_ExecuteTask(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 10},
	}
	th := newThread(exec, "t13", 13)
	_ = th.initEngine()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := &JsRequest{Id: "c"}
			task := newTask(context.Background(), req)
			th.executeTask(task)
			<-task.resultChan
		}(i)
	}
	wg.Wait()
}

// TestThread_Run_DeferCloseError tests error handling in deferred engine close.
func TestThread_Run_DeferCloseError(t *testing.T) {
	done := make(chan struct{})
	engine := &mockEngine{
		closeFunc: func() error {
			close(done)
			return errors.New("close error")
		},
		executeFunc: func(req *JsRequest) (*JsResponse, error) {
			panic("trigger panic in execute")
		},
	}
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return engine, nil },
		logger:        slog.Default(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t_defer", 1001)
	_ = th.initEngine()
	go th.run()
	time.Sleep(10 * time.Millisecond)
	// Trigger panic to enter defer
	task := newTask(context.Background(), &JsRequest{Id: "panic"})
	th.taskQueue <- task
	time.Sleep(20 * time.Millisecond)
	th.stop()
	select {
	case <-done:
		// ok, Close was called
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Close in defer")
	}
}

// TestThread_Run_PendingActionExecute tests that pending actions are executed after tasks.
func TestThread_Run_PendingActionExecute(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		logger:        slog.Default(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_pending", 1002)
	_ = th.initEngine()
	done := make(chan error, 1)
	go th.run()
	time.Sleep(10 * time.Millisecond)

	// 1. Enqueue a task to make the queue non-empty
	task := newTask(context.Background(), &JsRequest{Id: "a"})
	th.taskQueue <- task

	// 2. Send an action request, pendingAction will be set
	th.actionQueue <- &threadActionRequest{action: actionReload, done: done}
	time.Sleep(50 * time.Millisecond) // Give main loop a chance to schedule

	// 3. Consume the task, making the queue empty
	<-task.resultChan
	time.Sleep(50 * time.Millisecond)

	// 4. pendingAction will be executed in the next loop
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("pendingAction was not executed")
	}
	th.stop()
}

// TestThread_Run_CheckAndRetireIfNeeded_Logger tests retire logic and logger coverage.
func TestThread_Run_CheckAndRetireIfNeeded_Logger(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		logger:        slog.Default(),
		options: &JsExecutorOption{
			queueSize:     2,
			maxExecutions: 1, // Only allow 1 execution
		},
	}
	exec.pool = newPool(exec)
	exec.pool.setState(poolStateRunning)

	th := newThread(exec, "t_retire", 1004)
	exec.pool.threads.Store(th.threadId, th)
	ids := []uint32{th.threadId}
	exec.pool.threadIds.Store(&ids)
	atomic.StoreUint32(&exec.pool.threadCount, 1)
	_ = th.initEngine()
	go th.run()
	time.Sleep(10 * time.Millisecond)

	// One task is enough now that retire is executed at the next task boundary.
	task1 := newTask(context.Background(), &JsRequest{Id: "max1"})
	th.taskQueue <- task1
	<-task1.resultChan
	time.Sleep(100 * time.Millisecond)

	if state := th.getState(); state != threadStateStopped {
		t.Fatalf("expected thread to retire into stopped state, got %s", state)
	}
}

type reloadBoundaryTracker struct {
	mu           sync.Mutex
	created      int
	releaseFirst chan struct{}
	firstStarted chan struct{}
	firstOnce    sync.Once
}

func (r *reloadBoundaryTracker) newEngine() JsEngine {
	r.mu.Lock()
	r.created++
	id := r.created
	r.mu.Unlock()
	return &reloadBoundaryEngine{id: id, tracker: r}
}

type reloadBoundaryEngine struct {
	id      int
	tracker *reloadBoundaryTracker
}

func (e *reloadBoundaryEngine) Load(_ []*JsScript) error { return nil }

func (e *reloadBoundaryEngine) Execute(_ context.Context, req *JsRequest) (*JsResponse, error) {
	if e.id == 1 && req.Id == "first" {
		e.tracker.firstOnce.Do(func() {
			close(e.tracker.firstStarted)
		})
		<-e.tracker.releaseFirst
	}
	return &JsResponse{
		Id: req.Id,
		Context: map[string]interface{}{
			"generation": e.id,
		},
	}, nil
}

func (e *reloadBoundaryEngine) Close() error { return nil }

func TestThread_Reload_AtTaskBoundaryUnderLoad(t *testing.T) {
	tracker := &reloadBoundaryTracker{
		releaseFirst: make(chan struct{}),
		firstStarted: make(chan struct{}),
	}
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) { return tracker.newEngine(), nil },
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t_reload_boundary", 701)
	go th.run()
	time.Sleep(10 * time.Millisecond)
	defer th.stop()

	firstTask := newTask(context.Background(), &JsRequest{Id: "first"})
	th.taskQueue <- firstTask

	select {
	case <-tracker.firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first task to start")
	}

	reloadErr := make(chan error, 1)
	go func() {
		reloadErr <- th.reload()
	}()
	time.Sleep(20 * time.Millisecond)

	secondTask := newTask(context.Background(), &JsRequest{Id: "second"})
	th.taskQueue <- secondTask
	close(tracker.releaseFirst)

	<-firstTask.resultChan

	select {
	case err := <-reloadErr:
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for reload")
	}

	select {
	case result := <-secondTask.resultChan:
		generation, ok := result.response.Context["generation"].(int)
		if !ok {
			t.Fatalf("expected generation in response context, got %#v", result.response.Context)
		}
		if generation != 2 {
			t.Fatalf("expected second task to run after reload on generation 2, got %d", generation)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second task")
	}
}

func TestThread_Retire_AckTransitionsToStopped(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 2},
	}
	th := newThread(exec, "t_retire_ack", 702)
	go th.run()
	time.Sleep(10 * time.Millisecond)

	th.retire()

	if state := th.getState(); state != threadStateStopped {
		t.Fatalf("expected stopped state after retire, got %s", state)
	}
	select {
	case <-th.doneCh:
	default:
		t.Fatal("expected doneCh to be closed after retire")
	}
	if th.jsEngine != nil {
		t.Fatal("expected jsEngine to be released after retire")
	}
}

func TestThread_NotifyPoolReplenish_SafeDuringStop(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	exec.pool = &pool{replenishChan: make(chan struct{}, 1)}
	close(exec.pool.replenishChan)
	th := newThread(exec, "t_replenish_safe", 703)

	th.notifyPoolReplenish()
}

func TestThread_NotifyPoolReplenish_RunningPaths(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	exec.pool = &pool{replenishChan: make(chan struct{}, 1), state: uint32(poolStateRunning)}
	th := newThread(exec, "t_replenish_running", 704)

	th.notifyPoolReplenish()
	select {
	case <-exec.pool.replenishChan:
	default:
		t.Fatal("notifyPoolReplenish should signal running pool")
	}

	exec.pool.replenishChan <- struct{}{}
	th.notifyPoolReplenish()
	if got := len(exec.pool.replenishChan); got != 1 {
		t.Fatalf("notifyPoolReplenish should hit default when channel is full, len=%d", got)
	}

	close(exec.pool.replenishChan)
	th.notifyPoolReplenish()
}

func TestThread_NotifyPoolReplenish_ReturnsWhenDependenciesMissing(t *testing.T) {
	(&thread{}).notifyPoolReplenish()
	(&thread{executor: &JsExecutor{}}).notifyPoolReplenish()
	(&thread{executor: &JsExecutor{pool: &pool{}}}).notifyPoolReplenish()
}

// TestThread_ExecuteAction_Stop_CloseError tests error handling in stop action's engine close.
func TestThread_ExecuteAction_Stop_CloseError(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) {
			return &mockEngine{
				closeFunc: func() error { return errors.New("close error") },
			}, nil
		},
		logger:  slog.Default(),
		options: &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_stop_close_err", 100)
	_ = th.initEngine()
	req := &threadActionRequest{
		action: actionStop,
		done:   make(chan error, 1),
	}
	th.executeAction(req)
	<-req.done
}

// TestThread_ExecuteAction_Retire_CloseError tests error handling in retire action's engine close.
func TestThread_ExecuteAction_Retire_CloseError(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) {
			return &mockEngine{
				closeFunc: func() error { return errors.New("close error") },
			}, nil
		},
		logger:  slog.Default(),
		options: &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_retire_close_err", 101)
	_ = th.initEngine()
	req := &threadActionRequest{
		action: actionRetire,
		done:   make(chan error, 1),
	}
	th.executeAction(req)
	<-req.done
}

// TestThread_ExecuteAction_Reload_Error tests error handling in reload action.
func TestThread_ExecuteAction_Reload_Error(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: func() (JsEngine, error) {
			return &mockEngine{
				loadFunc: func(scripts []*JsScript) error { return errors.New("reload error") },
			}, nil
		},
		logger:  slog.Default(),
		options: &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_reload_err", 102)
	_ = th.initEngine()
	req := &threadActionRequest{
		action: actionReload,
		done:   make(chan error, 1),
	}
	th.executeAction(req)
	err := <-req.done
	if err == nil || err.Error() != "reload error" {
		t.Fatalf("expected reload error, got %v", err)
	}
}

// TestThread_ExecuteAction_Default tests the default branch of executeAction.
func TestThread_ExecuteAction_Default(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_default", 999)
	_ = th.initEngine()
	req := &threadActionRequest{
		action: threadAction(999), // Undefined action
		done:   make(chan error, 1),
	}
	th.executeAction(req)
	err := <-req.done
	if err != nil {
		t.Errorf("expected nil error for default action, got %v", err)
	}
}

// TestThread_ExecuteAction_NilRequest tests executeAction with nil request.
func TestThread_ExecuteAction_NilRequest(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		logger:        slog.Default(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_nil_req", 888)
	_ = th.initEngine()
	// Should not panic, should log error
	th.executeAction(nil)
}

func TestThread_RequestAction_ReturnsSendError(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_request_send_err", 889)
	close(th.actionQueue)

	if err := th.requestAction(actionReload); err == nil || err.Error() != "thread t_request_send_err is stopped" {
		t.Fatalf("expected send error from requestAction, got %v", err)
	}
}

func TestThread_Run_IgnoresNilActionAndStopsOnNilTask(t *testing.T) {
	exec := &JsExecutor{
		engineFactory: mockEngineFactory(),
		options:       &JsExecutorOption{queueSize: 1},
	}
	th := newThread(exec, "t_nil_action_nil_task", 890)
	go th.run()

	select {
	case err := <-th.initCh:
		if err != nil {
			t.Fatalf("thread init failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for thread init")
	}

	th.actionQueue <- nil
	time.Sleep(20 * time.Millisecond)
	th.taskQueue <- nil

	select {
	case <-th.doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for thread to stop on nil task")
	}

	if state := th.getState(); state != threadStateStopped {
		t.Fatalf("expected stopped state after nil task shutdown, got %s", state)
	}
}