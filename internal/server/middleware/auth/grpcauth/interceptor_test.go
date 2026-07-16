// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcauth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type grpcauthTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (f *grpcauthTestScope) Run(fn func(scope.Scope) error) error { return fn(f) }
func (f *grpcauthTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(f)
}
func (f *grpcauthTestScope) Session() *scope.Session { return nil }
func (f *grpcauthTestScope) WithContext(ctx context.Context) scope.Scope {
	return &grpcauthTestScope{ctx: ctx, cfg: f.cfg, logger: f.logger}
}
func (f *grpcauthTestScope) Context() context.Context { return f.ctx }
func (f *grpcauthTestScope) Logger() *slog.Logger     { return f.logger }
func (f *grpcauthTestScope) Config() *config.Config   { return f.cfg }

func (f *grpcauthTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(f.Config())
}

type grpcTestIdentity struct {
	userID  string
	tokenID string
	valid   bool
}

func (i grpcTestIdentity) GetUserID() string                   { return i.userID }
func (i grpcTestIdentity) GetTokenID() string                  { return i.tokenID }
func (i grpcTestIdentity) GetMetadata() map[string]interface{} { return nil }
func (i grpcTestIdentity) IsValid() bool                       { return i.valid }

type grpcFakeAuthenticator struct {
	validateFn func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error)
}

func (f grpcFakeAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	return f.validateFn(ctx, token, tokenType, checkRevoked)
}
func (f grpcFakeAuthenticator) CreateTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, errors.New("not implemented")
}
func (f grpcFakeAuthenticator) RefreshTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, errors.New("not implemented")
}
func (f grpcFakeAuthenticator) RevokeToken(context.Context, string, string) error {
	return errors.New("not implemented")
}
func (f grpcFakeAuthenticator) RevokeAllUserTokens(context.Context, string, string, string) (int, error) {
	return 0, errors.New("not implemented")
}
func (f grpcFakeAuthenticator) Close() error { return nil }

type testServerStream struct {
	ctx context.Context
}

func (s *testServerStream) SetHeader(metadata.MD) error  { return nil }
func (s *testServerStream) SendHeader(metadata.MD) error { return nil }
func (s *testServerStream) SetTrailer(metadata.MD)       {}
func (s *testServerStream) Context() context.Context     { return s.ctx }
func (s *testServerStream) SendMsg(interface{}) error    { return nil }
func (s *testServerStream) RecvMsg(interface{}) error    { return nil }

func newGRPCAuthScope() *grpcauthTestScope {
	return &grpcauthTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Auth:   config.NewDefaultAuthConfig(),
			Server: config.NewDefaultServerConfig(),
			Log:    config.NewDefaultLogConfig(),
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestHelpersDepthTLSAndSANMatching(t *testing.T) {
	if depth := getDepthFromIncomingMetadata(context.Background()); depth != 0 {
		t.Fatalf("depth = %d, want 0", depth)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "  "))
	if depth := getDepthFromIncomingMetadata(ctx); depth != 0 {
		t.Fatalf("blank depth = %d, want 0", depth)
	}
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "2"))
	if depth := getDepthFromIncomingMetadata(ctx); depth != 2 {
		t.Fatalf("depth = %d, want 2", depth)
	}
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "bad"))
	if depth := getDepthFromIncomingMetadata(ctx); depth != 0 {
		t.Fatalf("invalid depth = %d, want 0", depth)
	}

	tlsState := &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{DNSNames: []string{"task.choysum.internal", "api.internal"}}, {DNSNames: []string{"task.choysum.internal"}}}}
	sans := extractClientSANs(tlsState)
	if len(sans) != 2 {
		t.Fatalf("sans = %#v, want 2 unique SANs", sans)
	}
	if !hasAllowedSAN(tlsState, []string{"TASK.CHOYSUM.INTERNAL"}) {
		t.Fatal("expected SAN allowlist match")
	}
	if hasAllowedSAN(nil, []string{"task.choysum.internal"}) {
		t.Fatal("expected nil tls state not to match")
	}
	if sans := extractClientSANs(nil); sans != nil {
		t.Fatalf("nil tls state sans = %#v, want nil", sans)
	}
	if hasAllowedSAN(tlsState, nil) {
		t.Fatal("expected empty allowlist not to match")
	}
	if hasAllowedSAN(tlsState, []string{"   "}) {
		t.Fatal("expected blank allowlist entries not to match")
	}
	var nilCtx context.Context
	if info := tlsInfoFromContext(nilCtx); info != nil {
		t.Fatalf("nil ctx tls info = %#v, want nil", info)
	}
	if info := tlsInfoFromContext(context.Background()); info != nil {
		t.Fatalf("plain ctx tls info = %#v, want nil", info)
	}

	ctx = peer.NewContext(context.Background(), &peer.Peer{AuthInfo: credentials.TLSInfo{State: *tlsState}})
	if info := tlsInfoFromContext(ctx); info == nil || len(info.PeerCertificates) != 2 {
		t.Fatalf("tls info = %#v, want peer certs", info)
	}
}

