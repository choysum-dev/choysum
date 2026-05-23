// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

type HTTPServerHandle struct {
	Listener net.Listener
	Server   *http.Server
}

type HTTPServerOptions struct {
	Address          string
	ExistingListener net.Listener
	HasProxy         bool
	EnableTLS        bool
	TLSCertFile      string
	TLSKeyFile       string
	Logger           *slog.Logger
}

func StartHTTPServer(handler http.Handler, opts HTTPServerOptions) (*HTTPServerHandle, error) {
	listener := opts.ExistingListener
	if listener == nil {
		created, err := net.Listen("tcp", opts.Address)
		if err != nil {
			return nil, xfmt.Errorf("Failed to listen: %w", err)
		}
		listener = created
	}

	httpServer := &http.Server{Handler: handler}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	go func(srv *http.Server, ln net.Listener) {
		listenAddress := opts.Address
		if ln != nil && ln.Addr() != nil {
			listenAddress = ln.Addr().String()
		}
		scheme := "http"
		if opts.EnableTLS {
			scheme = "https"
			logger.Info("http server listening", httpServerListeningLogFields(opts.Address, listenAddress, scheme, opts.HasProxy)...)
			if err := srv.ServeTLS(ln, opts.TLSCertFile, opts.TLSKeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("http server serve failed", "error", err)
			}
			return
		}
		logger.Info("http server listening", httpServerListeningLogFields(opts.Address, listenAddress, scheme, opts.HasProxy)...)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server serve failed", "error", err)
		}
	}(httpServer, listener)

	return &HTTPServerHandle{Listener: listener, Server: httpServer}, nil
}

func httpServerListeningLogFields(configuredAddress, listenAddress, scheme string, hasProxy bool) []any {
	fields := []any{
		"address", listenAddress,
		"scheme", scheme,
		"grpc_web_proxy", hasProxy,
	}
	if accessURL := httpServerAccessURL(configuredAddress, listenAddress, scheme); accessURL != "" {
		fields = append(fields, "access_url", accessURL)
	}
	return fields
}

func httpServerAccessURL(configuredAddress, listenAddress, scheme string) string {
	if strings.TrimSpace(scheme) == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(strings.TrimSpace(listenAddress))
	if err != nil {
		return ""
	}
	host := httpServerAddressHost(configuredAddress)
	if host == "" || httpServerIsWildcardHost(host) {
		host = httpServerAddressHost(listenAddress)
	}
	if host == "" || httpServerIsWildcardHost(host) {
		host = "localhost"
	}
	return (&url.URL{Scheme: scheme, Host: net.JoinHostPort(host, port)}).String()
}

func httpServerAddressHost(address string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return ""
	}
	return host
}

func httpServerIsWildcardHost(host string) bool {
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsUnspecified()
}
