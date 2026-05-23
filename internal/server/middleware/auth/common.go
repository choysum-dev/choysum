// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ------ Shared Types ------

// ErrorResponse is the auth error response payload.
type ErrorResponse struct {
	Error       string `json:"error"`
	Message     string `json:"message"`
	Code        string `json:"code,omitempty"`
	RequestPath string `json:"path,omitempty"`
}

// ResponseFormat defines the response format.
type ResponseFormat int

const (
	// JSONResponseFormat returns JSON-formatted error responses.
	JSONResponseFormat ResponseFormat = iota
	// PlainTextResponseFormat returns plain-text error responses.
	PlainTextResponseFormat
)

// ------ Path And Method Matching ------

// IsPathExcluded reports whether a path is excluded.
func IsPathExcluded(urlPath string, excludePaths []string, excludeRegex []*regexp.Regexp) bool {
	// Check exact-match paths.
	for _, p := range excludePaths {
		if p == "/" {
			if urlPath == "/" {
				return true
			}
			continue
		}
		if p == urlPath || strings.HasPrefix(urlPath, p) {
			return true
		}
	}

	// Check glob-style path patterns.
	for _, p := range excludePaths {
		if strings.Contains(p, "*") {
			matched, err := path.Match(p, urlPath)
			if err == nil && matched {
				return true
			}
		}
	}

	// Check regex matches.
	for _, re := range excludeRegex {
		if re.MatchString(urlPath) {
			return true
		}
	}

	return false
}

// IsMethodExcluded reports whether a method is excluded.
func IsMethodExcluded(methodName string, excludedMethods []string) bool {
	normalized := strings.TrimPrefix(methodName, "/")
	for _, method := range excludedMethods {
		method = strings.TrimSpace(method)
		if method == "" {
			continue
		}

		// Glob support (e.g. "grpc.channelz.v1.Channelz/*").
		if strings.ContainsAny(method, "*?[") {
			if matched, err := path.Match(method, normalized); err == nil && matched {
				return true
			}
			continue
		}

		if method == methodName || method == normalized ||
			strings.HasSuffix(methodName, "/"+method) || strings.HasSuffix(normalized, "/"+method) {
			return true
		}
	}
	return false
}

// CompileRegexPatterns compiles regex patterns.
func CompileRegexPatterns(patterns []string) []*regexp.Regexp {
	var result []*regexp.Regexp
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			result = append(result, re)
		}
	}
	return result
}

// ------ Token Handling ------

// ExtractBearerToken extracts a Bearer token from the auth header.
func ExtractBearerToken(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return authHeader
}

// ------ Error Handling ------

// FormatHTTPError formats an HTTP auth error response.
func FormatHTTPError(w http.ResponseWriter, r *http.Request, err error, format ResponseFormat, runtimeScope scope.Scope) {
	var (
		statusCode int
		message    string
		code       string
	)

	// Use autherrors for error inspection.
	if autherrors.IsAuthError(err) {
		statusCode = http.StatusUnauthorized

		// Try extracting the auth error payload.
		if choysumErr, ok := autherrors.AsAuthError(err); ok {
			message = choysumErr.Message
			code = choysumErr.Code
		} else {
			// Fall back to the raw error.
			message = err.Error()
			code = "AUTH_ERROR"
		}
	} else if oerrors.Is(err, autherrors.Domain, "") {
		// Handle other auth-domain errors.
		statusCode = http.StatusUnauthorized
		if choysumErr := oerrors.As(err); choysumErr != nil {
			message = choysumErr.Message
			code = choysumErr.Code
		} else {
			message = err.Error()
			code = "AUTH_ERROR"
		}
	} else {
		statusCode = http.StatusUnauthorized
		message = "authentication failed"
		code = "AUTH_FAILURE"
	}

	// Return the response according to the configured format.
	switch format {
	case JSONResponseFormat:
		resp := ErrorResponse{
			Error:       http.StatusText(statusCode),
			Message:     message,
			Code:        code,
			RequestPath: r.URL.Path,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			runtimeScope.Logger().Error("json response write failed", "error", err)
		}

	case PlainTextResponseFormat:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		_, _ = fmt.Fprintf(w, "%s: %s", code, message)

	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(ErrorResponse{
			Error:   http.StatusText(statusCode),
			Message: message,
			Code:    code,
		}); err != nil {
			runtimeScope.Logger().Error("json response write failed", "error", err)
		}
	}
}

// AuthErrorToGRPCStatus converts auth errors to gRPC status values.
func AuthErrorToGRPCStatus(err error) error {
	var message string
	var code codes.Code = codes.Unauthenticated

	// Use autherrors for error inspection.
	if autherrors.IsAuthError(err) {
		if choysumErr, ok := autherrors.AsAuthError(err); ok {
			message = fmt.Sprintf("%s: %s", choysumErr.Code, choysumErr.Message)
		} else {
			message = err.Error()
		}
	} else if oerrors.Is(err, autherrors.Domain, "") {
		if choysumErr := oerrors.As(err); choysumErr != nil {
			message = fmt.Sprintf("%s: %s", choysumErr.Code, choysumErr.Message)
		} else {
			message = err.Error()
		}
	} else {
		message = "authentication failed"
	}

	return status.Error(code, message)
}

// ------ Config Helpers ------

// TokenExtractorsFromConfig builds token extractor definitions from config.
func TokenExtractorsFromConfig(extractorNames []string, cookieName string, queryParamName string) []interface{} {
	// Return interface{} to avoid depending on HTTP-specific structures.
	// The HTTP package resolves these values into concrete implementations.

	result := make([]interface{}, 0, len(extractorNames))
	for _, name := range extractorNames {
		switch strings.ToLower(name) {
		case "header":
			result = append(result, map[string]string{"type": "header"})
		case "cookie":
			result = append(result, map[string]interface{}{
				"type": "cookie",
				"name": cookieName,
			})
		case "query":
			result = append(result, map[string]interface{}{
				"type": "query",
				"name": queryParamName,
			})
		}
	}

	return result
}

// GetConfigResponseFormat resolves the configured response format.
func GetConfigResponseFormat(formatName string) ResponseFormat {
	if strings.ToLower(formatName) == "text" {
		return PlainTextResponseFormat
	}
	return JSONResponseFormat
}
