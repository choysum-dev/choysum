// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nservice_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	i18nservice "github.com/choysum-dev/choysum/internal/i18n/service"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *testScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *testScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *testScope) Session() *scope.Session { return s.session }
func (s *testScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &testScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *testScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *testScope) Logger() *slog.Logger { return s.logger }

func newTestScope(t *testing.T) *testScope {
	t.Helper()
	store.ResetSharedRegistryForTests()
	i18nservice.ResetDescriptorCacheForTests()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "i18n_svc.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &testScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func seedTerm(t *testing.T, rs scope.Scope, module, scopeKey, src, value string) {
	t.Helper()
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      module,
		Lang:        "zh_CN",
		Scope:       scopeKey,
		Src:         src,
		Value:       value,
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func invokeGetTranslations(t *testing.T, svc *i18nservice.Service, lang string, modules []string, hash string) map[string]any {
	t.Helper()
	out, err := invokeMethod(t, svc, i18nservice.MethodGetTranslations, map[string]any{
		"lang":         lang,
		"module_names": stringListAny(modules),
		"hash":         hash,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return out
}

func stringListAny(modules []string) []any {
	out := make([]any, 0, len(modules))
	for _, m := range modules {
		out = append(out, m)
	}
	return out
}

func invokeMethod(t *testing.T, svc *i18nservice.Service, methodName string, payload map[string]any) (map[string]any, error) {
	t.Helper()
	desc, err := svc.ServiceDesc()
	if err != nil {
		t.Fatal(err)
	}
	var handler func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error)
	for _, m := range desc.Methods {
		if m.MethodName == methodName {
			handler = m.Handler
			break
		}
	}
	if handler == nil {
		t.Fatalf("method %s not found", methodName)
	}
	resp, err := handler(nil, context.Background(), func(v any) error {
		msg := v.(*dynamicpb.Message)
		return converter.MapToMessage(payload, msg)
	}, nil)
	if err != nil {
		return nil, err
	}
	msg, ok := resp.(*dynamicpb.Message)
	if !ok {
		t.Fatalf("resp type %T", resp)
	}
	out, err := converter.MessageToMap(msg)
	if err != nil {
		t.Fatal(err)
	}
	return out, nil
}

func TestGetTranslationsFiltersModulesAndHash(t *testing.T) {
	rs := newTestScope(t)
	seedTerm(t, rs, "auth", "web/a@title", "Hello", "你好")
	seedTerm(t, rs, "auth", "web/a@ok", "OK", "好的")
	reg := store.RegistryFor(rs)
	reg.RememberModuleApplication("auth", "auth")
	reg.RememberModuleApplication("other", "partner")

	svc := i18nservice.New("auth", rs)
	if got := i18nservice.FullMethodGetTranslations("auth"); got != "/auth.I18n/GetTranslations" {
		t.Fatalf("FullMethodGetTranslations = %q", got)
	}
	if got := i18nservice.FullMethod("auth", i18nservice.MethodSearchTerms); got != "/auth.I18n/SearchTerms" {
		t.Fatalf("FullMethod SearchTerms = %q", got)
	}
	if got := i18nservice.FullMethod("auth", i18nservice.MethodUpdateTerm); got != "/auth.I18n/UpdateTerm" {
		t.Fatalf("FullMethod UpdateTerm = %q", got)
	}
	desc, err := svc.ServiceDesc()
	if err != nil {
		t.Fatal(err)
	}
	if desc.ServiceName != "auth.I18n" {
		t.Fatalf("ServiceName = %q", desc.ServiceName)
	}
	if len(desc.Methods) != 3 {
		t.Fatalf("Methods len = %d, want 3", len(desc.Methods))
	}

	out := invokeGetTranslations(t, svc, "zh_CN", []string{"auth", "other", "missing"}, "")
	if out["unchanged"] == true {
		t.Fatalf("expected changed: %#v", out)
	}
	terms, ok := out["terms_by_module"].(map[string]any)
	if !ok {
		t.Fatalf("terms_by_module type %#v", out["terms_by_module"])
	}
	if _, ok := terms["other"]; ok {
		t.Fatalf("other app module must be filtered: %#v", terms)
	}
	if _, ok := terms["missing"]; ok {
		t.Fatalf("missing module must be omitted: %#v", terms)
	}
	authTerms, ok := terms["auth"].(map[string]any)
	if !ok {
		t.Fatalf("auth terms missing: %#v", terms)
	}
	title, ok := authTerms["web/a@title"].(map[string]any)
	if !ok || title["Hello"] != "你好" {
		t.Fatalf("unexpected title: %#v", authTerms)
	}

	hash, _ := out["hash"].(string)
	if hash == "" {
		t.Fatal("expected termHash")
	}
	unchanged := invokeGetTranslations(t, svc, "zh_CN", []string{"auth"}, hash)
	if unchanged["unchanged"] != true {
		t.Fatalf("expected unchanged: %#v", unchanged)
	}
	if unchanged["terms_by_module"] != nil {
		t.Fatalf("terms must be null/absent when unchanged: %#v", unchanged["terms_by_module"])
	}

	// Framework module "core" is hosted in the app table even when mapped to application=core.
	seedTerm(t, rs, "core", "service/a@m", "Denied", "拒绝")
	reg.RememberModuleApplication("core", "core")
	withCore := invokeGetTranslations(t, svc, "zh_CN", []string{"auth", "core"}, "")
	coreTerms, ok := withCore["terms_by_module"].(map[string]any)
	if !ok {
		t.Fatalf("terms_by_module type %#v", withCore["terms_by_module"])
	}
	if _, ok := coreTerms["core"]; !ok {
		t.Fatalf("framework module terms missing: %#v", coreTerms)
	}
}

func TestGetTranslationsEmptyAppStableHash(t *testing.T) {
	rs := newTestScope(t)
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	svc := i18nservice.New("auth", rs)
	out := invokeGetTranslations(t, svc, "zh_CN", []string{"auth"}, "")
	if out["unchanged"] == true {
		t.Fatal("empty without client hash should not be unchanged")
	}
	hash, _ := out["hash"].(string)
	if hash != store.EmptyTermHash() {
		t.Fatalf("hash = %q, want empty stable %q", hash, store.EmptyTermHash())
	}
	terms, _ := out["terms_by_module"].(map[string]any)
	if len(terms) != 0 {
		t.Fatalf("expected empty terms: %#v", terms)
	}
}

func TestServiceDescName(t *testing.T) {
	rs := newTestScope(t)
	svc := i18nservice.New("web", rs)
	desc, err := svc.ServiceDesc()
	if err != nil {
		t.Fatal(err)
	}
	if desc.ServiceName != "web.I18n" {
		t.Fatalf("ServiceName = %q", desc.ServiceName)
	}
}

func TestSearchTermsPaginationAndStatus(t *testing.T) {
	rs := newTestScope(t)
	seedTerm(t, rs, "auth", "web/a@title", "Hello", "你好")
	seedTerm(t, rs, "auth", "web/a@ok", "OK", "")
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/a@bye",
		Src:         "Bye",
		Value:       "再见",
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourcePackaged,
		Comments:    "fuzzy",
	}).Error; err != nil {
		t.Fatal(err)
	}
	reg := store.RegistryFor(rs)
	reg.RememberModuleApplication("auth", "auth")

	svc := i18nservice.New("auth", rs)
	out, err := invokeMethod(t, svc, i18nservice.MethodSearchTerms, map[string]any{
		"lang":    "zh_CN",
		"modules": stringListAny([]string{"auth"}),
		"q":       "Hello",
		"limit":   10,
		"offset":  0,
	})
	if err != nil {
		t.Fatalf("SearchTerms: %v", err)
	}
	items, _ := out["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 Hello hit, got %#v", out)
	}
	first := items[0].(map[string]any)
	if first["status"] != "translated" || first["value"] != "你好" {
		t.Fatalf("unexpected item: %#v", first)
	}

	all, err := invokeMethod(t, svc, i18nservice.MethodSearchTerms, map[string]any{
		"lang":   "zh_CN",
		"limit":  50,
		"offset": 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	allItems, _ := all["items"].([]any)
	if len(allItems) != 3 {
		t.Fatalf("expected 3 items, got %#v", all)
	}
	bySrc := map[string]string{}
	for _, raw := range allItems {
		item := raw.(map[string]any)
		bySrc[fmt.Sprintf("%v", item["src"])] = fmt.Sprintf("%v", item["status"])
	}
	if bySrc["OK"] != "missing" || bySrc["Bye"] != "fuzzy" || bySrc["Hello"] != "translated" {
		t.Fatalf("status map: %#v", bySrc)
	}
}

func TestUpdateTermSetsOverrideAndCache(t *testing.T) {
	rs := newTestScope(t)
	seedTerm(t, rs, "auth", "web/a@title", "Hello", "你好")
	reg := store.RegistryFor(rs)
	reg.RememberModuleApplication("auth", "auth")
	ts := reg.StoreFor("auth")
	if err := ts.WarmLanguage("zh_CN"); err != nil {
		t.Fatal(err)
	}
	beforeHash := ts.TermHash("zh_CN")

	svc := i18nservice.New("auth", rs)
	out, err := invokeMethod(t, svc, i18nservice.MethodUpdateTerm, map[string]any{
		"module": "auth",
		"lang":   "zh_CN",
		"scope":  "web/a@title",
		"src":    "Hello",
		"value":  "您好",
	})
	if err != nil {
		t.Fatalf("UpdateTerm: %v", err)
	}
	item, _ := out["item"].(map[string]any)
	if item["source"] != i18nmodels.SourceOverride || item["value"] != "您好" {
		t.Fatalf("unexpected item: %#v", item)
	}
	if got, ok := ts.Lookup("auth", "zh_CN", "web/a@title", "Hello", ""); !ok || got != "您好" {
		t.Fatalf("cache lookup = %q ok=%v", got, ok)
	}
	if ts.TermHash("zh_CN") == beforeHash {
		t.Fatal("termHash must bump after UpdateTerm")
	}

	// Foreign module must be denied.
	reg.RememberModuleApplication("partner", "partner")
	_, err = invokeMethod(t, svc, i18nservice.MethodUpdateTerm, map[string]any{
		"module": "partner",
		"lang":   "zh_CN",
		"scope":  "a@b",
		"src":    "X",
		"value":  "Y",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}
