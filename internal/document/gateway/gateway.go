// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"io"
	"net/http"

	"github.com/choysum-dev/choysum/pkg/scope"
)

const (
	documentUploadPrefix         = "/_document/uploads/"
	documentBindingContentPrefix = "/_document/bindings/"

	documentAttachmentContentServiceName = "document.AttachmentContent"
	documentAttachmentBindingServiceName = "document.AttachmentBinding"
	documentAuthorizeUploadPutMethod     = "AuthorizeUploadPut"
	documentCommitUploadPutMethod        = "CommitUploadPut"
	documentResolveDownloadContentMethod = "ResolveDownloadContent"

	defaultMaxUploadBytes int64 = 20 * 1024 * 1024
)

var (
	payloadPutFunc  = defaultPayloadPut
	payloadOpenFunc = defaultPayloadOpen
)

type payloadPutRequest struct {
	uploadID           string
	payloadWriteTicket string
	contentType        string
	body               []byte
	checksumSHA256     string
}

type payloadPutReceipt struct {
	payloadID      string
	sizeBytes      int64
	checksumSHA256 string
	contentType    string
}

type payloadOpenRequest struct {
	bindingID         string
	payloadReadTicket string
}

type payloadOpenResult struct {
	body      io.ReadCloser
	sizeBytes int64
}

type authorizeUploadPutResult struct {
	maxUploadBytes     int64
	payloadWriteTicket string
}

type resolveDownloadContentResult struct {
	payloadReadTicket   string
	mimeType            string
	sizeBytes           int64
	checksumSHA256      string
	fileName            string
	downloadDisposition string
	etag                string
}

// RegisterSkeletonHandlers mounts document data-plane skeleton routes.
func RegisterSkeletonHandlers(mux *http.ServeMux, envs ...scope.Scope) {
	if mux == nil {
		return
	}

	var runtimeScope scope.Scope
	if len(envs) > 0 {
		runtimeScope = envs[0]
	}

	mux.HandleFunc(documentUploadPrefix, func(w http.ResponseWriter, r *http.Request) {
		uploadHandler(w, r, runtimeScope)
	})
	mux.HandleFunc(documentBindingContentPrefix, func(w http.ResponseWriter, r *http.Request) {
		bindingContentHandler(w, r, runtimeScope)
	})
}
