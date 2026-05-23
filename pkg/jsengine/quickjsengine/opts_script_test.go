// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"bytes"
	"sync"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

func TestScriptCacheKeyAndWithScript(t *testing.T) {
	scriptCache = sync.Map{}
	script := &jsengine.JsScript{FileName: "cached.js", Content: `globalThis.cached = (globalThis.cached || 0) + 1;`}
	if keyA, keyB := scriptCacheKey(script), scriptCacheKey(script); keyA != keyB {
		t.Fatalf("expected stable cache key, got %q and %q", keyA, keyB)
	}
	if scriptCacheKey(script) == scriptCacheKey(&jsengine.JsScript{FileName: "cached.js", Content: `globalThis.cached = 2;`}) {
		t.Fatal("expected different script content to produce a different cache key")
	}

	engineA := newTestQuickjsEngine(t)
	bytecodeA, err := compileScriptBytecode(engineA, script)
	if err != nil {
		t.Fatalf("compileScriptBytecode(engineA): %v", err)
	}
	if len(bytecodeA) == 0 {
		t.Fatal("expected compiled bytecode")
	}
	bytecodeB, err := compileScriptBytecode(engineA, script)
	if err != nil {
		t.Fatalf("compileScriptBytecode(engineA second call): %v", err)
	}
	if !bytes.Equal(bytecodeA, bytecodeB) {
		t.Fatal("expected cached bytecode to be reused")
	}

	engineB := newTestQuickjsEngine(t, WithScript(script))
	if got := evalInt64(t, engineB, `globalThis.cached`); got != 1 {
		t.Fatalf("globalThis.cached = %d, want 1", got)
	}
	if err := WithScript(nil)(engineB); err == nil {
		t.Fatal("expected nil script error")
	}
	if err := WithScript(&jsengine.JsScript{FileName: "broken.js", Content: `globalThis.bad = {`})(engineB); err == nil {
		t.Fatal("expected broken script error")
	}
}
