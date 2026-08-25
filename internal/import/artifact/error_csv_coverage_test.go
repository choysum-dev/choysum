// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	documentdb "github.com/choysum-dev/choysum/internal/document/storage/db"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
)

func TestWriteErrorArtifact_NilReport(t *testing.T) {
	if err := WriteErrorArtifact(context.Background(), nil, "cmp-1", nil); err != nil {
		t.Fatalf("WriteErrorArtifact(nil report): %v", err)
	}
}

func TestWriteErrorArtifact_RequiresCompanyID(t *testing.T) {
	report := importpkg.Report{
		Messages: []importpkg.Message{{Type: importpkg.MessageError, Text: "boom"}},
	}
	err := WriteErrorArtifact(context.Background(), nil, " ", &report)
	if err == nil {
		t.Fatal("expected company id error")
	}
}

func TestWriteErrorArtifact_EncodeFailure(t *testing.T) {
	orig := messagesToCSVBytes
	t.Cleanup(func() { messagesToCSVBytes = orig })
	messagesToCSVBytes = func([]importpkg.Message) ([]byte, error) {
		return nil, errors.New("encode failed")
	}

	report := importpkg.Report{
		Messages: []importpkg.Message{{Type: importpkg.MessageError, Text: "boom"}},
	}
	err := WriteErrorArtifact(context.Background(), nil, "cmp-1", &report)
	if err == nil || report.ArtifactRef != "" {
		t.Fatalf("err=%v artifact_ref=%q", err, report.ArtifactRef)
	}
}

func TestDefaultStoreArtifact_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	if _, err := defaultStoreArtifact(ctx, nil, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected nil scope error")
	}

	runtimeScope := defaultscope.NewDefaultScope(
		ctx,
		scopetest.FactoryInputFromConfig(&config.Config{
			Db: &config.DbConfig{
				Dialect: "sqlite",
				DSN:     filepath.Join(t.TempDir(), "artifact-errors.db"),
			},
		}),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if _, err := defaultStoreArtifact(ctx, runtimeScope, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected missing tables error")
	}

	scopeWithTables := newArtifactCoverageScope(t)
	if err := scopeWithTables.Session().DB.Exec(`DROP TABLE document_attachment_content`).Error; err != nil {
		t.Fatalf("drop attachment table: %v", err)
	}
	if _, err := defaultStoreArtifact(ctx, scopeWithTables, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected attachment create error")
	}

	if err := scopeWithTables.Session().DB.Exec(`DROP TABLE document_stored_content`).Error; err != nil {
		t.Fatalf("drop stored content table: %v", err)
	}
	if err := scopeWithTables.Session().DB.Exec(`CREATE TABLE document_attachment_content (
		id TEXT PRIMARY KEY,
		stored_content_id TEXT NOT NULL,
		company_id TEXT NOT NULL,
		status TEXT NOT NULL,
		mime_type TEXT,
		size_bytes INTEGER,
		checksum_sha256 TEXT
	)`).Error; err != nil {
		t.Fatalf("recreate attachment table: %v", err)
	}
	if _, err := defaultStoreArtifact(ctx, scopeWithTables, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected stored content create error")
	}
}

func TestWriteErrorArtifact_StoreFailure(t *testing.T) {
	orig := storeArtifact
	t.Cleanup(func() { storeArtifact = orig })
	storeArtifact = func(context.Context, scope.Scope, string, []byte, string) (string, error) {
		return "", errors.New("store failed")
	}

	report := importpkg.Report{
		Messages: []importpkg.Message{{Type: importpkg.MessageError, Text: "boom"}},
	}
	err := WriteErrorArtifact(context.Background(), nil, "cmp-1", &report)
	if err == nil {
		t.Fatal("expected store error")
	}
}

func TestDefaultStoreArtifact_DriverAndPutErrors(t *testing.T) {
	ctx := context.Background()
	scopeWithTables := newArtifactCoverageScope(t)

	origDriver := newStoredContentDriver
	t.Cleanup(func() { newStoredContentDriver = origDriver })
	newStoredContentDriver = func(*config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
		return nil, errors.New("driver failed")
	}
	if _, err := defaultStoreArtifact(ctx, scopeWithTables, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected driver error")
	}

	newStoredContentDriver = origDriver
	origPut := putStoredContent
	t.Cleanup(func() { putStoredContent = origPut })
	putStoredContent = func(pkgstorage.StoredContentDriver, context.Context, pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
		return pkgstorage.PayloadMutation{}, errors.New("put failed")
	}
	if _, err := defaultStoreArtifact(ctx, scopeWithTables, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected put error")
	}

	putStoredContent = origPut
	origRepo := newStoredContentRepository
	t.Cleanup(func() { newStoredContentRepository = origRepo })
	newStoredContentRepository = func(documentdb.RepositoryDeps) pkgstorage.StoredContentRepository {
		return nil
	}
	if _, err := defaultStoreArtifact(ctx, scopeWithTables, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected repository unavailable error")
	}
}

