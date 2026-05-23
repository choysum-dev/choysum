// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package payload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubStoredContentDriverFactory struct {
	newDriverFunc func(provider string, att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error)
}

func (f stubStoredContentDriverFactory) NewDriver(provider string, att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
	return f.newDriverFunc(provider, att)
}

type stubStoredContentDriver struct {
	providerName string
	putFunc      func(ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error)
	openFunc     func(ctx context.Context, record pkgstorage.StoredContentRecord) ([]byte, error)
	deleteFunc   func(ctx context.Context, record pkgstorage.StoredContentRecord) error
}

func (d stubStoredContentDriver) Provider() string { return d.providerName }

func (d stubStoredContentDriver) Put(ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
	if d.putFunc != nil {
		return d.putFunc(ctx, input)
	}
	return pkgstorage.PayloadMutation{}, nil
}

func (d stubStoredContentDriver) Open(ctx context.Context, record pkgstorage.StoredContentRecord) ([]byte, error) {
	if d.openFunc != nil {
		return d.openFunc(ctx, record)
	}
	return nil, nil
}

func (d stubStoredContentDriver) Delete(ctx context.Context, record pkgstorage.StoredContentRecord) error {
	if d.deleteFunc != nil {
		return d.deleteFunc(ctx, record)
	}
	return nil
}

type payloadTestS3Locator struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

func buildPayloadTestUploadStagingKey(uploadID string, checksum string) string {
	id := strings.TrimSpace(uploadID)
	if id == "" {
		id = "upload"
	}

	normalizedChecksum := strings.ToLower(strings.TrimSpace(checksum))
	if len(normalizedChecksum) > 12 {
		normalizedChecksum = normalizedChecksum[:12]
	}
	if normalizedChecksum == "" {
		return fmt.Sprintf("staging/%s", id)
	}

	return fmt.Sprintf("staging/%s/%s", id, normalizedChecksum)
}

func parsePayloadTestS3Locator(locatorJSON string) (payloadTestS3Locator, error) {
	text := strings.TrimSpace(locatorJSON)
	if text == "" {
		return payloadTestS3Locator{}, fmt.Errorf("stored content locator json is required for s3")
	}

	var locator payloadTestS3Locator
	if err := json.Unmarshal([]byte(text), &locator); err != nil {
		return payloadTestS3Locator{}, fmt.Errorf("parse s3 locator json: %w", err)
	}
	locator.Bucket = strings.TrimSpace(locator.Bucket)
	locator.Key = strings.TrimSpace(locator.Key)
	if locator.Bucket == "" || locator.Key == "" {
		return payloadTestS3Locator{}, fmt.Errorf("s3 locator json requires bucket and key")
	}
	return locator, nil
}

type payloadTestScope struct {
	ctx     context.Context
	session *scope.Session
	cfg     *config.Config
}

func (e *payloadTestScope) Run(fn func(runtimeScope scope.Scope) error) error {
	return fn(e)
}

func (e *payloadTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *payloadTestScope) Session() *scope.Session {
	return e.session
}

func (e *payloadTestScope) WithContext(ctx context.Context) scope.Scope {
	return &payloadTestScope{ctx: ctx, session: e.session, cfg: e.cfg}
}

func (e *payloadTestScope) Context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

func (e *payloadTestScope) Logger() *slog.Logger {
	return slog.Default()
}

func (e *payloadTestScope) Config() *config.Config {
	if e.cfg == nil {
		e.cfg = &config.Config{Document: config.NewDefaultDocumentConfig()}
	}
	return e.cfg
}

