// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"os"
	"path/filepath"

	xfmt "golang.org/x/exp/errors/fmt"
)

func (s *GRPCWebServer) applyRegistrationWatchPlans(plans []registrationWatchPlan) error {
	if s == nil {
		return xfmt.Errorf("applyRegistrationWatchPlans called on nil receiver")
	}
	return s.applyRegistrationWatchPlansWithHandler(plans, s.handleWatchedModuleUpgrade)
}

func (s *GRPCWebServer) applyRegistrationWatchPlansWithHandler(plans []registrationWatchPlan, handle watchTargetHandler) error {
	if s == nil {
		return xfmt.Errorf("applyRegistrationWatchPlansWithHandler called on nil receiver")
	}
	targets := s.buildRegisteredWatchTargets(plans, handle)
	s.hotreload.storeWatchTargets(targets)
	if !s.resolvedRuntimeOptions().hotReload {
		return nil
	}
	ctx := context.Background()
	if s.runtimeScope != nil && s.runtimeScope.Context() != nil {
		ctx = s.runtimeScope.Context()
	}
	// Cancel any still-running priming from a previous lifecycle before
	// starting a new one (e.g. during hot-reload restart).
	s.hotreload.fingerprintsMu.Lock()
	if s.hotreload.primeCancel != nil {
		s.hotreload.primeCancel()
	}
	primeCtx, primeCancel := context.WithCancel(ctx)
	s.hotreload.primeCancel = primeCancel
	s.hotreload.fingerprintsMu.Unlock()

	if !s.hasHotreloadWatcher() {
		s.hotreload.primeFingerprintsForTargets(primeCtx, targets)
		primeCancel()
		return nil
	}
	for _, target := range targets {
		if err := s.registerWatchTarget(target); err != nil {
			return xfmt.Errorf("Failed to register watch dir: %w", err)
		}
	}
	s.hotreload.primeWg.Add(1)
	go func() {
		defer s.hotreload.primeWg.Done()
		s.hotreload.primeFingerprintsForTargets(primeCtx, targets)
	}()
	return nil
}

func (s *GRPCWebServer) buildRegisteredWatchTargets(plans []registrationWatchPlan, handle watchTargetHandler) []registeredWatchTarget {
	if s == nil {
		return nil
	}
	targets := make([]registeredWatchTarget, 0, len(plans))
	seen := map[string]struct{}{}
	for _, plan := range plans {
		resolvedRoot, err := resolveWatchPath(plan.Root)
		if err != nil {
			s.runtimeScope.Logger().Warn("watch root unavailable", "app", plan.ServiceName, "module", plan.ModuleName, "root", plan.Root, "error", err)
			continue
		}
		key := plan.ServiceName + "\x00" + plan.ModuleName + "\x00" + resolvedRoot
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		targets = append(targets, registeredWatchTarget{
			serviceName: plan.ServiceName,
			moduleName:  plan.ModuleName,
			root:        resolvedRoot,
			handle:      handle,
		})
	}
	return targets
}

func (s *GRPCWebServer) registerWatchTarget(target registeredWatchTarget) error {
	if s == nil {
		return xfmt.Errorf("registerWatchTarget called on nil receiver")
	}
	registeredRoots := s.registeredWatchRoots()
	if _, exists := registeredRoots[target.root]; exists {
		s.runtimeScope.Logger().Debug("watch root skipped as duplicate", "app", target.serviceName, "module", target.moduleName, "root", target.root)
		return nil
	}
	skipRoot, coveredRoots, err := overlappingWatchRoots(target.root, registeredRoots)
	if err != nil {
		s.runtimeScope.Logger().Warn("watch root overlap evaluation failed", "app", target.serviceName, "module", target.moduleName, "root", target.root, "error", err)
		return nil
	}
	if skipRoot {
		s.runtimeScope.Logger().Debug("watch root skipped as covered", "app", target.serviceName, "module", target.moduleName, "root", target.root)
		return nil
	}
	if err := s.removeWatchRoots(coveredRoots); err != nil {
		return xfmt.Errorf("Failed to prune covered watch roots: %w", err)
	}
	for _, coveredRoot := range coveredRoots {
		delete(registeredRoots, coveredRoot)
	}

	st, err := os.Stat(target.root)
	if err != nil || !st.IsDir() {
		s.runtimeScope.Logger().Warn("watch root unavailable", "app", target.serviceName, "module", target.moduleName, "root", target.root, "error", err)
		return nil
	}

	if err := filepath.Walk(target.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = s.hotreload.addWatch(path)
			if err != nil {
				return xfmt.Errorf("Failed to watch directory: %w", err)
			}
		}
		return nil
	}); err != nil {
		// Best-effort: do not block startup due to a bad watch dir.
		s.runtimeScope.Logger().Warn("watch root walk failed", "app", target.serviceName, "module", target.moduleName, "root", target.root, "error", err)
		return nil
	}
	registeredRoots[target.root] = struct{}{}
	return nil
}

func (s *GRPCWebServer) registeredWatchRoots() map[string]struct{} {
	if s == nil {
		return nil
	}
	roots := map[string]struct{}{}
	watchList := s.hotreload.watchList()
	if len(watchList) == 0 {
		return roots
	}
	for _, watchPath := range watchList {
		roots[watchPath] = struct{}{}
	}
	return roots
}

func overlappingWatchRoots(candidateRoot string, registeredRoots map[string]struct{}) (skipRoot bool, coveredRoots []string, err error) {
	for registeredRoot := range registeredRoots {
		containedByExisting, err := isWatchedPath(registeredRoot, candidateRoot)
		if err != nil {
			return false, nil, err
		}
		if containedByExisting {
			return true, nil, nil
		}
		containsExisting, err := isWatchedPath(candidateRoot, registeredRoot)
		if err != nil {
			return false, nil, err
		}
		if containsExisting {
			coveredRoots = append(coveredRoots, registeredRoot)
		}
	}
	return false, coveredRoots, nil
}

func (s *GRPCWebServer) removeWatchRoots(roots []string) error {
	if s == nil {
		return xfmt.Errorf("removeWatchRoots called on nil receiver")
	}
	if !s.hasHotreloadWatcher() {
		return nil
	}
	for _, root := range roots {
		if err := s.hotreload.removeWatch(root); err != nil {
			return err
		}
	}
	return nil
}

func (s *GRPCWebServer) clearWatchRegistrations() error {
	if s == nil {
		return xfmt.Errorf("clearWatchRegistrations called on nil receiver")
	}
	s.hotreload.clearWatchTargets()
	if !s.hasHotreloadWatcher() {
		return nil
	}
	watchList := s.hotreload.watchList()
	for _, watchPath := range watchList {
		if err := s.hotreload.removeWatch(watchPath); err != nil {
			return xfmt.Errorf("Failed to remove watch: %w", err)
		}
	}
	return nil
}
