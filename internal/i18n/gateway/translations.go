// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
	"golang.org/x/sync/errgroup"
)

type handler struct {
	runtimeScope scope.Scope
	// fetch is injectable for tests; nil uses real gRPC dial.
	fetch func(ctx context.Context, app, lang string, moduleNames []string) (*appTranslations, error)
	// listModules is injectable for tests; nil queries IrModule.
	listModules func() (map[string][]string, error)
	// search / update are injectable for terms routes (user-identity dial).
	search func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error)
	update func(ctx context.Context, accessToken, app, lang string, item termItem) (*termItem, string, error)
}

func newHandler(runtimeScope scope.Scope) *handler {
	return &handler{runtimeScope: runtimeScope}
}

func (h *handler) serveTranslations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if lang == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "lang is required",
		})
		return
	}
	// moduleNames query param is rejected/ignored (§8.2.1 / D3): host fills it.
	_ = r.URL.Query().Get("moduleNames")
	clientHash := strings.TrimSpace(r.URL.Query().Get("hash"))

	byApp, err := h.modulesByApp()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	type appResult struct {
		app   string
		trans *appTranslations
	}

	apps := make([]string, 0, len(byApp))
	for app := range byApp {
		apps = append(apps, app)
	}

	results := make([]appResult, len(apps))
	g, gctx := errgroup.WithContext(r.Context())
	var mu sync.Mutex
	for i, app := range apps {
		i, app := i, app
		modules := byApp[app]
		g.Go(func() error {
			trans, err := h.fetchApp(gctx, app, lang, modules)
			if err != nil {
				return err
			}
			mu.Lock()
			results[i] = appResult{app: app, trans: trans}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	appHashes := make(map[string]string, len(results))
	messages := map[string]map[string]map[string]string{}
	for _, item := range results {
		if item.trans == nil {
			continue
		}
		appHashes[item.app] = item.trans.Hash
		for mod, byScope := range item.trans.Terms {
			messages[mod] = byScope
		}
	}

	catalogHash := CatalogHash(appHashes)
	etag := catalogETag(catalogHash)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", etag)
	if ifNoneMatch(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	resp := map[string]any{
		"lang":   lang,
		"locale": LangToLocale(lang),
		"hash":   catalogHash,
	}
	if clientHash != "" && clientHash == catalogHash {
		resp["unchanged"] = true
		resp["messages"] = nil
	} else {
		resp["unchanged"] = false
		resp["messages"] = messages
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) modulesByApp() (map[string][]string, error) {
	if h.listModules != nil {
		return h.listModules()
	}
	return installedModulesByApp(h.runtimeScope)
}

func (h *handler) fetchApp(ctx context.Context, app, lang string, moduleNames []string) (*appTranslations, error) {
	if h.fetch != nil {
		return h.fetch(ctx, app, lang, moduleNames)
	}
	return fetchAppTranslations(ctx, h.runtimeScope, app, lang, moduleNames)
}

func catalogETag(catalogHash string) string {
	return `W/"i18n-` + catalogHash + `"`
}

func ifNoneMatch(header, currentETag string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}
	current := strings.TrimPrefix(strings.TrimSpace(currentETag), "W/")
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		if strings.TrimPrefix(candidate, "W/") == current {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
