// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
		t.Fatal("expected dial or descriptor error")
	}
}

func TestFetchAppTranslationsDialFailure(t *testing.T) {
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial boom")
	})
	if _, err := fetchAppTranslations(ctx, nil, "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected dial or descriptor error")
	}
}

func TestOutgoingContextForInternalRPC(t *testing.T) {
	rs := &rpcTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg: &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "test-internal-key"},
			Server: &config.ServerConfig{Environment: "development"},
		},
	}
	md, ok := metadata.FromOutgoingContext(outgoingContextForInternalRPC(context.Background(), rs))
	if !ok || len(md.Get(internalKeyHeader)) == 0 {
		t.Fatalf("expected internal key metadata, md=%v", md)
	}
}

func TestSearchAppUsesInjectedHook(t *testing.T) {
	called := false
	h := &handler{
		search: func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
			called = true
			return &searchTermsResult{Lang: lang, Total: 0}, nil
		},
	}
	got, err := h.searchApp(context.Background(), "tok", "auth", "zh_CN", []string{"auth"}, "", 10, 0)
	if err != nil || !called || got == nil || got.Lang != "zh_CN" {
		t.Fatalf("injected searchApp failed: called=%v got=%#v err=%v", called, got, err)
	}
}
