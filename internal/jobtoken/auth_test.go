// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type jobtokenTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (e *jobtokenTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *jobtokenTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *jobtokenTestScope) Session() *scope.Session { return nil }
func (e *jobtokenTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = e.ctx
	}
	return &jobtokenTestScope{ctx: ctx, cfg: e.cfg, logger: e.logger}
}
func (e *jobtokenTestScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}
func (e *jobtokenTestScope) Logger() *slog.Logger   { return e.logger }
func (e *jobtokenTestScope) Config() *config.Config { return e.cfg }
func (e *jobtokenTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

type fakeAuthenticator struct {
	issueToken string
	issueExp   int64
	createPair *auth.TokenPair
	lastUserID string
	lastTTL    time.Duration
	lastMeta   map[string]interface{}
	issueErr   error
	createErr  error
}

func (f *fakeAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	return nil, nil
}
func (f *fakeAuthenticator) CreateTokens(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	f.lastUserID = userID
	f.lastMeta = metadata
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createPair != nil {
		return f.createPair, nil
	}
	return &auth.TokenPair{AccessToken: "pair-token", ExpiresAt: 123}, nil
}
func (f *fakeAuthenticator) RefreshTokens(ctx context.Context, refreshToken string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	return nil, nil
}
func (f *fakeAuthenticator) RevokeToken(ctx context.Context, token string, reason string) error {
	return nil
}
func (f *fakeAuthenticator) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	return 0, nil
}
func (f *fakeAuthenticator) Close() error { return nil }
func (f *fakeAuthenticator) CreateAccessTokenWithTTL(ctx context.Context, userID string, metadata map[string]interface{}, ttl time.Duration) (string, int64, error) {
	f.lastUserID = userID
	f.lastMeta = metadata
	f.lastTTL = ttl
	if f.issueErr != nil {
		return "", 0, f.issueErr
	}
	if f.issueToken == "" {
		f.issueToken = "issued-token"
	}
	if f.issueExp == 0 {
		f.issueExp = 321
	}
	return f.issueToken, f.issueExp, nil
}

type pairOnlyAuthenticator struct{ pair *auth.TokenPair }

func (p *pairOnlyAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	return nil, nil
}
func (p *pairOnlyAuthenticator) CreateTokens(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	if p.pair != nil {
		return p.pair, nil
	}
	return &auth.TokenPair{AccessToken: "pair-token", ExpiresAt: 123}, nil
}
func (p *pairOnlyAuthenticator) RefreshTokens(ctx context.Context, refreshToken string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	return nil, nil
}
func (p *pairOnlyAuthenticator) RevokeToken(ctx context.Context, token string, reason string) error {
	return nil
}
func (p *pairOnlyAuthenticator) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	return 0, nil
}
func (p *pairOnlyAuthenticator) Close() error { return nil }

func newJobTokenScope(environmentName string) *jobtokenTestScope {
	cfg := &config.Config{
		Server: config.NewDefaultServerConfig(),
		Auth:   config.NewDefaultAuthConfig(),
		Log:    config.NewDefaultLogConfig(),
	}
	cfg.Server.Environment = environmentName
	return &jobtokenTestScope{ctx: context.Background(), cfg: cfg, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestAuthHelpers(t *testing.T) {
	if isProduction(nil) {
		t.Fatal("expected nil env not to be production")
	}
	if !isProduction(newJobTokenScope("production")) {
		t.Fatal("expected production environment to be detected")
	}

	tlsState := &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{DNSNames: []string{"task.choysum.internal", " api.internal "}}, {DNSNames: []string{"task.choysum.internal"}}}}
	sans := extractClientSANs(tlsState)
	if len(sans) != 2 {
		t.Fatalf("expected deduplicated SANs, got %#v", sans)
	}
	if !hasAllowedSAN(tlsState, []string{"API.INTERNAL"}) {
		t.Fatalf("expected SAN allowlist match, got %#v", sans)
	}
	if hasAllowedSAN(nil, []string{"task.choysum.internal"}) || hasAllowedSAN(tlsState, nil) {
		t.Fatal("did not expect SAN match with missing inputs")
	}

	runtimeScope := newJobTokenScope("development")
	runtimeScope.cfg.Auth.InternalKey = "secret"
	if err := authorizeInternalCaller(nil, metadata.Pairs(internalKeyHeader, "secret"), runtimeScope); err != nil {
		t.Fatalf("expected internal key auth to succeed, got %v", err)
	}
	runtimeScope.cfg.Auth.JobTokenAllowedSANs = []string{"task.choysum.internal"}
	if err := authorizeInternalCaller(tlsState, metadata.MD{}, runtimeScope); err != nil {
		t.Fatalf("expected SAN auth to succeed, got %v", err)
	}
	prodRuntimeScope := newJobTokenScope("production")
	prodRuntimeScope.cfg.Auth.InternalKey = "secret"
	if err := authorizeInternalCaller(nil, metadata.Pairs(internalKeyHeader, "secret"), prodRuntimeScope); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected production internal key fallback to be rejected, got %v", err)
	}
	if err := authorizeInternalCaller(nil, metadata.MD{}, &jobtokenTestScope{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected missing auth config error, got %v", err)
	}

	ctx := peer.NewContext(context.Background(), &peer.Peer{AuthInfo: credentials.TLSInfo{State: *tlsState}})
	loaded := tlsInfoFromContext(ctx)
	if loaded == nil || len(loaded.PeerCertificates) != 2 {
		t.Fatalf("expected TLS info from context, got %#v", loaded)
	}
	if err := authorizeInternalCallerFromContext(ctx, metadata.MD{}, runtimeScope); err != nil {
		t.Fatalf("expected authorizeInternalCallerFromContext to use TLS peer info, got %v", err)
	}
}