func TestIsProductionAndAuthorizeInternalCallerRejectInvalidInputs(t *testing.T) {
	if isProduction(nil) {
		t.Fatal("expected nil env not to be production")
	}
	runtimeScope := newGRPCAuthScope()
	runtimeScope.cfg.Server = nil
	if isProduction(runtimeScope) {
		t.Fatal("expected nil server config not to be production")
	}
	runtimeScope.cfg.Server = config.NewDefaultServerConfig()
	runtimeScope.cfg.Server.Environment = " Production "
	if !isProduction(runtimeScope) {
		t.Fatal("expected trimmed case-insensitive production environment to match")
	}

	if authorizeInternalCaller(context.Background(), nil) {
		t.Fatal("expected nil env to reject internal caller")
	}
	runtimeScope.cfg.Auth = nil
	if authorizeInternalCaller(context.Background(), runtimeScope) {
		t.Fatal("expected nil auth config to reject internal caller")
	}

	runtimeScope = newGRPCAuthScope()
	runtimeScope.cfg.Auth.InternalKey = "secret"
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "wrong"))
	if authorizeInternalCaller(ctx, runtimeScope) {
		t.Fatal("expected wrong internal key to be rejected")
	}
}

func TestAuthorizeInternalCallerHonorsSANAndInternalKeyRules(t *testing.T) {
	runtimeScope := newGRPCAuthScope()
	runtimeScope.cfg.Auth.JobTokenAllowedSANs = []string{"task.choysum.internal"}
	tlsState := &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{DNSNames: []string{"task.choysum.internal"}}}}
	ctx := peer.NewContext(context.Background(), &peer.Peer{AuthInfo: credentials.TLSInfo{State: *tlsState}})
	if !authorizeInternalCaller(ctx, runtimeScope) {
		t.Fatal("expected SAN-authenticated internal caller to pass")
	}

	runtimeScope.cfg.Auth.JobTokenAllowedSANs = nil
	runtimeScope.cfg.Auth.InternalKey = "secret"
	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))
	if !authorizeInternalCaller(ctx, runtimeScope) {
		t.Fatal("expected internal key to authorize in non-production")
	}

	runtimeScope.cfg.Server.Environment = "production"
	if authorizeInternalCaller(ctx, runtimeScope) {
		t.Fatal("expected internal key to be rejected in production")
	}
}

func TestUnaryInterceptorBypassesSystemAndInternalMethods(t *testing.T) {
	runtimeScope := newGRPCAuthScope()
	nilAuth := NewAuthInterceptor(runtimeScope, nil).UnaryInterceptor()
	called := 0
	resp, err := nilAuth(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Any/Call"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called++
		return "nil-auth", nil
	})
	if err != nil || resp != "nil-auth" || called != 1 {
		t.Fatalf("nil auth resp=%v err=%v called=%d", resp, err, called)
	}

	runtimeScope.cfg.Auth.InternalKey = "secret"
	interceptor := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
		return nil, errors.New("should not validate")
	}}).UnaryInterceptor()

	called = 0
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called++
		return "ok", nil
	}

	resp, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, handler)
	if err != nil || resp != "ok" || called != 1 {
		t.Fatalf("system method resp=%v err=%v called=%d", resp, err, called)
	}

	called = 0
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))
	resp, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/task.Job/GetJob"}, handler)
	if err != nil || resp != "ok" || called != 1 {
		t.Fatalf("internal method resp=%v err=%v called=%d", resp, err, called)
	}

	called = 0
	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth.I18n/GetTranslations"}, handler)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("I18n without auth: want Unauthenticated, got %v", err)
	}

	called = 0
	resp, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/auth.I18n/GetTranslations"}, handler)
	if err != nil || resp != "ok" || called != 1 {
		t.Fatalf("I18n with internal key resp=%v err=%v called=%d", resp, err, called)
	}
}

