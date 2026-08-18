// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/internal/server/middleware/auth/httpauth"
	gzipmiddleware "github.com/choysum-dev/choysum/internal/server/middleware/gzip"
	"github.com/choysum-dev/choysum/internal/server/middleware/security/cspmiddleware"
	"github.com/choysum-dev/choysum/internal/server/middleware/security/csrfmiddleware"
	"github.com/choysum-dev/choysum/pkg/auth"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

type ProtocolRouterOptions struct {
	RuntimeScope    scope.Scope
	Mux             *http.ServeMux
	GRPCServer      *grpc.Server
	GRPCWebProxy    *grpcweb.WrappedGrpcServer
	GRPCClientPool  *GRPCClientPool
	Authenticator   auth.Authenticator
	CSPEnabled      bool
	CSRFEnabled     bool
	AuthEnabled     bool
	HTTPAuthEnabled bool
	EnableGzip      bool
}

func NewProtocolRouter(opts ProtocolRouterOptions) http.Handler {
	isGRPCWebRequest := func(r *http.Request) bool {
		contentType := strings.ToLower(r.Header.Get("content-type"))
		return r.Method == http.MethodPost && strings.HasPrefix(contentType, "application/grpc-web")
	}

	isGRPCRequest := func(r *http.Request) bool {
		contentType := strings.ToLower(r.Header.Get("content-type"))
		return r.ProtoMajor == 2 && strings.HasPrefix(contentType, "application/grpc")
	}

	var httpHandler http.Handler = opts.Mux

	if opts.CSPEnabled {
		cspHandler := cspmiddleware.NewCSPHandler(opts.RuntimeScope)
		httpHandler = cspHandler.Handler(httpHandler)
		opts.RuntimeScope.Logger().Debug("csp protection enabled")
	}

	if opts.CSRFEnabled {
		csrfHandler := csrfmiddleware.NewCSRFHandler(opts.RuntimeScope)
		httpHandler = csrfHandler.Handler(httpHandler)
		opts.RuntimeScope.Logger().Debug("csrf protection enabled")
	}

	if opts.Authenticator != nil && opts.AuthEnabled && opts.HTTPAuthEnabled {
		httpHandler = httpauth.AuthHandlerFromConfig(opts.RuntimeScope, opts.Authenticator)(httpHandler)
	}

	if opts.EnableGzip {
		gzipHandler := gzipmiddleware.NewGzipHandler(opts.RuntimeScope)
		httpHandler = gzipHandler.Handler(httpHandler)
		opts.RuntimeScope.Logger().Debug("http gzip enabled")
	}

	var grpcWebHandler http.Handler
	if opts.GRPCWebProxy != nil {
		grpcWebHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Expose-Headers", "server-timing, grpc-status, grpc-message, grpc-status-details-bin, traceparent")
			opts.GRPCWebProxy.ServeHTTP(w, r)
		})
		// Do not gzip grpc-web: gzip buffers until the handler returns, which
		// stalls long-lived server streams. Regular HTTP gzip remains above.
	}

	grpcHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		opts.GRPCServer.ServeHTTP(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if opts.GRPCClientPool != nil {
			r = r.WithContext(grpcclient.ContextWithServiceDialer(r.Context(), opts.GRPCClientPool.Dial))
		}

		if opts.GRPCWebProxy != nil && isGRPCWebRequest(r) {
			if strings.Contains(r.URL.Path, ".TaskWorker/ExecuteJob") || r.URL.Path == "/auth.JobTokenService/IssueTaskJobToken" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			grpcWebHandler.ServeHTTP(w, r)
			return
		}

		if isGRPCRequest(r) {
			grpcHandler.ServeHTTP(w, r)
			return
		}

		httpHandler.ServeHTTP(w, r)
	})
}
