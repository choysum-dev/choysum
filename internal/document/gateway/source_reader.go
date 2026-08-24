// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	documentpayload "github.com/choysum-dev/choysum/internal/document/payload"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReadSourceRefBytes loads import source bytes for an attachment binding id or attachment content id.
func ReadSourceRefBytes(ctx context.Context, runtimeScope scope.Scope, sourceRef string, identity auth.Identity) ([]byte, error) {
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceRef == "" {
		return nil, fmt.Errorf("source ref is required")
	}
	if runtimeScope == nil {
		return nil, fmt.Errorf("runtime scope is required")
	}
	if identity == nil || !identity.IsValid() {
		return nil, fmt.Errorf("authentication is required")
	}

	if bytes, err := readBindingSourceBytes(ctx, runtimeScope, sourceRef, identity); err == nil {
		return bytes, nil
	} else if !isSourceNotFound(err) {
		return nil, err
	}

	return readAttachmentContentSourceBytes(ctx, runtimeScope, sourceRef, identity)
}

func readBindingSourceBytes(ctx context.Context, runtimeScope scope.Scope, bindingID string, identity auth.Identity) ([]byte, error) {
	rpcCtx := outgoingContextForAuthRPC(ctx)
	principal := buildUploadPrincipal(identity)
	resolveResult, err := resolveDownloadContent(rpcCtx, bindingID, principal)
	if err != nil {
		return nil, err
	}

	openedPayload, err := payloadOpenFunc(ctx, runtimeScope, payloadOpenRequest{
		bindingID:         bindingID,
		payloadReadTicket: resolveResult.payloadReadTicket,
	})
	if err != nil {
		return nil, err
	}
	if openedPayload.body == nil {
		return nil, fmt.Errorf("attachment payload body is empty")
	}
	defer openedPayload.body.Close()

	return io.ReadAll(openedPayload.body)
}

func readAttachmentContentSourceBytes(ctx context.Context, runtimeScope scope.Scope, attachmentContentID string, identity auth.Identity) ([]byte, error) {
	activeCompanyID := activeCompanyIDFromIdentity(identity)
	if activeCompanyID == "" {
		return nil, fmt.Errorf("active company is required")
	}

	session, ok := scope.SessionForScope(ctx, runtimeScope)
	if !ok || session == nil || session.DB == nil {
		return nil, fmt.Errorf("database session is required")
	}

	var row struct {
		StoredContentID string `gorm:"column:stored_content_id"`
		CompanyID       string `gorm:"column:company_id"`
		Status          string `gorm:"column:status"`
	}
	err := session.DB.WithContext(ctx).
		Table("document_attachment_content").
		Select("stored_content_id", "company_id", "status").
		Where("id = ?", attachmentContentID).
		First(&row).Error
	if err != nil {
		return nil, fmt.Errorf("attachment content not found: %w", err)
	}
	if strings.TrimSpace(row.Status) != "active" {
		return nil, fmt.Errorf("attachment content is not active")
	}
	if strings.TrimSpace(row.CompanyID) != activeCompanyID {
		return nil, fmt.Errorf("attachment content company mismatch")
	}

	storedContentID := strings.TrimSpace(row.StoredContentID)
	if storedContentID == "" {
		return nil, fmt.Errorf("attachment content missing stored content id")
	}

	ticket, err := jsonMarshal(map[string]string{
		"attachmentBindingId": attachmentContentID,
		"attachmentContentId": attachmentContentID,
		"storedContentId":     storedContentID,
	})
	if err != nil {
		return nil, fmt.Errorf("build payload read ticket: %w", err)
	}

	body, err := openAttachmentContentBody(ctx, runtimeScope, attachmentContentID, string(ticket))
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("stored content body is empty")
	}
	defer body.Close()

	return io.ReadAll(body)
}

var jsonMarshal = json.Marshal

var openAttachmentContentBody = defaultOpenAttachmentContentBody

func defaultOpenAttachmentContentBody(ctx context.Context, runtimeScope scope.Scope, attachmentContentID, ticket string) (io.ReadCloser, error) {
	adapter := documentpayload.NewAdapter(runtimeScope, documentpayload.Options{})
	opened, err := adapter.Open(ctx, documentpayload.OpenRequest{
		BindingID:         attachmentContentID,
		PayloadReadTicket: ticket,
	})
	if err != nil {
		return nil, err
	}
	return opened.Body, nil
}

func isSourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		return true
	}
	if code, ok := documentpayload.CodeOf(err); ok && code == documentpayload.CodeNotFound {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "record not found")
}
