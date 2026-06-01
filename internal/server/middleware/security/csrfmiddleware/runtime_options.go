// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csrfmiddleware

import (
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	cookieName    string
	headerName    string
	cookiePath    string
	cookieDomain  string
	sameSite      http.SameSite
	maxAge        int
	excludedPaths []string
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	csrfCfg := config.NewDefaultCSRFConfig()
	if runtimeScope != nil {
		if serverOpts, ok := scope.ServerRuntimeOptionsFromScope(runtimeScope); ok && serverOpts.CSRF != nil {
			csrfCfg = serverOpts.CSRF
		}
	}

	sameSite := http.SameSiteStrictMode
	switch strings.ToLower(strings.TrimSpace(csrfCfg.SameSite)) {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "none":
		sameSite = http.SameSiteNoneMode
	case "strict":
		sameSite = http.SameSiteStrictMode
	}

	return runtimeOptions{
		cookieName:    csrfCfg.CookieName,
		headerName:    csrfCfg.HeaderName,
		cookiePath:    csrfCfg.CookiePath,
		cookieDomain:  csrfCfg.CookieDomain,
		sameSite:      sameSite,
		maxAge:        csrfCfg.MaxAge,
		excludedPaths: csrfCfg.ExcludedPaths,
	}
}
