// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
)

type cachedScript struct {
	once     sync.Once
	bytecode []byte
	err      error
}

var scriptCache sync.Map // key(string) -> *cachedScript

func scriptCacheKey(s *jsengine.JsScript) string {
	h := sha256.New()
	h.Write([]byte(s.FileName))
	h.Write([]byte{'\n'})
	h.Write([]byte(s.Content))
	return hex.EncodeToString(h.Sum(nil))
}

// compileScriptBytecode precompiles and caches script bytecode keyed by content and file name.
func compileScriptBytecode(jse *QuickjsEngine, s *jsengine.JsScript) ([]byte, error) {
	key := scriptCacheKey(s)
	v, _ := scriptCache.LoadOrStore(key, &cachedScript{})
	entry := v.(*cachedScript)
	entry.once.Do(func() {
		b, err := jse.Ctx.Compile(s.Content, quickjs.EvalFileName(s.FileName))
		if err != nil {
			entry.err = err
			return
		}
		entry.bytecode = b
	})
	return entry.bytecode, entry.err
}

// WithScript precompiles a script and loads it into the current QuickJS context, sharing cached bytecode across goroutines.
func WithScript(s *jsengine.JsScript) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		if s == nil {
			return fmt.Errorf("script is nil")
		}
		jse := engine.(*QuickjsEngine)

		bytecode, err := compileScriptBytecode(jse, s)
		if err != nil {
			return err
		}

		ret := jse.Ctx.EvalBytecode(bytecode)
		defer ret.Free()
		if ret.IsException() {
			return jse.Ctx.Exception()
		}
		return nil
	}
}
