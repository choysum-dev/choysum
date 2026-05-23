// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type errorResponse struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func parseUploadIDFromPath(path string) (string, bool) {
	if !strings.HasPrefix(path, documentUploadPrefix) {
		return "", false
	}

	tail := strings.Trim(strings.TrimPrefix(path, documentUploadPrefix), "/")
	if tail == "" || strings.Contains(tail, "/") {
		return "", false
	}
	return tail, true
}

func parseBindingIDFromContentPath(path string) (string, bool) {
	if !strings.HasPrefix(path, documentBindingContentPrefix) {
		return "", false
	}

	tail := strings.Trim(strings.TrimPrefix(path, documentBindingContentPrefix), "/")
	parts := strings.Split(tail, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "content" {
		return "", false
	}
	return parts[0], true
}

func writeNotImplemented(w http.ResponseWriter, method string) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:    "document.SKELETON_NOT_IMPLEMENTED",
		Message: "Document gateway skeleton route is mounted but implementation is pending (" + method + ")",
	})
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	if status <= 0 {
		status = documentErrorStatus(code)
	}
	writeJSONErrorWithMetadata(w, status, code, message, nil)
}

func writeJSONErrorWithMetadata(w http.ResponseWriter, status int, code string, message string, metadata map[string]string) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:     code,
		Message:  message,
		Metadata: metadata,
	})
}

func writePermissionDenied(ctx context.Context, runtimeScope scope.Scope, w http.ResponseWriter, stage string, message string, metadata map[string]string) {
	normalizedStage := normalizePermissionDeniedStage(stage)
	md := map[string]string{"stage": normalizedStage}
	for k, v := range metadata {
		ks := strings.TrimSpace(k)
		vs := strings.TrimSpace(v)
		if ks == "" || vs == "" {
			continue
		}
		md[ks] = vs
	}
	md["reason"] = normalizePermissionDeniedReason(md["reason"])

	observePermissionDenied(ctx, runtimeScope, permissionDeniedObservationInput{
		stage:      normalizedStage,
		reason:     md["reason"],
		ownerModel: md["ownerModel"],
		fieldName:  md["fieldName"],
	})

	writeJSONErrorWithMetadata(w, http.StatusForbidden, "document.PERMISSION_DENIED", message, md)
}
