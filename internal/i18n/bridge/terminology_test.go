// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/i18n/bridge"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

func TestWithTerminologyLookupReplacesNonObjectChoysum(t *testing.T) {
	seedNonObject := func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globals := jse.Ctx.Globals()
		globals.Set("$choysum", jse.Ctx.String("not-an-object"))
		return nil
	}
	lookup := func(module, lang, scope, src, kind string) (string, bool) {
		if src == "Hello" {
			return "你好", true
		}
		return "", false
	}

	engineIface, err := quickjsengine.NewFactory(seedNonObject, bridge.WithTerminologyLookup(lookup))()
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

func TestWithTerminologyProviderInvalidateAndUpsert(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bridge.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &bridgeTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	provider := jsengine.StaticScopeProvider(rs)

	engineIface, err := quickjsengine.NewFactory(bridge.WithTerminologyProvider(provider))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	noop := engine.Ctx.Eval(`$choysum.i18n.invalidateModule('', 'auth')`)
	defer noop.Free()
	if noop.IsException() {
		t.Fatalf("invalidate empty app: %v", engine.Ctx.Exception())
	}
	if noop.ToBool() {
		t.Fatal("expected invalidateModule('', ...) to return false")
	}

	// No store yet: must not create one as a side effect of invalidate.
	cold := engine.Ctx.Eval(`$choysum.i18n.invalidateModule('auth', 'auth')`)
	defer cold.Free()
	if cold.IsException() {
		t.Fatalf("invalidate cold: %v", engine.Ctx.Exception())
	}
	if cold.ToBool() {
		t.Fatal("expected invalidateModule before store exists to return false")
	}
	nullish := engine.Ctx.Eval(`$choysum.i18n.invalidateModule(null, 'auth')`)
	defer nullish.Free()
	if nullish.IsException() {
		t.Fatalf("invalidate nullish: %v", engine.Ctx.Exception())
	}
	if nullish.ToBool() {
		t.Fatal("expected invalidateModule(null, ...) to return false")
	}

	promise := engine.Ctx.Eval(`$choysum.i18n.upsertPackagedTerms('auth', 'auth', 'zh_CN', ` + "`" + `
msgctxt "web/a@new"
msgid "Hello"
msgstr "你好"
` + "`" + `)`)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("upsertPackagedTerms eval: %v", engine.Ctx.Exception())
	}
	result, err := awaitPromise(engine, promise)
	if err != nil {
		t.Fatalf("upsertPackagedTerms: %v", err)
	}
	defer result.Free()
	upserted := result.Get("upserted")
	defer upserted.Free()
	if int(upserted.ToInt64()) != 1 {
		t.Fatalf("upserted=%v, want 1", upserted.ToInt64())
	}

	var count int64
	if err := rs.Session().Table("auth_translation_term").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("db rows=%d, want 1", count)
	}

	// The engine-captured registry (not RegistryFor(rs)) holds the warm cache.
	hit := engine.Ctx.Eval(`$choysum.i18n.t('auth', 'zh_CN', 'web/a@new', 'Hello')`)
	defer hit.Free()
	if hit.IsException() {
		t.Fatalf("lookup after upsert: %v", engine.Ctx.Exception())
	}
	if hit.String() != "你好" {
		t.Fatalf("t after upsert = %q, want 你好", hit.String())
	}

	warmInv := engine.Ctx.Eval(`$choysum.i18n.invalidateModule('auth', 'auth')`)
	defer warmInv.Free()
	if warmInv.IsException() {
		t.Fatalf("invalidate warm: %v", engine.Ctx.Exception())
	}
	if !warmInv.ToBool() {
		t.Fatal("expected invalidateModule after upsert to return true")
	}

	bad := engine.Ctx.Eval(`$choysum.i18n.upsertPackagedTerms('auth', 'auth', '', 'x')`)
	defer bad.Free()
	if _, err := awaitPromise(engine, bad); err == nil {
		t.Fatal("expected upsertPackagedTerms validation error")
	}

	nullApp := engine.Ctx.Eval(`$choysum.i18n.upsertPackagedTerms(null, 'auth', 'zh_CN', 'x')`)
	defer nullApp.Free()
	if _, err := awaitPromise(engine, nullApp); err == nil {
		t.Fatal("expected upsertPackagedTerms null application to reject")
	}

	coreInv := engine.Ctx.Eval(`$choysum.i18n.invalidateModule('core', 'auth')`)
	defer coreInv.Free()
	if coreInv.ToBool() {
		t.Fatal("expected invalidateModule('core', ...) to return false")
	}
	emptyMod := engine.Ctx.Eval(`$choysum.i18n.invalidateModule('auth', '')`)
	defer emptyMod.Free()
	if emptyMod.ToBool() {
		t.Fatal("expected invalidateModule(..., '') to return false")
	}
	undefMod := engine.Ctx.Eval(`$choysum.i18n.invalidateModule('auth', undefined)`)
	defer undefMod.Free()
	if undefMod.ToBool() {
		t.Fatal("expected invalidateModule(..., undefined) to return false")
	}
	shortInv := engine.Ctx.Eval(`$choysum.i18n.invalidateModule('auth')`)
	defer shortInv.Free()
	if shortInv.ToBool() {
		t.Fatal("expected invalidateModule with one arg to return false")
	}

	u8 := engine.Ctx.Eval(`$choysum.i18n.upsertPackagedTerms('auth', 'web', 'zh_CN', new TextEncoder().encode('msgctxt "s"\nmsgid "X"\nmsgstr "Y"\n'))`)
	// TextEncoder may be unavailable — fall back to Uint8Array of a minimal PO.
	if u8.IsException() {
		u8.Free()
		_ = engine.Ctx.Exception()
		u8 = engine.Ctx.Eval(`(() => {
			const s = 'msgctxt "s"\nmsgid "X"\nmsgstr "Y"\n';
			const a = new Uint8Array(s.length);
			for (let i = 0; i < s.length; i++) a[i] = s.charCodeAt(i);
			return $choysum.i18n.upsertPackagedTerms('auth', 'web', 'zh_CN', a);
		})()`)
	}
	defer u8.Free()
	u8Result, err := awaitPromise(engine, u8)
	if err != nil {
		t.Fatalf("upsertPackagedTerms Uint8Array: %v", err)
	}
	u8Result.Free()

	fewArgs := engine.Ctx.Eval(`$choysum.i18n.upsertPackagedTerms('auth', 'auth', 'zh_CN')`)
	defer fewArgs.Free()
	if _, err := awaitPromise(engine, fewArgs); err == nil {
		t.Fatal("expected upsertPackagedTerms with missing poText to reject")
	}
}

type bridgeTestScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *bridgeTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *bridgeTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *bridgeTestScope) Session() *scope.Session { return s.session }
func (s *bridgeTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &bridgeTestScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *bridgeTestScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *bridgeTestScope) Logger() *slog.Logger { return s.logger }

func awaitPromise(engine *quickjsengine.QuickjsEngine, promise *quickjs.Value) (*quickjs.Value, error) {
	if promise == nil {
		return nil, fmt.Errorf("nil promise")
	}
	result := promise.Await()
	if result == nil {
		return nil, fmt.Errorf("await returned nil")
	}
	if result.IsException() {
		defer result.Free()
		return nil, engine.Ctx.Exception()
	}
	return result, nil
}
