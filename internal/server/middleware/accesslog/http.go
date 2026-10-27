// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Middleware logs finished non-gRPC HTTP requests for the mux chain.
func Middleware(opts Options, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r == nil {
				next.ServeHTTP(w, r)
				return
			}
			path := ""
			if r.URL != nil {
				path = r.URL.Path
			}
			if ShouldSkipHTTP(path) {
				next.ServeHTTP(w, r)
				return
			}
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			started := time.Now()
			next.ServeHTTP(rec, r)
			duration := time.Since(started)
			decision := ClassifyHTTP(opts, rec.status, duration)
			if !decision.Log {
				return
			}
			ctx := r.Context()
			LogRequest(ctx, logger, Record{
				Protocol: "http",
				Method:   r.Method,
				Path:     path,
				Code:     fmt.Sprintf("%d", rec.status),
				Duration: duration,
				User:     observedUserID(ctx),
				Slow:     decision.Slow,
				Level:    decision.Level,
			})
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}
