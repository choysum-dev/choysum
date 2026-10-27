// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
)

type resolveLogCapture struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *resolveLogCapture) Enabled(context.Context, slog.Level) bool { return true }

func (h *resolveLogCapture) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *resolveLogCapture) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *resolveLogCapture) WithGroup(string) slog.Handler      { return h }

func (h *resolveLogCapture) levelsFor(msg string) []slog.Level {
	h.mu.Lock()
	defer h.mu.Unlock()
	var levels []slog.Level
	for _, r := range h.records {
		if r.Message == msg {
			levels = append(levels, r.Level)
		}
	}
	return levels
}

type resolveLogTestScope struct {
	sourceTestScope
	logger *slog.Logger
}

func (s *resolveLogTestScope) Logger() *slog.Logger { return s.logger }

func TestLogResolveInstallOutcomeLevels(t *testing.T) {
	t.Parallel()
	capture := &resolveLogCapture{}
	runtimeScope := &resolveLogTestScope{
		sourceTestScope: sourceTestScope{
			ctx: context.Background(),
			cfg: &config.Config{},
		},
		logger: slog.New(capture),
	}
	coordinator := &Coordinator{runtimeScope: runtimeScope}
	parsed := ParsedInput{Kind: InputKindLocal, LocalName: "auth"}

	coordinator.logResolveInstallOutcome(parsed, "local", false, time.Now(), nil)
	if levels := capture.levelsFor("origin install resolve succeeded"); len(levels) != 1 || levels[0] != slog.LevelDebug {
		t.Fatalf("succeeded levels=%v, want [Debug]", levels)
	}

	coordinator.logResolveInstallOutcome(parsed, "registry", true, time.Now(), nil)
	if levels := capture.levelsFor("origin install resolve fallback"); len(levels) != 1 || levels[0] != slog.LevelInfo {
		t.Fatalf("fallback levels=%v, want [Info]", levels)
	}

	coordinator.logResolveInstallOutcome(parsed, "local", false, time.Now(), errors.New("boom"))
	if levels := capture.levelsFor("origin install resolve failed"); len(levels) != 1 || levels[0] != slog.LevelWarn {
		t.Fatalf("failed levels=%v, want [Warn]", levels)
	}
}
