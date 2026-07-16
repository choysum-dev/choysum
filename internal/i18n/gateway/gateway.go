// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package i18ngateway mounts host HTTP routes that aggregate {app}.I18n RPCs.
package i18ngateway

import (
	"net/http"

	"github.com/choysum-dev/choysum/pkg/scope"
)

const translationsPath = "/web/i18n/translations"

// RegisterHandlers mounts i18n host gateway routes on mux (D3: server mux, not WebHandlers).
func RegisterHandlers(mux *http.ServeMux, envs ...scope.Scope) {
	if mux == nil {
		return
	}
	var runtimeScope scope.Scope
	if len(envs) > 0 {
		runtimeScope = envs[0]
	}
	h := newHandler(runtimeScope)
	mux.HandleFunc(translationsPath, h.serveTranslations)
	mux.HandleFunc(termsPath, h.serveTerms)
}
