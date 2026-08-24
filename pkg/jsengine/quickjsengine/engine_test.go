// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

func TestQuickjsEngine_InterruptsOnContextDeadline(t *testing.T) {
	engine, err := NewFactory()()
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	scripts := []*jsengine.JsScript{{
		FileName: "interrupt_test.js",
		Content: `
      globalThis.$choysum = {
        __rpc__: function(req) {
          while (true) { /* spin */ }
        }
      };
    `,
	}}
	if err := engine.Load(scripts); err != nil {
		t.Fatalf("load scripts: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, execErr := engine.Execute(ctx, &jsengine.JsRequest{Id: "1", Service: "x"})
	elapsed := time.Since(start)

	if execErr == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(execErr, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", execErr)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("interrupt took too long: %s", elapsed)
	}
}

func TestQuickjsEngine_InterruptsOnContextCancel(t *testing.T) {
	engine, err := NewFactory()()
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	scripts := []*jsengine.JsScript{{
		FileName: "interrupt_cancel_test.js",
		Content: `
      globalThis.$choysum = {
        __rpc__: function(req) {
          while (true) { /* spin */ }
        }
      };
    `,
	}}
	if err := engine.Load(scripts); err != nil {
		t.Fatalf("load scripts: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, execErr := engine.Execute(ctx, &jsengine.JsRequest{Id: "1", Service: "x"})
	elapsed := time.Since(start)

	if execErr == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(execErr, context.Canceled) {
		t.Fatalf("expected canceled, got: %v", execErr)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("interrupt took too long: %s", elapsed)
	}
}

func TestSwapExecContext(t *testing.T) {
	engineIface, err := NewFactory()()
	if err != nil {
		t.Fatal(err)
	}
	engine := engineIface.(*QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	type key struct{}
	outer := context.WithValue(context.Background(), key{}, "outer")
	inner := context.WithValue(context.Background(), key{}, "inner")
	restoreOuter := engine.SwapExecContext(outer)
	restoreInner := engine.SwapExecContext(inner)
	if engine.ExecContext().Value(key{}) != "inner" {
		t.Fatal("inner not set")
	}
	restoreInner()
	if engine.ExecContext().Value(key{}) != "outer" {
		t.Fatal("outer not restored")
	}
	restoreOuter()
}
