// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"context"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

func TestEngineOptionsApplyAndValidate(t *testing.T) {
	engine := newTestQuickjsEngine(t)

	if err := WithGCThreshold(256)(engine); err != nil {
		t.Fatalf("WithGCThreshold: %v", err)
	}
	if engine.Option.GCThreshold != 256 {
		t.Fatalf("GCThreshold = %d, want 256", engine.Option.GCThreshold)
	}
	if err := WithGCThreshold(-2)(engine); err == nil || !strings.Contains(err.Error(), "invalid GC threshold") {
		t.Fatalf("expected invalid GC threshold error, got %v", err)
	}

	for _, tc := range []struct {
		name  string
		apply jsengine.JsEngineOption
		check func(*testing.T, *QuickjsEngine)
	}{
		{
			name:  "memory limit",
			apply: WithMemoryLimit(1024),
			check: func(t *testing.T, engine *QuickjsEngine) {
				t.Helper()
				if engine.Option.MemoryLimit != 1024 {
					t.Fatalf("MemoryLimit = %d, want 1024", engine.Option.MemoryLimit)
				}
			},
		},
		{
			name:  "timeout",
			apply: WithTimeout(7),
			check: func(t *testing.T, engine *QuickjsEngine) {
				t.Helper()
				if engine.Option.Timeout != 7 {
					t.Fatalf("Timeout = %d, want 7", engine.Option.Timeout)
				}
			},
		},
		{
			name:  "max stack size",
			apply: WithMaxStackSize(2048),
			check: func(t *testing.T, engine *QuickjsEngine) {
				t.Helper()
				if engine.Option.MaxStackSize != 2048 {
					t.Fatalf("MaxStackSize = %d, want 2048", engine.Option.MaxStackSize)
				}
			},
		},
		{
			name:  "can block",
			apply: WithCanBlock(true),
			check: func(t *testing.T, engine *QuickjsEngine) {
				t.Helper()
				if !engine.Option.CanBlock {
					t.Fatal("CanBlock = false, want true")
				}
			},
		},
		{
			name:  "module import",
			apply: WithEnableModuleImport(true),
			check: func(t *testing.T, engine *QuickjsEngine) {
				t.Helper()
				if !engine.Option.EnableModuleImport {
					t.Fatal("EnableModuleImport = false, want true")
				}
			},
		},
		{
			name:  "strip",
			apply: WithStrip(2),
			check: func(t *testing.T, engine *QuickjsEngine) {
				t.Helper()
				if engine.Option.Strip != 2 {
					t.Fatalf("Strip = %d, want 2", engine.Option.Strip)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.apply(engine); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			tc.check(t, engine)
		})
	}

	if err := WithStrip(-1)(engine); err == nil || !strings.Contains(err.Error(), "invalid strip level") {
		t.Fatalf("expected invalid strip error, got %v", err)
	}
	if err := WithStrip(3)(engine); err == nil || !strings.Contains(err.Error(), "invalid strip level") {
		t.Fatalf("expected invalid strip error, got %v", err)
	}
	if _, err := newEngine(WithGCThreshold(-2)); err == nil || !strings.Contains(err.Error(), "invalid GC threshold") {
		t.Fatalf("expected newEngine option error, got %v", err)
	}
}

func TestQuickjsEngineLoadAndExecutePaths(t *testing.T) {
	engine := newTestQuickjsEngine(t)

	if err := engine.Load([]*jsengine.JsScript{{
		FileName: "rpc.js",
		Content: `
			globalThis.$choysum = {
				__rpc__: async function(req) {
					return {
						id: req.id,
						result: { service: req.service, firstArg: req.args[0] },
						context: { seen: true }
					};
				}
			};
		`,
	}}); err != nil {
		t.Fatalf("Load(valid): %v", err)
	}

	resp, err := engine.Execute(context.TODO(), &jsengine.JsRequest{Id: "req-1", Service: "demo", Args: []interface{}{"value"}})
	if err != nil {
		t.Fatalf("Execute(valid): %v", err)
	}
	if resp.Id != "req-1" {
		t.Fatalf("response id = %q, want req-1", resp.Id)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("response result type = %T", resp.Result)
	}
	if result["service"] != "demo" || result["firstArg"] != "value" {
		t.Fatalf("unexpected response result: %#v", result)
	}
	if seen, ok := resp.Context["seen"].(bool); !ok || !seen {
		t.Fatalf("unexpected response context: %#v", resp.Context)
	}

	if _, err := engine.Execute(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "request cannot be nil") {
		t.Fatalf("expected nil request error, got %v", err)
	}

	bad := newTestQuickjsEngine(t)
	if err := bad.Load([]*jsengine.JsScript{{FileName: "bad.js", Content: `globalThis.$choysum = {`}}); err == nil || !strings.Contains(err.Error(), "failed to execute init script") {
		t.Fatalf("expected load error, got %v", err)
	}
	if _, err := bad.Execute(context.Background(), &jsengine.JsRequest{Id: "no-rpc", Service: "demo"}); err == nil || !strings.Contains(err.Error(), "failed to evaluate RPC script") {
		t.Fatalf("expected missing rpc error, got %v", err)
	}
}
