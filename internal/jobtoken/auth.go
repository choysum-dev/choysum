// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const internalKeyHeader = "x-choysum-internal-key"

func isProduction(runtimeScope scope.Scope) bool {
	return strings.EqualFold(strings.TrimSpace(runtimeOptionsFromScope(runtimeScope).serverEnvironment), "production")
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

func authorizeInternalCaller(ctxTLS *tls.ConnectionState, md metadata.MD, runtimeScope scope.Scope) error {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if !runtimeOpts.authConfigured {
		return status.Error(codes.Unauthenticated, "missing auth config")
	}
	allowed := runtimeOpts.authJobTokenAllowedSAN
	if hasAllowedSAN(ctxTLS, allowed) {
		return nil
	}

	key := strings.TrimSpace(runtimeOpts.authInternalKey)
	if key != "" && !isProduction(runtimeScope) {
		vals := md.Get(internalKeyHeader)
		if len(vals) > 0 {
			provided := strings.TrimSpace(vals[0])
			if subtle.ConstantTimeCompare([]byte(provided), []byte(key)) == 1 {
				return nil
			}
		}
	}

	return status.Error(codes.Unauthenticated, "internal authentication failed")
}

func authorizeInternalCallerFromContext(ctx context.Context, md metadata.MD, runtimeScope scope.Scope) error {
	return authorizeInternalCaller(tlsInfoFromContext(ctx), md, runtimeScope)
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
