// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/internal/export/runner"
	"github.com/choysum-dev/choysum/internal/i18n/po"
	"github.com/choysum-dev/choysum/internal/i18n/terms"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

const poPath = "/web/i18n/po"

func (h *handler) servePO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
	if application == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "application is required"})
		return
	}
	module := strings.TrimSpace(r.URL.Query().Get("module"))
	if module == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module is required"})
		return
	}

	byApp, err := h.modulesByApp()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if _, known := byApp[application]; !known {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown application"})
		return
	}

	modules := byApp[application]
	if !moduleBelongsToApp(modules, module) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module does not belong to application"})
		return
	}

	ctx := auth.ContextWithAccessToken(r.Context(), accessToken)
	if h.search != nil {
		ctx = terms.ContextWithCollectHooks(ctx, h.gatewaySearchHook(), nil)
	}

	_, result, err := runner.RunWithResult(ctx, h.runtimeScope, exportpkg.Spec{
		Profile:     exportpkg.ProfileTerminology,
		Caller:      exportpkg.CallerUser,
		Application: application,
		Module:      module,
		Lang:        lang,
		Format:      "po",
	})
	if err != nil {
		writeTermsRPCError(w, err)
		return
	}

	filename := fmt.Sprintf("%s-%s.po", module, lang)
	w.Header().Set("Content-Type", "text/x-po; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if result.Truncated {
		w.Header().Set("X-Choysum-PO-Truncated", "1")
		h.logger().Warn("i18n po export truncated",
			"application", application, "module", module, "lang", lang, "limit", terms.ExportMaxItems)
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(result.POBytes); err != nil {
		h.logger().Error("failed to write PO export", "error", err, "application", application, "module", module, "lang", lang)
	}
}

func (h *handler) collectAllTerms(ctx context.Context, accessToken, app, lang string, modules []string) ([]termItem, bool, error) {
	if h.search != nil {
		ctx = terms.ContextWithCollectHooks(ctx, h.gatewaySearchHook(), nil)
	}
	items, truncated, err := terms.CollectAll(ctx, accessToken, app, lang, modules)
	if err != nil {
		return nil, false, err
	}
	out := make([]termItem, len(items))
	copy(out, items)
	return out, truncated, nil
}

func buildPOEntries(lang string, items []termItem) []po.Entry {
	typed := make([]terms.Item, len(items))
	copy(typed, items)
	return terms.BuildPOEntries(lang, typed)
}

func (h *handler) gatewaySearchHook() terms.SearchPageFunc {
	return func(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*terms.SearchResult, error) {
		if h.search == nil {
			return nil, fmt.Errorf("search hook is not configured")
		}
		got, err := h.search(ctx, accessToken, app, lang, modules, q, limit, offset)
		if err != nil || got == nil {
			return nil, err
		}
		out := &terms.SearchResult{
			Lang:   got.Lang,
			Total:  got.Total,
			Limit:  got.Limit,
			Offset: got.Offset,
			Items:  make([]terms.Item, len(got.Items)),
		}
		for i, item := range got.Items {
			out.Items[i] = terms.Item(item)
		}
		return out, nil
	}
}
