// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"fmt"
	"strings"

	documentpayload "github.com/choysum-dev/choysum/internal/document/payload"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func defaultPayloadPut(ctx context.Context, runtimeScope scope.Scope, req payloadPutRequest) (payloadPutReceipt, error) {
	adapter := documentpayload.NewAdapter(runtimeScope, documentpayload.Options{})

	receipt, err := adapter.Put(ctx, documentpayload.PutRequest{
		UploadID:           req.uploadID,
		PayloadWriteTicket: req.payloadWriteTicket,
		ContentType:        req.contentType,
		Body:               req.body,
		ChecksumSHA256:     req.checksumSHA256,
	})
	if err != nil {
		return payloadPutReceipt{}, err
	}

	return payloadPutReceipt{
		payloadID:      strings.TrimSpace(receipt.PayloadID),
		sizeBytes:      receipt.SizeBytes,
		checksumSHA256: strings.TrimSpace(receipt.ChecksumSHA256),
		contentType:    strings.TrimSpace(receipt.ContentType),
	}, nil
}

func defaultPayloadOpen(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
	adapter := documentpayload.NewAdapter(runtimeScope, documentpayload.Options{})

	openedPayload, err := adapter.Open(ctx, documentpayload.OpenRequest{
		BindingID:         req.bindingID,
		PayloadReadTicket: req.payloadReadTicket,
	})
	if err != nil {
		return payloadOpenResult{}, err
	}

	return payloadOpenResult{
		body:      openedPayload.Body,
		sizeBytes: openedPayload.SizeBytes,
	}, nil
}

func normalizePayloadPutReceipt(receipt payloadPutReceipt, bodyLen int, fallbackChecksum string, fallbackContentType string) (payloadPutReceipt, error) {
	receipt.payloadID = strings.TrimSpace(receipt.payloadID)
	if receipt.payloadID == "" {
		return payloadPutReceipt{}, fmt.Errorf("payload receipt payloadId is required")
	}

	if receipt.sizeBytes < 0 {
		return payloadPutReceipt{}, fmt.Errorf("payload receipt sizeBytes must be non-negative")
	}
	if receipt.sizeBytes == 0 && bodyLen > 0 {
		receipt.sizeBytes = int64(bodyLen)
	}

	receipt.checksumSHA256 = strings.ToLower(strings.TrimSpace(receipt.checksumSHA256))
	if receipt.checksumSHA256 == "" {
		receipt.checksumSHA256 = strings.ToLower(strings.TrimSpace(fallbackChecksum))
	}
	if receipt.checksumSHA256 == "" {
		return payloadPutReceipt{}, fmt.Errorf("payload receipt checksumSha256 is required")
	}

	if receipt.contentType == "" {
		receipt.contentType = normalizeContentType(fallbackContentType)
	} else {
		receipt.contentType = normalizeContentType(receipt.contentType)
	}

	return receipt, nil
}
