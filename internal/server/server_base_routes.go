// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"net"
	"net/http"
	"strconv"

	documentgateway "github.com/choysum-dev/choysum/internal/document/gateway"
	i18ngateway "github.com/choysum-dev/choysum/internal/i18n/gateway"
	"google.golang.org/grpc/resolver"
)

func (s *GRPCWebServer) initBaseServerState(opts runtimeOptions) {
	s.address = &resolver.Address{
		Addr: net.JoinHostPort(opts.bindAddress, strconv.Itoa(opts.port)),
	}
	s.initBaseRoutes()
}

func (s *GRPCWebServer) initBaseRoutes() {
	s.mux = http.NewServeMux()

	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	s.mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/plain; charset=utf-8")
		if s.ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
	})

	documentgateway.RegisterSkeletonHandlers(s.mux, s.runtimeScope)
	i18ngateway.RegisterHandlers(s.mux, s.runtimeScope)
}
