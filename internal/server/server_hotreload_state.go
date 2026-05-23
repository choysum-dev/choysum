// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	xfmt "golang.org/x/exp/errors/fmt"
)

const defaultHotreloadQueueSize = 1

type watchTargetHandler func(moduleName string, file string) error

type registeredWatchTarget struct {
	serviceName string
	moduleName  string
	root        string
	handle      watchTargetHandler
}

// hotreloadState keeps hotreload-specific counters and in-flight state out of the server root.
type hotreloadState struct {
	watcher      *fsnotify.Watcher
	queue        chan string
	watchMu      sync.RWMutex
	watchTargets []registeredWatchTarget
	loopMu       sync.Mutex
	loopStop     chan struct{}
	loopDone     chan struct{}
	loopRunning  bool
	loopStopping bool
	eventMu      sync.Mutex
	eventBusy    bool
	dropped      atomic.Uint64
	coalesced    atomic.Uint64
}

func (s *GRPCWebServer) hotreloadWatcher() *fsnotify.Watcher {
	return s.hotreload.watcher
}

func (s *GRPCWebServer) hotreloadQueue() chan string {
	return s.hotreload.queue
}

func (s *GRPCWebServer) hasHotreloadWatcher() bool {
	return s.hotreload.watcher != nil
}

func (s *GRPCWebServer) startHotreloadLifecycle() error {
	if !s.shouldRunHotreloadLifecycle() {
		return nil
	}
	if err := s.hotreload.ensureLifecycle(); err != nil {
		return xfmt.Errorf("Failed to create file watcher: %w", err)
	}
	if !s.hotreload.prepareWatchLoop() {
		return nil
	}
	go s.watch()
	return nil
}

func (s *GRPCWebServer) stopHotreloadLifecycle() {
	s.hotreload.stopLoop()
	_ = s.hotreload.closeWatcher()
	s.hotreload.resetLifecycle()
}

func (s *GRPCWebServer) beginWatchEvent() bool {
	return s.hotreload.beginEvent()
}

func (s *GRPCWebServer) finishWatchEvent() {
	s.hotreload.finishEvent()
}

func (s *GRPCWebServer) watchDroppedCount() uint64 {
	return s.hotreload.droppedCount()
}

func (s *GRPCWebServer) watchCoalescedCount() uint64 {
	return s.hotreload.coalescedCount()
}

func (s *GRPCWebServer) shouldRunHotreloadLifecycle() bool {
	if s == nil {
		return false
	}
	if s.hasHotreloadWatcher() || s.hotreloadQueue() != nil {
		return true
	}
	return s.resolvedRuntimeOptions().hotReload
}

func (h *hotreloadState) beginEvent() bool {
	h.eventMu.Lock()
	defer h.eventMu.Unlock()
	if h.eventBusy {
		return false
	}
	h.eventBusy = true
	return true
}

func (h *hotreloadState) finishEvent() {
	h.eventMu.Lock()
	h.eventBusy = false
	h.eventMu.Unlock()
}

func (h *hotreloadState) recordDropped() uint64 {
	return h.dropped.Add(1)
}

func (h *hotreloadState) recordCoalesced() uint64 {
	return h.coalesced.Add(1)
}

func (h *hotreloadState) droppedCount() uint64 {
	return h.dropped.Load()
}

func (h *hotreloadState) coalescedCount() uint64 {
	return h.coalesced.Load()
}

func (h *hotreloadState) ensureLifecycle() error {
	h.ensureQueue()
	return h.ensureWatcher()
}

func (h *hotreloadState) ensureWatcher() error {
	if h.watcher != nil {
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	h.watcher = watcher
	return nil
}

func (h *hotreloadState) ensureQueue() {
	if h.queue != nil {
		return
	}
	h.queue = make(chan string, defaultHotreloadQueueSize)
}

func (h *hotreloadState) addWatch(path string) error {
	if h.watcher == nil {
		return nil
	}
	return h.watcher.Add(path)
}

func (h *hotreloadState) removeWatch(path string) error {
	if h.watcher == nil {
		return nil
	}
	return h.watcher.Remove(path)
}

func (h *hotreloadState) watchList() []string {
	if h.watcher == nil {
		return nil
	}
	return h.watcher.WatchList()
}

func (h *hotreloadState) watchEvents() <-chan fsnotify.Event {
	if h.watcher == nil {
		return nil
	}
	return h.watcher.Events
}

func (h *hotreloadState) watchErrors() <-chan error {
	if h.watcher == nil {
		return nil
	}
	return h.watcher.Errors
}

func (h *hotreloadState) closeWatcher() error {
	if h.watcher == nil {
		return nil
	}
	return h.watcher.Close()
}

func (h *hotreloadState) prepareWatchLoop() bool {
	h.loopMu.Lock()
	defer h.loopMu.Unlock()
	if h.loopRunning {
		return false
	}
	h.loopStop = make(chan struct{})
	h.loopDone = make(chan struct{})
	h.loopRunning = true
	h.loopStopping = false
	return true
}

func (h *hotreloadState) watchStopSignal() <-chan struct{} {
	h.loopMu.Lock()
	defer h.loopMu.Unlock()
	return h.loopStop
}

func (h *hotreloadState) finishWatchLoop() {
	h.loopMu.Lock()
	done := h.loopDone
	h.loopDone = nil
	h.loopStop = nil
	h.loopRunning = false
	h.loopStopping = false
	h.loopMu.Unlock()
	if done != nil {
		close(done)
	}
}

func (h *hotreloadState) stopLoop() {
	h.loopMu.Lock()
	if !h.loopRunning || h.loopStopping || h.loopStop == nil {
		h.loopMu.Unlock()
		return
	}
	stop := h.loopStop
	done := h.loopDone
	h.loopStopping = true
	h.loopMu.Unlock()
	if stop != nil {
		close(stop)
	}
	if done != nil {
		<-done
	}
}

func (h *hotreloadState) resetLifecycle() {
	h.watcher = nil
	h.drainQueue()
	h.clearWatchTargets()
	h.resetEventState()
}

func (h *hotreloadState) storeWatchTargets(targets []registeredWatchTarget) {
	h.watchMu.Lock()
	h.watchTargets = append([]registeredWatchTarget{}, targets...)
	h.watchMu.Unlock()
}

func (h *hotreloadState) watchTargetsSnapshot() []registeredWatchTarget {
	h.watchMu.RLock()
	defer h.watchMu.RUnlock()
	return append([]registeredWatchTarget{}, h.watchTargets...)
}

func (h *hotreloadState) clearWatchTargets() {
	h.watchMu.Lock()
	h.watchTargets = nil
	h.watchMu.Unlock()
}

func (h *hotreloadState) drainQueue() {
	if h.queue == nil {
		return
	}
	for {
		select {
		case <-h.queue:
		default:
			return
		}
	}
}

func (h *hotreloadState) resetEventState() {
	h.eventMu.Lock()
	h.eventBusy = false
	h.eventMu.Unlock()
}
