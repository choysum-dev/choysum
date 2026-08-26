// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/internal/export/plan"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Sink serializes reader output for one format.
type Sink interface {
	Write(ctx context.Context, runtimeScope scope.Scope, p plan.Plan, result *Result) error
}

var (
	sinkMu sync.RWMutex
	sinks  = make(map[string]Sink)
)

// RegisterSink binds a format to a sink implementation.
func RegisterSink(format string, s Sink) {
	sinkMu.Lock()
	defer sinkMu.Unlock()
	sinks[normalizeFormat(format)] = s
}

// SinkFor returns the sink for format.
func SinkFor(format string) (Sink, error) {
	sinkMu.RLock()
	s, ok := sinks[normalizeFormat(format)]
	sinkMu.RUnlock()
	if !ok || s == nil {
		return nil, exportpkg.Errorf(exportpkg.CodeInvalidFormat, "sink is not registered for format")
	}
	return s, nil
}

// ResetSinksForTest clears registered sinks. Tests only.
func ResetSinksForTest() {
	sinkMu.Lock()
	sinks = make(map[string]Sink)
	sinkMu.Unlock()
}

func normalizeFormat(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}