func (e *payloadTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newPayloadTestScope(t *testing.T, backend string, db *gorm.DB) *payloadTestScope {
	t.Helper()
	cfg := &config.Config{Document: config.NewDefaultDocumentConfig()}
	if cfg.Document != nil && cfg.Document.Attachment != nil {
		cfg.Document.Attachment.Backend = backend
		if cfg.Document.Attachment.S3 != nil {
			cfg.Document.Attachment.S3.Bucket = "choysum-attachments-test"
			cfg.Document.Attachment.S3.AccessKey = "ak"
			cfg.Document.Attachment.S3.SecretKey = "sk"
			cfg.Document.Attachment.S3.Endpoint = "127.0.0.1:9000"
			cfg.Document.Attachment.S3.UseTLS = false
		}
	}
	return &payloadTestScope{
		ctx:     context.Background(),
		session: &scope.Session{DB: db},
		cfg:     cfg,
	}
}

func newPayloadTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	err = db.Exec(`
		CREATE TABLE document_stored_content (
			id TEXT PRIMARY KEY,
			provider TEXT,
			locator_json TEXT,
			blob_data BLOB,
			status TEXT,
			company_id TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create document_stored_content: %v", err)
	}
	return db
}

func TestAdapterPutAndOpenDB(t *testing.T) {
	db := newPayloadTestDB(t)
	runtimeScope := newPayloadTestScope(t, "db", db)
	adapter := NewAdapter(runtimeScope, Options{})

	body := []byte("hello-db")
	putReq := PutRequest{
		UploadID:           "up_db_adapter_001",
		PayloadWriteTicket: `{"uploadId":"up_db_adapter_001","companyId":"cmp-001","activeCompanyId":"cmp-001","userId":"usr-001","status":"prepared"}`,
		ContentType:        "text/plain",
		ChecksumSHA256:     "abc123",
		Body:               body,
	}

	receipt, err := adapter.Put(context.Background(), putReq)
	if err != nil {
		t.Fatalf("adapter.Put() error = %v", err)
	}
	if !strings.HasPrefix(receipt.PayloadID, "sc:") {
		t.Fatalf("receipt.PayloadID = %q, want sc:*", receipt.PayloadID)
	}
	if receipt.SizeBytes != int64(len(body)) {
		t.Fatalf("receipt.SizeBytes = %d, want %d", receipt.SizeBytes, len(body))
	}

	storedContentID := strings.TrimSpace(strings.TrimPrefix(receipt.PayloadID, "sc:"))
	if storedContentID == "" {
		t.Fatal("stored content id should not be empty")
	}

	var row struct {
		Provider  string `gorm:"column:provider"`
		Status    string `gorm:"column:status"`
		CompanyID string `gorm:"column:company_id"`
		BlobData  []byte `gorm:"column:blob_data"`
	}
	err = db.Table(documentStoredContentTable).
		Select("provider, status, company_id, blob_data").
		Where("id = ?", storedContentID).
		Take(&row).Error
	if err != nil {
		t.Fatalf("query stored content row: %v", err)
	}
	if row.Provider != "db" {
		t.Fatalf("provider = %q, want db", row.Provider)
	}
	if row.Status != "active" {
		t.Fatalf("status = %q, want active", row.Status)
	}
	if row.CompanyID != "cmp-001" {
		t.Fatalf("company_id = %q, want cmp-001", row.CompanyID)
	}
	if !bytes.Equal(row.BlobData, body) {
		t.Fatalf("blob_data = %q, want %q", string(row.BlobData), string(body))
	}

	openReq := OpenRequest{
		BindingID:         "bnd-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-001","storedContentId":"%s"}`, storedContentID),
	}
	opened, err := adapter.Open(context.Background(), openReq)
	if err != nil {
		t.Fatalf("adapter.Open() error = %v", err)
	}
	defer opened.Body.Close()

	openedBody, err := io.ReadAll(opened.Body)
	if err != nil {
		t.Fatalf("read opened body: %v", err)
	}
	if !bytes.Equal(openedBody, body) {
		t.Fatalf("opened body = %q, want %q", string(openedBody), string(body))
	}
	if opened.SizeBytes != int64(len(body)) {
		t.Fatalf("opened.SizeBytes = %d, want %d", opened.SizeBytes, len(body))
	}
}

