// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcauth

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"strconv"
	"strings"

	middleware "github.com/choysum-dev/choysum/internal/server/middleware/auth"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// AuthInterceptor is a gRPC authentication interceptor.
type AuthInterceptor struct {
	authenticator auth.Authenticator
	runtimeScope  scope.Scope
	runtimeOpts   runtimeOptions
	// entryAuthSkipMethods are methods allowed to skip authentication (anonymous entry)
	// but only for depth=0 inbound requests.
	entryAuthSkipMethods []string
	headerName           string // Defaults to "authorization".
}

var systemAuthSkipMethods = []string{
	// Standard gRPC health service.
	"grpc.health.v1.Health/*",
	// Compatibility aliases (non-standard service naming).
	"Health/*",

	// gRPC server reflection.
	"grpc.reflection.v1alpha.ServerReflection/*",
	"grpc.reflection.v1.ServerReflection/*",

	// gRPC channelz (debugging / observability).
	"grpc.channelz.v1.Channelz/*",

	// Internal job token issuance (authenticated via mTLS/internal key).
	"auth.JobTokenService/*",
}

var internalAuthPreferredMethods = []string{
	"task.Job/GetJob",
	"*.TaskWorker/*",
	"*.I18n/GetTranslations",
	"*.TranslationTerm/GetTranslations",
	"meta.MetaApplication/*",
	"meta.MetaField/*",
	"meta.MetaModel/*",
	"meta.MetaService/*",
}

const internalKeyHeader = "x-choysum-internal-key"

// NewAuthInterceptor creates a gRPC authentication interceptor.
func NewAuthInterceptor(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...Option) *AuthInterceptor {
	interceptor := &AuthInterceptor{
		authenticator:        authenticator,
		runtimeScope:         runtimeScope,
		runtimeOpts:          runtimeOptionsFromScope(runtimeScope),
		entryAuthSkipMethods: []string{},
		headerName:           "authorization",
	}

	for _, opt := range opts {
		opt(interceptor)
	}

	return interceptor
}

func getDepthFromIncomingMetadata(ctx context.Context) int {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0
	}
	vals := md.Get("x-choysum-depth")
	if len(vals) == 0 {
		return 0
	}
	v := strings.TrimSpace(vals[0])
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return n
	}
	return 0
}

func tlsInfoFromContext(ctx context.Context) *tls.ConnectionState {
	if ctx == nil {
		return nil
	}
	if p, ok := peer.FromContext(ctx); ok {
		if p.AuthInfo != nil {
			if ti, ok := p.AuthInfo.(credentials.TLSInfo); ok {
				return &ti.State
			}
		}
	}
	return nil
}

func extractClientSANs(ctxTLS *tls.ConnectionState) []string {
	if ctxTLS == nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, cert := range ctxTLS.PeerCertificates {
		for _, dns := range cert.DNSNames {
			dns = strings.TrimSpace(dns)
			if dns != "" {
				set[dns] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for dns := range set {
		out = append(out, dns)
	}
	return out
}

func hasAllowedSAN(ctxTLS *tls.ConnectionState, allowed []string) bool {
	if ctxTLS == nil || len(allowed) == 0 {
		return false
	}
	allowedSet := map[string]struct{}{}
	for _, a := range allowed {
		a = strings.TrimSpace(strings.ToLower(a))
		if a != "" {
			allowedSet[a] = struct{}{}
		}
	}
	for _, dns := range extractClientSANs(ctxTLS) {
		if _, ok := allowedSet[strings.ToLower(dns)]; ok {
			return true
		}
	}
	return false
}

func isProduction(runtimeScope scope.Scope) bool {
	return isProductionWithOptions(runtimeOptionsFromScope(runtimeScope))
}

func authorizeInternalCaller(ctx context.Context, runtimeScope scope.Scope) bool {
	return authorizeInternalCallerWithOptions(ctx, runtimeOptionsFromScope(runtimeScope))
}

func isProductionWithOptions(opts runtimeOptions) bool {
	return strings.EqualFold(strings.TrimSpace(opts.serverEnvironment), "production")
}

func authorizeInternalCallerWithOptions(ctx context.Context, opts runtimeOptions) bool {
	md, _ := metadata.FromIncomingContext(ctx)
	ctxTLS := tlsInfoFromContext(ctx)
	allowed := opts.jobTokenAllowedSANs
	if hasAllowedSAN(ctxTLS, allowed) {
		return true
	}

	key := strings.TrimSpace(opts.internalKey)
	if key != "" && !isProductionWithOptions(opts) {
		vals := md.Get(internalKeyHeader)
		if len(vals) > 0 {
			provided := strings.TrimSpace(vals[0])
			if subtle.ConstantTimeCompare([]byte(provided), []byte(key)) == 1 {
				return true
			}
		}
	}
	return false
}

// UnaryInterceptor creates a unary RPC interceptor.
func (i *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Handle the request directly when authentication is disabled.
		if i.authenticator == nil {
			return handler(ctx, req)
		}

		// System methods are always allowed without authentication.
		if middleware.IsMethodExcluded(info.FullMethod, systemAuthSkipMethods) {
			return handler(ctx, req)
		}

		// Internal service identity (mTLS/internal key) for selected methods.
		if middleware.IsMethodExcluded(info.FullMethod, internalAuthPreferredMethods) {
			if authorizeInternalCallerWithOptions(ctx, i.runtimeOpts) {
				return handler(ctx, req)
			}
		}

		// Entry-derived anonymous methods apply to inbound (depth=0) requests only.
		if getDepthFromIncomingMetadata(ctx) == 0 {
			if middleware.IsMethodExcluded(info.FullMethod, i.entryAuthSkipMethods) {
				return handler(ctx, req)
			}
		}

		// Read metadata.
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Read the token from the Authorization header.
		tokens := md.Get(i.headerName)
		if len(tokens) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authentication token")
		}

		// Validate the token and always check revocation.
		token := middleware.ExtractBearerToken(tokens[0])
		identity, err := i.authenticator.ValidateToken(ctx, token, auth.AccessToken, true)
		if err != nil {
			// Convert auth errors to gRPC status errors.
			return nil, middleware.AuthErrorToGRPCStatus(err)
		}

		// Write identity and request access token into trusted context (Go-only).
		newCtx := auth.ContextWithIdentity(ctx, identity)
		newCtx = auth.ContextWithAccessToken(newCtx, token)

		// Call the next handler.
		return handler(newCtx, req)
	}
}

// StreamInterceptor creates a streaming RPC interceptor.
func (i *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Handle the request directly when authentication is disabled.
		if i.authenticator == nil {
			return handler(srv, ss)
		}

		// System methods are always allowed without authentication.
		if middleware.IsMethodExcluded(info.FullMethod, systemAuthSkipMethods) {
			return handler(srv, ss)
		}

		// Entry-derived anonymous methods apply to inbound (depth=0) requests only.
		if getDepthFromIncomingMetadata(ss.Context()) == 0 {
			if middleware.IsMethodExcluded(info.FullMethod, i.entryAuthSkipMethods) {
				return handler(srv, ss)
			}
		}

		// Read metadata from context.
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Read the token from the Authorization header.
		tokens := md.Get(i.headerName)
		if len(tokens) == 0 {
			return status.Errorf(codes.Unauthenticated, "missing authentication token")
		}

		// Validate the token and always check revocation.
		token := middleware.ExtractBearerToken(tokens[0])
		identity, err := i.authenticator.ValidateToken(ctx, token, auth.AccessToken, true)
		if err != nil {
			// Convert auth errors to gRPC status errors.
			return middleware.AuthErrorToGRPCStatus(err)
		}

		// Create a new context carrying identity and the request access token (Go-only).
		newCtx := auth.ContextWithIdentity(ctx, identity)
		newCtx = auth.ContextWithAccessToken(newCtx, token)

		// Wrap the stream with the new context.
		wrappedStream := WrapServerStream(ss, newCtx)

		// Call the next handler.
		return handler(srv, wrappedStream)
	}
}

