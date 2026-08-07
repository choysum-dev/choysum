// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/po"
)

const poPath = "/web/i18n/po"

// Tunable for tests; production defaults keep PO downloads bounded.
var (
	// Keep in sync with typical ORM Search page sizes so export pages stay bounded.
	poExportPageSize = 500
	poExportMaxItems = 10000
)

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
	modules = []string{module}

	items, truncated, err := h.collectAllTerms(r.Context(), accessToken, application, lang, modules)
	if err != nil {
		writeTermsRPCError(w, err)
		return
	}

	entries := buildPOEntries(lang, items)
	filename := fmt.Sprintf("%s-%s.po", module, lang)
	w.Header().Set("Content-Type", "text/x-po; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if truncated {
		w.Header().Set("X-Choysum-PO-Truncated", "1")
		h.logger().Warn("i18n po export truncated",
			"application", application, "module", module, "lang", lang, "limit", poExportMaxItems, "exported", len(items))
	}
	w.WriteHeader(http.StatusOK)
	if err := po.Write(w, entries); err != nil {
		// Headers already sent; cannot change status — log for ops visibility.
		h.logger().Error("failed to write PO export", "error", err, "application", application, "module", module, "lang", lang)
		return
	}
}

func (h *handler) collectAllTerms(ctx context.Context, accessToken, app, lang string, modules []string) ([]termItem, bool, error) {
	if poExportMaxItems <= 0 {
		return nil, true, nil
	}

	var (
		total int64
		err   error
	)
	if h.search != nil {
		// Injected search hooks historically return Total per page; ask once with limit=1.
		probe, err := h.search(ctx, accessToken, app, lang, modules, "", 1, 0)
		if err != nil {
			return nil, false, err
		}
		if probe != nil {
			total = probe.Total
		}
	} else {
		total, err = countAppTerms(ctx, accessToken, app, lang, modules, "")
		if err != nil {
			return nil, false, err
		}
	}

	var all []termItem
	offset := 0
	truncated := false
	for {
		remaining := poExportMaxItems - len(all)
		page := poExportPageSize
		if page > remaining {
			page = remaining
		}
		var result *searchTermsResult
		if h.search != nil {
			result, err = h.search(ctx, accessToken, app, lang, modules, "", page, offset)
		} else {
			result, err = searchAppTermsPage(ctx, accessToken, app, lang, modules, "", total, page, offset)
		}
		if err != nil {
			return nil, false, err
		}
		if result == nil || len(result.Items) == 0 {
			break
		}
		all = append(all, result.Items...)
		offset += len(result.Items)
		if len(all) >= poExportMaxItems {
			if total > int64(len(all)) {
				truncated = true
			}
			break
		}
		// Hooks may omit Total on the probe; adopt a later page total when present.
		if total <= 0 && result.Total > 0 {
			total = result.Total
		}
		// When total is still unknown, keep paging until a short page.
		if len(result.Items) < page {
			break
		}
		if total > 0 && int64(offset) >= total {
			break
		}
	}
	return all, truncated, nil
}

func buildPOEntries(lang string, items []termItem) []po.Entry {
	entries := make([]po.Entry, 0, len(items)+1)
	entries = append(entries, po.Entry{
		Msgid: "",
		Msgstr: "Content-Type: text/plain; charset=UTF-8\n" +
			"Content-Transfer-Encoding: 8bit\n" +
			"Language: " + lang + "\n" +
			"X-Generator: choysum-i18n-gateway\n",
	})
	for _, item := range items {
		e := po.Entry{
			Msgctxt: item.Scope,
			Msgid:   item.Src,
			Msgstr:  item.Value,
		}
		if item.Module != "" {
			e.ExtractedComments = append(e.ExtractedComments, "module: "+item.Module)
		}
		if item.Source != "" {
			e.TranslatorComments = append(e.TranslatorComments, "source: "+item.Source)
		}
		if strings.EqualFold(item.Status, "fuzzy") {
			e.Flags = append(e.Flags, "fuzzy")
		}
		entries = append(entries, e)
	}
	po.SortEntries(entries)
	// Keep header first after sort (SortEntries treats empty msgid as normal).
	return moveHeaderFirst(entries)
}

func moveHeaderFirst(entries []po.Entry) []po.Entry {
	var header []po.Entry
	var rest []po.Entry
	for _, e := range entries {
		if po.IsHeader(e) {
			header = append(header, e)
			continue
		}
		rest = append(rest, e)
	}
	return append(header, rest...)
}
