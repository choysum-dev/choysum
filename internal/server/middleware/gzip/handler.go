// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gzipmiddleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = statusCode
	// Downstream handlers may set Content-Length for the uncompressed body.
	// Gzip changes the wire size, so drop it and let the transport chunk.
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.Writer.Write(b)
}

// GzipHandler implements gzip middleware.
type GzipHandler struct {
	runtimeScope scope.Scope
}

func NewGzipHandler(runtimeScope scope.Scope) *GzipHandler {
	return &GzipHandler{
		runtimeScope: runtimeScope,
	}
}

func (h *GzipHandler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Document payload streams are already compressed media or large binaries;
		// gzip would rewrite Content-Length and hurt <img>/fetch consumers.
		cleanedPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		if strings.HasPrefix(cleanedPath, "/_document/") {
			next.ServeHTTP(w, r)
			return
		}

		// Check whether the client supports gzip.
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Create the gzip writer.
		gz, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
		if err != nil {
			h.runtimeScope.Logger().Error("gzip writer creation failed", "error", err)
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()

		// Set response headers.
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		// Clear any premature Content-Length before the handler runs.
		w.Header().Del("Content-Length")

		// Create the gzip response writer.
		gzipResponseWriter := &gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Handle the request.
		next.ServeHTTP(gzipResponseWriter, r)
	})
}
