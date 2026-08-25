// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact_test

import (
	"context"
	"encoding/csv"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/import/artifact"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestWriteErrorArtifact_RoundTrip(t *testing.T) {
	runtimeScope := newArtifactTestScope(t)
	companyID := "cmp-artifact-1"
	messages := []importpkg.Message{
		{Type: importpkg.MessageError, Row: 42, Field: "email", Code: "constraint", Text: "duplicate email"},
		{Type: importpkg.MessageSkip, Row: 7, Code: "empty_required", Text: "name is required"},
	}

	report := importpkg.Report{Messages: messages}
	if err := artifact.WriteErrorArtifact(context.Background(), runtimeScope, companyID, &report); err != nil {
		t.Fatalf("WriteErrorArtifact: %v", err)
	}
	if report.ArtifactRef == "" {
		t.Fatal("expected artifact_ref to be set")
	}

	var row struct {
		BlobData []byte `gorm:"column:blob_data"`
	}
	err := runtimeScope.Session().DB.WithContext(context.Background()).
		Table("document_stored_content").
		Select("document_stored_content.blob_data").
		Joins("JOIN document_attachment_content ON document_attachment_content.stored_content_id = document_stored_content.id").
		Where("document_attachment_content.id = ?", report.ArtifactRef).
		Take(&row).Error
	if err != nil {
		t.Fatalf("load stored artifact bytes: %v", err)
	}
	raw := row.BlobData

	reader := csv.NewReader(strings.NewReader(string(raw)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("csv rows = %d, want 3 (header + 2 messages)", len(records))
	}
	if records[0][0] != "type" || records[0][5] != "record_ref" {
		t.Fatalf("unexpected header: %#v", records[0])
	}
	if records[1][0] != "error" || records[1][1] != "42" || records[1][2] != "email" {
		t.Fatalf("unexpected error row: %#v", records[1])
	}
	if records[2][0] != "skip" || records[2][1] != "7" {
		t.Fatalf("unexpected skip row: %#v", records[2])
	}
}

func TestWriteErrorArtifact_SkipsWhenNoMessages(t *testing.T) {
	report := importpkg.Report{}
	if err := artifact.WriteErrorArtifact(context.Background(), nil, "cmp-1", &report); err != nil {
		t.Fatalf("WriteErrorArtifact: %v", err)
	}
	if report.ArtifactRef != "" {
		t.Fatalf("artifact_ref = %q, want empty", report.ArtifactRef)
	}
}

func newArtifactTestScope(t *testing.T) scope.Scope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "artifact.db")
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
