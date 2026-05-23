// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

func (s *GRPCWebServer) dispatchWatchHandler(file string) error {
	file, err := filepath.Abs(file)
	if err != nil {
		return xfmt.Errorf("Failed to get absolute path of file: %w", err)
	}
	file, err = resolveWatchPath(file)
	if err != nil {
		return xfmt.Errorf("Failed to resolve watched file path: %w", err)
	}
	s.runtimeScope.Logger().Debug("watch file detected", "file", file)

	seenModules := map[string]struct{}{}
	for _, target := range s.hotreload.watchTargetsSnapshot() {
		contained, err := isWatchedPath(target.root, file)
		if err != nil {
			return xfmt.Errorf("Failed to evaluate watched path containment: %w", err)
		}
		if !contained {
			continue
		}
		if _, seen := seenModules[target.moduleName]; seen {
			continue
		}
		seenModules[target.moduleName] = struct{}{}
		if err := s.dispatchWatchTarget(target, file); err != nil {
			return err
		}
	}

	return nil
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
	ctx := context.Background()
	txRoot := s.runtimeScope.WithContext(ctx)
	if err := txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
		moduleLifecycle := lifecycle.NewService(txScope, s.jsExecutor)
		if err := moduleLifecycle.Upgrade(tx.Context(), lifecycle.UpgradeRequest{Input: moduleName}); err != nil {
			return xfmt.Errorf("error upgrading module %s: %w", moduleName, err)
		}
		return nil
	}); err != nil {
		return xfmt.Errorf("error running watch handler: %w", err)
	}
	s.runtimeScope.Logger().Debug("watch module upgraded", "module", moduleName, "file", file)
	return nil
}

func (s *GRPCWebServer) handleWatchedFileChange(file string) error {
	if err := s.dispatchWatchHandler(file); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(s.runtimeScope.Context().Err(), context.Canceled) {
			s.runtimeScope.Logger().Debug("watch handler canceled", "error", err)
			return nil
		}
		return xfmt.Errorf("Failed to dispatch watch handler: %w", err)
	}
	if err := s.restart(); err != nil {
		return xfmt.Errorf("Failed to restart server: %w", err)
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
	defer s.finishWatchEvent()
	if err := s.waitForWatchDebounce(file); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return xfmt.Errorf("Failed to wait for watch debounce: %w", err)
	}
	return s.handleWatchedFileChange(file)
}