func TestAdapterUsesContextScopeBeforeRuntimeScopeSession(t *testing.T) {
	baseDB := newPayloadTestDB(t)
	ctxDB := newPayloadTestDB(t)
	runtimeScope := newPayloadTestScope(t, "db", baseDB)
	adapter := NewAdapter(runtimeScope, Options{})
	ctx := scope.ContextWithScope(context.Background(), newPayloadTestScope(t, "db", ctxDB))

	body := []byte("ctx-db")
	receipt, err := adapter.Put(ctx, PutRequest{
		UploadID:           "up_db_ctx_001",
		PayloadWriteTicket: `{"uploadId":"up_db_ctx_001","companyId":"cmp-ctx","activeCompanyId":"cmp-ctx","userId":"usr-ctx","status":"prepared"}`,
		ContentType:        "text/plain",
		ChecksumSHA256:     "ctxdb123",
		Body:               body,
	})
	if err != nil {
		t.Fatalf("adapter.Put() error = %v", err)
	}

	storedContentID := strings.TrimSpace(strings.TrimPrefix(receipt.PayloadID, "sc:"))
	if storedContentID == "" {
		t.Fatal("stored content id should not be empty")
	}

	var ctxCount int64
	err = ctxDB.Table(documentStoredContentTable).Where("id = ?", storedContentID).Count(&ctxCount).Error
	if err != nil {
		t.Fatalf("count ctx stored content row: %v", err)
	}
	if ctxCount != 1 {
		t.Fatalf("ctx row count = %d, want 1", ctxCount)
	}

	var baseCount int64
	err = baseDB.Table(documentStoredContentTable).Where("id = ?", storedContentID).Count(&baseCount).Error
	if err != nil {
		t.Fatalf("count base stored content row: %v", err)
	}
	if baseCount != 0 {
		t.Fatalf("base row count = %d, want 0", baseCount)
	}

	opened, err := adapter.Open(ctx, OpenRequest{
		BindingID:         "bnd-ctx-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-ctx-001","storedContentId":"%s"}`, storedContentID),
	})
	if err != nil {
		t.Fatalf("adapter.Open() error = %v", err)
	}
	defer opened.Body.Close()

	openedBody, err := io.ReadAll(opened.Body)
	if err != nil {
		t.Fatalf("read opened body: %v", err)
	}
	if !bytes.Equal(openedBody, body) {
		t.Fatalf("opened body = %q, want %q", string(openedBody), string(body))
	}

	err = adapter.Delete(ctx, DeleteRequest{
		BindingID:         "bnd-ctx-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-ctx-001","storedContentId":"%s"}`, storedContentID),
	})
	if err != nil {
		t.Fatalf("adapter.Delete() error = %v", err)
	}

	var deletedRow struct {
		Status string `gorm:"column:status"`
	}
	err = ctxDB.Table(documentStoredContentTable).
		Select("status").
		Where("id = ?", storedContentID).
		Take(&deletedRow).Error
	if err != nil {
		t.Fatalf("query deleted ctx stored content row: %v", err)
	}
	if deletedRow.Status != "deleted" {
		t.Fatalf("status after delete = %q, want deleted", deletedRow.Status)
	}

	err = baseDB.Table(documentStoredContentTable).
		Select("status").
		Where("id = ?", storedContentID).
		Take(&deletedRow).Error
	if err == nil {
		t.Fatal("expected base scope session to remain untouched")
	}
}

