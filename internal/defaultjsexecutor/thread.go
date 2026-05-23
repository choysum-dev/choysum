// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultjsexecutor

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

type threadState uint32

const (
	threadStateInit threadState = iota
	threadStateRunning
	threadStateDraining
	threadStateReloading
	threadStateRetiring
	threadStateStopped
)

func (s threadState) String() string {
	switch s {
	case threadStateInit:
		return "init"
	case threadStateRunning:
		return "running"
	case threadStateDraining:
		return "draining"
	case threadStateReloading:
		return "reloading"
	case threadStateRetiring:
		return "retiring"
	case threadStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// threadAction represents an action that can be performed on a thread.
type threadAction int

const (
	actionStop   threadAction = iota // Stop the thread
	actionReload                     // Reload the thread's JavaScript engine
	actionRetire                     // Retire the thread
)

// String returns the string representation of a threadAction.
func (a threadAction) String() string {
	switch a {
	case actionStop:
		return "stop"
	case actionReload:
		return "reload"
	case actionRetire:
		return "retire"
	default:
		return "unknown"
	}
}

// threadActionRequest represents a request to perform an action on a thread.
type threadActionRequest struct {
	action threadAction // The action to perform
	done   chan error   // Channel to signal completion and return any error
}

// thread represents a single JavaScript execution thread.
type thread struct {
	executor *JsExecutor // Reference to the parent executor
	name     string      // Human-readable name for the thread
	threadId uint32      // Unique identifier for the thread

	taskQueue   chan *task                // Channel for receiving tasks to execute
	actionQueue chan *threadActionRequest // Channel for receiving control actions
	initCh      chan error                // Channel to signal initialization completion
	doneCh      chan struct{}             // Closed when the thread goroutine exits
	closeOnce   sync.Once                 // Ensures thread-owned channels are closed once
	state       uint32                    // Atomic threadState

	lastUsedNano int64  // Timestamp of last task execution (atomic, nanoseconds)
	taskID       uint32 // Number of tasks executed by this thread (atomic)

	jsEngine jsengine.JsEngine // JavaScript engine instance
}

// newThread creates a new thread instance.
func newThread(executor *JsExecutor, name string, threadId uint32) *thread {
	return &thread{
		executor:     executor,
		name:         name,
		threadId:     threadId,
		taskQueue:    make(chan *task, executor.options.queueSize),
		actionQueue:  make(chan *threadActionRequest, 1),
		initCh:       make(chan error, 1),
		doneCh:       make(chan struct{}),
		state:        uint32(threadStateInit),
		lastUsedNano: time.Now().UnixNano(),
		taskID:       0,
	}
}

func (t *thread) getState() threadState {
	return threadState(atomic.LoadUint32(&t.state))
}

func (t *thread) setState(state threadState) {
	atomic.StoreUint32(&t.state, uint32(state))
}

func (t *thread) isStopped() bool {
	select {
	case <-t.doneCh:
		return true
	default:
		return false
	}
}

func (t *thread) waitStopped() {
	<-t.doneCh
}

func (t *thread) closeOwnedChannels() {
	t.closeOnce.Do(func() {
		close(t.taskQueue)
		close(t.actionQueue)
		close(t.doneCh)
	})
}

func (t *thread) completeAction(req *threadActionRequest, err error) {
	if req == nil || req.done == nil {
		return
	}
	req.done <- err
}

func (t *thread) closeCurrentEngine() error {
	if t.jsEngine == nil {
		return nil
	}
	err := t.jsEngine.Close()
	if err != nil && t.executor != nil && t.executor.logger != nil {
		t.executor.logger.Error("js engine close failed",
			"thread", t.name,
			"error", err)
	}
	t.jsEngine = nil
	return err
}

func (t *thread) sendActionRequest(req *threadActionRequest) (err error) {
	if req == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("thread %s is stopped", t.name)
		}
	}()

	select {
	case <-t.doneCh:
		return fmt.Errorf("thread %s is stopped", t.name)
	case t.actionQueue <- req:
		return nil
	}
}

func (t *thread) requestAction(action threadAction) error {
	if t.isStopped() {
		return fmt.Errorf("thread %s is stopped", t.name)
	}
	req := &threadActionRequest{
		action: action,
		done:   make(chan error, 1),
	}
	if err := t.sendActionRequest(req); err != nil {
		return err
	}

	select {
	case err := <-req.done:
		return err
	case <-t.doneCh:
		return fmt.Errorf("thread %s is stopped", t.name)
	}
}

