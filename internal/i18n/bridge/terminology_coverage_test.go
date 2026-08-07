// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/buke/quickjs-go"
	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type coverageScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *coverageScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *coverageScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *coverageScope) Session() *scope.Session { return s.session }
func (s *coverageScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &coverageScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *coverageScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *coverageScope) Logger() *slog.Logger { return s.logger }

func newCoverageEngine(t *testing.T, opts ...jsengine.JsEngineOption) *quickjsengine.QuickjsEngine {
	t.Helper()
	engineIface, err := quickjsengine.NewFactory(opts...)()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })
	return engine
}

func TestWithTerminologyNilAndNonNil(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	engine := newCoverageEngine(t, WithTerminology(nil))
	empty := engine.Ctx.Eval(`$choysum.i18n.t('a', 'zh_CN', 's', 'Hello')`)
	defer empty.Free()
	if empty.String() != "" {
		t.Fatalf("nil terminology t = %q, want empty", empty.String())
	}
	few := engine.Ctx.Eval(`$choysum.i18n.t('a')`)
	defer few.Free()
	if few.String() != "" {
		t.Fatalf("short-args t = %q, want empty", few.String())
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "with-term.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &coverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	reg := store.NewRegistry(rs)
	reg.RememberModuleApplication("auth", "auth")
	engine2 := newCoverageEngine(t, WithTerminology(reg))
	miss := engine2.Ctx.Eval(`$choysum.i18n.t('auth', 'zh_CN', 's', 'Hello')`)
	defer miss.Free()
	if miss.IsException() {
		t.Fatalf("WithTerminology lookup: %v", engine2.Ctx.Exception())
	}
}

func TestWithTerminologyProviderNilScope(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	engine := newCoverageEngine(t, WithTerminologyProvider(nil))
	hit := engine.Ctx.Eval(`typeof $choysum.i18n.t === 'function' && typeof $choysum.i18n.invalidateModule === 'undefined'`)
	defer hit.Free()
	if !hit.ToBool() {
		t.Fatal("expected t present and invalidateModule absent when provider is nil")
	}
}

func TestInvalidateModuleFuncNilRegistry(t *testing.T) {
	engine := newCoverageEngine(t)
	fn := invalidateModuleFunc(nil)
	ret := fn(engine.Ctx, nil, []*quickjs.Value{engine.Ctx.String("auth"), engine.Ctx.String("auth")})
	defer ret.Free()
	if ret.ToBool() {
		t.Fatal("expected false for nil registry")
	}
	short := fn(engine.Ctx, nil, []*quickjs.Value{engine.Ctx.String("auth")})
	defer short.Free()
	if short.ToBool() {
		t.Fatal("expected false for short args")
	}
}

func TestPoTextBytesVariants(t *testing.T) {
	engine := newCoverageEngine(t)
	ctx := engine.Ctx

	if _, err := poTextBytes(nil); err == nil {
		t.Fatal("expected nil poText error")
	}
	undef := ctx.Undefined()
	defer undef.Free()
	if _, err := poTextBytes(undef); err == nil {
		t.Fatal("expected undefined poText error")
	}
	null := ctx.Null()
	defer null.Free()
	if _, err := poTextBytes(null); err == nil {
		t.Fatal("expected null poText error")
	}

	str := ctx.String("hello")
	defer str.Free()
	got, err := poTextBytes(str)
	if err != nil || string(got) != "hello" {
		t.Fatalf("string poText = %q err=%v", got, err)
	}

	u8 := ctx.NewUint8Array([]byte("abc"))
	defer u8.Free()
	got, err = poTextBytes(u8)
	if err != nil || string(got) != "abc" {
		t.Fatalf("Uint8Array poText = %q err=%v", got, err)
	}

	clamped := ctx.Eval(`new Uint8ClampedArray([65, 66])`)
	defer clamped.Free()
	got, err = poTextBytes(clamped)
	if err != nil || string(got) != "AB" {
		t.Fatalf("Uint8ClampedArray poText = %q err=%v", got, err)
	}

	buf := ctx.NewArrayBuffer([]byte("xy"))
	defer buf.Free()
	got, err = poTextBytes(buf)
	if err != nil || string(got) != "xy" {
		t.Fatalf("ArrayBuffer poText = %q err=%v", got, err)
	}

	num := ctx.Int32(1)
	defer num.Free()
	if _, err := poTextBytes(num); err == nil {
		t.Fatal("expected number poText error")
	}
}

func TestPerformUpsertPackagedTermsBranches(t *testing.T) {
	store.ResetSharedRegistryForTests()
	t.Cleanup(store.ResetSharedRegistryForTests)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "upsert-cov.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &coverageScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	provider := jsengine.StaticScopeProvider(rs)
	reg := store.RegistryFor(rs)
	engine := newCoverageEngine(t, WithTerminologyProvider(provider))

	short := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, nil)
	defer short.Free()
	if !short.IsError() {
		t.Fatal("expected error for short args")
	}

	badType := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.String("auth"),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.Int32(1),
	})
	defer badType.Free()
	if !badType.IsError() {
		t.Fatal("expected error for non-string poText")
	}

	coreApp := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, []*quickjs.Value{
		engine.Ctx.String("core"),
		engine.Ctx.String("auth"),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.String("x"),
	})
	defer coreApp.Free()
	if !coreApp.IsError() {
		t.Fatal("expected error for core application")
	}

	nilSessionProvider := jsengine.ScopeProvider(func(ctx context.Context) scope.Scope {
		return &coverageScope{ctx: ctx, logger: rs.logger, session: nil}
	})
	noSession := performUpsertPackagedTerms(engine.Ctx, engine, nilSessionProvider, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.String("auth"),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.String("x"),
	})
	defer noSession.Free()
	if !noSession.IsError() {
		t.Fatal("expected error for missing session")
	}

	nilProvider := performUpsertPackagedTerms(engine.Ctx, engine, nil, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.String("auth"),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.String("x"),
	})
	defer nilProvider.Free()
	if !nilProvider.IsError() {
		t.Fatal("expected error for nil scope provider")
	}

	prevUpsert := upsertPackagedTermsFn
	upsertPackagedTermsFn = func(runtimeScope scope.Scope, reg *store.Registry, application, module, lang string, poText []byte) (*i18nimport.ImportStats, error) {
		return nil, fmt.Errorf("forced upsert failure")
	}
	t.Cleanup(func() { upsertPackagedTermsFn = prevUpsert })
	failUpsert := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.String("auth"),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.String("x"),
	})
	defer failUpsert.Free()
	if !failUpsert.IsError() {
		t.Fatal("expected error from upsert failure")
	}

	upsertPackagedTermsFn = func(runtimeScope scope.Scope, reg *store.Registry, application, module, lang string, poText []byte) (*i18nimport.ImportStats, error) {
		return nil, nil
	}
	nilStats := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.String("auth"),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.String("x"),
	})
	defer nilStats.Free()
	if nilStats.IsError() {
		t.Fatal("expected success when stats are nil")
	}
	lang := nilStats.Get("lang")
	defer lang.Free()
	if lang.String() != "zh_CN" {
		t.Fatalf("lang=%q, want zh_CN", lang.String())
	}

	upsertPackagedTermsFn = func(runtimeScope scope.Scope, reg *store.Registry, application, module, lang string, poText []byte) (*i18nimport.ImportStats, error) {
		return &i18nimport.ImportStats{Lang: lang, Upserted: 2}, nil
	}
	prevMarshal := marshalFn
	marshalFn = func(ctx *quickjs.Context, v any) (*quickjs.Value, error) {
		return nil, fmt.Errorf("forced marshal failure")
	}
	t.Cleanup(func() { marshalFn = prevMarshal })
	failMarshal := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.String("auth"),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.String("x"),
	})
	defer failMarshal.Free()
	if !failMarshal.IsError() {
		t.Fatal("expected error from marshal failure")
	}

	nullModule := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.Null(),
		engine.Ctx.String("zh_CN"),
		engine.Ctx.String("x"),
	})
	defer nullModule.Free()
	if !nullModule.IsError() {
		t.Fatal("expected error for null module")
	}
	undefLang := performUpsertPackagedTerms(engine.Ctx, engine, provider, reg, []*quickjs.Value{
		engine.Ctx.String("auth"),
		engine.Ctx.String("auth"),
		engine.Ctx.Undefined(),
		engine.Ctx.String("x"),
	})
	defer undefLang.Free()
	if !undefLang.IsError() {
		t.Fatal("expected error for undefined lang")
	}
}
