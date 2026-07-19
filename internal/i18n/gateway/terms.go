// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	termsPath          = "/web/i18n/terms"
	defaultTermsLimit  = 50
	maxTermsLimit      = 100
	allAppsSearchLimit = 100
)

func (h *handler) serveTerms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.serveTermsList(w, r)
	case http.MethodPatch:
		h.serveTermsPatch(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *handler) serveTermsList(w http.ResponseWriter, r *http.Request) {
	accessToken, ok := requireTermsAuth(r.Context(), r.Header.Get("Authorization"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication is required"})
		return
	}

	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if lang == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "lang is required"})
		return
	}
	if !validLang(lang) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid lang format"})
		return
	}
	application := strings.TrimSpace(r.URL.Query().Get("application"))
	module := strings.TrimSpace(r.URL.Query().Get("module"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseQueryInt(r.URL.Query().Get("limit"), defaultTermsLimit)
	offset := parseQueryInt(r.URL.Query().Get("offset"), 0)
	if limit <= 0 {
		limit = defaultTermsLimit
	}
	if limit > maxTermsLimit {
		limit = maxTermsLimit
	}
	if offset < 0 {
		offset = 0
	}

	byApp, err := h.modulesByApp()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if module != "" && application == "" {
		for app, mods := range byApp {
			for _, m := range mods {
				if m == module {
					application = app
					break
				}
			}
			if application != "" {
				break
			}
		}
	}

	// D8: without q, application is required (no All-apps pagination).
	if q == "" && application == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "application is required when q is empty",
		})
		return
	}

	if application != "" {
		if _, known := byApp[application]; !known {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown application"})
			return
		}
		modules := byApp[application]
		if module != "" {
			modules = []string{module}
		}
		result, err := h.searchApp(r.Context(), accessToken, application, lang, modules, q, limit, offset)
		if err != nil {
			writeTermsRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"lang":   lang,
			"items":  result.Items,
			"total":  result.Total,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	// All-apps search with q: concurrent fan-out, limit capped, no pagination (D8).
	limit = allAppsSearchLimit
	apps := sortedAppNames(byApp)
	type appSearchOutcome struct {
		items  []termItem
		denied bool
		failed error
	}
	outcomes := make([]appSearchOutcome, len(apps))
	g, gctx := errgroup.WithContext(r.Context())
	for i, app := range apps {
		i, app := i, app
		modules := byApp[app]
		if module != "" {
			modules = []string{module}
		}
		g.Go(func() error {
			result, err := h.searchApp(gctx, accessToken, app, lang, modules, q, limit, 0)
			if err != nil {
				if status.Code(err) == codes.PermissionDenied {
					outcomes[i] = appSearchOutcome{denied: true}
					return nil
				}
				// Degrade like serveTranslations: one offline app must not 502 the fan-out.
				h.logger().Warn("i18n terms search: skipping failed application",
					"application", app, "lang", lang, "error", err)
				outcomes[i] = appSearchOutcome{failed: err}
				return nil
			}
			if result != nil {
				outcomes[i] = appSearchOutcome{items: result.Items}
			}
			return nil
		})
	}
	_ = g.Wait()

	var items []termItem
	var denied bool
	var firstFail error
	for _, outcome := range outcomes {
		if outcome.denied {
			denied = true
			continue
		}
		if outcome.failed != nil {
			if firstFail == nil {
				firstFail = outcome.failed
			}
			continue
		}
		items = append(items, outcome.items...)
	}
	if len(items) == 0 {
		if denied {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "terminology editor permission required"})
			return
		}
		if firstFail != nil {
			writeTermsRPCError(w, firstFail)
			return
		}
	}
	truncated := len(items) >= limit
	if len(items) > limit {
		items = items[:limit]
		truncated = true
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lang":      lang,
		"items":     items,
		"total":     len(items),
		"limit":     limit,
		"offset":    0,
		"truncated": truncated,
	})
}