func (t *thread) markDraining(action threadAction) {
	switch action {
	case actionStop, actionReload, actionRetire:
		if t.getState() == threadStateRunning {
			t.setState(threadStateDraining)
		}
	}
}

// getTaskCount returns the number of tasks executed by this thread (thread-safe).
func (t *thread) getTaskCount() uint32 {
	return atomic.LoadUint32(&t.taskID)
}

// getLastUsed returns the timestamp of the last task execution (thread-safe).
func (t *thread) getLastUsed() time.Time {
	return time.Unix(0, atomic.LoadInt64(&t.lastUsedNano))
}

// initEngine initializes the JavaScript engine for this thread.
func (t *thread) initEngine() error {
	// Create a new JavaScript engine instance
	jsEngine, err := t.executor.engineFactory()
	if err != nil {
		return fmt.Errorf("failed to create JS engine: %w", err)
	}
	t.jsEngine = jsEngine

	// Get initialization scripts safely
	scripts := t.executor.GetJsScripts()

	// Initialize the engine with scripts
	if err := t.jsEngine.Load(scripts); err != nil {
		return fmt.Errorf("failed to init JS engine: %w", err)
	}

	return nil
}

// run is the main thread loop that processes tasks and actions.
func (t *thread) run() {
	// Lock this goroutine to an OS thread for consistent execution environment
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Cleanup when the thread exits
	defer func() {
		t.setState(threadStateStopped)
		if t.jsEngine != nil {
			if err := t.jsEngine.Close(); err != nil {
				if t.executor != nil && t.executor.logger != nil {
					t.executor.logger.Error("js engine close failed",
						"thread", t.name,
						"error", err)
				}
			}
		}
		t.jsEngine = nil
		t.closeOwnedChannels()
	}()

	// Use a queue to store all pending actions
	var pendingActions []*threadActionRequest

	// Initialize the JavaScript engine
	if err := t.initEngine(); err != nil {
		t.setState(threadStateStopped)
		t.initCh <- err
		close(t.initCh)
		if t.executor != nil && t.executor.logger != nil {
			t.executor.logger.Error("js engine initialization failed",
				"thread", t.name,
				"error", err,
			)
		}
		return
	}
	// Signal successful initialization
	t.setState(threadStateRunning)
	t.initCh <- nil
	close(t.initCh)

	// Main execution loop
	for {
		for len(pendingActions) > 0 {
			action := pendingActions[0]
			pendingActions = pendingActions[1:]
			if t.executeAction(action) {
				return
			}
			continue
		}

		select {
		case actionReq, ok := <-t.actionQueue:
			if !ok {
				return
			}
			if actionReq != nil {
				t.markDraining(actionReq.action)
				pendingActions = append(pendingActions, actionReq)
			}
			continue
		default:
		}

		select {
		case actionReq, ok := <-t.actionQueue:
			if !ok {
				return
			}
			if actionReq != nil {
				t.markDraining(actionReq.action)
				pendingActions = append(pendingActions, actionReq)
			}
			continue
		case task, ok := <-t.taskQueue:
			if !ok || task == nil {
				return // Channel closed, exit the thread
			}
			t.executeTask(task)
			if t.checkAndRetireIfNeeded(&pendingActions) {
				if t.executor.logger != nil {
					t.executor.logger.Debug("thread retirement requested",
						"thread", t.name,
						"reason", "max_executions_reached",
						"task_count", t.getTaskCount(),
					)
				}
				t.notifyPoolReplenish()
				continue
			}
		}
	}
}

