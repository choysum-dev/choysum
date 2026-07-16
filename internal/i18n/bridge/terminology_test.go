// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge_test

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/bridge"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestWithTerminologyLookupSync(t *testing.T) {
	lookup := func(module, lang, scope, src, kind string) (string, bool) {
		if module == "auth" && lang == "zh_CN" && scope == "a@b" && src == "Hello" && kind == "literal" {
			return "你好", true
		}
		return "", false
	}

	engineIface, err := quickjsengine.NewFactory(bridge.WithTerminologyLookup(lookup))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	hit := engine.Ctx.Eval(`$choysum.i18n.t('auth', 'zh_CN', 'a@b', 'Hello')`)
	defer hit.Free()
	if hit.IsException() {
		t.Fatalf("Eval hit: %v", engine.Ctx.Exception())
	}
	if hit.String() != "你好" {
		t.Fatalf("hit = %q, want 你好", hit.String())
	}

	miss := engine.Ctx.Eval(`$choysum.i18n.t('auth', 'zh_CN', 'a@b', 'Missing')`)
	defer miss.Free()
	if miss.IsException() {
		t.Fatalf("Eval miss: %v", engine.Ctx.Exception())
	}
	if miss.String() != "" {
		t.Fatalf("miss = %q, want empty", miss.String())
	}
}
