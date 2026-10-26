// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

func TestWaitWaitGroup_CompletesBeforeTimeout(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
	}()
	if !waitWaitGroup(&wg, time.Second) {
		t.Fatal("expected WaitGroup to finish before timeout")
	}
}

func TestWaitWaitGroup_TimesOut(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	start := time.Now()
	if waitWaitGroup(&wg, 30*time.Millisecond) {
		t.Fatal("expected timeout while WaitGroup still held")
	}
	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Fatalf("timeout returned too early: %v", elapsed)
	}
	wg.Done()
}

func TestWaitWaitGroup_NilAndNonPositive(t *testing.T) {
	if !waitWaitGroup(nil, time.Millisecond) {
		t.Fatal("nil WaitGroup should succeed")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	var finishedOK bool
	go func() {
		finishedOK = waitWaitGroup(&wg, 0)
		close(done)
	}()
	wg.Done()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("unbounded wait did not return")
	}
	if !finishedOK {
		t.Fatal("non-positive timeout should wait unbounded and succeed")
	}
}

func TestDispatcherStop_TimesOutWaitingForInFlightWork(t *testing.T) {
	prev := stopWaitTimeout
	stopWaitTimeout = 40 * time.Millisecond
	t.Cleanup(func() { stopWaitTimeout = prev })

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	dispatcher := NewDispatcherWithRuntime(
		&testScope{ctx: context.Background(), logger: logger, cfg: &config.Config{Task: config.NewDefaultTaskConfig()}},
		nil,
		taskcontract.Runtime{Queue: cancelAwareTaskQueue{}},
	)
	dispatcher.interval = time.Hour
	dispatcher.Start()
	// Simulate a hung job worker that never calls Done.
	dispatcher.wg.Add(1)

	start := time.Now()
	dispatcher.Stop()
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Fatalf("Stop returned too quickly (%v); expected stopWaitTimeout", elapsed)
	}
	if elapsed > time.Second {
		t.Fatalf("Stop hung too long: %v", elapsed)
	}
	if !strings.Contains(buf.String(), "task dispatcher stop timed out waiting for in-flight work") {
		t.Fatalf("expected timeout WARN, got logs: %s", buf.String())
	}
}