func TestUnaryInterceptorHandlesEntrySkipErrorsAndSuccess(t *testing.T) {
	runtimeScope := newGRPCAuthScope()
	var gotCtx context.Context
	interceptor := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		if token != "token-1" || tokenType != auth.AccessToken || !checkRevoked {
			t.Fatalf("unexpected validate args token=%q type=%q revoked=%v", token, tokenType, checkRevoked)
		}
		return grpcTestIdentity{userID: "u1", tokenID: "t1", valid: true}, nil
	}}, WithEntryAuthSkipMethods("svc.Open/Login")).UnaryInterceptor()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		gotCtx = ctx
		return "ok", nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "0"))
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Open/Login"}, handler)
	if err != nil || resp != "ok" {
		t.Fatalf("entry-skip resp=%v err=%v", resp, err)
	}

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "1"))
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Open/Login"}, handler)
	if status.Code(err) != codes.Unauthenticated || !strings.Contains(err.Error(), "missing authentication token") {
		t.Fatalf("expected unauthenticated missing token, got %v", err)
	}

	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Secure/Call"}, handler)
	if status.Code(err) != codes.Unauthenticated || !strings.Contains(err.Error(), "missing metadata") {
		t.Fatalf("expected missing metadata error, got %v", err)
	}

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token-1"))
	resp, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Secure/Call"}, handler)
	if err != nil || resp != "ok" {
		t.Fatalf("secure resp=%v err=%v", resp, err)
	}
	if id := auth.IdentityFromContext(gotCtx); id == nil || id.GetUserID() != "u1" {
		t.Fatalf("identity in ctx = %#v, want u1", id)
	}
	if token, ok := auth.AccessTokenFromContext(gotCtx); !ok || token != "token-1" {
		t.Fatalf("access token = %q ok=%v, want token-1", token, ok)
	}
}

func TestUnaryInterceptorMapsAuthErrorsAndCanBeDisabledFromConfig(t *testing.T) {
	runtimeScope := newGRPCAuthScope()
	interceptor := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
		return nil, autherrors.NewAuthError(autherrors.ErrPermissionDenied, "access denied")
	}}).UnaryInterceptor()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer denied"))
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Secure/Call"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "bad", nil
	})
	if status.Code(err) != codes.Unauthenticated || !strings.Contains(err.Error(), autherrors.ErrPermissionDenied.String()) {
		t.Fatalf("expected mapped auth error, got %v", err)
	}

	runtimeScope.cfg.Auth.Enabled = false
	called := 0
	resp, err := AuthInterceptorFromConfig(runtimeScope, grpcFakeAuthenticator{})(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Any/Call"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called++
		return "ok", nil
	})
	if err != nil || resp != "ok" || called != 1 {
		t.Fatalf("disabled from config resp=%v err=%v called=%d", resp, err, called)
	}
}