func TestDefaultStoreArtifact_RequiresSession(t *testing.T) {
	if _, err := defaultStoreArtifact(context.Background(), noSessionArtifactScope{}, "cmp-1", []byte("x"), errorCSVMimeType); err == nil {
		t.Fatal("expected database session error")
	}
}

type noSessionArtifactScope struct{}

func (noSessionArtifactScope) Run(func(scope.Scope) error) error { return nil }
func (noSessionArtifactScope) Transactor() scope.Transactor      { return nil }
func (noSessionArtifactScope) Session() *scope.Session           { return nil }
func (noSessionArtifactScope) WithContext(context.Context) scope.Scope {
	return noSessionArtifactScope{}
}
func (noSessionArtifactScope) Context() context.Context { return context.Background() }
func (noSessionArtifactScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (noSessionArtifactScope) Config() *config.Config { return nil }
func (noSessionArtifactScope) FactoryInput() scope.FactoryInput { return nil }

func TestDefaultMessagesToCSVBytes_HeaderWriteError(t *testing.T) {
	orig := writeCSVRecord
	t.Cleanup(func() { writeCSVRecord = orig })
	writeCSVRecord = func(*csv.Writer, []string) error {
		return errors.New("header write failed")
	}
	if _, err := defaultMessagesToCSVBytes([]importpkg.Message{{Type: importpkg.MessageError, Text: "boom"}}); err == nil {
		t.Fatal("expected header write error")
	}
}

func TestDefaultMessagesToCSVBytes_RowWriteError(t *testing.T) {
	orig := writeCSVRecord
	t.Cleanup(func() { writeCSVRecord = orig })
	calls := 0
	writeCSVRecord = func(*csv.Writer, []string) error {
		calls++
		if calls > 1 {
			return errors.New("row write failed")
		}
		return nil
	}
	if _, err := defaultMessagesToCSVBytes([]importpkg.Message{{Type: importpkg.MessageError, Text: "boom"}}); err == nil {
		t.Fatal("expected row write error")
	}
}

func TestDefaultMessagesToCSVBytes_WriteError(t *testing.T) {
	orig := ioWriterForErrorCSV
	t.Cleanup(func() { ioWriterForErrorCSV = orig })
	ioWriterForErrorCSV = func(io.Writer) io.Writer {
		return errWriter{err: errors.New("csv write failed")}
	}
	if _, err := defaultMessagesToCSVBytes([]importpkg.Message{{Type: importpkg.MessageError, Text: "boom"}}); err == nil {
		t.Fatal("expected csv encode error")
	}
}

type errWriter struct {
	err error
}

func (e errWriter) Write([]byte) (int, error) {
	return 0, e.err
}

func TestDefaultMessagesToCSVBytes_RoundTrip(t *testing.T) {
	raw, err := defaultMessagesToCSVBytes([]importpkg.Message{
		{Type: importpkg.MessageError, Row: 1, Text: "boom"},
	})
	if err != nil {
		t.Fatalf("defaultMessagesToCSVBytes: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected csv bytes")
	}
}

func newArtifactCoverageScope(t *testing.T) scope.Scope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "artifact-coverage.db")
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     dbPath,
		},
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	db := runtimeScope.Session().DB
	if err := db.Exec(`CREATE TABLE document_stored_content (
		id TEXT PRIMARY KEY,
		provider TEXT,
		locator_json TEXT,
		blob_data BLOB,
		status TEXT,
		company_id TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create stored content table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE document_attachment_content (
		id TEXT PRIMARY KEY,
		stored_content_id TEXT NOT NULL,
		company_id TEXT NOT NULL,
		status TEXT NOT NULL,
		mime_type TEXT,
		size_bytes INTEGER,
		checksum_sha256 TEXT
	)`).Error; err != nil {
		t.Fatalf("create attachment content table: %v", err)
	}
	return runtimeScope
}
