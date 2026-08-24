// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	documentpayload "github.com/choysum-dev/choysum/internal/document/payload"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReadBindingSourceBytes_Errors(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	identity := gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}}
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveResult = map[string]any{
		"attachmentBindingId": "bnd-1",
		"payloadReadTicket":   `{"attachmentBindingId":"bnd-1"}`,
		"sizeBytes":           int64(1),
	}
	ctx := auth.ContextWithIdentity(context.Background(), identity)
	ctx = auth.ContextWithAccessToken(ctx, "token")
	ctx = grpcclientWithDialer(newGatewayDocumentDialer(t, docFixture))

	prevPayloadOpen := payloadOpenFunc
	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{}, errors.New("open failed")
	}
	t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })

	if _, err := readBindingSourceBytes(ctx, runtimeScope, "bnd-1", identity); err == nil {
		t.Fatal("expected open error")
	}

	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{body: nil}, nil
	}
	if _, err := readBindingSourceBytes(ctx, runtimeScope, "bnd-1", identity); err == nil {
		t.Fatal("expected empty body error")
	}
}

func TestReadSourceRefBytes_NonNotFoundBindingError(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	identity := gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}}
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveErr = status.Error(codes.PermissionDenied, "denied")
	dialer := newGatewayDocumentDialer(t, docFixture)
	ctx := grpcclientWithDialer(dialer)

	if _, err := ReadSourceRefBytes(ctx, runtimeScope, "bnd-denied", identity); err == nil {
		t.Fatal("expected permission error")
	}
}

func TestReadAttachmentContentSourceBytes_MoreErrors(t *testing.T) {
	db := newAttachmentContentTestDB(t)
	runtimeScope := &gatewayTestScope{
		ctx:     context.Background(),
		session: &scope.Session{DB: db},
		cfg:     &config.Config{Document: config.NewDefaultDocumentConfig()},
	}
	identity := gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}}

	if _, err := readAttachmentContentSourceBytes(context.Background(), runtimeScope, "att-1", gatewayFakeIdentity{}); err == nil {
		t.Fatal("expected active company error")
	}
	if _, err := readAttachmentContentSourceBytes(context.Background(), nil, "att-1", identity); err == nil {
		t.Fatal("expected session error")
	}

	if err := db.Exec(`INSERT INTO document_attachment_content (id, stored_content_id, company_id, status) VALUES (?, ?, ?, ?)`,
		"att-empty-stored", "", "c1", "active").Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := readAttachmentContentSourceBytes(context.Background(), runtimeScope, "att-empty-stored", identity); err == nil {
		t.Fatal("expected missing stored content id")
	}
}

func TestReadAttachmentContentSourceBytes_OpenErrors(t *testing.T) {
	db := newAttachmentContentTestDB(t)
	if err := db.Exec(`INSERT INTO document_attachment_content (id, stored_content_id, company_id, status) VALUES (?, ?, ?, ?)`,
		"att-open-fail", "stored-missing", "c1", "active").Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	runtimeScope := &gatewayTestScope{
		ctx:     context.Background(),
		session: &scope.Session{DB: db},
		cfg:     &config.Config{Document: config.NewDefaultDocumentConfig()},
	}
	identity := gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}}
	if _, err := readAttachmentContentSourceBytes(context.Background(), runtimeScope, "att-open-fail", identity); err == nil {
		t.Fatal("expected adapter open error")
	}
}

func newAttachmentContentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE document_attachment_content (
		id TEXT PRIMARY KEY,
		stored_content_id TEXT NOT NULL,
		company_id TEXT NOT NULL,
		status TEXT NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func grpcclientWithDialer(dialer grpcclient.ServiceDialer) context.Context {
	ctx := auth.ContextWithIdentity(context.Background(), gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}})
	return grpcclient.ContextWithServiceDialer(ctx, dialer)
}

// Ensure payload not-found classification stays aligned with adapter errors.
func TestIsSourceNotFound_PayloadNotFoundWrap(t *testing.T) {
	if !isSourceNotFound(documentpayload.NotFoundWrap("missing", fmt.Errorf("record not found"))) {
		t.Fatal("expected wrapped not found")
	}
}

// Cover read-all path when body closes cleanly.
func TestReadBindingSourceBytes_ReadAllError(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	identity := gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}}
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveResult = map[string]any{
		"attachmentBindingId": "bnd-1",
		"payloadReadTicket":   `{"attachmentBindingId":"bnd-1"}`,
		"sizeBytes":           int64(1),
	}
	ctx := auth.ContextWithIdentity(context.Background(), identity)
	ctx = auth.ContextWithAccessToken(ctx, "token")
	ctx = grpcclientWithDialer(newGatewayDocumentDialer(t, docFixture))

	prevPayloadOpen := payloadOpenFunc
	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{body: io.NopCloser(&errReader{})}, nil
	}
	t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })
	if _, err := readBindingSourceBytes(ctx, runtimeScope, "bnd-1", identity); err == nil {
		t.Fatal("expected read error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReader) Close() error             { return nil }
