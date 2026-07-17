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

func TestWithTerminologyLookupExplicitKind(t *testing.T) {
	lookup := func(module, lang, scope, src, kind string) (string, bool) {
		if kind == "custom" && src == "Company" {
			return "公司", true
		}
		if kind == "literal" && src == "Hello" {
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

	custom := engine.Ctx.Eval(`$choysum.i18n.t('auth', 'zh_CN', 'm@id', 'Company', 'custom')`)
	defer custom.Free()
	if custom.IsException() {
		t.Fatalf("Eval custom kind: %v", engine.Ctx.Exception())
	}
	if custom.String() != "公司" {
		t.Fatalf("custom kind = %q, want 公司", custom.String())
	}
	lit := engine.Ctx.Eval(`$choysum.i18n.t('auth', 'zh_CN', 'a@b', 'Hello')`)
	defer lit.Free()
	if lit.IsException() {
		t.Fatalf("Eval literal: %v", engine.Ctx.Exception())
	}
	if lit.String() != "你好" {
		t.Fatalf("literal = %q, want 你好", lit.String())
	}
}
