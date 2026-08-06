// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Shared helpers for PO export (TranslationTerm Search dial). Terms HTTP routes were removed in P3.

func (h *handler) searchApp(ctx context.Context, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
	if h.search != nil {
		return h.search(ctx, accessToken, app, lang, modules, q, limit, offset)
	}
	return fetchAppSearchTerms(ctx, h.runtimeScope, accessToken, app, lang, modules, q, limit, offset)
}

func writeTermsRPCError(w http.ResponseWriter, err error) {
	code := status.Code(err)
	switch code {
	case codes.PermissionDenied:
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "permission denied"})
	case codes.Unauthenticated:
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication is required"})
	case codes.InvalidArgument:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
	}
}
