// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

type stubJsEngine struct{}

func (stubJsEngine) Load([]*jsengine.JsScript) error { return nil }
func (stubJsEngine) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (stubJsEngine) Close() error { return nil }

func TestQuickJSCoder_ErrorPaths(t *testing.T) {
	t.Run("engine create error", func(t *testing.T) {
		orig := newVueVirtualEngine
		t.Cleanup(func() { newVueVirtualEngine = orig })
		newVueVirtualEngine = func(string) (jsengine.JsEngine, error) {
			return nil, errors.New("engine boom")
		}
		c := NewQuickJSCoder()
		_, err := c.CreateServiceScript("a.vue", "<script setup lang=\"ts\"></script>", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "engine boom") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("unexpected engine type", func(t *testing.T) {
		orig := newVueVirtualEngine
		t.Cleanup(func() { newVueVirtualEngine = orig })
		newVueVirtualEngine = func(string) (jsengine.JsEngine, error) {
			return stubJsEngine{}, nil
		}
		c := NewQuickJSCoder()
		_, err := c.CreateServiceScript("a.vue", "<script setup lang=\"ts\"></script>", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "unexpected engine type") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("close nil engine", func(t *testing.T) {
		c := NewQuickJSCoder()
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not a function", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `var vuevirtual = { createServiceScript: 123 };`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "not a function") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("eval exception", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `var vuevirtual = null;`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil {
			t.Fatal("expected eval error")
		}
	})

	t.Run("createServiceScript throws", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `var vuevirtual = { createServiceScript: function() { throw new Error("codegen boom"); } };`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{CurrentDirectory: "/tmp"})
		if err == nil || !strings.Contains(err.Error(), "codegen boom") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `var vuevirtual = { createServiceScript: function() {
				return { embeddedId: "script_ts", scriptKind: "ts", content: "", mappings: [] };
			} };`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "empty service script") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("json stringify unavailable", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `
				delete JSON.stringify;
				var vuevirtual = { createServiceScript: function() {
					return { embeddedId: "script_ts", scriptKind: "ts", content: "export {}", mappings: [] };
				} };
			`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "JSON.stringify unavailable") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("json stringify eval exception", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `
				Object.defineProperty(JSON, "stringify", {
					configurable: true,
					get: function() { throw new Error("stringify getter boom"); }
				});
				var vuevirtual = { createServiceScript: function() {
					return { embeddedId: "script_ts", scriptKind: "ts", content: "export {}", mappings: [] };
				} };
			`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "JSON.stringify unavailable") {
			t.Fatalf("err=%v", err)
		}
		if !strings.Contains(err.Error(), "stringify getter boom") {
			t.Fatalf("expected drained exception in err, got %v", err)
		}
	})

	t.Run("stringify throws", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `
				JSON.stringify = function() { throw new Error("stringify boom"); };
				var vuevirtual = { createServiceScript: function() {
					return { embeddedId: "script_ts", scriptKind: "ts", content: "export {}", mappings: [] };
				} };
			`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "stringify") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("decode invalid json", func(t *testing.T) {
		orig := vueVirtualScriptContent
		t.Cleanup(func() { vueVirtualScriptContent = orig })
		vueVirtualScriptContent = func() string {
			return `
				JSON.stringify = function() { return "{not-json"; };
				var vuevirtual = { createServiceScript: function() {
					return { embeddedId: "script_ts", scriptKind: "ts", content: "export {}", mappings: [] };
				} };
			`
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "decode") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("marshal path error", func(t *testing.T) {
		orig := marshalJSValue
		t.Cleanup(func() { marshalJSValue = orig })
		n := 0
		marshalJSValue = func(eng *quickjsengine.QuickjsEngine, v any) (*quickjs.Value, error) {
			n++
			if n == 1 {
				return nil, errors.New("path marshal boom")
			}
			return defaultMarshalJSValue(eng, v)
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "marshal path") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("marshal source error", func(t *testing.T) {
		orig := marshalJSValue
		t.Cleanup(func() { marshalJSValue = orig })
		n := 0
		marshalJSValue = func(eng *quickjsengine.QuickjsEngine, v any) (*quickjs.Value, error) {
			n++
			if n == 2 {
				return nil, errors.New("source marshal boom")
			}
			return defaultMarshalJSValue(eng, v)
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{})
		if err == nil || !strings.Contains(err.Error(), "marshal source") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("marshal opts error", func(t *testing.T) {
		orig := marshalJSValue
		t.Cleanup(func() { marshalJSValue = orig })
		n := 0
		marshalJSValue = func(eng *quickjsengine.QuickjsEngine, v any) (*quickjs.Value, error) {
			n++
			if n == 3 {
				return nil, errors.New("opts marshal boom")
			}
			return defaultMarshalJSValue(eng, v)
		}
		c := NewQuickJSCoder()
		t.Cleanup(func() { _ = c.Close() })
		_, err := c.CreateServiceScript("a.vue", "x", CodegenOptions{CurrentDirectory: "/x"})
		if err == nil || !strings.Contains(err.Error(), "marshal opts") {
			t.Fatalf("err=%v", err)
		}
	})
}