// executeAction executes a thread action (stop, reload, retire, or default).
func (t *thread) executeAction(req *threadActionRequest) (terminal bool) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in executeAction: %v", r)
			if t.executor != nil && t.executor.logger != nil {
				attrs := []any{"thread", t.name, "panic", r}
				if req != nil {
					attrs = append(attrs, "action", req.action.String())
				}
				t.executor.logger.Error("thread action panic recovered", attrs...)
			}
			if req != nil {
				switch req.action {
				case actionReload:
					t.setState(threadStateRunning)
				case actionStop, actionRetire:
					t.setState(threadStateStopped)
					t.jsEngine = nil
					terminal = true
				}
			}
			t.completeAction(req, err)
		}
	}()

	if req == nil {
		if t.executor != nil && t.executor.logger != nil {
			t.executor.logger.Error("thread action request missing", "thread", t.name)
		}
		return false
	}

	switch req.action {
	case actionReload:
		if t.executor == nil || t.executor.engineFactory == nil {
			err := fmt.Errorf("thread %s has no engine factory", t.name)
			t.setState(threadStateRunning)
			if t.executor != nil && t.executor.logger != nil {
				t.executor.logger.Error("thread reload failed",
					"thread", t.name,
					"error", err)
			}
			t.completeAction(req, err)
			return false
		}

		// Recreate engine on reload to avoid carrying over stale runtime state.
		t.setState(threadStateReloading)
		newEngine, err := t.executor.engineFactory()
		if err != nil {
			t.setState(threadStateRunning)
			if t.executor.logger != nil {
				t.executor.logger.Error("thread reload failed",
					"thread", t.name,
					"error", err)
			}
			t.completeAction(req, err)
			return false
		}

		scripts := t.executor.GetJsScripts()
		if err := newEngine.Load(scripts); err != nil {
			_ = newEngine.Close()
			t.setState(threadStateRunning)
			if t.executor.logger != nil {
				t.executor.logger.Error("thread reload failed",
					"thread", t.name,
					"error", err)
			}
			t.completeAction(req, err)
			return false
		}

		oldEngine := t.jsEngine
		t.jsEngine = newEngine
		if oldEngine != nil {
			if closeErr := oldEngine.Close(); closeErr != nil && t.executor.logger != nil {
				t.executor.logger.Error("previous js engine close failed during reload",
					"thread", t.name,
					"error", closeErr)
			}
		}

		t.setState(threadStateRunning)
		t.completeAction(req, nil)
		return false

	case actionStop:
		err := t.closeCurrentEngine()
		t.setState(threadStateStopped)
		t.completeAction(req, err)
		return true

	case actionRetire:
		t.setState(threadStateRetiring)
		err := t.closeCurrentEngine()
		t.setState(threadStateStopped)
		t.completeAction(req, err)
		return true

	default:
		// Handle unknown action types
		t.completeAction(req, nil)
		return false
	}
}

// executeTask executes a single JavaScript task.
func (t *thread) executeTask(task *task) {
	// Update task execution statistics on completion
	defer func() {
		if r := recover(); r != nil {
			// Handle panic during task execution
			task.resultChan <- &taskResult{
				response: nil,
				err:      fmt.Errorf("panic in thread %s: %v", t.name, r),
			}
			if t.executor != nil && t.executor.logger != nil {
				t.executor.logger.Error("task execution panic",
					"thread", t.name,
					"task_id", t.getTaskCount(),
					"panic", r)
			}
		}
		// Update thread statistics atomically
		atomic.StoreInt64(&t.lastUsedNano, time.Now().UnixNano())
		atomic.AddUint32(&t.taskID, 1)
	}()

	task.status = taskStatusRunning

	// Execute the JavaScript request
	response, err := t.jsEngine.Execute(task.ctx, task.request)
	if response == nil {
		response = &jsengine.JsResponse{
			Id:      task.request.Id,
			Result:  nil,
			Context: task.request.Context,
		}
	}

	if response.Routing == nil {
		response.Routing = &jsengine.JsExecutionRouting{}
	}
	response.Routing.ThreadID = t.threadId
	task.resultChan <- &taskResult{
		response: response,
		err:      err,
	}
	task.status = taskStatusCompleted
}

// reload sends a reload request to the thread and waits for completion.
func (t *thread) reload() error {
	return t.requestAction(actionReload)
}

// stop gracefully stops the thread and waits for the goroutine to exit.
func (t *thread) stop() {
	_ = t.requestAction(actionStop)
	t.waitStopped()
}

// retire gracefully retires the thread and waits for the goroutine to exit.
func (t *thread) retire() {
	_ = t.requestAction(actionRetire)
	t.waitStopped()
}

// checkAndRetireIfNeeded checks if the thread has reached maxExecutions and retires if needed.
// If retire is needed, it appends a retire request to pendingActions and returns true.
func (t *thread) checkAndRetireIfNeeded(pendingActions *[]*threadActionRequest) bool {
	if t.executor.options.maxExecutions > 0 && t.getTaskCount() >= t.executor.options.maxExecutions {
		if t.executor != nil && t.executor.pool != nil && t.executor.pool.beginThreadRetire(t) {
			t.setState(threadStateDraining)
			*pendingActions = append(*pendingActions, &threadActionRequest{
				action: actionRetire,
				done:   make(chan error, 1),
			})
			return true
		}
	}
	return false
}

// notifyPoolReplenish notifies the pool to check and replenish threads if needed.
func (t *thread) notifyPoolReplenish() {
	if t.executor == nil || t.executor.pool == nil || t.executor.pool.replenishChan == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	select {
	case t.executor.pool.replenishChan <- struct{}{}:
	default:
	}
}