func TestConvenienceAndConfigInterceptors(t *testing.T) {
	runtimeScope := newGRPCAuthScope()
	configured := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{}, WithHeaderName("x-auth-token"), WithEntryAuthSkipMethods("svc.Open/Login"))
	if configured.headerName != "x-auth-token" {
		t.Fatalf("header name = %q, want x-auth-token", configured.headerName)
	}
	if len(configured.entryAuthSkipMethods) != 1 || configured.entryAuthSkipMethods[0] != "svc.Open/Login" {
		t.Fatalf("entry skip methods = %#v, want svc.Open/Login", configured.entryAuthSkipMethods)
	}

	called := 0
	unary := AuthInterceptorFunc(runtimeScope, grpcFakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		if token != "wrapped-token" {
			t.Fatalf("token = %q, want wrapped-token", token)
		}
		return grpcTestIdentity{userID: "wrapped", tokenID: "tok", valid: true}, nil
	}})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer wrapped-token"))
	resp, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Secure/Call"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called++
		return auth.IdentityFromContext(ctx).GetUserID(), nil
	})
	if err != nil || resp != "wrapped" || called != 1 {
		t.Fatalf("wrapped unary resp=%v err=%v called=%d", resp, err, called)
	}

	streamCalled := 0
	streamInterceptor := StreamInterceptorFunc(runtimeScope, grpcFakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		if token != "stream-wrapped" {
			t.Fatalf("token = %q, want stream-wrapped", token)
		}
		return grpcTestIdentity{userID: "stream-wrapped", tokenID: "tok", valid: true}, nil
	}})
	stream := &testServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer stream-wrapped"))}
	err = streamInterceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc.Stream/Call"}, func(srv interface{}, ss grpc.ServerStream) error {
		streamCalled++
		if id := auth.IdentityFromContext(ss.Context()); id == nil || id.GetUserID() != "stream-wrapped" {
			t.Fatalf("stream identity = %#v, want stream-wrapped", id)
		}
		return nil
	})
	if err != nil || streamCalled != 1 {
		t.Fatalf("wrapped stream err=%v called=%d", err, streamCalled)
	}

	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.GrpcAuthentication = true
	runtimeScope.cfg.Auth.GrpcEntryPolicy = map[string]*config.EntryMethodConfig{
		" svc.Open/Login ": {SkipAuthentication: true},
		"svc.Nil/Skip":     nil,
	}

	called = 0
	resp, err = AuthInterceptorFromConfig(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
		return nil, errors.New("should skip auth")
	}})(metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "0")), nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Open/Login"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called++
		return "config-unary", nil
	})
	if err != nil || resp != "config-unary" || called != 1 {
		t.Fatalf("config unary resp=%v err=%v called=%d", resp, err, called)
	}

	streamCalled = 0
	err = StreamInterceptorFromConfig(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
		return nil, errors.New("should skip auth")
	}})(nil, &testServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "0"))}, &grpc.StreamServerInfo{FullMethod: "/svc.Open/Login"}, func(srv interface{}, ss grpc.ServerStream) error {
		streamCalled++
		return nil
	})
	if err != nil || streamCalled != 1 {
		t.Fatalf("config stream err=%v called=%d", err, streamCalled)
	}
}

func TestStreamInterceptorSuccessAndHelpers(t *testing.T) {
	runtimeScope := newGRPCAuthScope()
	interceptor := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
		return grpcTestIdentity{userID: "stream", tokenID: "tok", valid: true}, nil
	}}).StreamInterceptor()
	baseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer stream-token"))
	stream := &testServerStream{ctx: baseCtx}
	var gotCtx context.Context
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc.Stream/Call"}, func(srv interface{}, ss grpc.ServerStream) error {
		gotCtx = ss.Context()
		return nil
	})
	if err != nil {
		t.Fatalf("stream interceptor error: %v", err)
	}
	if id := auth.IdentityFromContext(gotCtx); id == nil || id.GetUserID() != "stream" {
		t.Fatalf("stream identity = %#v, want stream", id)
	}
	if token, ok := auth.AccessTokenFromContext(gotCtx); !ok || token != "stream-token" {
		t.Fatalf("stream token = %q ok=%v, want stream-token", token, ok)
	}

	wrapped := WrapServerStream(stream, context.WithValue(context.Background(), struct{}{}, "wrapped"))
	if wrapped.Context() == stream.Context() {
		t.Fatal("expected wrapped stream context to differ from original")
	}

	runtimeScope.cfg.Auth.Enabled = false
	called := 0
	err = StreamInterceptorFromConfig(runtimeScope, grpcFakeAuthenticator{})(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc.Stream/Call"}, func(srv interface{}, ss grpc.ServerStream) error {
		called++
		return nil
	})
	if err != nil || called != 1 {
		t.Fatalf("disabled stream err=%v called=%d", err, called)
	}
}