func (h *handler) serveTermsPatch(w http.ResponseWriter, r *http.Request) {
	accessToken, ok := requireTermsAuth(r.Context(), r.Header.Get("Authorization"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication is required"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	items, lang, err := parsePatchBody(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if lang == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "lang is required"})
		return
	}
	if !validLang(lang) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid lang format"})
		return
	}
	if len(items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "items are required"})
		return
	}

	byApp, err := h.modulesByApp()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Pre-validate all items before any write (reject bad requests early).
	// Multi-app patches are not distributed-transactional: each updateApp is an
	// independent per-app RPC, so a later failure leaves earlier apps committed.
	grouped := map[string][]termItem{}
	for i, item := range items {
		app := strings.TrimSpace(item.Application)
		if app == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "application is required", "index": i})
			return
		}
		if _, known := byApp[app]; !known {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown application", "application": app, "index": i})
			return
		}
		if strings.TrimSpace(item.Module) == "" || strings.TrimSpace(item.Scope) == "" || strings.TrimSpace(item.Src) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module, scope, and src are required", "index": i})
			return
		}
		grouped[app] = append(grouped[app], item)
	}

	updated := make([]termItem, 0, len(items))
	for _, app := range sortedKeys(grouped) {
		for _, item := range grouped[app] {
			out, _, err := h.updateApp(r.Context(), accessToken, app, lang, item)
			if err != nil {
				writeTermsRPCError(w, err)
				return
			}
			if out != nil {
				updated = append(updated, *out)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"lang":  lang,
		"items": updated,
	})
}

func (h *handler) searchApp(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
	if h.search != nil {
		return h.search(ctx, accessToken, app, lang, modules, q, limit, offset)
	}
	return fetchAppSearchTerms(ctx, h.runtimeScope, accessToken, app, lang, modules, q, limit, offset)
}

func (h *handler) updateApp(ctx context.Context, accessToken, app, lang string, item termItem) (*termItem, string, error) {
	if h.update != nil {
		return h.update(ctx, accessToken, app, lang, item)
	}
	return invokeAppUpdateTerm(ctx, h.runtimeScope, accessToken, app, lang, item)
}

func parsePatchBody(body []byte) ([]termItem, string, error) {
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) == 0 {
		return nil, "", fmt.Errorf("empty body")
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, "", fmt.Errorf("invalid json")
	}
	lang := jsonMapTrimmedString(raw, "lang")

	if itemsRaw, ok := raw["items"]; ok && itemsRaw != nil {
		encoded, err := json.Marshal(itemsRaw)
		if err != nil {
			return nil, "", fmt.Errorf("invalid items")
		}
		var items []termItem
		if err := json.Unmarshal(encoded, &items); err != nil {
			return nil, "", fmt.Errorf("invalid items")
		}
		return items, lang, nil
	}

	item := termItem{
		Application: jsonMapTrimmedString(raw, "application"),
		Module:      jsonMapTrimmedString(raw, "module"),
		Scope:       jsonMapTrimmedString(raw, "scope"),
		Src:         jsonMapString(raw, "src"),
		Value:       jsonMapString(raw, "value"),
		Kind:        jsonMapTrimmedString(raw, "kind"),
	}
	if item.Application == "" && item.Module == "" {
		return nil, "", fmt.Errorf("items or term object required")
	}
	return []termItem{item}, lang, nil
}

// jsonMapString returns a string JSON field, or "" when missing/null/non-string.
func jsonMapString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func jsonMapTrimmedString(m map[string]any, key string) string {
	return strings.TrimSpace(jsonMapString(m, key))
}

func writeTermsRPCError(w http.ResponseWriter, err error) {
	code := status.Code(err)
	switch code {
	case codes.PermissionDenied:
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "terminology editor permission required"})
	case codes.Unauthenticated:
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication is required"})
	case codes.InvalidArgument:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
	}
}

func parseQueryInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func sortedAppNames(byApp map[string][]string) []string {
	apps := make([]string, 0, len(byApp))
	for app := range byApp {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	return apps
}

func sortedKeys(m map[string][]termItem) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
