// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package db_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"testing"
	"time"

	documentdb "github.com/choysum-dev/choysum/internal/document/storage/db"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type repositoryTestScope struct {
	session *scope.Session
}

func (s *repositoryTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *repositoryTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *repositoryTestScope) Session() *scope.Session { return s.session }
func (s *repositoryTestScope) WithContext(context.Context) scope.Scope {
	return &repositoryTestScope{session: s.session}
}
func (s *repositoryTestScope) Context() context.Context { return context.Background() }
func (s *repositoryTestScope) Logger() *slog.Logger     { return nil }
func (s *repositoryTestScope) Config() *config.Config   { return nil }
func (s *repositoryTestScope) FactoryInput() scope.FactoryInput {
	return nil
}

func TestStoredContentRepositoryDBBlobRoundTrip(t *testing.T) {
	repo := documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: newStoredContentTestDB(t)})
	now := time.Now().UTC().Truncate(time.Second)

	err := repo.Create(context.Background(), pkgstorage.CreateStoredContentInput{
		ID:        "stored-db-001",
		Provider:  "db",
		BlobData:  []byte("hello-db"),
		Status:    "active",
		CompanyID: "cmp-001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	record, err := repo.GetByID(context.Background(), "stored-db-001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if record.Provider != "db" {
		t.Fatalf("record.Provider = %q, want db", record.Provider)
	}
	if string(record.BlobData) != "hello-db" {
		t.Fatalf("record.BlobData = %q, want hello-db", string(record.BlobData))
	}
	if record.LocatorJSON != "" {
		t.Fatalf("record.LocatorJSON = %q, want empty", record.LocatorJSON)
	}
}

func TestStoredContentRepositoryS3LocatorRoundTrip(t *testing.T) {
	db := newStoredContentTestDB(t)
	repo := documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: db})
	locatorJSON := `{"bucket":"choysum-attachments-test","key":"staging/up-001/abc"}`
	now := time.Now().UTC().Truncate(time.Second)

	err := repo.Create(context.Background(), pkgstorage.CreateStoredContentInput{
		ID:          "stored-s3-001",
		Provider:    "s3",
		LocatorJSON: locatorJSON,
		Status:      "active",
		CompanyID:   "cmp-001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var row struct {
		LocatorJSON string `gorm:"column:locator_json"`
	}
	if err := db.Table(pkgstorage.DocumentStoredContentTable).
		Select("locator_json").
		Where("id = ?", "stored-s3-001").
		Take(&row).Error; err != nil {
		t.Fatalf("query stored row: %v", err)
	}
	assertJSONEqual(t, row.LocatorJSON, locatorJSON)

	record, err := repo.GetByID(context.Background(), "stored-s3-001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	assertJSONEqual(t, record.LocatorJSON, locatorJSON)
	if len(record.BlobData) != 0 {
		t.Fatalf("record.BlobData = %q, want empty", string(record.BlobData))
	}
}

func TestStoredContentRepositoryGenericLocatorRoundTrip(t *testing.T) {
	db := newStoredContentTestDB(t)
	repo := documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: db})
	locatorJSON := `{"path":"company/cmp-001/files/a.bin"}`
	now := time.Now().UTC().Truncate(time.Second)

	err := repo.Create(context.Background(), pkgstorage.CreateStoredContentInput{
		ID:          "stored-file-001",
		Provider:    "file",
		LocatorJSON: locatorJSON,
		Status:      "active",
		CompanyID:   "cmp-001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var row struct {
		LocatorJSON string `gorm:"column:locator_json"`
	}
	if err := db.Table(pkgstorage.DocumentStoredContentTable).
		Select("locator_json").
		Where("id = ?", "stored-file-001").
		Take(&row).Error; err != nil {
		t.Fatalf("query stored row: %v", err)
	}
	assertJSONEqual(t, row.LocatorJSON, locatorJSON)

	record, err := repo.GetByID(context.Background(), "stored-file-001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	assertJSONEqual(t, record.LocatorJSON, locatorJSON)
	if record.Provider != "file" {
		t.Fatalf("record.Provider = %q, want file", record.Provider)
	}
}

func TestStoredContentRepositoryMarkDeletedClearsBlobData(t *testing.T) {
	db := newStoredContentTestDB(t)
	repo := documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: db})
	now := time.Now().UTC().Truncate(time.Second)

	err := repo.Create(context.Background(), pkgstorage.CreateStoredContentInput{
		ID:        "stored-delete-001",
		Provider:  "db",
		BlobData:  []byte("delete-me"),
		Status:    "active",
		CompanyID: "cmp-001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	rowsAffected, err := repo.MarkDeleted(context.Background(), "stored-delete-001", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("MarkDeleted() error = %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("rowsAffected = %d, want 1", rowsAffected)
	}

	record, err := repo.GetByID(context.Background(), "stored-delete-001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if record.Status != "deleted" {
		t.Fatalf("record.Status = %q, want deleted", record.Status)
	}
	if len(record.BlobData) != 0 {
		t.Fatalf("record.BlobData = %q, want empty", string(record.BlobData))
	}
}

