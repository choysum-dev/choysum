// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	logutil "github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	xfmt "golang.org/x/exp/errors/fmt"
)

func (s *GRPCWebServer) dispatchWatchHandler(file string) (int, error) {
	if s == nil {
		return 0, xfmt.Errorf("dispatchWatchHandler called on nil receiver")
	}
	resolvedFile, err := resolveWatchPath(file)
	if err != nil {
		return 0, xfmt.Errorf("Failed to resolve watched file path: %w", err)
	}
	return s.dispatchWatchHandlerResolved(resolvedFile)
}

func (s *GRPCWebServer) dispatchWatchHandlerResolved(resolvedFile string) (int, error) {
	if s == nil {
		return 0, xfmt.Errorf("dispatchWatchHandlerResolved called on nil receiver")
	}
	if s.runtimeScope == nil {
		return 0, xfmt.Errorf("runtime scope is nil")
	}
	s.runtimeScope.Logger().Debug("watch file detected", "file", resolvedFile)

	seenModules := map[string]struct{}{}
	dispatched := 0
	for _, target := range s.hotreload.watchTargetsSnapshot() {
		contained, err := isWatchedPath(target.root, resolvedFile)
		if err != nil {
			return dispatched, xfmt.Errorf("Failed to evaluate watched path containment: %w", err)
		}
		if !contained {
			continue
		}
		if _, seen := seenModules[target.moduleName]; seen {
			continue
		}
		seenModules[target.moduleName] = struct{}{}
		if err := s.dispatchWatchTarget(target, resolvedFile); err != nil {
			return dispatched, err
		}
		dispatched++
	}

	return dispatched, nil
}

func (s *GRPCWebServer) dispatchWatchTarget(target registeredWatchTarget, file string) error {
	if s == nil {
		return xfmt.Errorf("dispatchWatchTarget called on nil receiver")
	}
	handle := target.handle
	if handle == nil {
		handle = s.handleWatchedModuleUpgrade
	}
	if err := handle(target.moduleName, file); err != nil {
		return xfmt.Errorf("Failed to call watch callback for %s: %w", target.serviceName, err)
	}
	return nil
}

func (s *GRPCWebServer) handleWatchedModuleUpgrade(moduleName string, file string) error {
	if s == nil {
		return xfmt.Errorf("handleWatchedModuleUpgrade called on nil receiver")
	}
	if s.runtimeScope == nil {
		return xfmt.Errorf("runtime scope is nil")
	}
	s.runtimeScope.Logger().Debug("watch module upgrade started", "module", moduleName, "file", file)
	s.hotreload.progressMu.Lock()
	line := s.hotreload.progressLine
	if line != nil {
		line.Update(0, fmt.Sprintf("Upgrading module: %s", moduleName))
	}
	s.hotreload.progressMu.Unlock()
	ctx := s.runtimeScope.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if line != nil {
		ctx = logutil.WithProgressLine(ctx, line)
	}
	moduleLifecycle := lifecycle.NewService(s.runtimeScope.WithContext(ctx), s.jsExecutor)
	if err := moduleLifecycle.Upgrade(ctx, lifecycle.UpgradeRequest{Input: moduleName}); err != nil {
		return xfmt.Errorf("error running watch handler: %w", err)
	}
	s.runtimeScope.Logger().Debug("watch module upgraded", "module", moduleName, "file", file)
	return nil
}

func (s *GRPCWebServer) handleWatchedFileChange(file string) error {
	if s == nil {
		return xfmt.Errorf("handleWatchedFileChange called on nil receiver")
	}
	resolvedFile, err := resolveWatchPath(file)
	if err != nil {
		return xfmt.Errorf("Failed to resolve watched file path: %w", err)
	}
	return s.handleWatchedFileChangeResolved(resolvedFile)
}

