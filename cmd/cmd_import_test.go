// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	_ "github.com/choysum-dev/choysum/internal/import/adapter/stub"
	"github.com/choysum-dev/choysum/internal/import/registry"
	_ "github.com/choysum-dev/choysum/internal/import/runner"
	stubwriter "github.com/choysum-dev/choysum/internal/import/writer/stub"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileRecord, stubwriter.Writer{})
}

func TestCLI_BestEffort_PartialOk(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	envGetter := func() scope.Scope { return runtimeScope }

	cmd := newImportRecordCmd(envGetter)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--model", "base.Country",
		"--source", filepath.Join(t.TempDir(), "unused.csv"),
		"--format", "stub",
		"--policy", "best_effort",
		"--company-id", "cmp-cli-1",
		"--stub-unit-count", "3",
		"--stub-fail-unit-index", "2",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var report importpkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\nstdout=%q", err, stdout.String())
	}
	if report.Stats.Ok != 2 || report.Stats.Error != 1 {
		t.Fatalf("unexpected stats: %+v", report.Stats)
	}
	if report.Policy != importpkg.PolicyBestEffort {
		t.Fatalf("policy = %q, want best_effort", report.Policy)
	}

	var count int64
	if err := runtimeScope.Session().Model(&stubwriter.Row{}).Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Fatalf("row count = %d, want 2", count)
	}
}

func TestParseImportPolicy_AcceptsKnownPolicies(t *testing.T) {
	for _, policy := range []string{"atomic", "stop_keep", "best_effort"} {
		if _, err := parseImportPolicy(policy); err != nil {
			t.Fatalf("parseImportPolicy(%q): %v", policy, err)
		}
	}
}

func TestParseImportPolicy_RejectsUnknown(t *testing.T) {
	if _, err := parseImportPolicy("bogus"); err == nil {
		t.Fatal("expected error")
	}
}

func newImportCLITestScope(t *testing.T) scope.Scope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "import-cli.db")
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
	if err := runtimeScope.Session().AutoMigrate(&stubwriter.Row{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	ensureImportCLIDocumentTables(t, runtimeScope.Session().DB)
	return runtimeScope
}

func ensureImportCLIDocumentTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS document_stored_content (
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
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS document_attachment_content (
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
}