func TestStoredContentRepositoryUsesContextScopeBeforeBaseDB(t *testing.T) {
	baseDB := newStoredContentTestDB(t)
	ctxDB := newStoredContentTestDB(t)
	repo := documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: baseDB})
	now := time.Now().UTC().Truncate(time.Second)
	ctx := scope.ContextWithScope(context.Background(), &repositoryTestScope{session: &scope.Session{DB: ctxDB}})

	err := repo.Create(ctx, pkgstorage.CreateStoredContentInput{
		ID:        "stored-ctx-001",
		Provider:  "db",
		BlobData:  []byte("hello-ctx"),
		Status:    "active",
		CompanyID: "cmp-ctx",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var ctxCount int64
	if err := ctxDB.Table(pkgstorage.DocumentStoredContentTable).Where("id = ?", "stored-ctx-001").Count(&ctxCount).Error; err != nil {
		t.Fatalf("count ctx rows: %v", err)
	}
	if ctxCount != 1 {
		t.Fatalf("ctx row count = %d, want 1", ctxCount)
	}

	var baseCount int64
	if err := baseDB.Table(pkgstorage.DocumentStoredContentTable).Where("id = ?", "stored-ctx-001").Count(&baseCount).Error; err != nil {
		t.Fatalf("count base rows: %v", err)
	}
	if baseCount != 0 {
		t.Fatalf("base row count = %d, want 0", baseCount)
	}

	record, err := repo.GetByID(ctx, "stored-ctx-001")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if string(record.BlobData) != "hello-ctx" {
		t.Fatalf("record.BlobData = %q, want hello-ctx", string(record.BlobData))
	}

	rowsAffected, err := repo.MarkDeleted(ctx, "stored-ctx-001", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("MarkDeleted() error = %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("rowsAffected = %d, want 1", rowsAffected)
	}

	record, err = repo.GetByID(ctx, "stored-ctx-001")
	if err != nil {
		t.Fatalf("GetByID(after delete) error = %v", err)
	}
	if record.Status != "deleted" {
		t.Fatalf("record.Status = %q, want deleted", record.Status)
	}
	if len(record.BlobData) != 0 {
		t.Fatalf("record.BlobData = %q, want empty", string(record.BlobData))
	}
}

func TestStoredContentRepositoryRejectsInvalidS3LocatorJSON(t *testing.T) {
	repo := documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: newStoredContentTestDB(t)})
	now := time.Now().UTC().Truncate(time.Second)

	err := repo.Create(context.Background(), pkgstorage.CreateStoredContentInput{
		ID:          "stored-invalid-s3-001",
		Provider:    "s3",
		LocatorJSON: `{"bucket":"choysum-attachments-test"}`,
		Status:      "active",
		CompanyID:   "cmp-001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want invalid s3 locator error")
	}
}

func newStoredContentTestDB(t *testing.T) *gorm.DB {
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

func assertJSONEqual(t *testing.T, actual string, expected string) {
	t.Helper()

	var actualValue any
	if err := json.Unmarshal([]byte(actual), &actualValue); err != nil {
		t.Fatalf("parse actual json: %v", err)
	}
	var expectedValue any
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		t.Fatalf("parse expected json: %v", err)
	}

	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("json = %#v, want %#v", actualValue, expectedValue)
	}
}
