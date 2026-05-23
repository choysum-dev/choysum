// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"net/http"
	"strings"

	documentpayload "github.com/choysum-dev/choysum/internal/document/payload"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func writeDocumentRPCError(ctx context.Context, runtimeScope scope.Scope, w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, string(documentErrorInternal), "document service call failed")
		return
	}

	info := extractDocumentErrorInfo(st)
	code := mapDocumentErrorCodeFromRPC(st, info)
	message := strings.TrimSpace(st.Message())
	metadata := map[string]string{}

	if info != nil {
		if text := strings.TrimSpace(info.GetMessage()); text != "" {
			message = text
		}
		for k, v := range info.GetMetadata() {
			key := strings.TrimSpace(k)
			value := strings.TrimSpace(v)
			if key == "" || value == "" {
				continue
			}
			metadata[key] = value
		}
	}

	if message == "" {
		message = "document service call failed"
	}

	if code == string(documentErrorPermissionDenied) {
		writePermissionDenied(ctx, runtimeScope, w, metadata["stage"], message, metadata)
		return
	}

	writeJSONErrorWithMetadata(w, documentErrorStatus(code), code, message, metadata)
}

func writePayloadAdapterError(w http.ResponseWriter, err error, fallbackMessage string) {
	code := string(documentErrorInternal)
	message := strings.TrimSpace(fallbackMessage)
	if message == "" {
		message = "document payload adapter failed"
	}

	if payloadCode, ok := documentpayload.CodeOf(err); ok {
		code = mapPayloadAdapterCode(payloadCode)
		if code != string(documentErrorInternal) {
			if text := strings.TrimSpace(err.Error()); text != "" {
				message = text
			}
		}
	}

	writeJSONError(w, documentErrorStatus(code), code, message)
}

func mapPayloadAdapterCode(code documentpayload.Code) string {
	switch code {
	case documentpayload.CodeInvalidArgument:
		return string(documentErrorInvalidArgument)
	case documentpayload.CodeFailedPrecondition:
		return string(documentErrorFailedPrecondition)
	case documentpayload.CodeNotFound:
		return string(documentErrorNotFound)
	default:
		return string(documentErrorInternal)
	}
}

func extractDocumentErrorInfo(st *status.Status) *oerrors.ErrorInfo {
	if st == nil {
		return nil
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*oerrors.ErrorInfo)
		if ok && info != nil {
			return info
		}
	}
	return nil
}

func mapDocumentErrorCodeFromRPC(st *status.Status, info *oerrors.ErrorInfo) string {
	if code, ok := normalizeDocumentErrorCodeFromInfo(info); ok {
		return code
	}
	if st == nil {
		return string(documentErrorInternal)
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return string(documentErrorInvalidArgument)
	case codes.Unauthenticated:
		return string(documentErrorUnauthenticated)
	case codes.PermissionDenied:
		return string(documentErrorPermissionDenied)
	case codes.NotFound:
		return string(documentErrorNotFound)
	case codes.FailedPrecondition:
		return string(documentErrorFailedPrecondition)
	case codes.ResourceExhausted:
		return string(documentErrorUploadTooLarge)
	case codes.Unimplemented:
		return string(documentErrorSkeletonNotImplemented)
	default:
		return string(documentErrorInternal)
	}
}

func normalizeDocumentErrorCodeFromInfo(info *oerrors.ErrorInfo) (string, bool) {
	if info == nil {
		return "", false
	}

	rawCode := strings.TrimSpace(info.GetCode())
	if rawCode == "" {
		return "", false
	}

	upperCode := ""
	if strings.HasPrefix(strings.ToLower(rawCode), "document.") {
		upperCode = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(rawCode, "document.")))
	} else {
		domain := strings.TrimSpace(info.GetDomain())
		if !strings.EqualFold(domain, "document") {
			return "", false
		}
		upperCode = strings.ToUpper(rawCode)
	}

	if upperCode == "" {
		return "", false
	}
	return "document." + upperCode, true
}
