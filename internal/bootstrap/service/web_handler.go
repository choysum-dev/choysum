// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	bootstrapweb "github.com/choysum-dev/choysum/internal/bootstrap/web"
	"github.com/choysum-dev/choysum/pkg/scope"
)

const (
	bootstrapBasePath   = "/bootstrap"
	bootstrapAssetPath  = "/bootstrap/assets/"
	immutableAssetCache = "public, max-age=31536000, immutable"
	noCacheHeader       = "no-cache"
)

func newBootstrapWebHandler(runtimeScope scope.Scope) (http.Handler, error) {
	source := os.Getenv(bootstrapweb.EnvBootstrapWebSource)
	distFS, sourceName, sourceRoot, err := bootstrapweb.LoadDistFS(source)
	if err != nil {
		return nil, err
	}

	runtimeScope.Logger().Debug("bootstrap web source selected", "source", sourceName, "root", sourceRoot)

	fileServer := http.FileServer(http.FS(distFS))
	stripPrefixHandler := http.StripPrefix(bootstrapBasePath+"/", fileServer)

	serveIndex := func(w http.ResponseWriter, _ *http.Request) {
		body, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			runtimeScope.Logger().Error("bootstrap index read failed", "error", err)
			http.Error(w, "bootstrap index not available", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", noCacheHeader)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == bootstrapBasePath || p == bootstrapBasePath+"/" {
			serveIndex(w, r)
			return
		}
		if !strings.HasPrefix(p, bootstrapBasePath+"/") {
			http.NotFound(w, r)
			return
		}

		relPath := strings.TrimPrefix(p, bootstrapBasePath+"/")
		relPath = path.Clean(relPath)
		if relPath == "." || relPath == "" {
			serveIndex(w, r)
			return
		}
		if strings.HasPrefix(relPath, "../") {
			http.NotFound(w, r)
			return
		}

		if strings.HasPrefix(p, bootstrapAssetPath) {
			w.Header().Set("Cache-Control", immutableAssetCache)
			serveBootstrapAssetRequest(runtimeScope, w, r, stripPrefixHandler)
			return
		}

		if st, err := fs.Stat(distFS, relPath); err == nil && !st.IsDir() {
			if strings.HasSuffix(relPath, ".html") {
				w.Header().Set("Cache-Control", noCacheHeader)
			}
			stripPrefixHandler.ServeHTTP(w, r)
			return
		}

		serveIndex(w, r)
	}), nil
}

func serveBootstrapAssetRequest(runtimeScope scope.Scope, w http.ResponseWriter, r *http.Request, handler http.Handler) {
	writer := &bootstrapAssetStatusResponseWriter{ResponseWriter: w, status: http.StatusOK}
	handler.ServeHTTP(writer, r)
	logBootstrapAssetRequestResult(runtimeScope, r, writer.status)
}

func logBootstrapAssetRequestResult(runtimeScope scope.Scope, r *http.Request, status int) {
	attrs := []any{"method", r.Method, "path", r.URL.Path, "status", status}
	if status < http.StatusBadRequest {
		runtimeScope.Logger().Debug("web asset served", attrs...)
		return
	}
	if status >= http.StatusInternalServerError {
		runtimeScope.Logger().Error("web asset request failed", attrs...)
		return
	}
	runtimeScope.Logger().Warn("web asset request failed", attrs...)
}

type bootstrapAssetStatusResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *bootstrapAssetStatusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *bootstrapAssetStatusResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
