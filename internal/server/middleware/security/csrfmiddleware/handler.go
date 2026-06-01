// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csrfmiddleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
)

// CSRFHandler applies CSRF protection.
type CSRFHandler struct {
	runtimeScope  scope.Scope
	cookieName    string
	headerName    string
	cookiePath    string
	cookieDomain  string
	sameSite      http.SameSite
	maxAge        int
	excludedPaths []string
}

// NewCSRFHandler creates a CSRF handler.
func NewCSRFHandler(runtimeScope scope.Scope) *CSRFHandler {
	opts := runtimeOptionsFromScope(runtimeScope)

	return &CSRFHandler{
		runtimeScope:  runtimeScope,
		cookieName:    opts.cookieName,
		headerName:    opts.headerName,
		cookiePath:    opts.cookiePath,
		cookieDomain:  opts.cookieDomain,
		sameSite:      opts.sameSite,
		maxAge:        opts.maxAge,
		excludedPaths: opts.excludedPaths,
	}
}

// Handler returns the CSRF middleware handler.
func (h *CSRFHandler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check whether the path is excluded.
		for _, path := range h.excludedPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// 2. Try setting the CSRF cookie for every request.
		h.ensureCSRFCookie(w, r)

		// 3. Safe methods do not require validation.
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		// 4. Validate the CSRF token for unsafe methods.
		if !h.validateCSRFToken(w, r) {
			return // Validation failed and the error response has already been sent.
		}

		// 5. Continue when validation succeeds.
		next.ServeHTTP(w, r)
	})
}

// ensureCSRFCookie makes sure the request has a CSRF cookie.
func (h *CSRFHandler) ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	// Check whether the cookie already exists.
	_, err := r.Cookie(h.cookieName)
	if err == http.ErrNoCookie {
		// Generate a new token.
		token, err := h.generateToken()
		if err != nil {
			h.runtimeScope.Logger().Error("csrf token generation failed", "error", err)
			return
		}

		// Set the cookie.
		cookie := &http.Cookie{
			Name:     h.cookieName,
			Value:    token,
			Path:     h.cookiePath,
			Domain:   h.cookieDomain,
			MaxAge:   h.maxAge,
			Secure:   true,
			HttpOnly: false, // JavaScript access is required.
			SameSite: h.sameSite,
		}
		http.SetCookie(w, cookie)

		// Log the cookie creation.
		h.runtimeScope.Logger().Debug("csrf cookie set", "token", token[:8]+"...")
	}
}

// validateCSRFToken validates the CSRF token.
func (h *CSRFHandler) validateCSRFToken(w http.ResponseWriter, r *http.Request) bool {
	// Read the token from the cookie.
	cookie, err := r.Cookie(h.cookieName)
	if err != nil {
		// Return an error when the cookie is missing.
		h.sendCSRFError(w, r, "CSRF protection requires cookies to be enabled in the browser")
		return false
	}

	// Read the token from the request header.
	requestToken := r.Header.Get(h.headerName)

	// Fall back to the form value when the header is absent.
	if requestToken == "" && r.Method == "POST" {
		if err := r.ParseForm(); err == nil {
			requestToken = r.FormValue("csrf_token")
		}
	}

	// Ensure the token is present.
	if requestToken == "" {
		h.sendCSRFError(w, r, "CSRF validation failed: missing CSRF token")
		return false
	}

	// Ensure the token matches the cookie value.
	if requestToken != cookie.Value {
		h.sendCSRFError(w, r, "CSRF validation failed: invalid CSRF token")
		return false
	}

	return true
}

// sendCSRFError sends a CSRF error response.
func (h *CSRFHandler) sendCSRFError(w http.ResponseWriter, r *http.Request, message string) {
	// Set the shared error header.
	w.Header().Set("X-CSRF-Error", "true")

	// Check whether the client expects JSON.
	isXHR := r.Header.Get("X-Requested-With") == "XMLHttpRequest"
	wantsJSON := strings.Contains(r.Header.Get("Accept"), "application/json")

	if isXHR || wantsJSON {
		// Return a JSON error.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)

		response := map[string]interface{}{
			"error":   "csrf_error",
			"message": message,
		}
		json.NewEncoder(w).Encode(response)
	} else {
		// Return an HTML error.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)

		// Render a simple HTML error page.
		html := `<!DOCTYPE html>
<html>
<head>
    <title>Security Validation Failed</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        h1 { color: #c00; }
        .box { border: 1px solid #ddd; padding: 20px; border-radius: 5px; background-color: #f9f9f9; }
        .help { margin-top: 20px; font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <h1>Security Validation Failed</h1>
    <div class="box">
        <p>` + message + `</p>
        <p>Please refresh the page or make sure cookies are enabled in your browser.</p>
    </div>
    <div class="help">
        <p>If the problem persists, contact your system administrator.</p>
    </div>
</body>
</html>`
		w.Write([]byte(html))
	}

	// Log the failed validation.
	h.runtimeScope.Logger().Warn("csrf validation failed",
		"path", r.URL.Path,
		"method", r.Method,
		"ip", r.RemoteAddr,
		"message", message)
}

// generateToken generates a random CSRF token.
func (h *CSRFHandler) generateToken() (string, error) {
	// Use 32 random bytes (256 bits).
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// isSafeMethod reports whether the HTTP method is safe.
func isSafeMethod(method string) bool {
	return method == "GET" || method == "HEAD" || method == "OPTIONS"
}
