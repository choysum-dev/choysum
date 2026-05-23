// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gzipmiddleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
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
