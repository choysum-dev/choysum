// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"bytes"
	"context"
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

func TestIsSourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "grpc not found", err: status.Error(codes.NotFound, "missing"), want: true},
		{name: "payload not found", err: documentpayload.NotFound("missing"), want: true},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "denied"), want: false},
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "attachment binding id is required"), want: false},
		{name: "payload invalid argument", err: documentpayload.InvalidArgument("attachment binding id is required"), want: false},
		{name: "record not found text", err: fmt.Errorf("record not found"), want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSourceNotFound(tc.err); got != tc.want {
				t.Fatalf("isSourceNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestReadSourceRefBytes_Validation(t *testing.T) {
	ctx := context.Background()
	runtimeScope := newGatewayTestScope(t)
	identity := gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}}

	if _, err := ReadSourceRefBytes(ctx, runtimeScope, "", identity); err == nil {
		t.Fatal("expected empty source ref error")
	}
	if _, err := ReadSourceRefBytes(ctx, nil, "src-1", identity); err == nil {
		t.Fatal("expected nil scope error")
	}
	if _, err := ReadSourceRefBytes(ctx, runtimeScope, "src-1", gatewayFakeIdentity{}); err == nil {
		t.Fatal("expected auth error")
	}
}

func TestReadSourceRefBytes_BindingPath(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	bindingID := "bnd_source_reader"
	body := []byte("name,code\nAcme,ACME\n")
	ticket := fmt.Sprintf(`{"attachmentBindingId":"%s","attachmentContentId":"att-1","storedContentId":"stored-1"}`, bindingID)

	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveResult = map[string]any{
		"attachmentBindingId": bindingID,
		"payloadReadTicket":   ticket,
		"mimeType":            "text/csv",
		"sizeBytes":           int64(len(body)),
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadOpen := payloadOpenFunc
	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{body: io.NopCloser(bytes.NewReader(body)), sizeBytes: int64(len(body))}, nil
	}
	t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })

	ctx := auth.ContextWithIdentity(context.Background(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)

	got, err := ReadSourceRefBytes(ctx, runtimeScope, bindingID, auth.IdentityFromContext(ctx))
	if err != nil {
		t.Fatalf("ReadSourceRefBytes: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("bytes = %q, want %q", string(got), string(body))
	}
}

func TestReadSourceRefBytes_AttachmentContentFallback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	body := []byte("csv,data\n1,2\n")
	if err := db.Exec(`CREATE TABLE document_attachment_content (
		id TEXT PRIMARY KEY,
		stored_content_id TEXT NOT NULL,
		company_id TEXT NOT NULL,
		status TEXT NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create attachment table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE document_stored_content (
		id TEXT PRIMARY KEY,
		provider TEXT,
		locator_json TEXT,
		blob_data BLOB,
		status TEXT,
		company_id TEXT
	)`).Error; err != nil {
		t.Fatalf("create stored content table: %v", err)
	}
	contentID := "att-content-1"
	if err := db.Exec(`INSERT INTO document_attachment_content (id, stored_content_id, company_id, status) VALUES (?, ?, ?, ?)`,
		contentID, "stored-content-1", "c1", "active").Error; err != nil {
		t.Fatalf("seed attachment row: %v", err)
	}
	if err := db.Exec(`INSERT INTO document_stored_content (id, provider, locator_json, blob_data, status, company_id) VALUES (?, ?, ?, ?, ?, ?)`,
		"stored-content-1", "db", "{}", body, "active", "c1").Error; err != nil {
		t.Fatalf("seed stored content: %v", err)
	}

	runtimeScope := &gatewayTestScope{
		ctx:     context.Background(),
		session: &scope.Session{DB: db},
		cfg:     &config.Config{Document: config.NewDefaultDocumentConfig()},
	}

	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveErr = status.Error(codes.NotFound, "binding missing")
	dialer := newGatewayDocumentDialer(t, docFixture)

	ctx := auth.ContextWithIdentity(context.Background(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)

	got, err := ReadSourceRefBytes(ctx, runtimeScope, contentID, auth.IdentityFromContext(ctx))
	if err != nil {
		t.Fatalf("ReadSourceRefBytes fallback: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("bytes = %q, want %q", string(got), string(body))
	}
}

func TestReadSourceRefBytes_AttachmentContentErrors(t *testing.T) {
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
	if err := db.Exec(`INSERT INTO document_attachment_content (id, stored_content_id, company_id, status) VALUES (?, ?, ?, ?)`,
		"att-inactive", "stored-1", "c1", "draft").Error; err != nil {
		t.Fatalf("seed inactive: %v", err)
	}
	if err := db.Exec(`INSERT INTO document_attachment_content (id, stored_content_id, company_id, status) VALUES (?, ?, ?, ?)`,
		"att-other-co", "stored-2", "c2", "active").Error; err != nil {
		t.Fatalf("seed other company: %v", err)
	}

	runtimeScope := &gatewayTestScope{ctx: context.Background(), session: &scope.Session{DB: db}}
	identity := gatewayFakeIdentity{userID: "u1", tokenID: "tok", metadata: map[string]any{"activeCompanyId": "c1"}}
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveErr = status.Error(codes.NotFound, "binding missing")
	dialer := newGatewayDocumentDialer(t, docFixture)
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), dialer)

	if _, err := ReadSourceRefBytes(ctx, runtimeScope, "att-inactive", identity); err == nil {
		t.Fatal("expected inactive error")
	}
	if _, err := ReadSourceRefBytes(ctx, runtimeScope, "att-other-co", identity); err == nil {
		t.Fatal("expected company mismatch error")
	}
	if _, err := ReadSourceRefBytes(ctx, runtimeScope, "att-missing", identity); err == nil {
		t.Fatal("expected missing content error")
	}
}