func TestAdapterPutAndOpenS3(t *testing.T) {
	db := newPayloadTestDB(t)
	runtimeScope := newPayloadTestScope(t, "s3", db)

	putCalls := 0
	getCalls := 0
	var putBucket string
	var putKey string
	var putLocatorJSON string
	var putBody []byte
	var putContentType string

	adapter := NewAdapter(runtimeScope, Options{
		DriverFactory: stubStoredContentDriverFactory{newDriverFunc: func(provider string, att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
			if provider != "s3" {
				return nil, fmt.Errorf("unexpected provider %q", provider)
			}
			return stubStoredContentDriver{
				providerName: "s3",
				putFunc: func(ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
					putCalls += 1
					putBucket = "choysum-attachments-test"
					putKey = buildPayloadTestUploadStagingKey(input.UploadID, input.ChecksumSHA256)
					putBody = append([]byte(nil), input.Body...)
					putContentType = input.ContentType
					locatorJSON, err := json.Marshal(payloadTestS3Locator{Bucket: putBucket, Key: putKey})
					if err != nil {
						return pkgstorage.PayloadMutation{}, err
					}
					putLocatorJSON = string(locatorJSON)
					return pkgstorage.PayloadMutation{LocatorJSON: putLocatorJSON}, nil
				},
				openFunc: func(ctx context.Context, record pkgstorage.StoredContentRecord) ([]byte, error) {
					getCalls += 1
					return []byte("hello-s3"), nil
				},
			}, nil
		}},
	})

	body := []byte("hello-s3")
	receipt, err := adapter.Put(context.Background(), PutRequest{
		UploadID:           "up_s3_adapter_001",
		PayloadWriteTicket: `{"uploadId":"up_s3_adapter_001","companyId":"cmp-001","activeCompanyId":"cmp-001","userId":"usr-001","status":"prepared"}`,
		ContentType:        "application/octet-stream",
		ChecksumSHA256:     "a1b2c3d4e5f6",
		Body:               body,
	})
	if err != nil {
		t.Fatalf("adapter.Put() error = %v", err)
	}
	if putCalls != 1 {
		t.Fatalf("put s3 calls = %d, want 1", putCalls)
	}
	if putBucket != "choysum-attachments-test" {
		t.Fatalf("put bucket = %q, want choysum-attachments-test", putBucket)
	}
	if putContentType != "application/octet-stream" {
		t.Fatalf("put contentType = %q, want application/octet-stream", putContentType)
	}
	if !bytes.Equal(putBody, body) {
		t.Fatalf("put body = %q, want %q", string(putBody), string(body))
	}
	if !strings.HasPrefix(putKey, "staging/up_s3_adapter_001/") {
		t.Fatalf("put key = %q, want staging/up_s3_adapter_001/*", putKey)
	}
	if !strings.HasPrefix(receipt.PayloadID, "sc:") {
		t.Fatalf("receipt.PayloadID = %q, want sc:*", receipt.PayloadID)
	}

	storedContentID := strings.TrimSpace(strings.TrimPrefix(receipt.PayloadID, "sc:"))
	if storedContentID == "" {
		t.Fatal("stored content id should not be empty")
	}

	var row struct {
		Provider    string `gorm:"column:provider"`
		LocatorJSON string `gorm:"column:locator_json"`
		Status      string `gorm:"column:status"`
	}
	err = db.Table(documentStoredContentTable).
		Select("provider, locator_json, status").
		Where("id = ?", storedContentID).
		Take(&row).Error
	if err != nil {
		t.Fatalf("query stored content row: %v", err)
	}
	if row.Provider != "s3" {
		t.Fatalf("provider = %q, want s3", row.Provider)
	}
	if row.LocatorJSON != putLocatorJSON {
		t.Fatalf("locator_json = %q, want %q", row.LocatorJSON, putLocatorJSON)
	}
	if row.Status != "active" {
		t.Fatalf("status = %q, want active", row.Status)
	}

	opened, err := adapter.Open(context.Background(), OpenRequest{
		BindingID:         "bnd-s3-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-s3-001","storedContentId":"%s"}`, storedContentID),
	})
	if err != nil {
		t.Fatalf("adapter.Open() error = %v", err)
	}
	defer opened.Body.Close()

	openedBody, err := io.ReadAll(opened.Body)
	if err != nil {
		t.Fatalf("read opened body: %v", err)
	}
	if string(openedBody) != "hello-s3" {
		t.Fatalf("opened body = %q, want hello-s3", string(openedBody))
	}
	if opened.SizeBytes != int64(len(openedBody)) {
		t.Fatalf("opened.SizeBytes = %d, want %d", opened.SizeBytes, len(openedBody))
	}
	if getCalls != 1 {
		t.Fatalf("get s3 calls = %d, want 1", getCalls)
	}
}

func TestAdapterRoutesRegisteredCustomProviderThroughFactory(t *testing.T) {
	db := newPayloadTestDB(t)
	runtimeScope := newPayloadTestScope(t, "file", db)

	factoryCalls := 0
	deleteCalls := 0
	var lastDeletedLocator string
	seenBackends := make([]string, 0, 3)

	adapter := NewAdapter(runtimeScope, Options{
		DriverFactory: stubStoredContentDriverFactory{newDriverFunc: func(provider string, att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
			factoryCalls += 1
			if provider != "file" {
				return nil, fmt.Errorf("unexpected provider %q", provider)
			}
			if att == nil {
				return nil, fmt.Errorf("attachment config is required")
			}
			seenBackends = append(seenBackends, att.Backend)
			return stubStoredContentDriver{
				providerName: "file",
				putFunc: func(ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
					return pkgstorage.PayloadMutation{LocatorJSON: `{"path":"company/cmp-001/files/payload.bin"}`}, nil
				},
				openFunc: func(ctx context.Context, record pkgstorage.StoredContentRecord) ([]byte, error) {
					if record.Provider != "file" {
						return nil, fmt.Errorf("unexpected record provider %q", record.Provider)
					}
					return []byte("hello-file"), nil
				},
				deleteFunc: func(ctx context.Context, record pkgstorage.StoredContentRecord) error {
					deleteCalls += 1
					lastDeletedLocator = record.LocatorJSON
					return nil
				},
			}, nil
		}},
	})

	receipt, err := adapter.Put(context.Background(), PutRequest{
		UploadID:           "up_file_adapter_001",
		PayloadWriteTicket: `{"uploadId":"up_file_adapter_001","companyId":"cmp-001","activeCompanyId":"cmp-001","userId":"usr-001","status":"prepared"}`,
		ContentType:        "application/octet-stream",
		ChecksumSHA256:     "file-checksum-001",
		Body:               []byte("hello-file"),
	})
	if err != nil {
		t.Fatalf("adapter.Put() error = %v", err)
	}
	if !strings.HasPrefix(receipt.PayloadID, "sc:") {
		t.Fatalf("receipt.PayloadID = %q, want sc:*", receipt.PayloadID)
	}

	storedContentID := strings.TrimSpace(strings.TrimPrefix(receipt.PayloadID, "sc:"))
	if storedContentID == "" {
		t.Fatal("stored content id should not be empty")
	}

	var row struct {
		Provider    string `gorm:"column:provider"`
		LocatorJSON string `gorm:"column:locator_json"`
		Status      string `gorm:"column:status"`
	}
	err = db.Table(documentStoredContentTable).
		Select("provider, locator_json, status").
		Where("id = ?", storedContentID).
		Take(&row).Error
	if err != nil {
		t.Fatalf("query stored content row: %v", err)
	}
	if row.Provider != "file" {
		t.Fatalf("provider = %q, want file", row.Provider)
	}
	if row.LocatorJSON != `{"path":"company/cmp-001/files/payload.bin"}` {
		t.Fatalf("locator_json = %q, want file locator", row.LocatorJSON)
	}
	if row.Status != "active" {
		t.Fatalf("status = %q, want active", row.Status)
	}

	opened, err := adapter.Open(context.Background(), OpenRequest{
		BindingID:         "bnd-file-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-file-001","storedContentId":"%s"}`, storedContentID),
	})
	if err != nil {
		t.Fatalf("adapter.Open() error = %v", err)
	}
	defer opened.Body.Close()

	openedBody, err := io.ReadAll(opened.Body)
	if err != nil {
		t.Fatalf("read opened body: %v", err)
	}
	if string(openedBody) != "hello-file" {
		t.Fatalf("opened body = %q, want hello-file", string(openedBody))
	}

	err = adapter.Delete(context.Background(), DeleteRequest{
		BindingID:         "bnd-file-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-file-001","storedContentId":"%s"}`, storedContentID),
	})
	if err != nil {
		t.Fatalf("adapter.Delete() error = %v", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", deleteCalls)
	}
	if lastDeletedLocator != `{"path":"company/cmp-001/files/payload.bin"}` {
		t.Fatalf("deleted locator = %q, want file locator", lastDeletedLocator)
	}
	if factoryCalls != 3 {
		t.Fatalf("factory calls = %d, want 3", factoryCalls)
	}
	for index, backend := range seenBackends {
		if backend != "file" {
			t.Fatalf("factory backend[%d] = %q, want file", index, backend)
		}
	}

	var deletedRow struct {
		Status string `gorm:"column:status"`
	}
	err = db.Table(documentStoredContentTable).
		Select("status").
		Where("id = ?", storedContentID).
		Take(&deletedRow).Error
	if err != nil {
		t.Fatalf("query deleted stored content row: %v", err)
	}
	if deletedRow.Status != "deleted" {
		t.Fatalf("status after delete = %q, want deleted", deletedRow.Status)
	}
}

func TestAdapterPutRejectsUploadIDMismatch(t *testing.T) {
	runtimeScope := newPayloadTestScope(t, "db", newPayloadTestDB(t))
	adapter := NewAdapter(runtimeScope, Options{})

	_, err := adapter.Put(context.Background(), PutRequest{
		UploadID:           "up-db-actual",
		PayloadWriteTicket: `{"uploadId":"up-db-other","companyId":"cmp-001"}`,
		Body:               []byte("hello"),
	})
	if err == nil || !strings.Contains(err.Error(), "payload write ticket upload id mismatch") {
		t.Fatalf("error = %v, want upload id mismatch", err)
	}
}

func TestAdapterDeleteDBMarksPayloadDeleted(t *testing.T) {
	db := newPayloadTestDB(t)
	runtimeScope := newPayloadTestScope(t, "db", db)
	adapter := NewAdapter(runtimeScope, Options{})

	body := []byte("delete-db")
	receipt, err := adapter.Put(context.Background(), PutRequest{
		UploadID:           "up_db_delete_001",
		PayloadWriteTicket: `{"uploadId":"up_db_delete_001","companyId":"cmp-001","activeCompanyId":"cmp-001","userId":"usr-001","status":"prepared"}`,
		ContentType:        "text/plain",
		ChecksumSHA256:     "aabbcc",
		Body:               body,
	})
	if err != nil {
		t.Fatalf("adapter.Put() error = %v", err)
	}

	storedContentID := strings.TrimSpace(strings.TrimPrefix(receipt.PayloadID, "sc:"))
	if storedContentID == "" {
		t.Fatal("stored content id should not be empty")
	}

	err = adapter.Delete(context.Background(), DeleteRequest{
		BindingID:         "bnd-delete-db-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-delete-db-001","storedContentId":"%s"}`, storedContentID),
	})
	if err != nil {
		t.Fatalf("adapter.Delete() error = %v", err)
	}

	var row struct {
		Status   string `gorm:"column:status"`
		BlobData []byte `gorm:"column:blob_data"`
	}
	err = db.Table(documentStoredContentTable).
		Select("status, blob_data").
		Where("id = ?", storedContentID).
		Take(&row).Error
	if err != nil {
		t.Fatalf("query stored content row: %v", err)
	}
	if row.Status != "deleted" {
		t.Fatalf("status = %q, want deleted", row.Status)
	}
	if len(row.BlobData) != 0 {
		t.Fatalf("blob_data len = %d, want 0", len(row.BlobData))
	}

	_, err = adapter.Open(context.Background(), OpenRequest{
		BindingID:         "bnd-delete-db-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-delete-db-001","storedContentId":"%s"}`, storedContentID),
	})
	code, ok := CodeOf(err)
	if err == nil || !ok || code != CodeNotFound {
		t.Fatalf("open after delete err = %v, code=%q, want code=%q", err, code, CodeNotFound)
	}
}

func TestAdapterDeleteS3RemovesObject(t *testing.T) {
	db := newPayloadTestDB(t)
	runtimeScope := newPayloadTestScope(t, "s3", db)

	putCalls := 0
	removeCalls := 0
	var removeBucket string
	var removeKey string

	adapter := NewAdapter(runtimeScope, Options{
		DriverFactory: stubStoredContentDriverFactory{newDriverFunc: func(provider string, att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
			if provider != "s3" {
				return nil, fmt.Errorf("unexpected provider %q", provider)
			}
			return stubStoredContentDriver{
				providerName: "s3",
				putFunc: func(ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
					putCalls += 1
					locatorJSON, err := json.Marshal(payloadTestS3Locator{Bucket: "choysum-attachments-test", Key: "staging/up_s3_delete_001/abc"})
					if err != nil {
						return pkgstorage.PayloadMutation{}, err
					}
					return pkgstorage.PayloadMutation{LocatorJSON: string(locatorJSON)}, nil
				},
				deleteFunc: func(ctx context.Context, record pkgstorage.StoredContentRecord) error {
					removeCalls += 1
					locator, err := parsePayloadTestS3Locator(record.LocatorJSON)
					if err != nil {
						return err
					}
					removeBucket = locator.Bucket
					removeKey = locator.Key
					return nil
				},
			}, nil
		}},
	})

	receipt, err := adapter.Put(context.Background(), PutRequest{
		UploadID:           "up_s3_delete_001",
		PayloadWriteTicket: `{"uploadId":"up_s3_delete_001","companyId":"cmp-001","activeCompanyId":"cmp-001","userId":"usr-001","status":"prepared"}`,
		ContentType:        "application/octet-stream",
		ChecksumSHA256:     "delete-s3-checksum",
		Body:               []byte("delete-s3"),
	})
	if err != nil {
		t.Fatalf("adapter.Put() error = %v", err)
	}
	if putCalls != 1 {
		t.Fatalf("put s3 calls = %d, want 1", putCalls)
	}

	storedContentID := strings.TrimSpace(strings.TrimPrefix(receipt.PayloadID, "sc:"))
	if storedContentID == "" {
		t.Fatal("stored content id should not be empty")
	}

	err = adapter.Delete(context.Background(), DeleteRequest{
		BindingID:         "bnd-delete-s3-001",
		PayloadReadTicket: fmt.Sprintf(`{"attachmentBindingId":"bnd-delete-s3-001","storedContentId":"%s"}`, storedContentID),
	})
	if err != nil {
		t.Fatalf("adapter.Delete() error = %v", err)
	}
	if removeCalls != 1 {
		t.Fatalf("remove s3 calls = %d, want 1", removeCalls)
	}
	if removeBucket != "choysum-attachments-test" {
		t.Fatalf("remove bucket = %q, want choysum-attachments-test", removeBucket)
	}
	if removeKey != "staging/up_s3_delete_001/abc" {
		t.Fatalf("remove key = %q, want staging/up_s3_delete_001/abc", removeKey)
	}
}

func TestAdapterDeleteRejectsBindingMismatch(t *testing.T) {
	runtimeScope := newPayloadTestScope(t, "db", newPayloadTestDB(t))
	adapter := NewAdapter(runtimeScope, Options{})

	err := adapter.Delete(context.Background(), DeleteRequest{
		BindingID:         "bnd-delete-mismatch-actual",
		PayloadReadTicket: `{"attachmentBindingId":"bnd-delete-mismatch-ticket","storedContentId":"sc_001"}`,
	})
	code, ok := CodeOf(err)
	if err == nil || !ok || code != CodeFailedPrecondition {
		t.Fatalf("error = %v, code=%q, want code=%q", err, code, CodeFailedPrecondition)
	}
}

func TestAdapterPromoteAcceptsOpaquePayloadHandles(t *testing.T) {
	runtimeScope := newPayloadTestScope(t, "db", newPayloadTestDB(t))
	adapter := NewAdapter(runtimeScope, Options{})

	for _, payloadID := range []string{"sc:stored_001"} {
		receipt, err := adapter.Promote(context.Background(), PromoteRequest{PayloadID: payloadID})
		if err != nil {
			t.Fatalf("Promote(%q) error = %v", payloadID, err)
		}
		if receipt.PayloadID != payloadID {
			t.Fatalf("Promote(%q) payloadID = %q, want %q", payloadID, receipt.PayloadID, payloadID)
		}
	}
}

func TestAdapterPromoteRejectsLegacyS3PayloadHandles(t *testing.T) {
	runtimeScope := newPayloadTestScope(t, "db", newPayloadTestDB(t))
	adapter := NewAdapter(runtimeScope, Options{})

	for _, payloadID := range []string{
		"s3://choysum-attachments-test/staging/up_001/abc",
		"s3:staging/up_001/abc",
	} {
		_, err := adapter.Promote(context.Background(), PromoteRequest{PayloadID: payloadID})
		code, ok := CodeOf(err)
		if err == nil || !ok || code != CodeInvalidArgument {
			t.Fatalf("Promote(%q) error = %v, code=%q, want code=%q", payloadID, err, code, CodeInvalidArgument)
		}
	}
}

func TestAdapterPromoteRejectsInlinePayloadHandle(t *testing.T) {
	runtimeScope := newPayloadTestScope(t, "db", newPayloadTestDB(t))
	adapter := NewAdapter(runtimeScope, Options{})

	_, err := adapter.Promote(context.Background(), PromoteRequest{PayloadID: "inline_base64:AAECAw=="})
	code, ok := CodeOf(err)
	if err == nil || !ok || code != CodeInvalidArgument {
		t.Fatalf("error = %v, code=%q, want code=%q", err, code, CodeInvalidArgument)
	}
}