func (s *GRPCWebServer) handleWatchedFileChangeResolved(resolvedFile string) error {
	if s == nil {
		return xfmt.Errorf("handleWatchedFileChangeResolved called on nil receiver")
	}
	if s.runtimeScope == nil {
		return xfmt.Errorf("runtime scope is nil")
	}
	// Skip reload when file content hasn't actually changed (e.g. no-op
	// save or atomic-save that produces an identical file). The check runs
	// after debounce so temp-file rename sequences have already settled.
	if !s.hotreload.contentChangedResolved(resolvedFile) {
		return nil
	}

	s.hotreload.progressMu.Lock()
	line := s.hotreload.progressLine
	if line != nil {
		line.Update(0, fmt.Sprintf("Detected change: %s", filepath.Base(resolvedFile)))
	}
	s.hotreload.progressMu.Unlock()

	dispatched, err := s.dispatchWatchHandlerResolved(resolvedFile)
	if err != nil {
		if errors.Is(err, context.Canceled) || (s.runtimeScope.Context() != nil && errors.Is(s.runtimeScope.Context().Err(), context.Canceled)) {
			s.runtimeScope.Logger().Debug("watch handler canceled", "error", err)
			return nil
		}
		// Evict fingerprint so the user can retry by saving again.
		s.hotreload.clearFingerprint(resolvedFile)
		if line != nil {
			s.hotreload.progressMu.Lock()
			line.Clear()
			line.Done("✗", fmt.Sprintf("Hotreload failed: %s", filepath.Base(resolvedFile)))
			s.hotreload.progressMu.Unlock()
		}
		return xfmt.Errorf("Failed to dispatch watch handler: %w", err)
	}
	// Skip restart when no module matched the changed file (e.g. file outside
	// any registered watch root).
	if dispatched == 0 {
		if line != nil {
			s.hotreload.progressMu.Lock()
			line.Done("→", fmt.Sprintf("Skipped (no module matched): %s", filepath.Base(resolvedFile)))
			s.hotreload.progressMu.Unlock()
		}
		s.runtimeScope.Logger().Debug("watch file did not match any module, skipping restart", "file", resolvedFile)
		return nil
	}

	if line != nil {
		s.hotreload.progressMu.Lock()
		line.Update(1, "Restarting runtime...")
		s.hotreload.progressMu.Unlock()
	}
	if err := s.restart(); err != nil {
		// Evict fingerprint so the user can retry by saving again.
		s.hotreload.clearFingerprint(resolvedFile)
		if line != nil {
			s.hotreload.progressMu.Lock()
			line.Clear()
			line.Done("✗", "Restart failed")
			s.hotreload.progressMu.Unlock()
		}
		return xfmt.Errorf("Failed to restart server: %w", err)
	}

	if line != nil {
		s.hotreload.progressMu.Lock()
		line.Done("✓", fmt.Sprintf("Hotreload done (dropped/coalesced: %d/%d)", s.watchDroppedCount(), s.watchCoalescedCount()))
		s.hotreload.progressMu.Unlock()
	}
	s.runtimeScope.Logger().Info(
		"watch reload completed",
		"file", resolvedFile,
		"watch_dropped_count", s.watchDroppedCount(),
		"watch_coalesced_count", s.watchCoalescedCount(),
	)
	return nil
}

func (s *GRPCWebServer) handleQueuedWatchEvent(eventInfo string) error {
	if s == nil {
		return xfmt.Errorf("handleQueuedWatchEvent called on nil receiver")
	}
	if s.runtimeScope == nil {
		return xfmt.Errorf("runtime scope is nil")
	}
	// Parse the packed file path and module name to maintain deduplication consistency.
	// eventInfo format: "file|module"
	var file, module string
	if idx := strings.LastIndex(eventInfo, "|"); idx >= 0 {
		file = eventInfo[:idx]
		module = eventInfo[idx+1:]
	} else {
		file = eventInfo
		module = s.resolveWatchModule(file)
	}
	defer s.hotreload.finishModuleEvent(module)

	if err := s.waitForWatchDebounce(file); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return xfmt.Errorf("Failed to wait for watch debounce: %w", err)
	}
	return s.handleWatchedFileChange(file)
}
