// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultjsexecutor

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

var (
	errPoolStopping         = errors.New("thread pool is stopping")
	errPoolStopped          = errors.New("thread pool is stopped")
	errPoolReloadNotRunning = errors.New("thread pool is not running")
)

type poolState uint32

const (
	poolStateCold poolState = iota
	poolStateRunning
	poolStateStopping
	poolStateStopped
)

func (s poolState) String() string {
	switch s {
	case poolStateCold:
		return "cold"
	case poolStateRunning:
		return "running"
	case poolStateStopping:
		return "stopping"
	case poolStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// pool manages a collection of JavaScript execution threads using lock-free algorithms.
type pool struct {
	executor        *JsExecutor   // Reference to the parent executor
	threads         sync.Map      // Lock-free map of thread ID to thread instance
	threadIds       atomic.Value  // Stores *[]uint32 for round-robin selection (copy-on-write)
	threadCount     uint32        // Atomic: current number of threads in the pool
	roundRobinIndex uint32        // Current index for round-robin selection (atomic)
	threadIdCounter uint32        // Counter for generating unique thread IDs (atomic)
	stopCleanup     chan struct{} // Channel to signal cleanup goroutine to stop
	replenishChan   chan struct{} // Channel to signal thread replenishment
	cleanupDone     chan struct{} // Closed when the cleanup goroutine exits

	state          uint32 // Atomic poolState
	controlMu      sync.Mutex
	stopOnce       sync.Once
	stopDone       chan struct{}
	cleanupStarted uint32
}

// threadCleanupInfo holds information about a thread to be cleaned up.
type threadCleanupInfo struct {
	id     uint32  // Thread ID
	thread *thread // Thread instance
}

// newPool creates a new thread pool with lock-free data structures.
func newPool(e *JsExecutor) *pool {
	p := &pool{
		executor:        e,
		threadCount:     0,
		roundRobinIndex: 0,
		threadIdCounter: 0,
		stopCleanup:     make(chan struct{}),
		replenishChan:   make(chan struct{}, 1),
		cleanupDone:     make(chan struct{}),
		state:           uint32(poolStateCold),
		stopDone:        make(chan struct{}),
	}
	// Initialize empty thread ID list with pointer
	emptyIds := make([]uint32, 0)
	p.threadIds.Store(&emptyIds)
	return p
}

func (p *pool) getState() poolState {
	return poolState(atomic.LoadUint32(&p.state))
}

func (p *pool) setState(state poolState) {
	atomic.StoreUint32(&p.state, uint32(state))
}

func (p *pool) admissionError() error {
	switch p.getState() {
	case poolStateStopping:
		return errPoolStopping
	case poolStateStopped:
		return errPoolStopped
	default:
		return nil
	}
}

func (p *pool) snapshotThreads() []*thread {
	threads := make([]*thread, 0)
	p.threads.Range(func(_, value interface{}) bool {
		threads = append(threads, value.(*thread))
		return true
	})
	sort.Slice(threads, func(i, j int) bool {
		return threads[i].threadId < threads[j].threadId
	})
	return threads
}

func (p *pool) detachThread(threadId uint32) bool {
	if _, loaded := p.threads.LoadAndDelete(threadId); !loaded {
		return false
	}
	p.removeThreadFromList(threadId)
	atomic.AddUint32(&p.threadCount, ^uint32(0)) // -1
	return true
}

func (p *pool) requestReplenish() {
	if p == nil || p.replenishChan == nil || p.getState() != poolStateRunning {
		return
	}
	select {
	case p.replenishChan <- struct{}{}:
	default:
	}
}

func (p *pool) beginThreadRetire(t *thread) bool {
	if p == nil || t == nil {
		return false
	}
	if err := p.admissionError(); err != nil {
		return false
	}
	if !p.detachThread(t.threadId) {
		return false
	}
	p.requestReplenish()
	return true
}

func (p *pool) enqueueTask(t *thread, task *task) (err error) {
	if err := p.admissionError(); err != nil {
		return err
	}
	if t == nil || t.isStopped() {
		return errPoolStopping
	}
	defer func() {
		if r := recover(); r != nil {
			err = errPoolStopping
		}
	}()
	t.taskQueue <- task
	return nil
}

// start initializes the thread pool with the minimum number of threads.
func (p *pool) start() error {
	if err := p.admissionError(); err != nil {
		return err
	}
	if p.getState() == poolStateRunning {
		return nil
	}
	if err := p.warmupMinThreadsParallel(); err != nil {
		return err
	}
	p.setState(poolStateRunning)

	// Start cleanup goroutine if TTL or max executions are configured
	if (p.executor.options.threadTTL > 0 || p.executor.options.maxExecutions > 0) && atomic.CompareAndSwapUint32(&p.cleanupStarted, 0, 1) {
		go p.retireThreads()
	}

	if p.executor.logger != nil {
		p.executor.logger.Debug("thread pool started",
			"min_pool_size", p.executor.options.minPoolSize,
			"max_pool_size", p.executor.options.maxPoolSize,
			"queue_size", p.executor.options.queueSize,
			"thread_ttl_ms", p.executor.options.threadTTL.Milliseconds(),
			"max_executions", p.executor.options.maxExecutions,
			"execute_timeout_ms", p.executor.options.executeTimeout.Milliseconds(),
			"create_threshold", p.executor.options.createThreshold,
			"select_threshold", p.executor.options.selectThreshold,
			"initial_threads", p.threadCount,
		)
	}
	return nil
}

type threadWarmupResult struct {
	thread *thread
	err    error
}

// warmupMinThreadsParallel prewarms the minimum thread count in parallel.
// If any initialization fails, all successfully created warmup threads are rolled back.
func (p *pool) warmupMinThreadsParallel() error {
	min := p.executor.options.minPoolSize
	if min == 0 {
		return nil
	}

	results := make(chan threadWarmupResult, min)
	for i := uint32(0); i < min; i++ {
		go func() {
			t, err := p.createThread()
			results <- threadWarmupResult{thread: t, err: err}
		}()
	}

	created := make([]*thread, 0, min)
	failed := 0
	var firstErr error
	for i := uint32(0); i < min; i++ {
		res := <-results
		if res.err != nil {
			failed++
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		if res.thread != nil {
			created = append(created, res.thread)
		}
	}

	if failed == 0 {
		return nil
	}

	p.rollbackWarmupThreads(created)
	return fmt.Errorf("failed to warm up thread pool: target=%d created=%d failed=%d: %w", min, len(created), failed, firstErr)
}

// rollbackWarmupThreads removes and stops threads created during warmup.
func (p *pool) rollbackWarmupThreads(threads []*thread) {
	for _, t := range threads {
		if t == nil {
			continue
		}
		if _, loaded := p.threads.LoadAndDelete(t.threadId); loaded {
			p.removeThreadFromList(t.threadId)
			atomic.AddUint32(&p.threadCount, ^uint32(0)) // -1
		}
	}

	for _, t := range threads {
		if t == nil {
			continue
		}
		t.stop()
	}
}

// stop shuts down the thread pool and all threads.
func (p *pool) stop() error {
	p.stopOnce.Do(func() {
		p.setState(poolStateStopping)
		close(p.stopCleanup)

		p.controlMu.Lock()
		threads := p.snapshotThreads()
		for _, t := range threads {
			p.detachThread(t.threadId)
		}
		p.controlMu.Unlock()

		for _, t := range threads {
			t.stop()
		}

		if atomic.LoadUint32(&p.cleanupStarted) == 1 {
			<-p.cleanupDone
		}

		p.threads = sync.Map{}
		emptyIds := make([]uint32, 0)
		p.threadIds.Store(&emptyIds)
		atomic.StoreUint32(&p.threadCount, 0)
		p.setState(poolStateStopped)

		if p.executor.logger != nil {
			p.executor.logger.Debug("thread pool stopped")
		}

		close(p.stopDone)
	})

	<-p.stopDone
	return nil
}

// addThreadToList adds a thread ID to the round-robin list using copy-on-write.
func (p *pool) addThreadToList(threadId uint32) {
	for {
		oldIdsPtr := p.threadIds.Load().(*[]uint32)
		oldIds := *oldIdsPtr
		newIds := make([]uint32, len(oldIds)+1)
		copy(newIds, oldIds)
		newIds[len(oldIds)] = threadId

		// Use CompareAndSwap with pointer to slice
		if p.threadIds.CompareAndSwap(oldIdsPtr, &newIds) {
			break
		}
		// If CAS fails, retry with the updated list
	}
}

// removeThreadFromList removes a thread ID from the round-robin list using copy-on-write.
func (p *pool) removeThreadFromList(threadId uint32) {
	for {
		oldIdsPtr := p.threadIds.Load().(*[]uint32)
		oldIds := *oldIdsPtr
		newIds := make([]uint32, 0, len(oldIds))

		// Copy all IDs except the one to remove
		for _, id := range oldIds {
			if id != threadId {
				newIds = append(newIds, id)
			}
		}

		// Use CompareAndSwap with pointer to slice
		if p.threadIds.CompareAndSwap(oldIdsPtr, &newIds) {
			break
		}
		// If CAS fails, retry with the updated list
	}
}

// createThread creates and starts a new thread with atomic thread count control.
func (p *pool) createThread() (*thread, error) {
	if err := p.admissionError(); err != nil {
		return nil, err
	}
	newCount := atomic.AddUint32(&p.threadCount, 1)
	if newCount > p.executor.options.maxPoolSize {
		atomic.AddUint32(&p.threadCount, ^uint32(0)) // -1
		return nil, fmt.Errorf("max pool size reached")
	}

	threadId := atomic.AddUint32(&p.threadIdCounter, 1)
	t := newThread(p.executor, "thread-"+strconv.FormatUint(uint64(threadId), 10), threadId)

	// Start the thread goroutine
	go t.run()

	// Wait for the thread to finish initialization
	if err := <-t.initCh; err != nil {
		// Initialization failed, so we decrement the thread count and return the error.
		// The thread is not in the pool yet, so no need to remove it from maps.
		atomic.AddUint32(&p.threadCount, ^uint32(0)) // -1
		return nil, fmt.Errorf("thread initialization failed: %w", err)
	}

	// Add thread to sync.Map (lock-free)
	p.threads.Store(threadId, t)

	// Add thread ID to round-robin list (copy-on-write)
	p.addThreadToList(threadId)

	return t, nil
}

// selectThread selects an appropriate thread for executing a request using lock-free round-robin.
func (p *pool) selectThread(req *jsengine.JsRequest) *thread {
	if err := p.admissionError(); err != nil {
		return nil
	}
	// 1. Check whether a specific thread routing is requested.
	if req != nil && req.Routing != nil && req.Routing.ThreadID > 0 {
		if t, found := p.threads.Load(req.Routing.ThreadID); found {
			return t.(*thread)
		}
	}

	// 2. Get current thread ID snapshot for round-robin selection
	threadIds := *p.threadIds.Load().(*[]uint32)
	listLen := len(threadIds)
	if listLen == 0 {
		return nil
	}

	// 3. Try to find a thread that's not too busy (load balancing)
	startIndex := atomic.AddUint32(&p.roundRobinIndex, 1) % uint32(listLen)
	for i := 0; i < listLen; i++ {
		index := (startIndex + uint32(i)) % uint32(listLen)
		threadId := threadIds[index]

		if t, exists := p.threads.Load(threadId); exists {
			thread := t.(*thread)
			queueThreshold := int(float64(p.executor.options.queueSize) * p.executor.options.selectThreshold)
			if len(thread.taskQueue) < queueThreshold {
				return thread
			}
		}
	}

	// 4. If all threads are busy, return the next thread in round-robin order
	threadId := threadIds[startIndex]
	if t, exists := p.threads.Load(threadId); exists {
		return t.(*thread)
	}

	return nil
}

// getOrCreateThread gets an existing thread or creates a new one if needed.
func (p *pool) getOrCreateThread(req *jsengine.JsRequest) (*thread, error) {
	if err := p.admissionError(); err != nil {
		return nil, err
	}
	// Try to get an existing thread first
	if t := p.selectThread(req); t != nil {
		queueThreshold := int(float64(p.executor.options.queueSize) * p.executor.options.createThreshold)
		if len(t.taskQueue) < queueThreshold {
			return t, nil
		}
	}

	// Use atomic threadCount for concurrency safety
	currentThreadCount := atomic.LoadUint32(&p.threadCount)
	if currentThreadCount < p.executor.options.maxPoolSize {
		if p.executor.logger != nil {
			p.executor.logger.Debug("thread creation triggered by load",
				"current_threads", currentThreadCount,
				"max_pool_size", p.executor.options.maxPoolSize)
		}
		if t, err := p.createThread(); err == nil {
			return t, nil
		}
	}

	// At max capacity, return an existing thread
	t := p.selectThread(req)
	if t == nil {
		return nil, fmt.Errorf("no available thread in pool")
	}
	return t, nil
}

// execute executes a task using an appropriate thread.
func (p *pool) execute(task *task) (*jsengine.JsResponse, error) {
	if err := p.admissionError(); err != nil {
		return nil, err
	}
	// Get a thread for execution
	t, err := p.getOrCreateThread(task.request)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	// Enqueue the task
	if err := p.enqueueTask(t, task); err != nil {
		return nil, err
	}

	// Wait for the result. Cancellation/timeouts are enforced via task.ctx
	// (and should interrupt the JS runtime), so we can safely return ctx.Err()
	// without relying on a separate wait-timeout.
	select {
	case result := <-task.resultChan:
		return result.response, result.err
	case <-task.ctx.Done():
		// If the task finished concurrently, prefer its result.
		select {
		case result := <-task.resultChan:
			return result.response, result.err
		default:
			return nil, task.ctx.Err()
		}
	}
}

// reload reloads all threads with new scripts.
func (p *pool) reload() error {
	p.controlMu.Lock()
	defer p.controlMu.Unlock()

	if err := p.admissionError(); err != nil {
		return err
	}

	threads := p.snapshotThreads()
	var reloadError error
	for _, t := range threads {
		if err := p.admissionError(); err != nil {
			return err
		}
		if err := t.reload(); err != nil {
			reloadError = fmt.Errorf("failed to reload thread %s: %w", t.name, err)
			break
		}
	}

	if p.executor.logger != nil {
		if reloadError == nil {
			p.executor.logger.Debug("thread pool reloaded",
				"thread_count", p.threadCount,
			)
		}
	}

	return reloadError
}

// retireThreads runs the background cleanup process for idle or overused threads.
func (p *pool) retireThreads() {
	defer close(p.cleanupDone)
	var ticker *time.Ticker
	if p.executor.options.threadTTL > 0 {
		ticker = time.NewTicker(p.executor.options.threadTTL / 2)
	} else {
		ticker = time.NewTicker(time.Minute)
	}
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performCleanup()
		case <-p.replenishChan:
			p.replenish()
		case <-p.stopCleanup:
			return
		}
	}
}

// shouldRemoveThread determines if a thread should be removed based on TTL and execution count.
func (p *pool) shouldRemoveThread(t *thread, now time.Time) bool {
	// Check TTL if configured
	if p.executor.options.threadTTL > 0 {
		if now.Sub(t.getLastUsed()) > p.executor.options.threadTTL {
			return true
		}
	}

	// Check max executions if configured
	if p.executor.options.maxExecutions > 0 {
		if t.getTaskCount() >= p.executor.options.maxExecutions {
			return true
		}
	}

	return false
}

// performCleanup performs the actual cleanup of idle or overused threads using lock-free operations.
func (p *pool) performCleanup() {
	if p.getState() != poolStateRunning {
		return
	}
	now := time.Now()
	currentThreadCount := atomic.LoadUint32(&p.threadCount)

	if currentThreadCount <= p.executor.options.minPoolSize {
		return
	}

	// Collect threads to remove by ranging over sync.Map
	var threadsToRemove []threadCleanupInfo
	p.threads.Range(func(key, value interface{}) bool {
		threadId := key.(uint32)
		t := value.(*thread)

		if p.shouldRemoveThread(t, now) {
			// Ensure we don't remove too many threads
			if currentThreadCount-uint32(len(threadsToRemove)) > p.executor.options.minPoolSize {
				threadsToRemove = append(threadsToRemove, threadCleanupInfo{
					id:     threadId,
					thread: t,
				})
			}
		}
		return true // Continue iteration
	})

	if len(threadsToRemove) == 0 {
		return
	}

	// Remove threads from pool and update counters
	retireCandidates := make([]threadCleanupInfo, 0, len(threadsToRemove))
	for _, info := range threadsToRemove {
		if p.detachThread(info.id) {
			retireCandidates = append(retireCandidates, info)
		}
	}

	newThreadCount := atomic.LoadUint32(&p.threadCount)

	// Stop threads asynchronously (no locks needed)
	for _, info := range retireCandidates {
		taskCount := info.thread.getTaskCount()
		lastUsed := info.thread.getLastUsed()

		go func(th *thread, tc uint32, lu time.Time) {
			th.retire()
			reason := "idle timeout"
			if p.executor.options.maxExecutions > 0 && tc >= p.executor.options.maxExecutions {
				reason = "max executions reached"
			}
			if p.executor.logger != nil {
				p.executor.logger.Debug("thread retired",
					"thread", th.name,
					"reason", reason,
					"execution_count", tc,
					"idle_time_ms", now.Sub(lu).Milliseconds(),
					"remaining_threads", newThreadCount)
			}
		}(info.thread, taskCount, lastUsed)
	}
}

// replenish checks if the pool needs more threads and creates them if necessary.
func (p *pool) replenish() {
	if p.getState() != poolStateRunning {
		return
	}
	for {
		if p.getState() != poolStateRunning {
			break
		}
		current := atomic.LoadUint32(&p.threadCount)
		if current >= p.executor.options.minPoolSize {
			break
		}
		if _, err := p.createThread(); err != nil {
			if p.executor.logger != nil {
				p.executor.logger.Error("replenishment thread creation failed", "error", err)
			}
			break
		}
	}
}