// WrappedServerStream wraps grpc.ServerStream with a custom context.
type WrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context.
func (w *WrappedServerStream) Context() context.Context {
	return w.ctx
}

// WrapServerStream wraps a ServerStream with a new context.
func WrapServerStream(ss grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &WrappedServerStream{
		ServerStream: ss,
		ctx:          ctx,
	}
}

// AuthInterceptorFunc creates a convenience gRPC auth interceptor.
func AuthInterceptorFunc(runtimeScope scope.Scope, authenticator auth.Authenticator) grpc.UnaryServerInterceptor {
	return NewAuthInterceptor(runtimeScope, authenticator).UnaryInterceptor()
}

// StreamInterceptorFunc creates a convenience gRPC streaming auth interceptor.
func StreamInterceptorFunc(runtimeScope scope.Scope, authenticator auth.Authenticator) grpc.StreamServerInterceptor {
	return NewAuthInterceptor(runtimeScope, authenticator).StreamInterceptor()
}

// AuthInterceptorFromConfig creates a gRPC auth interceptor from config.
func AuthInterceptorFromConfig(runtimeScope scope.Scope, authenticator auth.Authenticator) grpc.UnaryServerInterceptor {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if authenticator == nil || !runtimeOpts.authEnabled || !runtimeOpts.grpcAuthentication {
		// Return a no-op interceptor.
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	opts := []Option{}

	// Entry-derived anonymous methods (depth=0 only)
	if len(runtimeOpts.entryAuthSkipMethods) > 0 {
		opts = append(opts, WithEntryAuthSkipMethods(runtimeOpts.entryAuthSkipMethods...))
	}

	// Configure the header name.
	opts = append(opts, WithHeaderName("authorization"))

	return NewAuthInterceptor(runtimeScope, authenticator, opts...).UnaryInterceptor()
}

// StreamInterceptorFromConfig creates a gRPC streaming auth interceptor from config.
func StreamInterceptorFromConfig(runtimeScope scope.Scope, authenticator auth.Authenticator) grpc.StreamServerInterceptor {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if authenticator == nil || !runtimeOpts.authEnabled || !runtimeOpts.grpcAuthentication {
		// Return a no-op interceptor.
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	opts := []Option{}

	// Entry-derived anonymous methods (depth=0 only)
	if len(runtimeOpts.entryAuthSkipMethods) > 0 {
		opts = append(opts, WithEntryAuthSkipMethods(runtimeOpts.entryAuthSkipMethods...))
	}

	// Configure the header name.
	opts = append(opts, WithHeaderName("authorization"))

	return NewAuthInterceptor(runtimeScope, authenticator, opts...).StreamInterceptor()
}
