// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"time"

	"github.com/fsnotify/fsnotify"
	xfmt "golang.org/x/exp/errors/fmt"
)

var watchDebounceWindow = 75 * time.Millisecond

func (s *GRPCWebServer) watch() {
	if s == nil {
		return
	}
	defer s.hotreload.finishWatchLoop()
	stopSignal := s.hotreload.watchStopSignal()
	watchEvents := s.hotreload.watchEvents()
	watchErrors := s.hotreload.watchErrors()
	for {
		select {
		case <-stopSignal:
			return
		case event, ok := <-watchEvents:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				s.enqueueWatchEvent(event.Name)
			}
		case err, ok := <-watchErrors:
			if !ok {
				return
			}
			s.runtimeScope.Logger().Error("watch error", "error", err)
		}
	}
}

func (s *GRPCWebServer) enqueueWatchEvent(file string) bool {
	if s == nil {
		return false
	}
	resolvedFile, err := resolveWatchPath(file)
	if err != nil {
		s.recordDroppedWatchEvent("path_unresolved", file, err)
		return false
	}
	file = resolvedFile
	watchQueue := s.hotreloadQueue()
	if watchQueue == nil {
		s.recordDroppedWatchEvent("queue_unavailable", file, nil)
		return false
	}

	// Resolve which module this file belongs to (retained for dispatch use).
	module := s.resolveWatchModule(file)
	// Per-file dedup: only coalesce when the same file is already in-flight.
	// Different files within the same module are both queued so a no-op save
	// on file A cannot starve a real change to file B in the same module.
	if !s.hotreload.beginModuleEvent(file) {
		s.recordCoalescedWatchEvent(file)
		return false
	}
	// Pack both the file path and resolved module name together to ensure dequeue parity.
	eventInfo := file + "|" + module
	select {
	case watchQueue <- eventInfo:
		return true
	default:
		s.hotreload.finishModuleEvent(file)
		s.recordDroppedWatchEvent("queue_full", file, nil)
		return false
	}
}

func (s *GRPCWebServer) recordDroppedWatchEvent(reason string, file string, err error) {
	total := s.hotreload.recordDropped()
	if s.runtimeScope == nil {
		return
	}
	attrs := []any{"file", file, "reason", reason, "dropped_count", total}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	s.runtimeScope.Logger().Warn("watch event dropped", attrs...)
}

func (s *GRPCWebServer) recordCoalescedWatchEvent(file string) {
	total := s.hotreload.recordCoalesced()
	if s.runtimeScope == nil {
		return
	}
	s.runtimeScope.Logger().Debug("watch event coalesced", "file", file, "coalesced_count", total)
}

func (s *GRPCWebServer) waitForWatchDebounce(file string) error {
	if s == nil {
		return xfmt.Errorf("waitForWatchDebounce called on nil receiver")
	}
	if s.runtimeScope == nil {
		return xfmt.Errorf("runtime scope is nil")
	}
	if watchDebounceWindow <= 0 {
		return nil
	}
	timer := time.NewTimer(watchDebounceWindow)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	select {
	case <-s.runtimeScope.Context().Done():
		return s.runtimeScope.Context().Err()
	case <-timer.C:
		return nil
	}
}
