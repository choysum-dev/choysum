// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func uploadHandler(w http.ResponseWriter, r *http.Request, runtimeScope scope.Scope) {
	if _, ok := parseUploadIDFromPath(r.URL.Path); !ok {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if runtimeScope == nil {
		writeNotImplemented(w, "document.AttachmentContent/AuthorizeUploadPut")
		return
	}

	uploadID, _ := parseUploadIDFromPath(r.URL.Path)
	identity := auth.IdentityFromContext(r.Context())
	if identity == nil || !identity.IsValid() {
		writeJSONError(w, http.StatusUnauthorized, "document.UNAUTHENTICATED", "authentication is required")
		return
	}
	rpcCtx := outgoingContextForAuthRPC(r.Context())
	principal := buildUploadPrincipal(identity)
	contentType := normalizeContentType(r.Header.Get("Content-Type"))

	requestMeta := map[string]any{}
	if contentType != "" {
		requestMeta["contentType"] = contentType
	}
	if r.ContentLength >= 0 {
		requestMeta["contentLength"] = r.ContentLength
	}

	authorizeResult, err := authorizeUploadPut(rpcCtx, uploadID, principal, requestMeta)
	if err != nil {
		writeDocumentRPCError(r.Context(), runtimeScope, w, err)
		return
	}

	body, tooLarge, err := readRequestBody(r, authorizeResult.maxUploadBytes)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "document.INVALID_UPLOAD_BODY", "failed to read upload body")
		return
	}
	if tooLarge {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "document.UPLOAD_TOO_LARGE", "upload body exceeds maxUploadBytes")
		return
	}

	checksum := checksumSHA256Hex(body)
	receipt, err := payloadPutFunc(r.Context(), runtimeScope, payloadPutRequest{
		uploadID:           uploadID,
		payloadWriteTicket: authorizeResult.payloadWriteTicket,
		contentType:        contentType,
		body:               body,
		checksumSHA256:     checksum,
	})
	if err != nil {
		writePayloadAdapterError(w, err, "failed to persist uploaded payload")
		return
	}

	receipt, err = normalizePayloadPutReceipt(receipt, len(body), checksum, contentType)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "document.INTERNAL", "failed to persist uploaded payload")
		return
	}

	if err := commitUploadPut(rpcCtx, uploadID, principal, receipt); err != nil {
		writeDocumentRPCError(r.Context(), runtimeScope, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func buildUploadPrincipal(identity auth.Identity) map[string]any {
	activeCompanyID := activeCompanyIDFromIdentity(identity)
	principal := map[string]any{
		"userId":          strings.TrimSpace(identity.GetUserID()),
		"activeCompanyId": activeCompanyID,
	}

	companyIDs := enabledCompanyIDsFromIdentity(identity, activeCompanyID)
	if len(companyIDs) > 0 {
		enabled := make([]any, 0, len(companyIDs))
		for _, companyID := range companyIDs {
			enabled = append(enabled, companyID)
		}
		principal["enabledCompanyIds"] = enabled
	}

	return principal
}

func authorizeUploadPut(ctx context.Context, uploadID string, principal map[string]any, requestMeta map[string]any) (authorizeUploadPutResult, error) {
	req := map[string]any{
		"uploadId":  strings.TrimSpace(uploadID),
		"principal": principal,
	}
	if len(requestMeta) > 0 {
		req["requestMeta"] = requestMeta
	}

	resp, err := callDocumentAttachmentContentMethod(ctx, documentAuthorizeUploadPutMethod, map[string]any{"req": req})
	if err != nil {
		return authorizeUploadPutResult{}, err
	}

	result := asRecord(resp["result"])
	if result == nil {
		return authorizeUploadPutResult{}, fmt.Errorf("authorize upload response missing result payload")
	}

	payloadWriteTicket := normalizeOptionalText(result["payloadWriteTicket"])
	if payloadWriteTicket == "" {
		return authorizeUploadPutResult{}, fmt.Errorf("authorize upload response missing payloadWriteTicket")
	}

	maxUploadBytes := defaultMaxUploadBytes
	if parsed, ok := parseOptionalInt64(result["maxUploadBytes"]); ok && parsed > 0 {
		maxUploadBytes = parsed
	}

	return authorizeUploadPutResult{
		maxUploadBytes:     maxUploadBytes,
		payloadWriteTicket: payloadWriteTicket,
	}, nil
}

func commitUploadPut(ctx context.Context, uploadID string, principal map[string]any, receipt payloadPutReceipt) error {
	payloadReceipt := map[string]any{
		"payloadId":      receipt.payloadID,
		"sizeBytes":      receipt.sizeBytes,
		"checksumSha256": receipt.checksumSHA256,
	}
	if receipt.contentType != "" {
		payloadReceipt["contentType"] = receipt.contentType
	}

	req := map[string]any{
		"uploadId":       strings.TrimSpace(uploadID),
		"principal":      principal,
		"payloadReceipt": payloadReceipt,
	}

	resp, err := callDocumentAttachmentContentMethod(ctx, documentCommitUploadPutMethod, map[string]any{"req": req})
	if err != nil {
		return err
	}

	result := asRecord(resp["result"])
	if result == nil {
		return fmt.Errorf("commit upload response missing result payload")
	}

	_ = normalizeOptionalText(result["attachmentUploadSessionStatus"])
	return nil
}
