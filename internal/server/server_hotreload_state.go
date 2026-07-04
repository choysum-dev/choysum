// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/fsnotify/fsnotify"
	xfmt "golang.org/x/exp/errors/fmt"
)

const defaultHotreloadQueueSize = 8

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
	queueSize    int
	watchMu      sync.RWMutex
	watchTargets []registeredWatchTarget
	loopMu       sync.Mutex
	loopStop     chan struct{}
	loopDone     chan struct{}
	loopRunning  bool
	loopStopping bool

	// Per-module busy tracking replaces the previous global eventBusy flag.
	// Only one event per module is kept in-flight; files for different modules
	// can be queued concurrently up to the channel capacity.
	moduleMu    sync.Mutex
	busyModules map[string]struct{}

	// progressMu protects progressLine method calls.
	progressMu sync.Mutex

	// progressLine writes single-line hotreload status updates to stderr.
	progressLine *logger.ProgressLine

	// fingerprints caches file content hashes to detect no-op saves.
	// Key: resolved absolute path. Value: hex-encoded SHA-256.
	fingerprints   map[string]string
	fingerprintsMu sync.Mutex

	dropped   atomic.Uint64
	coalesced atomic.Uint64
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
	if s.hotreload.progressLine == nil && s.runtimeScope != nil {
		s.hotreload.progressLine = logger.ProgressLineFromContext(s.runtimeScope.Context())
	}
	s.hotreload.queueSize = s.resolvedQueueSize()
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

func (s *GRPCWebServer) resolvedQueueSize() int {
	if s == nil {
		return defaultHotreloadQueueSize
	}
	size := s.resolvedRuntimeOptions().hotReloadQueueSize
	if size <= 0 {
		return defaultHotreloadQueueSize
	}
	return size
}

func (h *hotreloadState) beginEvent() bool {
	h.moduleMu.Lock()
	defer h.moduleMu.Unlock()
	if h.busyModules == nil {
		h.busyModules = make(map[string]struct{})
	}
	// Backward-compatible global gate: any module string works;
	// callers should migrate to beginModuleEvent for per-module dedup.
	if _, busy := h.busyModules[""]; busy {
		return false
	}
	h.busyModules[""] = struct{}{}
	return true
}

func (h *hotreloadState) finishEvent() {
	h.moduleMu.Lock()
	delete(h.busyModules, "")
	h.moduleMu.Unlock()
}

func (h *hotreloadState) beginModuleEvent(module string) bool {
	if module == "" {
		return h.beginEvent()
	}
	h.moduleMu.Lock()
	defer h.moduleMu.Unlock()
	if h.busyModules == nil {
		h.busyModules = make(map[string]struct{})
	}
	if _, busy := h.busyModules[module]; busy {
		return false
	}
	h.busyModules[module] = struct{}{}
	return true
}

func (h *hotreloadState) finishModuleEvent(module string) {
	if module == "" {
		h.finishEvent()
		return
	}
	h.moduleMu.Lock()
	delete(h.busyModules, module)
	h.moduleMu.Unlock()
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
	h.ensureProgressLine()
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
	size := h.queueSize
	if size <= 0 {
		size = defaultHotreloadQueueSize
	}
	h.queue = make(chan string, size)
}

func (h *hotreloadState) ensureProgressLine() {
	if h.progressLine != nil {
		return
	}
	h.progressLine = logger.NewProgressLine(os.Stderr)
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
	h.clearFingerprints()
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
	h.moduleMu.Lock()
	h.busyModules = nil
	h.moduleMu.Unlock()
}

func (h *hotreloadState) primeFingerprintsForTargets(targets []registeredWatchTarget) {
	roots := make([]string, 0, len(targets))
	for _, target := range targets {
		roots = append(roots, target.root)
	}
	h.primeFingerprintsForRoots(roots)
}

func (h *hotreloadState) primeFingerprintsForRoots(roots []string) {
	seenRoots := map[string]struct{}{}
	for _, root := range roots {
		if _, seen := seenRoots[root]; seen {
			continue
		}
		seenRoots[root] = struct{}{}
		h.primeFingerprintsForRoot(root)
	}
}

func (h *hotreloadState) primeFingerprintsForRoot(root string) {
	if root == "" {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		canonical := canonicalWatchPath(path)
		if h.hasFingerprint(canonical) {
			return nil
		}
		hash, ok := fileFingerprint(canonical)
		if !ok {
			return nil
		}
		h.setFingerprintIfAbsentResolved(canonical, hash)
		return nil
	})
}

// contentChanged returns true when the file content differs from the cached
// fingerprint. The fingerprint is always updated after the comparison so the
// next no-op save is ignored. Non-regular files are treated as changed to
// stay on the safe side.
func (h *hotreloadState) contentChanged(resolvedPath string) bool {
	return h.contentChangedResolved(canonicalWatchPath(resolvedPath))
}

func (h *hotreloadState) contentChangedResolved(resolvedPath string) bool {
	path := canonicalWatchPath(resolvedPath)
	hash, ok := fileFingerprint(path)
	if !ok {
		h.fingerprintsMu.Lock()
		if h.fingerprints != nil {
			delete(h.fingerprints, path)
		}
		h.fingerprintsMu.Unlock()
		return true
	}

	h.fingerprintsMu.Lock()
	prev, ok := h.fingerprints[path]
	if h.fingerprints == nil {
		h.fingerprints = make(map[string]string)
	}
	h.fingerprints[path] = hash
	h.fingerprintsMu.Unlock()

	return !ok || hash != prev
}

func fileFingerprint(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", false
	}
	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum), true
}

func (h *hotreloadState) hasFingerprint(path string) bool {
	h.fingerprintsMu.Lock()
	defer h.fingerprintsMu.Unlock()
	if h.fingerprints == nil {
		return false
	}
	_, exists := h.fingerprints[canonicalWatchPath(path)]
	return exists
}

func (h *hotreloadState) setFingerprintIfAbsent(path string, hash string) {
	h.setFingerprintIfAbsentResolved(canonicalWatchPath(path), hash)
}

func (h *hotreloadState) setFingerprintIfAbsentResolved(path string, hash string) {
	h.fingerprintsMu.Lock()
	if h.fingerprints == nil {
		h.fingerprints = make(map[string]string)
	}
	if _, exists := h.fingerprints[path]; !exists {
		h.fingerprints[path] = hash
	}
	h.fingerprintsMu.Unlock()
}

func canonicalWatchPath(path string) string {
	resolved, err := resolveWatchPath(path)
	if err == nil {
		return resolved
	}
	return path
}

// clearFingerprints drops the cached file fingerprints so the next write
// after a hotreload lifecycle reset always triggers.
func (h *hotreloadState) clearFingerprints() {
	h.fingerprintsMu.Lock()
	h.fingerprints = nil
	h.fingerprintsMu.Unlock()
}

// clearFingerprint removes a single cached fingerprint so that a failed
// hot reload can be retried by saving the same file again.
func (h *hotreloadState) clearFingerprint(path string) {
	h.fingerprintsMu.Lock()
	if h.fingerprints != nil {
		delete(h.fingerprints, path)
	}
	h.fingerprintsMu.Unlock()
}