func TestStreamInterceptorBypassesAndMapsErrors(t *testing.T) {
	t.Run("bypasses nil authenticator and system methods", func(t *testing.T) {
		runtimeScope := newGRPCAuthScope()
		called := 0
		nilAuth := NewAuthInterceptor(runtimeScope, nil).StreamInterceptor()
		err := nilAuth(nil, &testServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/svc.Stream/Call"}, func(srv interface{}, ss grpc.ServerStream) error {
			called++
			return nil
		})
		if err != nil || called != 1 {
			t.Fatalf("nil auth err=%v called=%d", err, called)
		}

		called = 0
		system := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
			return nil, errors.New("should not validate")
		}}).StreamInterceptor()
		err = system(nil, &testServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, func(srv interface{}, ss grpc.ServerStream) error {
			called++
			return nil
		})
		if err != nil || called != 1 {
			t.Fatalf("system stream err=%v called=%d", err, called)
		}
	})

	t.Run("honors entry skip only for depth zero", func(t *testing.T) {
		runtimeScope := newGRPCAuthScope()
		interceptor := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
			return nil, errors.New("should not validate")
		}}, WithEntryAuthSkipMethods("svc.Open/Stream")).StreamInterceptor()

		called := 0
		err := interceptor(nil, &testServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "0"))}, &grpc.StreamServerInfo{FullMethod: "/svc.Open/Stream"}, func(srv interface{}, ss grpc.ServerStream) error {
			called++
			return nil
		})
		if err != nil || called != 1 {
			t.Fatalf("depth zero skip err=%v called=%d", err, called)
		}

		err = interceptor(nil, &testServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "1"))}, &grpc.StreamServerInfo{FullMethod: "/svc.Open/Stream"}, func(srv interface{}, ss grpc.ServerStream) error {
			return nil
		})
		if status.Code(err) != codes.Unauthenticated || !strings.Contains(err.Error(), "missing authentication token") {
			t.Fatalf("expected unauthenticated missing token, got %v", err)
		}
	})

	t.Run("returns metadata and token errors before validation", func(t *testing.T) {
		runtimeScope := newGRPCAuthScope()
		interceptor := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
			return nil, errors.New("should not validate")
		}}).StreamInterceptor()

		err := interceptor(nil, &testServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/svc.Stream/Call"}, func(srv interface{}, ss grpc.ServerStream) error {
			return nil
		})
		if status.Code(err) != codes.Unauthenticated || !strings.Contains(err.Error(), "missing metadata") {
			t.Fatalf("expected missing metadata error, got %v", err)
		}

		err = interceptor(nil, &testServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("other", "value"))}, &grpc.StreamServerInfo{FullMethod: "/svc.Stream/Call"}, func(srv interface{}, ss grpc.ServerStream) error {
			return nil
		})
		if status.Code(err) != codes.Unauthenticated || !strings.Contains(err.Error(), "missing authentication token") {
			t.Fatalf("expected missing token error, got %v", err)
		}
	})

	t.Run("maps validation errors to grpc status", func(t *testing.T) {
		runtimeScope := newGRPCAuthScope()
		interceptor := NewAuthInterceptor(runtimeScope, grpcFakeAuthenticator{validateFn: func(context.Context, string, auth.TokenType, bool) (auth.Identity, error) {
			return nil, autherrors.NewAuthError(autherrors.ErrPermissionDenied, "access denied")
		}}).StreamInterceptor()

		err := interceptor(nil, &testServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer denied-stream"))}, &grpc.StreamServerInfo{FullMethod: "/svc.Stream/Call"}, func(srv interface{}, ss grpc.ServerStream) error {
			return nil
		})
		if status.Code(err) != codes.Unauthenticated || !strings.Contains(err.Error(), autherrors.ErrPermissionDenied.String()) {
			t.Fatalf("expected mapped auth error, got %v", err)
		}
	})
}
