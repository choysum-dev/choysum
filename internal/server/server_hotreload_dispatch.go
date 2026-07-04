// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	logutil "github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	xfmt "golang.org/x/exp/errors/fmt"
)

func (s *GRPCWebServer) dispatchWatchHandler(file string) (int, error) {
	file, err := filepath.Abs(file)
	if err != nil {
		return 0, xfmt.Errorf("Failed to get absolute path of file: %w", err)
	}
	file, err = resolveWatchPath(file)
	if err != nil {
		return 0, xfmt.Errorf("Failed to resolve watched file path: %w", err)
	}
	s.runtimeScope.Logger().Debug("watch file detected", "file", file)

	seenModules := map[string]struct{}{}
	dispatched := 0
	for _, target := range s.hotreload.watchTargetsSnapshot() {
		contained, err := isWatchedPath(target.root, file)
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
		if err := s.dispatchWatchTarget(target, file); err != nil {
			return dispatched, err
		}
		dispatched++
	}

	return dispatched, nil
}

func (s *GRPCWebServer) dispatchWatchTarget(target registeredWatchTarget, file string) error {
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
	s.runtimeScope.Logger().Debug("watch module upgrade started", "module", moduleName, "file", file)
	line := s.hotreload.progressLine
	if line != nil {
		line.Update(0, fmt.Sprintf("Upgrading module: %s", moduleName))
	}
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
	line := s.hotreload.progressLine
	if line != nil {
		line.Update(0, fmt.Sprintf("Detected change: %s", filepath.Base(file)))
	}

	dispatched, err := s.dispatchWatchHandler(file)
	if err != nil {
		if line != nil {
			line.Clear()
			line.Done("✗", fmt.Sprintf("Hotreload failed: %s", filepath.Base(file)))
		}
		if errors.Is(err, context.Canceled) || errors.Is(s.runtimeScope.Context().Err(), context.Canceled) {
			s.runtimeScope.Logger().Debug("watch handler canceled", "error", err)
			return nil
		}
		return xfmt.Errorf("Failed to dispatch watch handler: %w", err)
	}
	// Skip restart when no module matched the changed file (e.g. file outside
	// any registered watch root).
	if dispatched == 0 {
		if line != nil {
			line.Done("→", fmt.Sprintf("Skipped (no module matched): %s", filepath.Base(file)))
		}
		s.runtimeScope.Logger().Debug("watch file did not match any module, skipping restart", "file", file)
		return nil
	}

	if line != nil {
		line.Update(1, "Restarting runtime...")
	}
	if err := s.restart(); err != nil {
		if line != nil {
			line.Clear()
			line.Done("✗", "Restart failed")
		}
		return xfmt.Errorf("Failed to restart server: %w", err)
	}

	if line != nil {
		line.Done("✓", fmt.Sprintf("Hotreload done (dropped/coalesced: %d/%d)", s.watchDroppedCount(), s.watchCoalescedCount()))
	}
	s.runtimeScope.Logger().Info(
		"watch reload completed",
		"file", file,
		"watch_dropped_count", s.watchDroppedCount(),
		"watch_coalesced_count", s.watchCoalescedCount(),
	)
	return nil
}

func (s *GRPCWebServer) handleQueuedWatchEvent(file string) error {
	module := s.resolveWatchModule(file)
	defer s.hotreload.finishModuleEvent(module)
	if err := s.waitForWatchDebounce(file); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return xfmt.Errorf("Failed to wait for watch debounce: %w", err)
	}
	return s.handleWatchedFileChange(file)
}
