// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"testing"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	i18nservice "github.com/choysum-dev/choysum/internal/i18n/service"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type rpcTestScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
	cfg     *config.Config
}

func (s *rpcTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *rpcTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *rpcTestScope) Session() *scope.Session { return s.session }
func (s *rpcTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &rpcTestScope{ctx: ctx, logger: s.logger, session: s.session, cfg: s.cfg}
}
func (s *rpcTestScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *rpcTestScope) Logger() *slog.Logger { return s.logger }
func (s *rpcTestScope) Config() *config.Config {
	if s.cfg == nil {
		s.cfg = &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "test-internal-key"},
			Server: &config.ServerConfig{Environment: "development"},
		}
	}
	return s.cfg
}
func (s *rpcTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(s.Config())
}

func newRPCTestScope(t *testing.T) *rpcTestScope {
	t.Helper()
	store.ResetSharedRegistryForTests()
	i18nservice.ResetDescriptorCacheForTests()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "i18n_rpc.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &rpcTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
		cfg: &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "test-internal-key"},
			Server: &config.ServerConfig{Environment: "development"},
		},
	}
}

func seedAuthTerm(t *testing.T, rs scope.Scope, scopeKey, src, value string) {
	t.Helper()
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "auth",
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

func newAuthI18nDialer(t *testing.T, rs scope.Scope) grpcclient.ServiceDialer {
	t.Helper()
	svc := i18nservice.New("auth", rs)
	desc, err := svc.ServiceDesc()
	if err != nil {
		t.Fatal(err)
	}
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	server.RegisterService(desc, svc)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	var conn *grpc.ClientConn
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})
	return func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "auth.I18n" {
			return nil, errors.New("unexpected service: " + serviceName)
		}
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
}

// newAuthTranslationTermDialer serves GetTranslations under auth.TranslationTerm,
// delegating to the Go I18n TermStore handlers (wire-compatible field layout).
func newAuthTranslationTermDialer(t *testing.T, rs scope.Scope) grpcclient.ServiceDialer {
	t.Helper()
	svc := i18nservice.New("auth", rs)
	i18nDesc, err := svc.ServiceDesc()
	if err != nil {
		t.Fatal(err)
	}
	var getMethod *grpc.MethodDesc
	for i := range i18nDesc.Methods {
		if i18nDesc.Methods[i].MethodName == i18nservice.MethodGetTranslations {
			getMethod = &i18nDesc.Methods[i]
			break
		}
	}
	if getMethod == nil {
		t.Fatal("I18n ServiceDesc missing GetTranslations")
	}
	ttDesc := &grpc.ServiceDesc{
		ServiceName: "auth.TranslationTerm",
		HandlerType: (*interface{})(nil),
		Methods:     []grpc.MethodDesc{*getMethod},
		Streams:     []grpc.StreamDesc{},
		Metadata:    "auth/translation_term.proto",
	}

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	server.RegisterService(ttDesc, svc)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	var conn *grpc.ClientConn
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})
	return func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "auth.TranslationTerm" {
			return nil, fmt.Errorf("unexpected service: %s", serviceName)
		}
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
}

func TestFetchAppSearchTermsViaBufconn(t *testing.T) {
	rs := newRPCTestScope(t)
	seedAuthTerm(t, rs, "web/a@title", "Hello", "你好")
	store.RegistryFor(rs).RememberModuleApplication("auth", "auth")

	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newAuthI18nDialer(t, rs))

	result, err := fetchAppSearchTerms(ctx, rs, "user-token", "auth", "zh_CN", []string{"auth"}, "Hello", 10, 0)
	if err != nil {
		t.Fatalf("fetchAppSearchTerms: %v", err)
	}
	if result.Lang != "zh_CN" || result.Total < 1 || len(result.Items) < 1 {
		t.Fatalf("unexpected search result: %#v", result)
	}
	if result.Items[0].Src != "Hello" || result.Items[0].Value != "你好" {
		t.Fatalf("unexpected item: %#v", result.Items[0])
	}
}

func TestFetchAppTranslationsViaBufconn(t *testing.T) {
	rs := newRPCTestScope(t)
	seedAuthTerm(t, rs, "web/a@title", "Hello", "你好")
	store.RegistryFor(rs).RememberModuleApplication("auth", "auth")

	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newAuthTranslationTermDialer(t, rs))
	got, err := fetchAppTranslations(ctx, rs, "auth", "zh_CN", []string{"auth"})
	if err != nil {
		t.Fatalf("fetchAppTranslations: %v", err)
	}
	if got.Hash == "" {
		t.Fatal("expected hash")
	}
	if got.Terms["auth"]["web/a@title"]["Hello"] != "你好" {
		t.Fatalf("terms = %#v", got.Terms)
	}

	md, ok := metadata.FromOutgoingContext(outgoingContextForInternalRPC(context.Background(), rs))
	if !ok || len(md.Get(internalKeyHeader)) == 0 {
		t.Fatalf("expected internal key metadata, md=%v", md)
	}
}

func TestParseAppTranslationsBranches(t *testing.T) {
	empty := parseAppTranslations(map[string]any{"hash": "<nil>"})
	if empty.Hash != "" || len(empty.Terms) != 0 {
		t.Fatalf("empty = %#v", empty)
	}

	got := parseAppTranslations(map[string]any{
		"hash": "abc",
		"terms_by_module": map[string]any{
			"auth": map[string]any{
				"a@t": map[string]any{"Hello": "你好"},
				"bad": "skip",
			},
			"skip-mod": "nope",
		},
	})
	if got.Hash != "abc" {
		t.Fatalf("hash = %q", got.Hash)
	}
	if got.Terms["auth"]["a@t"]["Hello"] != "你好" {
		t.Fatalf("terms = %#v", got.Terms)
	}
	if _, ok := got.Terms["auth"]["bad"]; ok {
		t.Fatal("non-map scope should be skipped")
	}
	if _, ok := got.Terms["skip-mod"]; ok {
		t.Fatal("non-map module should be skipped")
	}
}

func TestFetchAppTranslationsRequiresApplication(t *testing.T) {
	if _, err := fetchAppTranslations(context.Background(), nil, "  ", "zh_CN", nil); err == nil || !strings.Contains(err.Error(), "application is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchAppSearchTermsDialFailure(t *testing.T) {
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial boom")
	})
	if _, err := fetchAppSearchTerms(ctx, nil, "tok", "auth", "zh_CN", nil, "", 10, 0); err == nil {
		t.Fatal("expected dial error")
	}
}

func TestFetchAppTranslationsDialFailure(t *testing.T) {
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "auth.TranslationTerm" {
			t.Fatalf("unexpected dial target %q", serviceName)
		}
		return nil, errors.New("dial boom")
	})
	if _, err := fetchAppTranslations(ctx, nil, "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected dial error")
	}
}
