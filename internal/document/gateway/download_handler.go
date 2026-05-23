// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func resolveDownloadContent(ctx context.Context, bindingID string, principal map[string]any) (resolveDownloadContentResult, error) {
	req := map[string]any{
		"attachmentBindingId": strings.TrimSpace(bindingID),
		"principal":           principal,
	}

	resp, err := callDocumentAttachmentBindingMethod(ctx, documentResolveDownloadContentMethod, map[string]any{"req": req})
	if err != nil {
		return resolveDownloadContentResult{}, err
	}

	result := asRecord(resp["result"])
	if result == nil {
		return resolveDownloadContentResult{}, fmt.Errorf("resolve download response missing result payload")
	}

	payloadReadTicket := normalizeOptionalText(result["payloadReadTicket"])
	if payloadReadTicket == "" {
		return resolveDownloadContentResult{}, fmt.Errorf("resolve download response missing payloadReadTicket")
	}

	sizeBytes, ok := parseOptionalInt64(result["sizeBytes"])
	if !ok || sizeBytes < 0 {
		return resolveDownloadContentResult{}, fmt.Errorf("resolve download response has invalid sizeBytes")
	}

	mimeType := normalizeContentType(normalizeOptionalText(result["mimeType"]))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	downloadDisposition := strings.ToLower(strings.TrimSpace(normalizeOptionalText(result["downloadDisposition"])))
	if downloadDisposition != "inline" && downloadDisposition != "attachment" {
		downloadDisposition = "attachment"
	}

	fileName := strings.TrimSpace(normalizeOptionalText(result["fileName"]))
	if fileName == "" {
		fileName = fmt.Sprintf("attachment-%s%s", strings.TrimSpace(bindingID), mimeSuffix(mimeType))
	}

	return resolveDownloadContentResult{
		payloadReadTicket:   payloadReadTicket,
		mimeType:            mimeType,
		sizeBytes:           sizeBytes,
		checksumSHA256:      strings.ToLower(strings.TrimSpace(normalizeOptionalText(result["checksumSha256"]))),
		fileName:            fileName,
		downloadDisposition: downloadDisposition,
		etag:                strings.TrimSpace(normalizeOptionalText(result["etag"])),
	}, nil
}

func bindingContentHandler(w http.ResponseWriter, r *http.Request, runtimeScope scope.Scope) {
	if _, ok := parseBindingIDFromContentPath(r.URL.Path); !ok {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if runtimeScope == nil || !documentSessionAvailable(runtimeScope, r.Context()) {
		writeNotImplemented(w, "document.AttachmentBinding/DownloadContent")
		return
	}

	bindingID, _ := parseBindingIDFromContentPath(r.URL.Path)
	identity := auth.IdentityFromContext(r.Context())
	if identity == nil || !identity.IsValid() {
		writeJSONError(w, http.StatusUnauthorized, "document.UNAUTHENTICATED", "authentication is required")
		return
	}

	rpcCtx := outgoingContextForAuthRPC(r.Context())
	principal := buildUploadPrincipal(identity)
	resolveResult, err := resolveDownloadContent(rpcCtx, bindingID, principal)
	if err != nil {
		writeDocumentRPCError(r.Context(), runtimeScope, w, err)
		return
	}

	openedPayload, err := payloadOpenFunc(r.Context(), runtimeScope, payloadOpenRequest{
		bindingID:         bindingID,
		payloadReadTicket: resolveResult.payloadReadTicket,
	})
	if err != nil {
		writePayloadAdapterError(w, err, "failed to load attachment payload")
		return
	}
	if openedPayload.body == nil {
		writeJSONError(w, http.StatusInternalServerError, string(documentErrorInternal), "failed to load attachment payload")
		return
	}
	defer openedPayload.body.Close()

	w.Header().Set("Content-Type", resolveResult.mimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", buildContentDisposition(resolveResult.downloadDisposition, resolveResult.fileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", resolveResult.sizeBytes))
	if resolveResult.etag != "" {
		w.Header().Set("ETag", resolveResult.etag)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, openedPayload.body)
}

func documentSessionAvailable(runtimeScope scope.Scope, ctx context.Context) bool {
	if runtimeScope == nil {
		return false
	}
	_, ok := scope.SessionForScope(ctx, runtimeScope)
	return ok
}
