// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"net"
	"net/http"
	"time"
)

const (
	defaultDialTimeout         = 10 * time.Second
	defaultKeepAlive           = 30 * time.Second
	defaultTLSHandshakeTimeout = 10 * time.Second
	defaultIdleConnTimeout     = 90 * time.Second
	defaultMaxIdleConns        = 100
)

// defaultHTTPClient is a package-level shared *http.Client for registry HTTP
// requests (catalog index, npm metadata, and tarball downloads). It uses its
// own transport rather than http.DefaultTransport so that idle-connection
// lifecycle is scoped to registry operations.
//
// No client-level Timeout is set: tarball downloads may legitimately take
// longer than a fixed deadline. The transport-level timeouts (DialContext,
// TLSHandshakeTimeout, ResponseHeaderTimeout) cover connection establishment
// and response-headers arrival, and callers are expected to provide a
// context.Context for overall request cancellation.
var defaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          defaultMaxIdleConns,
		MaxIdleConnsPerHost:   defaultMaxIdleConns,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	},
}

// newDefaultHTTPClient returns the shared registry HTTP client so callers
// don't allocate a new transport on every invocation.
func newDefaultHTTPClient() *http.Client {
	return defaultHTTPClient
}
