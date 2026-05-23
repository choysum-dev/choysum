// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cspmiddleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// CSPHandler applies Content Security Policy headers.
type CSPHandler struct {
	excludedPaths []string
	reportOnly    bool
	reportURI     string
	cspConfig     *config.CSPConfig
	hstsConfig    *config.HSTSConfig
	environment   string
	useHTTPS      bool
}

// NewCSPHandler creates a CSP handler.
func NewCSPHandler(runtimeScope scope.Scope) *CSPHandler {
	opts := runtimeOptionsFromScope(runtimeScope)

	return &CSPHandler{
		excludedPaths: opts.excludedPaths,
		reportOnly:    opts.reportOnly,
		reportURI:     opts.reportURI,
		cspConfig:     opts.cspConfig,
		hstsConfig:    opts.hstsConfig,
		environment:   opts.environment,
		useHTTPS:      opts.useHTTPS,
	}
}

// Handler returns the CSP middleware handler.
func (h *CSPHandler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip processing when CSP is disabled.
		if h.cspConfig == nil || !h.cspConfig.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check whether the path is excluded.
		for _, path := range h.excludedPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		isProd := h.environment == "production" || h.environment == "prod"

		// Choose the CSP directive set for the current environment.
		var directives config.CSPDirectives
		if isProd {
			directives = h.cspConfig.Production
		} else {
			directives = h.cspConfig.Development
		}

		// Build the CSP header value.
		cspValue := h.buildCSPHeader(directives, h.useHTTPS)

		// Append the report URI when configured.
		if h.reportURI != "" {
			cspValue += "; report-uri " + h.reportURI
		}

		// Set the CSP header.
		headerName := "Content-Security-Policy"
		if h.reportOnly {
			headerName = "Content-Security-Policy-Report-Only"
		}
		w.Header().Set(headerName, cspValue)

		// Set the HSTS header when HTTPS is enabled.
		if h.useHTTPS && h.hstsConfig != nil && h.hstsConfig.Enabled {
			hstsValue := "max-age=" + strconv.Itoa(h.hstsConfig.MaxAge)
			if h.hstsConfig.IncludeSubdomains {
				hstsValue += "; includeSubDomains"
			}
			if h.hstsConfig.Preload {
				hstsValue += "; preload"
			}
			w.Header().Set("Strict-Transport-Security", hstsValue)
		}

		// Add other security headers.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY") // Backward-compatible frame-ancestors support.
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		referrerPolicy := "origin-when-cross-origin"
		if isProd {
			referrerPolicy = "strict-origin-when-cross-origin"
		}
		w.Header().Set("Referrer-Policy", referrerPolicy)

		// Continue handling the request.
		next.ServeHTTP(w, r)
	})
}

// buildCSPHeader builds the CSP header value.
func (h *CSPHandler) buildCSPHeader(directives config.CSPDirectives, useHTTPS bool) string {
	var parts []string

	// Append configured CSP directives.
	if len(directives.DefaultSrc) > 0 {
		parts = append(parts, "default-src "+strings.Join(directives.DefaultSrc, " "))
	}
	if len(directives.ScriptSrc) > 0 {
		parts = append(parts, "script-src "+strings.Join(directives.ScriptSrc, " "))
	}
	if len(directives.StyleSrc) > 0 {
		parts = append(parts, "style-src "+strings.Join(directives.StyleSrc, " "))
	}
	if len(directives.ImgSrc) > 0 {
		parts = append(parts, "img-src "+strings.Join(directives.ImgSrc, " "))
	}
	if len(directives.ConnectSrc) > 0 {
		parts = append(parts, "connect-src "+strings.Join(directives.ConnectSrc, " "))
	}
	if len(directives.FontSrc) > 0 {
		parts = append(parts, "font-src "+strings.Join(directives.FontSrc, " "))
	}
	if len(directives.ObjectSrc) > 0 {
		parts = append(parts, "object-src "+strings.Join(directives.ObjectSrc, " "))
	}
	if len(directives.MediaSrc) > 0 {
		parts = append(parts, "media-src "+strings.Join(directives.MediaSrc, " "))
	}
	if len(directives.FrameSrc) > 0 {
		parts = append(parts, "frame-src "+strings.Join(directives.FrameSrc, " "))
	}
	if len(directives.WorkerSrc) > 0 {
		parts = append(parts, "worker-src "+strings.Join(directives.WorkerSrc, " "))
	}
	if len(directives.FrameAncestors) > 0 {
		parts = append(parts, "frame-ancestors "+strings.Join(directives.FrameAncestors, " "))
	}
	if len(directives.FormAction) > 0 {
		parts = append(parts, "form-action "+strings.Join(directives.FormAction, " "))
	}
	if len(directives.BaseURI) > 0 {
		parts = append(parts, "base-uri "+strings.Join(directives.BaseURI, " "))
	}
	if len(directives.ChildSrc) > 0 {
		parts = append(parts, "child-src "+strings.Join(directives.ChildSrc, " "))
	}
	if len(directives.ManifestSrc) > 0 {
		parts = append(parts, "manifest-src "+strings.Join(directives.ManifestSrc, " "))
	}

	// Add HTTPS-only directives for mixed-content hardening.
	if useHTTPS {
		parts = append(parts, "upgrade-insecure-requests")
		parts = append(parts, "block-all-mixed-content")
	}

	return strings.Join(parts, "; ")
}
